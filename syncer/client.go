package syncer

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/ably/ably-go/ably"
	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/drive"
	live "github.com/emojihunt/emojihunt/live/client"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
	"github.com/getsentry/sentry-go"
	"golang.org/x/xerrors"
)

type Client struct {
	RestartDiscovery chan bool

	ably    *ably.RealtimeChannel
	discord *discord.Client
	drive   *drive.Client
	live    *live.Client
	state   *state.Client

	solvedCategories []string
	sortLock         sync.Mutex
}

const ablyChannelName = "huntbot"

func New(ably *ably.Realtime, discord *discord.Client, drive *drive.Client,
	live *live.Client, state *state.Client) *Client {
	return &Client{
		RestartDiscovery: make(chan bool),
		ably:             ably.Channels.Get(ablyChannelName),
		discord:          discord,
		drive:            drive,
		live:             live,
		state:            state,
	}
}

func (c *Client) Watch(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.HandleMetrics()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "sync")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	// The bot's status is reset when we connect to Discord
	if err := c.TriggerDiscovery(ctx); err != nil {
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
		case <-c.state.DiscoveryChange:
			err = c.TriggerDiscovery(ctx)
			c.RestartDiscovery <- true
		case change := <-c.state.PuzzleChange:
			err = c.TriggerPuzzle(ctx, change)
			if change.BotComplete != nil {
				change.BotComplete <- err
			}
		case change := <-c.state.RoundChange:
			err = c.TriggerRound(ctx, change)
		case <-ctx.Done():
			return
		}
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else if len(c.state.PuzzleChange) == 0 && len(c.state.RoundChange) == 0 {
			log.Printf("sync: up to date!")
		}
	}
}

func (c *Client) TriggerDiscovery(ctx context.Context) error {
	discoveryRestarts.Inc()
	config, err := c.state.DiscoveryConfig(ctx)
	if err != nil {
		return err
	}

	var message = c.live.ComputeMeta(config)
	c.state.LiveMessage <- state.LiveMessage{
		Event: state.EventTypeSettings,
		Data:  message,
	}
	if err := c.ably.Publish(ctx, state.EventTypeSettings, message); err != nil {
		return xerrors.Errorf("ably.Publish: %w", err)
	}

	var data discordgo.UpdateStatusData
	if config.PuzzlesURL == "" {
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
	puzzlesProcessed.Inc()
	if change.ChangeID > 0 {
		// Publish the update to Ably
		c.state.LiveMessage <- state.LiveMessage{
			Event: state.EventTypeSync,
			Data:  change.SyncMessage(),
		}
		err := c.ably.Publish(ctx, state.EventTypeSync, change.SyncMessage())
		if err != nil {
			return xerrors.Errorf("ably.Publish: %w", err)
		}
	}

	if change.After == nil {
		// Don't take any action when a puzzle is deleted. To avoid accidents, the
		// channel and spreadsheet should be cleaned up manually.
		return nil
	}

	var wg sync.WaitGroup
	var ch = make(chan error, 4)
	var puzzle = *change.After

	// Maybe sync updates to the Discord channel name and category
	if puzzle.Round.DiscordCategory == "" {
		// Round category needs to be lazily created...
		c.CheckDiscordRound(ctx, puzzle.Round) // (will trigger another change)
	} else {
		var c0 DiscordChannelFields
		if change.Before != nil {
			c0 = NewDiscordChannelFields(*change.Before)
		}
		var c1 = NewDiscordChannelFields(puzzle)
		if puzzle.DiscordChannel != "" && c0 != c1 {
			wg.Add(1)
			go func() { ch <- c.UpdateDiscordChannel(ctx, c1); wg.Done() }()
		}
	}

	// Maybe sync updates to the Discord pinned message
	var p0 DiscordPinFields
	if change.Before != nil {
		p0 = NewDiscordPinFields(*change.Before)
	}
	var p1 = NewDiscordPinFields(puzzle)
	if puzzle.DiscordChannel != "" && p0 != p1 {
		wg.Add(1)
		go func() { ch <- c.UpdateDiscordPin(ctx, p1); wg.Done() }()
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

	// Maybe sync updates to the voice room
	var v0 VoiceRoomFields
	if change.Before != nil {
		v0 = NewVoiceRoomFields(*change.Before)
	}
	var v1 = NewVoiceRoomFields(puzzle)
	if v0 != v1 {
		wg.Add(1)
		go func() { ch <- c.SyncVoiceRooms(ctx); wg.Done() }()
	}

	wg.Wait()
	close(ch)
	for err := range ch {
		var ic *invalidChannelError
		var code = discord.ErrCode(err)
		if errors.As(err, &ic) ||
			code == discordgo.ErrCodeUnknownChannel ||
			code == discordgo.ErrCodeInvalidFormBody {
			log.Printf("sync: discord error %#v on %q", err, puzzle.Name)
			c.CheckDiscordPuzzle(ctx, puzzle)
			return err
		} else if err != nil {
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
		if change.BotComplete == nil {
			err := c.NotifySolveInPuzzleChannel(puzzle)
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
	roundsProcessed.Inc()
	if change.ChangeID > 0 {
		// Publish the update to Ably
		c.state.LiveMessage <- state.LiveMessage{
			Event: state.EventTypeSync,
			Data:  change.SyncMessage(),
		}
		err := c.ably.Publish(ctx, state.EventTypeSync, change.SyncMessage())
		if err != nil {
			return xerrors.Errorf("ably.Publish: %w", err)
		}
	}

	if change.After == nil {
		return nil
	}

	var wg sync.WaitGroup
	var ch = make(chan error, 4)
	var round = *change.After

	// Maybe sync updates to the Discord category name
	var c0 DiscordCategoryFields
	if change.Before != nil {
		c0 = NewDiscordCategoryFields(*change.Before)
	}
	var c1 = NewDiscordCategoryFields(round)
	if round.DiscordCategory != "" && c0 != c1 {
		wg.Add(1)
		go func() { ch <- c.UpdateDiscordCategory(ctx, c1); wg.Done() }()
	}

	// Maybe sync updates to the Google Drive folder name
	var d0 DriveFolderFields
	if change.Before != nil {
		d0 = NewDriveFolderFields(*change.Before)
	}
	var d1 = NewDriveFolderFields(round)
	if round.DriveFolder != "" && d0 != d1 {
		wg.Add(1)
		go func() { ch <- c.UpdateDriveFolder(ctx, d1); wg.Done() }()
	}

	wg.Wait()
	close(ch)
	for err := range ch {
		var code = discord.ErrCode(err)
		if code == discordgo.ErrCodeUnknownChannel || code == discordgo.ErrCodeInvalidFormBody {
			c.CheckDiscordRound(ctx, round)
			return err
		} else if err != nil {
			return err
		}
	}
	return nil
}
