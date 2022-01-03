package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/syncer"
)

const (
	PollInterval        = 10 * time.Second
	InitialWarningDelay = 1 * time.Minute
	MinWarningFrequency = 10 * time.Minute
)

type Poller struct {
	airtable *client.Airtable
	discord  *client.Discord
	syncer   *syncer.Syncer

	mu           sync.Mutex           // hold while accessing everything below
	enabled      bool                 // global killswitch, toggle with !huntbot kill/!huntbot start
	lastWarnTime map[string]time.Time // airtable id -> when we last warned about a malformed puzzle
}

func NewPoller(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer) *Poller {
	return &Poller{
		airtable:     airtable,
		discord:      discord,
		syncer:       syncer,
		enabled:      true,
		lastWarnTime: map[string]time.Time{},
	}
}

func (p *Poller) Enable(enable bool) (changed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.enabled == enable {
		return false
	} else {
		p.enabled = enable
		return true
	}
}

func (p *Poller) isEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled
}

func (p *Poller) Poll(ctx context.Context) {
	// Airtable doesn't support webhooks, so we have to poll the database for
	// updates.
	failures := 0
	for {
		if p.isEnabled() {
			puzzles, err := p.airtable.ListRecords()
			if err != nil {
				// Log errors always, but ping after 3 consecutive failures,
				// then every 10, to avoid spam
				log.Printf("polling sheet failed: %v", err)
				failures++
				if failures%10 == 3 {
					p.discord.TechChannelSend(fmt.Sprintf("polling sheet failed: %v", err))
				}
			} else {
				failures = 0
			}

			for _, puzzle := range puzzles {
				if puzzle.Pending {
					// Skip auto-added records that haven't been confirmed by a
					// human
					continue
				}
				err := p.processPuzzle(ctx, &puzzle)
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
			log.Print("exiting database poller due to signal")
			return
		case <-time.After(PollInterval):
		}
	}
}

func (p *Poller) processPuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
	if !puzzle.IsValid() {
		// Occasionally warn the QM about puzzles that are missing fields.
		if puzzle.Name != "" {
			if err := p.warnPuzzle(ctx, puzzle); err != nil {
				return fmt.Errorf("error warning about malformed puzzle %q: %v", puzzle.Name, err)
			}
		}
		return nil
	}

	_, err := p.syncer.IdempotentCreateUpdate(ctx, puzzle)
	return err
}

func (p *Poller) warnPuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if lastWarning, ok := p.lastWarnTime[puzzle.AirtableRecord.ID]; !ok {
		p.lastWarnTime[puzzle.AirtableRecord.ID] = time.Now().Add(InitialWarningDelay - MinWarningFrequency)
	} else if time.Since(lastWarning) <= MinWarningFrequency {
		return nil
	}
	var msgs []string
	if puzzle.PuzzleURL == "" {
		msgs = append(msgs, "missing a URL")
	}
	if puzzle.Round.Name == "" {
		msgs = append(msgs, "missing a round")
	}
	if puzzle.Answer != "" && !puzzle.Status.IsSolved() {
		msgs = append(msgs, "has an answer even though it's not marked solved")
	}
	if len(msgs) == 0 {
		return fmt.Errorf("cannot warn about well-formatted puzzle %q: %v", puzzle.Name, puzzle)
	}
	if err := p.discord.QMChannelSend(fmt.Sprintf("Puzzle %q is %s", puzzle.Name, strings.Join(msgs, " and "))); err != nil {
		return err
	}
	p.lastWarnTime[puzzle.AirtableRecord.ID] = time.Now()
	return nil
}
