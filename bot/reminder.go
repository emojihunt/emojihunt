package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/huntyet"
	"github.com/emojihunt/emojihunt/state"
	"github.com/getsentry/sentry-go"
)

type ReminderBot struct {
	discord   *discord.Client
	state     *state.Client
	intervals []time.Duration
}

func NewReminderBot(main context.Context, discord *discord.Client, state *state.Client) discord.Bot {
	b := &ReminderBot{
		discord: discord,
		state:   state,
		intervals: []time.Duration{
			-2 * time.Hour,
			-1 * time.Hour,
			-30 * time.Minute,
		},
	}
	go b.worker(main)
	return b
}

func (b *ReminderBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "reminders",
		Description: "List all puzzle reminders ⏱️",
	}, false
}

func (b *ReminderBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	results, err := b.state.ListPuzzles(ctx)
	if err != nil {
		return "", err
	}
	var puzzles []state.Puzzle
	for _, puzzle := range results {
		if puzzle.HasReminder() {
			puzzles = append(puzzles, puzzle)
		}
	}

	if len(puzzles) < 1 {
		return ":zero: There are no puzzle reminders. Use the `Reminder` field in " +
			"the puzzle tracker to set a reminder.", nil
	}

	msg := ":calendar_spiral: Reminders:\n"
	for _, puzzle := range puzzles {
		suffix := ""
		if time.Now().After(puzzle.Reminder) {
			suffix = " (passed)"
		} else if time.Until(puzzle.Reminder) > 72*time.Hour {
			suffix = " (warning: in more than 3 days?!)"
		}
		msg += fmt.Sprintf(
			" • %s @ %s ET%s\n",
			puzzle.Name,
			puzzle.Reminder.In(huntyet.BostonTime).Format("Mon 3:04 PM"),
			suffix,
		)
	}
	return msg, nil
}

func (b *ReminderBot) worker(main context.Context) {
	ctx, cancel := context.WithCancel(main)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "reminders")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	for {
		if since, err := b.state.ReminderTimestamp(ctx); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else if next, err := b.notify(ctx, since); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else {
			b.state.SetReminderTimestamp(ctx, *next)
		}

		// Wake up on the next(-ish) 1-minute boundary
		wait := time.Until(time.Now().Add(time.Minute).Truncate(time.Minute))
		if wait < 30*time.Second {
			wait += 1 * time.Minute
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (b *ReminderBot) notify(ctx context.Context, since time.Time) (*time.Time, error) {
	now := time.Now()

	puzzles, err := b.state.ListPuzzles(ctx)
	if err != nil {
		return nil, err
	}

	for _, puzzle := range puzzles {
		if !puzzle.HasReminder() {
			continue
		}

		var msg string
		for _, delay := range b.intervals {
			target := puzzle.Reminder.Add(delay)
			if target.Before(now) && target.After(since) {
				msg = fmt.Sprintf(":hourglass_flowing_sand: Reminder: %q in %s",
					puzzle.Name, time.Until(puzzle.Reminder).Round(time.Minute))
			}
		}
		if puzzle.Reminder.Before(now) && puzzle.Reminder.After(since) {
			msg = fmt.Sprintf(":alarm_clock: It's time! Puzzle %q has a reminder set for "+
				"now (%s ET)",
				puzzle.Name, puzzle.Reminder.In(huntyet.BostonTime).Format("Mon 3:04 PM"))
		}

		if msg != "" {
			_, err = b.discord.ChannelSend(b.discord.QMChannel, msg)
			if err != nil {
				return nil, err
			}
			if puzzle.DiscordChannel != "" {
				err = b.discord.ChannelSendRawID(puzzle.DiscordChannel, msg)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return &now, nil
}

func (b *ReminderBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
