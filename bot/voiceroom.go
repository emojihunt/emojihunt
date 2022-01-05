package bot

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
)

const (
	roomStatusHeader = "Working Room"
	eventDescription = "🤖 Event managed by Huntbot. Use `/name` to modify!"
)

func MakeVoiceRoomCommand(air *client.Airtable, dis *client.Discord) *client.DiscordCommand {
	return &client.DiscordCommand{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "room",
			Description: "Assign puzzles to voice rooms",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Assign this puzzle to a voice room 🔔",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "in",
							Description: "Where are we going? 🗺️",
							Required:    true,
							Type:        discordgo.ApplicationCommandOptionChannel,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildVoice,
							},
						},
					},
				},
				{
					Name:        "stop",
					Description: "Remove this puzzle from its voice room 🔕",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			puzzle, err := air.FindByDiscordChannel(i.IC.ChannelID)
			if err != nil {
				return "", fmt.Errorf("unable to get puzzle for channel ID %q", i.IC.ChannelID)
			}

			var reply string
			switch i.Subcommand.Name {
			case "start":
				var channel *discordgo.Channel
				for _, opt := range i.Subcommand.Options {
					if opt.Name == "in" {
						channel = opt.ChannelValue(s)
					}
				}
				if channel == nil {
					return "", fmt.Errorf("could not find channel argument in options list")
				}
				puzzle, err = air.UpdateVoiceRoom(puzzle, channel.ID)
				if err != nil {
					return "", err
				}
				reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention())
			case "stop":
				puzzle, err = air.UpdateVoiceRoom(puzzle, "")
				if err != nil {
					return "", err
				}
				reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
			default:
				return "", fmt.Errorf("unexpected /room subcommand: %q", i.Subcommand.Name)
			}

			// Work around the three-second deadline. Discord will display
			// "huntbot is thinking..." until this finishes.
			go func() {
				err := voiceAsyncProcessing(air, dis, puzzle)
				if err != nil {
					log.Printf("discord: error handling interaction %q: %s", i.Command, spew.Sdump(err))
					reply = fmt.Sprintf("🚨 Bot Error! Please ping in %s for help.\n```\n%s\n```", dis.TechChannel.Mention(), spew.Sdump(err))
				}
				_, err = s.InteractionResponseEdit(
					s.State.User.ID, i.IC.Interaction, &discordgo.WebhookEdit{
						Content: reply,
					},
				)
				if err != nil {
					log.Printf("discord: error responding to interaction %q: %s", i.Command, spew.Sdump(err))
				} else {
					log.Printf("discord: finished async processing for interaction %q", i.Command)
				}
			}()
			return client.DiscordMagicReplyDefer, nil
		},
	}
}

func voiceAsyncProcessing(air *client.Airtable, dis *client.Discord, puzzle *schema.Puzzle) error {
	if err := voiceSyncPinnedMessage(dis, puzzle); err != nil {
		return err
	}
	puzzles, err := air.FindWithVoiceRoom()
	if err != nil {
		return err
	}
	events, err := dis.ListScheduledEvents()
	if err != nil {
		return err
	}
	return voiceSyncEvents(dis, puzzles, events)
}

func voiceSyncPinnedMessage(dis *client.Discord, puzzle *schema.Puzzle) error {
	msg := "No voice room set. Use `/room start $room` to start working in $room."
	if puzzle.VoiceRoom != "" {
		msg = fmt.Sprintf("Join us in <#%s>!", puzzle.VoiceRoom)
	}

	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: roomStatusHeader},
		Description: msg,
	}
	return dis.CreateUpdatePin(puzzle.DiscordChannel, roomStatusHeader, embed)
}

func voiceSyncEvents(dis *client.Discord, puzzles []*schema.Puzzle, eventsByID map[string]*client.DiscordScheduledEvent) error {
	var groupings = make(map[string][]string)
	for _, puzzle := range puzzles {
		if puzzle.VoiceRoom == "" {
			continue
		}
		groupings[puzzle.VoiceRoom] = append(groupings[puzzle.VoiceRoom], puzzle.Name)
	}

	var eventsByChannel = make(map[string]*client.DiscordScheduledEvent)
	for _, event := range eventsByID {
		if event.Description != eventDescription {
			// Skip events not created by the bot
			continue
		}
		eventsByChannel[event.ChannelID] = event
		if _, ok := groupings[event.ChannelID]; !ok {
			// Event has no more puzzles; delete
			if err := dis.DeleteScheduledEvent(event); err != nil {
				return err
			}
		}
	}

	for voiceRoom, puzzles := range groupings {
		eventTitle := strings.Join(sort.StringSlice(puzzles), " & ")
		if existing, ok := eventsByChannel[voiceRoom]; ok {
			// Update existing event if needed
			if eventTitle != existing.Name {
				_, err := dis.UpdateScheduledEvent(existing, map[string]interface{}{
					"name": eventTitle,
				})
				if err != nil {
					return err
				}
			}
		} else {
			// Create new event

			// TODO: if the event has been stopped manually, drop all puzzles
			// from it, update Airtable and update the pins. This means we need
			// to re-sync from Discord? Or no, log the event ID instead of the
			// voice room...

			event, err := dis.CreateScheduledEvent(&client.DiscordScheduledEvent{
				ChannelID:    voiceRoom,
				Name:         eventTitle,
				PrivacyLevel: 2, // guild-local, the only option
				StartTime:    time.Now().Add(5 * time.Minute),
				Description:  eventDescription,
				EntityType:   2, // voice room
			})
			if err != nil {
				return err
			}
			_, err = dis.UpdateScheduledEvent(event, map[string]interface{}{
				"status": 2, // active (start the event!)
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
