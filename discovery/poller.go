package discovery

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
)

type Discovery struct {
	cookie    *http.Cookie
	airtable  *client.Airtable
	discord   *client.Discord
	newRounds map[string]time.Time
}

type DiscoveredPuzzle struct {
	Name  string
	URL   *url.URL
	Round string
}

const (
	pollInterval         = 1 * time.Minute
	roundNotifyFrequency = 10 * time.Minute
)

func New(cookieName, cookieValue string, airtable *client.Airtable, discord *client.Discord) *Discovery {
	return &Discovery{
		cookie: &http.Cookie{
			Name:   cookieName,
			Value:  cookieValue,
			MaxAge: 0,
		},
		airtable:  airtable,
		discord:   discord,
		newRounds: make(map[string]time.Time),
	}
}

func (d *Discovery) Poll(ctx context.Context) {
	for {
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
