package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/state"
)

var timemit, _ = time.LoadLocation("America/New_York")
var notifications = []time.Duration{
	-2 * time.Hour,
	-1 * time.Hour,
	-30 * time.Minute,
}

const warnErrorFrequency = 10 * time.Minute

func RegisterReminderBot(airtable *client.Airtable, discord *client.Discord, state *state.State) {
	var bot = reminderBot{airtable, discord, state}
	discord.AddCommand(bot.makeSlashCommand())
	go bot.notificationLoop()
}

type reminderBot struct {
	airtable *client.Airtable
	discord  *client.Discord
	state    *state.State
}

func (bot *reminderBot) makeSlashCommand() *client.DiscordCommand {
	return &client.DiscordCommand{
		InteractionType: discordgo.InteractionApplicationCommand,
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "reminders",
			Description: "List all puzzle reminders ⏱️",
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			puzzles, err := bot.airtable.ListWithReminder()
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
					" • %s @ %s TIMEMIT%s\n",
					puzzle.Name,
					puzzle.Reminder.In(timemit).Format("Mon 3:04 PM"),
					suffix,
				)
			}
			return msg, nil
		},
	}
}

func (bot *reminderBot) notificationLoop() {
	for {
		bot.state.Lock()
		since := bot.state.ReminderTimestamp
		bot.state.Unlock()

		next, err := bot.processNotifications(since)
		bot.state.Lock()
		if err != nil {
			log.Printf("reminder: error: %s", spew.Sprint(err))
			if time.Since(bot.state.ReminderWarnError).Truncate(time.Minute) >= warnErrorFrequency {
				msg := fmt.Sprintf(":alarm_clock: Error processing reminders: ```\n%s\n```", spew.Sprint(err))
				err = bot.discord.ChannelSend(bot.discord.TechChannel, msg)
				if err != nil {
					log.Printf("reminder: error notifying #tech of error: %s", spew.Sprint(err))
				}
				bot.state.ReminderWarnError = time.Now()
			}
		} else {
			bot.state.ReminderTimestamp = *next
		}
		bot.state.CommitAndUnlock()

		// Wake up on next 1-minute boundary
		wait := time.Until(time.Now().Add(time.Minute).Truncate(time.Minute))
		if wait < 30*time.Second {
			wait += 1 * time.Minute
		}
		time.Sleep(wait)
	}
}

func (bot *reminderBot) processNotifications(since time.Time) (*time.Time, error) {
	now := time.Now()

	puzzles, err := bot.airtable.ListWithReminder()
	if err != nil {
		return nil, err
	}

	for _, puzzle := range puzzles {
		var msg string
		for _, delay := range notifications {
			target := puzzle.Reminder.Add(delay)
			if target.Before(now) && target.After(since) {
				msg = fmt.Sprintf(":hourglass_flowing_sand: Reminder: %q in %s",
					puzzle.Name, time.Until(puzzle.Reminder).Round(time.Minute))
			}
		}
		if puzzle.Reminder.Before(now) && puzzle.Reminder.After(since) {
			msg = fmt.Sprintf(":alarm_clock: It's time! Puzzle %q has a reminder set for "+
				"now (%s TIMEMIT)",
				puzzle.Name, puzzle.Reminder.In(timemit).Format("Mon 3:04 PM"))
		}

		if msg != "" {
			if len(puzzle.DiscordChannel) > 1 {
				err = bot.discord.ChannelSendRawID(puzzle.DiscordChannel, msg)
				if err != nil {
					return nil, err
				}
			}
			err = bot.discord.ChannelSend(bot.discord.QMChannel, msg)
			if err != nil {
				return nil, err
			}
		}
	}

	return &now, nil
}
