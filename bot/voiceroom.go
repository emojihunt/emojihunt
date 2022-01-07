package bot

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
)

const (
	roomStatusHeader = "Working Room"
	eventDescription = "🤖 Event managed by Huntbot. Use `/name` to modify!"
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
			var newChannelID string
			switch i.Subcommand.Name {
			case "start":
				channelOpt, err := dis.OptionByName(i.Subcommand.Options, "in")
				if err != nil {
					return "", err
				}
				channel := channelOpt.ChannelValue(s)
				newChannelID = channel.ID
				reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention())
			case "stop":
				puzzle, err = air.UpdateVoiceRoom(puzzle, "")
				if err != nil {
					return "", err
				}
				reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
			default:
				return "", fmt.Errorf("unexpected /voice subcommand: %q", i.Subcommand.Name)
			}

			return dis.ReplyAsync(s, i, func() (string, error) {
				puzzle, err := air.UpdateVoiceRoom(puzzle, newChannelID)
				if err != nil {
					return "", err
				}
				err = voiceSyncPinnedMessage(dis, puzzle)
				if err != nil {
					return "", err
				}
				puzzles, err := air.FindWithVoiceRoom()
				if err != nil {
					return "", err
				}
				events, err := dis.ListScheduledEvents()
				if err != nil {
					return "", err
				}
				err = voiceSyncEvents(air, dis, puzzles, puzzle, events)
				if err != nil {
					return "", err
				}
				return reply, nil
			})
		},
	}
}

func voiceSyncPinnedMessage(dis *client.Discord, puzzle *schema.Puzzle) error {
	msg := "No voice room set. Use `/voice start` to start working in $room."
	if puzzle.VoiceRoom != "" {
		msg = fmt.Sprintf("Join us in <#%s>!", puzzle.VoiceRoom)
	}

	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: roomStatusHeader},
		Description: msg,
	}
	return dis.CreateUpdatePin(puzzle.DiscordChannel, roomStatusHeader, embed)
}

func voiceSyncEvents(air *client.Airtable, dis *client.Discord, puzzles []*schema.Puzzle, newest *schema.Puzzle, eventsByID map[string]*client.DiscordScheduledEvent) error {
	var groupings = make(map[string][]*schema.Puzzle)
	for _, puzzle := range puzzles {
		if puzzle.VoiceRoom == "" {
			continue
		}
		groupings[puzzle.VoiceRoom] = append(groupings[puzzle.VoiceRoom], puzzle)
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
		var puzzleNames []string
		for _, puzzle := range puzzles {
			puzzleNames = append(puzzleNames, puzzle.Name)
		}
		eventTitle := strings.Join(sort.StringSlice(puzzleNames), " & ")

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
			if len(puzzles) > 1 {
				// There are other puzzles assigned to this room, but there's no
				// event. Someone must have stopped the event manually (or
				// Discord stopped it because the voice room emptied for more
				// than a few minutes). Handle this by un-assigning all of the
				// stale puzzles from the room.
				for _, puzzle := range puzzles {
					if puzzle.AirtableRecord.ID == newest.AirtableRecord.ID {
						continue
					}
					var err error
					if puzzle, err = air.UpdateVoiceRoom(puzzle, ""); err != nil {
						return err
					}
					if err = voiceSyncPinnedMessage(dis, puzzle); err != nil {
						return err
					}
				}
				eventTitle = newest.Name
			}

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
