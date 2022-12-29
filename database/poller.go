package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
)

const (
	pollInterval        = 15 * time.Second
	initialWarningDelay = 1 * time.Minute
	minWarningFrequency = 10 * time.Minute
)

type Poller struct {
	airtable *client.Airtable
	discord  *client.Discord
	syncer   *syncer.Syncer
	state    *state.State
}

func NewPoller(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer, state *state.State) *Poller {
	return &Poller{
		airtable: airtable,
		discord:  discord,
		syncer:   syncer,
		state:    state,
	}
}

func (p *Poller) Poll(ctx context.Context) {
	// Airtable doesn't support webhooks, so we have to poll the database for
	// updates.
	failures := 0
	for {
		if !p.state.IsKilled() {
			invalid, needsSync, err := p.airtable.ListPuzzlesToAction()
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

			for _, puzzle := range invalid {
				if err := p.warnPuzzle(ctx, &puzzle); err != nil {
					log.Printf("error warning about malformed puzzle %q: %v", puzzle.Name, err)
				}
			}

			for _, id := range needsSync {
				puzzle, err := p.airtable.LockByID(id)
				if err != nil {
					log.Printf("failed to reload puzzle: %v", err)
					continue
				}
				if puzzle.LastModified == nil || time.Since(*puzzle.LastModified) < p.airtable.ModifyGracePeriod {
					puzzle.Unlock()
					continue
				}
				if _, err = p.syncer.IdempotentCreateUpdate(ctx, puzzle); err != nil {
					// Log errors and keep going
					log.Printf("updating puzzle failed: %v", err)
				}
				puzzle.Unlock()
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

func (p *Poller) warnPuzzle(ctx context.Context, puzzle *schema.InvalidPuzzle) error {
	p.state.Lock()
	defer p.state.CommitAndUnlock()

	if lastWarning, ok := p.state.AirtableLastWarn[puzzle.RecordID]; !ok {
		p.state.AirtableLastWarn[puzzle.RecordID] = time.Now().Add(initialWarningDelay - minWarningFrequency)
	} else if time.Since(lastWarning) <= minWarningFrequency {
		return nil
	}
	msg := fmt.Sprintf("**:boom: Halp!** Errors with puzzle %q: %s.",
		puzzle.Name, strings.Join(puzzle.Problems, " and "))
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Edit in Airtable",
			Style: discordgo.LinkButton,
			Emoji: discordgo.ComponentEmoji{Name: "📝"},
			URL:   puzzle.EditURL,
		},
	}
	if _, err := p.discord.ChannelSendComponents(p.discord.QMChannel, msg, components); err != nil {
		return err
	}
	p.state.AirtableLastWarn[puzzle.RecordID] = time.Now()
	return nil
}
