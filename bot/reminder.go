package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/util"
	"github.com/getsentry/sentry-go"
)

type ReminderBot struct {
	db      *db.Client
	discord *discord.Client
	state   *state.State

	intervals          []time.Duration
	warnErrorFrequency time.Duration
}

func NewReminderBot(main context.Context, db *db.Client, discord *discord.Client, state *state.State) discord.Bot {
	b := &ReminderBot{
		db: db, discord: discord, state: state,
		intervals: []time.Duration{
			-2 * time.Hour,
			-1 * time.Hour,
			-30 * time.Minute,
		},
		warnErrorFrequency: 10 * time.Minute,
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
	puzzles, err := b.db.ListWithReminder(ctx)
	if err != nil {
		return "", err
	}

	if len(puzzles) < 1 {
		return ":zero: There are no puzzle reminders. Use the `Reminder` field in Airtable " +
			"to set a reminder.", nil
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
			puzzle.Reminder.In(util.BostonTime).Format("Mon 3:04 PM"),
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
		b.state.Lock()
		since := b.state.ReminderTimestamp
		b.state.Unlock()

		if next, err := b.notify(ctx, since); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else {
			b.state.Lock()
			b.state.ReminderTimestamp = *next
			b.state.CommitAndUnlock(ctx)
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

	puzzles, err := b.db.ListWithReminder(ctx)
	if err != nil {
		return nil, err
	}

	for _, puzzle := range puzzles {
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
				puzzle.Name, puzzle.Reminder.In(util.BostonTime).Format("Mon 3:04 PM"))
		}

		if msg != "" {
			if len(puzzle.DiscordChannel) > 1 {
				err = b.discord.ChannelSendRawID(puzzle.DiscordChannel, msg)
				if err != nil {
					return nil, err
				}
			}
			_, err = b.discord.ChannelSend(b.discord.QMChannel, msg)
			if err != nil {
				return nil, err
			}
		}
	}

	return &now, nil
}

func (b *ReminderBot) HandleReaction(context.Context,
	*discordgo.MessageReaction) error {
	return nil
}

func (b *ReminderBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
