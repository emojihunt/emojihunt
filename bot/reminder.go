package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
)

var eastern, _ = time.LoadLocation("America/New_York")
var notifications = []time.Duration{
	-2 * time.Hour,
	-1 * time.Hour,
	-30 * time.Minute,
}

const warnErrorFrequency = 10 * time.Minute

func RegisterReminderBot(database *db.Client, discord *discord.Client, state *state.State) {
	var bot = reminderBot{database, discord, state}
	discord.AddCommand(bot.makeSlashCommand())
	go bot.notificationLoop()
}

type reminderBot struct {
	database *db.Client
	discord  *discord.Client
	state    *state.State
}

func (bot *reminderBot) makeSlashCommand() *discord.Command {
	return &discord.Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "reminders",
			Description: "List all puzzle reminders ⏱️",
		},
		Handler: func(s *discordgo.Session, i *discord.CommandInput) (string, error) {
			puzzles, err := bot.database.ListWithReminder()
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
					puzzle.Reminder.In(eastern).Format("Mon 3:04 PM"),
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
				msg := fmt.Sprintf("```*** ERROR PROCESSING REMINDERS ***\n\n%s\n```", spew.Sprint(err))
				_, err = bot.discord.ChannelSend(bot.discord.TechChannel, msg)
				if err != nil {
					log.Printf(
						"reminder: error notifying #%s of error: %s",
						bot.discord.TechChannel.Name,
						spew.Sprint(err),
					)
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

	puzzles, err := bot.database.ListWithReminder()
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
				"now (%s ET)",
				puzzle.Name, puzzle.Reminder.In(eastern).Format("Mon 3:04 PM"))
		}

		if msg != "" {
			if len(puzzle.DiscordChannel) > 1 {
				err = bot.discord.ChannelSendRawID(puzzle.DiscordChannel, msg)
				if err != nil {
					return nil, err
				}
			}
			_, err = bot.discord.ChannelSend(bot.discord.QMChannel, msg)
			if err != nil {
				return nil, err
			}
		}
	}

	return &now, nil
}
