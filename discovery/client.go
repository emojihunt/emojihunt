package discovery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/emojiname"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/db"
	"github.com/emojihunt/emojihunt/syncer"
	"github.com/getsentry/sentry-go"
)

type Client struct {
	discord *discord.Client
	state   *state.Client
	syncer  *syncer.Client

	discovered chan []state.ScrapedPuzzle
}

func New(discord *discord.Client, s *state.Client, y *syncer.Client) *Client {
	return &Client{discord, s, y, make(chan []state.ScrapedPuzzle)}
}

func (c *Client) Watch(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "discovery.watch")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	for {
		ctx, cancel := context.WithCancel(ctx)
		if !c.state.IsEnabled(ctx) {
			log.Printf("discovery: disabled via kill switch")
		} else {
			err := func() error {
				config, err := c.state.DiscoveryConfig(ctx)
				if err != nil {
					return err
				}
				if config.PuzzlesURL == "" {
					log.Printf("discovery: disabled because puzzle_url is blank")
					return nil // disabled
				}
				poller, err := NewPoller(config)
				if err != nil {
					return err
				}
				go poller.Poll(ctx, c.discovered)
				return nil
			}()
			if err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		}

		select {
		case <-ctx.Done():
			cancel()
			return
		case <-c.syncer.RestartDiscovery:
			log.Printf("discovery: restarting...")
			cancel()
		}
	}
}

func (c *Client) SyncWorker(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "discovery.sync")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	var wakeup = time.Now().Add(roundCreationPause)
	for {
		select {
		case puzzles := <-c.discovered:
			for _, puzzle := range puzzles {
				if !c.state.IsEnabled(ctx) {
					break
				}
				if err := c.handleScrapedPuzzle(ctx, puzzle); err != nil {
					sentry.GetHubFromContext(ctx).CaptureException(err)
					break
				}
			}
		case <-time.After(time.Until(wakeup)):
		case <-ctx.Done():
			return
		}

		rounds, err := c.state.ListPendingDiscoveredRounds(ctx)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			continue
		}
		for _, round := range rounds {
			if !c.state.IsEnabled(ctx) {
				break
			}
			err := c.handleDiscoveredRound(ctx, round)
			if err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		}

		puzzles, err := c.state.ListCreatablePuzzles(ctx)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			continue
		}
		for _, puzzle := range puzzles {
			if !c.state.IsEnabled(ctx) {
				break
			}
			err := c.handleCreatablePuzzle(ctx, puzzle)
			if err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		}
		wakeup = wakeup.Add(roundCreationPause)
	}
}

func (c *Client) handleScrapedPuzzle(ctx context.Context, record state.ScrapedPuzzle) error {
	created, err := c.state.IsPuzzleCreated(ctx, record)
	if err != nil {
		return err
	} else if created {
		return nil // already created
	}
	discovered, err := c.state.IsPuzzleDiscovered(ctx, record)
	if err != nil {
		return err
	} else if discovered {
		return nil // already handled
	}

	var params = db.CreateDiscoveredPuzzleParams{
		PuzzleURL: record.PuzzleURL,
		Name:      record.Name,
	}
	round, err := c.state.GetCreatedRound(ctx, record.RoundName)
	if errors.Is(err, sql.ErrNoRows) {
		round, err := c.state.GetDiscoveredRound(ctx, record.RoundName)
		if errors.Is(err, sql.ErrNoRows) {
			// New round, hasn't even been logged yet
			new, err := c.state.CreateDiscoveredRound(ctx, record.RoundName)
			if err != nil {
				return err
			}
			params.DiscoveredRound = sql.NullInt64{Int64: new, Valid: true}
		} else if err != nil {
			return err
		} else {
			// New round, pending QM approval & creation
			params.DiscoveredRound = sql.NullInt64{Int64: round.ID, Valid: true}
		}
		log.Printf("discovery: logging scraped puzzle %q", record.Name)
		return c.state.CreateDiscoveredPuzzle(ctx, params)
	} else if err != nil {
		return err
	} else {
		// Ready to create puzzle
		err = c.state.CreateDiscoveredPuzzle(ctx, params)
		if err != nil {
			return err
		}
		return c.createPuzzle(ctx, record, round)
	}
}

func (c *Client) createPuzzle(ctx context.Context, record state.ScrapedPuzzle,
	round state.Round) error {
	log.Printf("discovery: creating puzzle %q (%s) in round %q",
		record.Name, record.PuzzleURL, round.Name)
	var err error
	var puzzle = state.RawPuzzle{
		Name:      record.Name,
		Round:     round.ID,
		PuzzleURL: record.PuzzleURL,
	}
	puzzle.SpreadsheetID, err = c.syncer.CreateSpreadsheet(ctx, puzzle, round)
	if err != nil {
		return err
	}
	puzzle.DiscordChannel, err = c.syncer.CreateDiscordChannel(ctx, puzzle, round)
	if err != nil {
		return err
	}
	_, _, err = c.state.CreatePuzzle(ctx, puzzle)
	return err
}

func (c *Client) handleDiscoveredRound(ctx context.Context, round db.DiscoveredRound) error {
	if round.MessageID == "" {
		log.Printf("discovery: notifying #qm of new round %q", round.Name)
		puzzles, err := c.state.ListDiscoveredPuzzlesForRound(ctx, round.ID)
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("```*** ❓ NEW ROUND: \"%s\" ***\n\n", round.Name)
		for _, puzzle := range puzzles {
			msg += fmt.Sprintf("%s\n%s\n\n", puzzle.Name, puzzle.PuzzleURL)
		}
		msg += "Reminder: use `/qm discovery pause` to stop the bot.\n\n"
		msg += fmt.Sprintf(
			"```\n%s please react to pick an emoji for this round\n",
			c.discord.QMRole.Mention(),
		)

		id, err := c.discord.ChannelSend(c.discord.QMChannel, msg)
		if err != nil {
			return err
		}
		round.MessageID = id
		round.NotifiedAt = time.Now()
		return c.state.UpdateDiscoveredRound(ctx, round)

	} else if time.Now().After(round.NotifiedAt.Add(roundCreationPause)) {
		// It's been a while, check Discord for round emoji
		emoji, err := c.discord.GetTopReaction(c.discord.QMChannel, round.MessageID)
		if err != nil {
			return err
		} else if emoji == "" {
			// QM hasn't assigned an emoji yet
			return nil
		}

		var hue = emojiname.EmojiHue(emoji)
		var record = state.Round{Name: round.Name, Emoji: emoji, Hue: int64(hue)}
		log.Printf("discovery: creating round %q with emoji %q, hue %d",
			record.Name, record.Emoji, record.Hue)
		record.DriveFolder, err = c.syncer.CreateDriveFolder(ctx, record)
		if err != nil {
			return err
		}
		created, _, err := c.state.CreateRound(ctx, record)
		if err != nil {
			return err
		}
		return c.state.CompleteDiscoveredRound(ctx, round.ID, created)
	}
	return nil
}

func (c *Client) handleCreatablePuzzle(ctx context.Context, row db.ListCreatablePuzzlesRow) error {
	var scraped = state.ScrapedPuzzle{
		Name:      row.Name,
		RoundName: row.Name_2,
		PuzzleURL: row.PuzzleURL,
	}
	created, err := c.state.IsPuzzleCreated(ctx, scraped)
	if err != nil {
		return err
	} else if created {
		// was manually created in the interim
		return c.state.CompleteDiscoveredPuzzle(ctx, row.ID)
	}

	round, err := c.state.GetCreatedRound(ctx, scraped.RoundName)
	if err != nil {
		return err
	}
	err = c.state.CompleteDiscoveredPuzzle(ctx, row.ID)
	if err != nil {
		return err
	}
	return c.createPuzzle(ctx, scraped, round)
}
