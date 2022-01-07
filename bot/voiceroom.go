package bot

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
)

const (
	roomStatusHeader = "Working Room"
	eventDescription = "🤖 Event managed by Huntbot. Use `/voice` to modify!"
)

func MakeVoiceRoomCommand(air *client.Airtable, dis *client.Discord) *client.DiscordCommand {
	return &client.DiscordCommand{
		InteractionType: discordgo.InteractionApplicationCommand,
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "voice",
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
			var channel *discordgo.Channel
			switch i.Subcommand.Name {
			case "start":
				channelOpt, err := dis.OptionByName(i.Subcommand.Options, "in")
				if err != nil {
					return "", err
				}
				channel = channelOpt.ChannelValue(s)
				reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention())
			case "stop":
				reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
			default:
				return "", fmt.Errorf("unexpected /voice subcommand: %q", i.Subcommand.Name)
			}

			return dis.ReplyAsync(s, i, func() (string, error) {
				// Search for an existing event for the given voice room
				events, err := dis.ListScheduledEvents()
				if err != nil {
					return "", err
				}
				var event *client.DiscordScheduledEvent
				for _, item := range events {
					if channel != nil && item.ChannelID == channel.ID {
						event = item
					}
				}

				// If there's no existing event in this voice room, create one
				if event == nil && channel != nil {
					log.Printf("creating scheduled event in %s", channel.Name)
					event, err = dis.CreateScheduledEvent(&client.DiscordScheduledEvent{
						ChannelID:    channel.ID,
						Name:         puzzle.Name,
						PrivacyLevel: 2, // guild-local, the only option
						StartTime:    time.Now().Add(5 * time.Minute),
						Description:  eventDescription,
						EntityType:   2, // voice room
					})
					if err != nil {
						return "", err
					}
					event, err = dis.UpdateScheduledEvent(event, map[string]interface{}{
						"status": 2, // active (start the event!)
					})
					if err != nil {
						return "", err
					}
				}

				// Update Discord and Airtable
				puzzle, err = voiceUpdateAirtableAndPinnedMessage(dis, air, puzzle, event)
				if err != nil {
					return "", err
				}

				// Sync existing events with Airtable
				puzzles, err := air.FindWithVoiceRoomEvent()
				if err != nil {
					return "", err
				}
				var eventsByID = make(map[string]*client.DiscordScheduledEvent)
				var puzzlesByEvent = make(map[string][]*schema.Puzzle)
				for _, puzzle := range puzzles {
					if puzzle.VoiceRoomEvent == "" {
						continue
					}
					puzzlesByEvent[puzzle.VoiceRoomEvent] = append(puzzlesByEvent[puzzle.VoiceRoomEvent], puzzle)
				}
				for _, event := range events {
					if event.Description != eventDescription {
						// Skip events not created by the bot
						continue
					}
					if _, ok := puzzlesByEvent[event.ID]; !ok {
						// Event has no more puzzles; delete
						log.Printf("deleting scheduled event %s in %s", event.ID, event.ChannelID)
						if err := dis.DeleteScheduledEvent(event); err != nil {
							return "", err
						}
					}
					eventsByID[event.ID] = event
				}
				for eventID, puzzles := range puzzlesByEvent {
					var puzzleNames []string
					for _, puzzle := range puzzles {
						puzzleNames = append(puzzleNames, puzzle.Name)
					}
					eventTitle := strings.Join(sort.StringSlice(puzzleNames), " & ")

					if existing, ok := events[eventID]; !ok {
						// Someone must have stopped the event manually (or
						// Discord stopped it because the voice room emptied for
						// more than a few minutes). Un-assign all of the stale
						// puzzles from the room.
						for _, puzzle := range puzzles {
							_, err = voiceUpdateAirtableAndPinnedMessage(dis, air, puzzle, nil)
							if err != nil {
								return "", err
							}
						}
					} else if eventTitle != existing.Name {
						// Update event name
						log.Printf("updating scheduled event %s in %s", event.ID, event.ChannelID)
						_, err := dis.UpdateScheduledEvent(existing, map[string]interface{}{
							"name": eventTitle,
						})
						if err != nil {
							return "", err
						}
					}
				}
				return reply, nil
			})
		},
	}
}

func voiceUpdateAirtableAndPinnedMessage(dis *client.Discord, air *client.Airtable, puzzle *schema.Puzzle, event *client.DiscordScheduledEvent) (*schema.Puzzle, error) {
	// Update pinned message in channel
	msg := "No voice room set. Use `/voice start` to start working in $room."
	eventDesc := "unset"
	if event != nil {
		msg = fmt.Sprintf("Join us in <#%s>!", event.ChannelID)
		eventDesc = event.ID
	}
	log.Printf("updating airtable and pinned message for %q: event %s", puzzle.Name, eventDesc)
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: roomStatusHeader},
		Description: msg,
	}
	err := dis.CreateUpdatePin(puzzle.DiscordChannel, roomStatusHeader, embed)
	if err != nil {
		return nil, err
	}

	// Update Airtable with new event
	var eventID string
	if event != nil {
		eventID = event.ID
	}
	return air.UpdateVoiceRoomEvent(puzzle, eventID)
}
