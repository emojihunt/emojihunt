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
	discord   *discord.Client
	discovery bool
	drive     *drive.Client
	state     *state.Client
}

func New(discord *discord.Client, discovery bool, drive *drive.Client, state *state.Client) *Client {
	return &Client{discord, discovery, drive, state}
}

func (c *Client) Watch(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "sync")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	// The bot's status is reset when we connect to Discord
	c.TriggerDiscoveryEnabled(ctx)
	c.RestorePlaceholderEvent()

	for {
		var err error
		select {
		case <-c.state.DiscoveryChange:
			err = c.TriggerDiscoveryEnabled(ctx)
		case change := <-c.state.PuzzleChange:
			err = c.TriggerPuzzle(ctx, change)
		case change := <-c.state.RoundChange:
			err = c.TriggerRound(ctx, change)
		case <-ctx.Done():
			return
		}
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}
	}
}

func (c *Client) TriggerDiscoveryEnabled(ctx context.Context) error {
	var data discordgo.UpdateStatusData
	if !c.discovery {
		data.Status = "idle"
	} else if c.state.IsEnabled(ctx) {
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

func (c *Client) TriggerPuzzle(ctx context.Context, change state.PuzzleChange) error {
	if change.After == nil {
		// Don't take any action when a puzzle is deleted. To avoid accidents, the
		// channel and spreadsheet should be cleaned up manually.
		return nil
	} else if !change.Sync {
		// To avoid infinite loops, don't take any action on changes made by
		// TriggerPuzzle.
		return nil
	}

	var err error
	var puzzle = *change.After
	if puzzle.SpreadsheetID == "" {
		puzzle, err = c.CreateSpreadsheet(ctx, puzzle)
		if err != nil {
			return err
		}
	}
	if puzzle.DiscordChannel == "" {
		puzzle, err = c.CreateDiscordChannel(ctx, puzzle)
		if err != nil {
			return err
		}
	}

	if change.Before == nil {
		err = c.UpdateDiscordChannel(NewDiscordChannelFields(puzzle))
		if err != nil {
			return err
		}
		err = c.UpdateDiscordPin(NewDiscordPinFields(puzzle))
		if err != nil {
			return err
		}
		err = c.UpdateSpreadsheet(ctx, NewSpreadsheetFields(puzzle))
		if err != nil {
			return err
		}
		return c.NotifyNewPuzzle(puzzle)
	} else {
		var wg sync.WaitGroup
		var ch = make(chan error, 4)

		// On solve, unset voice room
		if !change.Before.Status.IsSolved() && puzzle.Status.IsSolved() {
			var sync bool
			puzzle, err = c.state.UpdatePuzzle(ctx, puzzle.ID,
				func(puzzle *state.RawPuzzle) error {
					if puzzle.VoiceRoom != "" {
						puzzle.VoiceRoom = ""
						sync = true
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
			if sync {
				wg.Add(1)
				go func() { ch <- c.SyncVoiceRooms(ctx); wg.Done() }()
			}
		}

		// Sync updates to the Discord channel name and category:
		var c0, c1 = NewDiscordChannelFields(*change.Before), NewDiscordChannelFields(puzzle)
		if puzzle.HasDiscordChannel() && c0 != c1 {
			wg.Add(1)
			go func() { ch <- c.UpdateDiscordChannel(c1); wg.Done() }()
		}

		// Sync updates to the Discord pinned message:
		var p0, p1 = NewDiscordPinFields(*change.Before), NewDiscordPinFields(puzzle)
		if puzzle.HasDiscordChannel() && p0 != p1 {
			wg.Add(1)
			go func() { ch <- c.UpdateDiscordPin(p1); wg.Done() }()
		}

		// Sync updates to the spreadsheet name and folder:
		var s0, s1 = NewSpreadsheetFields(*change.Before), NewSpreadsheetFields(puzzle)
		if puzzle.HasSpreadsheetID() && s0 != s1 {
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
		if !change.Before.Status.IsSolved() && puzzle.Status.IsSolved() {
			return c.NotifyPuzzleSolved(puzzle, false) // TODO: support `botRequest` field
		} else if change.Before.Status == status.NotStarted && puzzle.Status == status.Working {
			return c.NotifyPuzzleWorking(puzzle)
		} else {
			return nil
		}
	}
}

func (c *Client) TriggerRound(ctx context.Context, change state.RoundChange) error {
	// TODO
	return nil
}