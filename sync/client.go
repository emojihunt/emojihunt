package sync

import (
	"context"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/drive"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
	"github.com/getsentry/sentry-go"
)

type Client struct {
	discord          *discord.Client
	discovery        bool
	drive            *drive.Client
	solvedCategories []string
}

func New(discord *discord.Client, discovery bool, drive *drive.Client) *Client {
	return &Client{discord, discovery, drive, nil}
}

func (c *Client) Watch(ctx context.Context, state *state.Client) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "sync")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	// The bot's status is reset when we connect to Discord
	if err := c.TriggerDiscoveryEnabled(ctx, state.IsEnabled(ctx)); err != nil {
		panic(err)
	}
	if err := c.RestoreSolvedCategories(); err != nil {
		panic(err)
	}
	if err := c.RestorePlaceholderEvent(); err != nil {
		panic(err)
	}

	for {
		var err error
		select {
		case enabled := <-state.DiscoveryChange:
			err = c.TriggerDiscoveryEnabled(ctx, enabled)
		case change := <-state.PuzzleChange:
			err = c.TriggerPuzzle(ctx, change)
		case change := <-state.RoundChange:
			err = c.TriggerRound(ctx, change)
		case <-ctx.Done():
			return
		}
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}
	}
}

func (c *Client) TriggerDiscoveryEnabled(ctx context.Context, enabled bool) error {
	var data discordgo.UpdateStatusData
	if !c.discovery {
		data.Status = "idle"
	} else if enabled {
		data.Status = "online"
	} else {
		data.Status = "dnd"
		data.Activities = []*discordgo.Activity{
			{
				Name:  "Huntbot",
				Type:  discordgo.ActivityTypeCustom,
				State: "puzzle discovery paused",
			},
		}
	}
	return c.discord.UpdateStatus(data)
}

func (c *Client) TriggerPuzzle(ctx context.Context, change state.PuzzleChange) (err error) {
	if change.After == nil {
		// Don't take any action when a puzzle is deleted. To avoid accidents, the
		// channel and spreadsheet should be cleaned up manually.
		return nil
	}

	var wg sync.WaitGroup
	var ch = make(chan error, 4)
	var puzzle = *change.After

	// Maybe sync updates to the Discord channel name and category
	var c0 DiscordChannelFields
	if change.Before != nil {
		c0 = NewDiscordChannelFields(*change.Before)
	}
	var c1 = NewDiscordChannelFields(puzzle)
	if puzzle.DiscordChannel != "" && c0 != c1 {
		wg.Add(1)
		go func() { ch <- c.UpdateDiscordChannel(c1); wg.Done() }()
	}

	// Maybe sync updates to the Discord pinned message
	var p0 DiscordPinFields
	if change.Before != nil {
		p0 = NewDiscordPinFields(*change.Before)
	}
	var p1 = NewDiscordPinFields(puzzle)
	if puzzle.DiscordChannel != "" && p0 != p1 {
		wg.Add(1)
		go func() { ch <- c.UpdateDiscordPin(p1); wg.Done() }()
	}

	// Maybe sync updates to the spreadsheet name and folder
	var s0 SpreadsheetFields
	if change.Before != nil {
		s0 = NewSpreadsheetFields(*change.Before)
	}
	var s1 = NewSpreadsheetFields(puzzle)
	if puzzle.SpreadsheetID != "" && s0 != s1 {
		wg.Add(1)
		go func() { ch <- c.UpdateSpreadsheet(ctx, s1); wg.Done() }()
	}

	wg.Wait()
	close(ch)
	for err := range ch {
		if err != nil {
			return err
		}
	}

	// Notify the puzzle channel and #more-eyes of significant status changes
	if change.Before == nil {
		if puzzle.DiscordChannel != "" {
			return c.NotifyNewPuzzle(puzzle)
		}
	} else if !change.Before.Status.IsSolved() && puzzle.Status.IsSolved() {
		// If the change was triggered by a bot, the bot's response will be visible
		// in the puzzle channel so there's no need to send a solve notification
		// there.
		if !change.Bot {
			err = c.NotifySolveInPuzzleChannel(puzzle)
			if err != nil {
				return err
			}
		}
		// Always notify on solve, even if the puzzle doesn't have a Discord
		// channel.
		return c.NotifySolveInHangingOut(puzzle)
	} else if change.Before.Status == status.NotStarted && puzzle.Status == status.Working {
		if puzzle.DiscordChannel != "" {
			return c.NotifyPuzzleWorking(puzzle)
		}
	}
	return nil
}

func (c *Client) TriggerRound(ctx context.Context, change state.RoundChange) error {
	if change.After == nil {
		return nil
	}
	// TODO
	return nil
}
