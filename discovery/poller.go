package discovery

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/syncer"
	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"
)

type DiscoveryConfig struct {
	CookieName  string `json:"cookie_name"`
	CookieValue string `json:"cookie_value"`
}

type Poller struct {
	cookie        *http.Cookie
	airtable      *client.Airtable
	discord       *client.Discord
	syncer        *syncer.Syncer
	wsLimiter     *rate.Limiter
	newRounds     map[string]time.Time
	lastWarnError time.Time

	mu      sync.Mutex // hold while accessing everything below
	enabled bool
}

type DiscoveredPuzzle struct {
	Name  string
	URL   *url.URL
	Round string
}

const (
	pollInterval         = 1 * time.Minute
	roundNotifyFrequency = 10 * time.Minute
	warnErrorFrequency   = 10 * time.Minute
	newPuzzleLimit       = 10
	websocketBurst       = 3
)

var websocketRate = rate.Every(1 * time.Minute)

func New(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer, config *DiscoveryConfig) *Poller {
	return &Poller{
		cookie: &http.Cookie{
			Name:   config.CookieName,
			Value:  config.CookieValue,
			MaxAge: 0,
		},
		airtable:      airtable,
		discord:       discord,
		syncer:        syncer,
		wsLimiter:     rate.NewLimiter(websocketRate, websocketBurst),
		newRounds:     make(map[string]time.Time),
		lastWarnError: time.Now().Add(-24 * time.Hour),
		enabled:       true,
	}
}

func (d *Poller) Poll(ctx context.Context) {
	ch, err := d.openWebsocket()
	if err != nil {
		log.Printf("discovery: failed to open websocket: %v", err)
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
		case <-ch:
		case <-time.After(pollInterval):
		}
	}
}

func (d *Poller) Enable(enable bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enable
}

func (d *Poller) isEnabled() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.enabled
}

func (d *Poller) logAndMaybeWarn(memo string, err error) {
	log.Printf("discovery: %s: %v", memo, err)
	if time.Since(d.lastWarnError) >= warnErrorFrequency {
		msg := fmt.Sprintf("discovery: %s: ```\n%s\n```", memo, spew.Sdump(err))
		d.discord.ChannelSend(d.discord.TechChannel, msg)
		d.lastWarnError = time.Now()
	}
}

func (d *Poller) openWebsocket() (chan bool, error) {
	if websocketURL == nil {
		return nil, nil
	}

	ch := make(chan bool)
	ws, err := websocket.Dial(websocketURL.String(), "", websocketOrigin)
	if err != nil {
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
	}(ws, ch)
	return ch, nil
}
