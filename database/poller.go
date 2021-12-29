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

	mu           sync.Mutex               // hold while accessing everything below
	enabled      bool                     // global killswitch, toggle with !huntbot kill/!huntbot start
	puzzleStatus map[string]schema.Status // name -> status (best-effort cache)
	archived     map[string]bool          // name -> channel was archived (best-effort cache)
	lastWarnTime map[string]time.Time     // name -> when we last warned about a malformed puzzle
}

func NewPoller(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer) *Poller {
	return &Poller{
		airtable:     airtable,
		discord:      discord,
		syncer:       syncer,
		enabled:      true,
		puzzleStatus: map[string]schema.Status{},
		archived:     map[string]bool{},
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
			log.Print("exiting watcher due to signal")
			return
		case <-time.After(PollInterval):
		}
	}
}

func (p *Poller) processPuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
	if puzzle.Name == "" || puzzle.PuzzleURL == "" || puzzle.Round.Name == "" {
		// Occasionally warn the QM about puzzles that are missing fields.
		if puzzle.Name != "" {
			if err := p.warnPuzzle(ctx, puzzle); err != nil {
				return fmt.Errorf("error warning about malformed puzzle %q: %v", puzzle.Name, err)
			}
		}
		return nil
	}

	var err error
	puzzle, err = p.syncer.IdempotentCreate(ctx, puzzle)
	if err != nil {
		return err
	}

	if p.setPuzzleStatus(puzzle.Name, puzzle.Status) != puzzle.Status ||
		puzzle.Answer != "" && puzzle.Status.IsSolved() && !p.isArchived(puzzle.Name) {
		// (potential) status change
		if puzzle.Status.IsSolved() {
			if err := p.syncer.MarkSolved(ctx, puzzle); err != nil {
				return fmt.Errorf("failed to mark puzzle %q solved: %v", puzzle.Name, err)
			}
			p.archive(puzzle.Name)
		} else {
			if err := p.logStatus(ctx, puzzle); err != nil {
				return fmt.Errorf("failed to mark puzzle %q %v: %v", puzzle.Name, puzzle.Status, err)
			}
		}
	}

	return nil
}

func (p *Poller) warnPuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if lastWarning, ok := p.lastWarnTime[puzzle.Name]; !ok {
		p.lastWarnTime[puzzle.Name] = time.Now().Add(InitialWarningDelay - MinWarningFrequency)
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
	if len(msgs) == 0 {
		return fmt.Errorf("cannot warn about well-formatted puzzle %q: %v", puzzle.Name, puzzle)
	}
	if err := p.discord.QMChannelSend(fmt.Sprintf("Puzzle %q is %s", puzzle.Name, strings.Join(msgs, " and "))); err != nil {
		return err
	}
	p.lastWarnTime[puzzle.Name] = time.Now()
	return nil
}

func (p *Poller) isArchived(puzzleName string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.archived[puzzleName]
}

func (p *Poller) archive(puzzleName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.archived[puzzleName] = true
}

func (p *Poller) setPuzzleStatus(name string, newStatus schema.Status) (oldStatus schema.Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	oldStatus = p.puzzleStatus[name]
	p.puzzleStatus[name] = newStatus
	return oldStatus
}

// logStatus marks the status; it is *not* called if the puzzle is solved
func (p *Poller) logStatus(ctx context.Context, puzzle *schema.Puzzle) error {
	didUpdate, err := p.syncer.SetPinnedStatusInfo(puzzle, puzzle.DiscordChannel)
	if err != nil {
		return fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
	}

	if didUpdate {
		if err := p.discord.StatusUpdateChannelSend(fmt.Sprintf("%s Puzzle <#%s> is now %v.", puzzle.Round.Emoji, puzzle.DiscordChannel, puzzle.Status.Pretty())); err != nil {
			return fmt.Errorf("error posting puzzle status announcement: %v", err)
		}
	}

	return nil
}
