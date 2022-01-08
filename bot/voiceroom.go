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
				var event *discordgo.GuildScheduledEvent
				for _, item := range events {
					if channel != nil && item.ChannelID != nil && *item.ChannelID == channel.ID {
						event = item
					}
				}

				// If there's no existing event in this voice room, create one
				if event == nil && channel != nil {
					log.Printf("creating scheduled event in %s", channel.Name)
					start := time.Now().Add(5 * time.Minute)
					event, err = dis.CreateScheduledEvent(&discordgo.GuildScheduledEvent{
						ChannelID:          &channel.ID,
						Name:               puzzle.Name,
						PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
						ScheduledStartTime: &start,
						Description:        eventDescription,
						EntityType:         discordgo.GuildScheduledEventEntityTypeVoice,
					})
					if err != nil {
						return "", err
					}
					// event.Status = discordgo.GuildScheduledEventStatusActive

					event, err = dis.UpdateScheduledEvent(event, map[string]interface{}{
						"status": discordgo.GuildScheduledEventStatusActive,
					})
					if err != nil {
						return "", err
					}
				}

				// Update Discord and Airtable
				puzzle, err = voiceUpdateAirtableAndPinnedMessage(air, dis, puzzle, event)
				if err != nil {
					return "", err
				}

				// Sync existing events with Airtable
				puzzles, err := air.FindWithVoiceRoomEvent()
				if err != nil {
					return "", err
				}
				var eventsByID = make(map[string]*discordgo.GuildScheduledEvent)
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
						log.Printf("deleting scheduled event %s in %s", event.ID, *event.ChannelID)
						// Mark so we ignore the event-deleted webhook for this
						// action.
						dis.MarkScheduledEventComplete(event)
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

					if event, ok := events[eventID]; !ok {
						// Someone must have stopped the event manually (or
						// Discord stopped it because the voice room emptied for
						// more than a few minutes). Un-assign all of the stale
						// puzzles from the room.
						for _, puzzle := range puzzles {
							_, err = voiceUpdateAirtableAndPinnedMessage(air, dis, puzzle, nil)
							if err != nil {
								return "", err
							}
						}
					} else if eventTitle != event.Name {
						// Update event name
						log.Printf("updating scheduled event %s in %s", event.ID, *event.ChannelID)
						_, err = dis.UpdateScheduledEvent(event, map[string]interface{}{
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

func MakeVoiceRoomScheduledEventUpdateHandler(air *client.Airtable, dis *client.Discord) func(s *discordgo.Session, i *discordgo.GuildScheduledEventUpdate) {
	return func(s *discordgo.Session, i *discordgo.GuildScheduledEventUpdate) {
		if i.Description != eventDescription || i.Status != discordgo.GuildScheduledEventStatusCompleted {
			return
		}

		if !dis.MarkScheduledEventComplete(i.GuildScheduledEvent) {
			log.Printf("discord: ignoring scheduled event completion event: already seen")
			return
		} else {
			log.Printf("discord: processing scheduled event completion event for %q", i.Name)
		}
		puzzles, err := air.FindWithVoiceRoomEvent()
		if err == nil {
			for _, puzzle := range puzzles {
				_, err = voiceUpdateAirtableAndPinnedMessage(air, dis, puzzle, nil)
				if err != nil {
					break
				}
			}
		}
		if err != nil {
			log.Printf("discord: error processing scheduled event completion event: %v", spew.Sdump(err))
		}
	}
}

func voiceUpdateAirtableAndPinnedMessage(air *client.Airtable, dis *client.Discord, puzzle *schema.Puzzle, event *discordgo.GuildScheduledEvent) (*schema.Puzzle, error) {
	// Update pinned message in channel
	msg := "No voice room set. Use `/voice start` to start working in $room."
	eventDesc := "unset"
	if event != nil {
		msg = fmt.Sprintf("Join us in <#%s>!", *event.ChannelID)
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
