package huntbot

import (
	"context"
	"fmt"
	"log"
	"time"
)

const pollInterval = 10 * time.Second

func (h *HuntBot) PollDatabase(ctx context.Context) {
	// Airtable doesn't support webhooks, so we have to poll the database for
	// updates.
	failures := 0
	for {
		if h.isEnabled() {
			puzzles, err := h.airtable.ListRecords()
			if err != nil {
				// Log errors always, but ping after 3 consecutive failures,
				// then every 10, to avoid spam
				log.Printf("polling sheet failed: %v", err)
				failures++
				if failures%10 == 3 {
					h.discord.TechChannelSend(fmt.Sprintf("polling sheet failed: %v", err))
				}
			} else {
				failures = 0
			}

			h.mu.Lock()
			h.channelToPuzzle = make(map[string]string)
			for _, puzzle := range puzzles {
				h.channelToPuzzle[puzzle.DiscordChannel] = puzzle.Name
			}
			h.mu.Unlock()

			for _, puzzle := range puzzles {
				err := h.updatePuzzle(ctx, &puzzle)
				if err != nil {
					// Log errors and keep going.
					log.Printf("updating puzzle failed: %v", err)
				}
			}
		} else {
			log.Printf("bot disabled, skipping update")
		}

		select {
		case <-ctx.Done():
			log.Print("exiting watcher due to signal")
			return
		case <-time.After(pollInterval):
		}
	}
}

func (h *HuntBot) isEnabled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.enabled
}
