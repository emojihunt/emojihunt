package sync

import (
	"context"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/drive"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
)

type Client struct {
	state     *state.Client
	discord   *discord.Client
	discovery bool
	drive     *drive.Client
}

func New(discord *discord.Client, discovery bool, drive *drive.Client, state *state.Client) *Client {
	return &Client{
		discord:   discord,
		discovery: discovery,
		drive:     drive,
		state:     state,
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

func (c *Client) TriggerPuzzle(ctx context.Context, previous *state.Puzzle, puzzle state.Puzzle) error {
	var err error
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

	if previous == nil {
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
		if !previous.Status.IsSolved() && puzzle.Status.IsSolved() {
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
		var c0, c1 = NewDiscordChannelFields(*previous), NewDiscordChannelFields(puzzle)
		if puzzle.HasDiscordChannel() && c0 != c1 {
			wg.Add(1)
			go func() { ch <- c.UpdateDiscordChannel(c1); wg.Done() }()
		}

		// Sync updates to the Discord pinned message:
		var p0, p1 = NewDiscordPinFields(*previous), NewDiscordPinFields(puzzle)
		if puzzle.HasDiscordChannel() && p0 != p1 {
			wg.Add(1)
			go func() { ch <- c.UpdateDiscordPin(p1); wg.Done() }()
		}

		// Sync updates to the spreadsheet name and folder:
		var s0, s1 = NewSpreadsheetFields(*previous), NewSpreadsheetFields(puzzle)
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
		if !previous.Status.IsSolved() && puzzle.Status.IsSolved() {
			return c.NotifyPuzzleSolved(puzzle, false) // TODO: support `botRequest` field
		} else if previous.Status == status.NotStarted && puzzle.Status == status.Working {
			return c.NotifyPuzzleWorking(puzzle)
		} else {
			return nil
		}
	}
}
