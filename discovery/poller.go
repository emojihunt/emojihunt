package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"
)

type DiscoveryConfig struct {
	CookieName  string `json:"cookie_name"`
	CookieValue string `json:"cookie_value"`
	AuthToken   string `json:"auth_token"`
}

type Poller struct {
	cookie    *http.Cookie
	token     string
	airtable  *client.Airtable
	discord   *client.Discord
	syncer    *syncer.Syncer
	state     *state.State
	wsLimiter *rate.Limiter
}

type DiscoveredPuzzle struct {
	Name  string
	URL   *url.URL
	Round string
}

const (
	pollInterval         = 20 * time.Second
	roundNotifyFrequency = 10 * time.Minute
	warnErrorFrequency   = 10 * time.Minute
	newPuzzleLimit       = 10
	websocketBurst       = 3
)

var websocketRate = rate.Every(1 * time.Minute)

func New(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer, config *DiscoveryConfig, state *state.State) *Poller {
	return &Poller{
		cookie: &http.Cookie{
			Name:   config.CookieName,
			Value:  config.CookieValue,
			MaxAge: 0,
		},
		token:     config.AuthToken,
		airtable:  airtable,
		discord:   discord,
		syncer:    syncer,
		state:     state,
		wsLimiter: rate.NewLimiter(websocketRate, websocketBurst),
	}
}

func (d *Poller) Poll(ctx context.Context) {
	for {
		ch, err := d.openWebsocket()
		if err != nil {
			log.Printf("discovery: failed to open websocket: %v", spew.Sprint(err))
		}

		for {
			if !d.isEnabled() {
				time.Sleep(2 * time.Second)
				continue
			}

			puzzles, err := d.Scrape()
			if err != nil {
				d.logAndMaybeWarn("scraping error", err)
			}

			if err := d.SyncPuzzles(puzzles); err != nil {
				d.logAndMaybeWarn("syncing error", err)
			}

			select {
			case <-ctx.Done():
				log.Print("exiting discovery poller due to signal")
				return
			case _, more := <-ch:
				if !more {
					break
				}
			case <-time.After(pollInterval):
			}
		}
	}
}

func (d *Poller) isEnabled() bool {
	d.state.Lock()
	defer d.state.Unlock()
	return !d.state.DiscoveryDisabled
}

func (d *Poller) logAndMaybeWarn(memo string, err error) {
	d.state.Lock()
	defer d.state.CommitAndUnlock()

	log.Printf("discovery: %s: %v", memo, err)
	if time.Since(d.state.DiscoveryLastWarn) >= warnErrorFrequency {
		msg := fmt.Sprintf("discovery: %s: ```\n%s\n```", memo, spew.Sdump(err))
		d.discord.ChannelSend(d.discord.TechChannel, msg)
		d.state.DiscoveryLastWarn = time.Now()
	}
}

func (d *Poller) openWebsocket() (chan bool, error) {
	if websocketURL == nil {
		return nil, nil
	}

	log.Printf("discovery: (re-)connecting to websocket...")
	ch := make(chan bool)
	ws, err := websocket.Dial(websocketURL.String(), "", websocketOrigin)
	if err != nil {
		return nil, err
	}
	log.Printf("discovery: opened websocket connection to %q", websocketURL.String())
	data, err := json.Marshal(map[string]interface{}{
		"type": "AUTH",
		"data": d.token,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("discovery: wrote AUTH message to websocket")
	if _, err := ws.Write(data); err != nil {
		return nil, err
	}
	go func(ws *websocket.Conn, ch chan bool) {
		defer close(ch)
		scanner := bufio.NewScanner(ws)
		for scanner.Scan() {
			if d.wsLimiter.Allow() {
				log.Printf("discovery: ws: %q", scanner.Text())
				ch <- true
			} else {
				log.Printf("discovery: ws (skipped due to rate limit): %q", scanner.Text())
			}
		}
		log.Printf("discovery: closing ws channel")
	}(ws, ch)
	return ch, nil
}
