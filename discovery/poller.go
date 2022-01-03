package discovery

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
)

type Poller struct {
	cookie    *http.Cookie
	airtable  *client.Airtable
	discord   *client.Discord
	newRounds map[string]time.Time

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
)

func New(cookieName, cookieValue string, airtable *client.Airtable, discord *client.Discord) *Poller {
	return &Poller{
		cookie: &http.Cookie{
			Name:   cookieName,
			Value:  cookieValue,
			MaxAge: 0,
		},
		airtable:  airtable,
		discord:   discord,
		newRounds: make(map[string]time.Time),
		enabled:   true,
	}
}

func (d *Poller) Poll(ctx context.Context) {
	for {
		if !d.isEnabled() {
			time.Sleep(2 * time.Second)
			continue
		}

		puzzles, err := d.Scrape()
		if err != nil {
			log.Printf("discovery: scraping error: %v", err)
			msg := fmt.Sprintf("discovery: scraping error: ```\n%s\n```", spew.Sdump(err))
			d.discord.TechChannelSend(msg)
		}

		if err := d.SyncPuzzles(puzzles); err != nil {
			log.Printf("discovery: syncing error: %v", err)
			msg := fmt.Sprintf("discovery: syncing error: ```\n%s\n```", spew.Sdump(err))
			d.discord.TechChannelSend(msg)
		}

		select {
		case <-ctx.Done():
			log.Print("exiting discovery poller due to signal")
			return
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
