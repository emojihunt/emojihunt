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
	"github.com/gauravjsingh/emojihunt/server"
	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"
)

type Poller struct {
	cookie    *http.Cookie
	airtable  *client.Airtable
	discord   *client.Discord
	server    *server.Server
	newRounds map[string]time.Time
	wsLimiter *rate.Limiter

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
	newPuzzleLimit       = 10
	websocketBurst       = 3
)

var websocketRate = rate.Every(1 * time.Minute)

func New(cookieName, cookieValue string, airtable *client.Airtable, discord *client.Discord, server *server.Server) *Poller {
	return &Poller{
		cookie: &http.Cookie{
			Name:   cookieName,
			Value:  cookieValue,
			MaxAge: 0,
		},
		airtable:  airtable,
		discord:   discord,
		server:    server,
		newRounds: make(map[string]time.Time),
		wsLimiter: rate.NewLimiter(websocketRate, websocketBurst),
		enabled:   true,
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
			log.Printf("discovery: scraping error: %v", err)
			msg := fmt.Sprintf("discovery: scraping error: ```\n%s\n```", spew.Sdump(err))
			d.discord.ChannelSend(d.discord.TechChannelID, msg)
		}

		if err := d.SyncPuzzles(puzzles); err != nil {
			log.Printf("discovery: syncing error: %v", err)
			msg := fmt.Sprintf("discovery: syncing error: ```\n%s\n```", spew.Sdump(err))
			d.discord.ChannelSend(d.discord.TechChannelID, msg)
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
