package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/state"
	"github.com/gauravjsingh/emojihunt/syncer"
)

const (
	pollInterval        = 10 * time.Second
	initialWarningDelay = 1 * time.Minute
	minWarningFrequency = 10 * time.Minute
	modifyGracePeriod   = 8 * time.Second
)

type Poller struct {
	airtable *client.Airtable
	discord  *client.Discord
	syncer   *syncer.Syncer
	state    *state.State

	mu           sync.Mutex           // hold while accessing everything below
	lastWarnTime map[string]time.Time // airtable id -> when we last warned about a malformed puzzle
}

func NewPoller(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer, state *state.State) *Poller {
	return &Poller{
		airtable:     airtable,
		discord:      discord,
		syncer:       syncer,
		state:        state,
		lastWarnTime: map[string]time.Time{},
	}
}

func (p *Poller) isEnabled() bool {
	p.state.Lock()
	defer p.state.Unlock()
	return !p.state.HuntbotDisabled
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
					msg := fmt.Sprintf("polling sheet failed: ```\n%s\n```", spew.Sdump(err))
					p.discord.ChannelSend(p.discord.TechChannel, msg)
				}
			} else {
				failures = 0
			}

			listTimestamp := time.Now()

			for _, puzzle := range puzzles {
				if puzzle.Pending {
					// Skip auto-added records that haven't been confirmed by a
					// human
					continue
				}
				// TODO: refresh puzzles from the API if our data is more than N
				// seconds stale...
				err := p.processPuzzle(ctx, &puzzle, &listTimestamp)
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
		case <-time.After(pollInterval):
		}
	}
}

func (p *Poller) processPuzzle(ctx context.Context, puzzle *schema.Puzzle, timestamp *time.Time) error {
	if !puzzle.IsValid() {
		// Occasionally warn the QM about puzzles that are missing fields.
		if puzzle.Name != "" {
			if err := p.warnPuzzle(ctx, puzzle); err != nil {
				return fmt.Errorf("error warning about malformed puzzle %q: %v", puzzle.Name, err)
			}
		}
		return nil
	}

	if timestamp.Sub(*puzzle.LastModified) < modifyGracePeriod {
		// Wait a few seconds after edits before processing a puzzle. Since
		// Airtable exposes as-you-type changes in the API, this delay is
		// necessary to avoid picking up puzzles with  partially-entered text.
		return nil
	}

	_, err := p.syncer.IdempotentCreateUpdate(ctx, puzzle)
	return err
}

func (p *Poller) warnPuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if lastWarning, ok := p.lastWarnTime[puzzle.AirtableRecord.ID]; !ok {
		p.lastWarnTime[puzzle.AirtableRecord.ID] = time.Now().Add(initialWarningDelay - minWarningFrequency)
	} else if time.Since(lastWarning) <= minWarningFrequency {
		return nil
	}
	var msgs []string
	if puzzle.PuzzleURL == "" {
		msgs = append(msgs, "missing a URL")
	}
	if len(puzzle.Rounds) == 0 {
		msgs = append(msgs, "missing a round")
	}
	if puzzle.Answer != "" && !puzzle.Status.IsSolved() {
		msgs = append(msgs, "has an answer even though it's not marked solved")
	}
	if len(msgs) == 0 {
		return fmt.Errorf("cannot warn about well-formatted puzzle %q: %v", puzzle.Name, puzzle)
	}
	msg := fmt.Sprintf("**:boom: Halp!** Errors with puzzle %q: %s.",
		puzzle.Name, strings.Join(msgs, " and "))
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Edit in Airtable",
			Style: discordgo.LinkButton,
			Emoji: discordgo.ComponentEmoji{Name: "📝"},
			URL:   p.airtable.EditURL(puzzle),
		},
	}
	if err := p.discord.ChannelSendComponents(p.discord.QMChannel, msg, components); err != nil {
		return err
	}
	p.lastWarnTime[puzzle.AirtableRecord.ID] = time.Now()
	return nil
}
