package sync

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

const (
	VoiceRoomEventDescription = "🤖 Event managed by Huntbot"
	VoiceRoomPlaceholderTitle = "🫥 Placeholder Event"

	eventDelay = 7 * 24 * time.Hour
)

type VoiceRoomFields struct {
	VoiceRoom string
	Name      string // only if VoiceRoom is set
}

func NewVoiceRoomFields(puzzle state.Puzzle) VoiceRoomFields {
	if puzzle.VoiceRoom == "" {
		return VoiceRoomFields{}
	} else {
		return VoiceRoomFields{
			VoiceRoom: puzzle.VoiceRoom,
			Name:      puzzle.Name,
		}
	}
}

// SyncVoiceRooms synchronizes all Discord scheduled events, creating and
// deleting events so that Discord matches the database state.
func (c *Client) SyncVoiceRooms(ctx context.Context) error {
	log.Printf("sync: syncing voice rooms")
	events, err := c.discord.ListScheduledEvents()
	if err != nil {
		return err
	}
	infos, err := c.state.ListVoiceRoomInfo(ctx)
	if err != nil {
		return err
	}

	var placeholderEvents []*discordgo.GuildScheduledEvent
	var puzzlesByChannel = make(map[string][]state.VoiceInfo)
	var eventsByChannel = make(map[string]*discordgo.GuildScheduledEvent)
	for _, puzzle := range infos {
		if puzzle.VoiceRoom == "" {
			continue
		}
		puzzlesByChannel[puzzle.VoiceRoom] = append(puzzlesByChannel[puzzle.VoiceRoom], puzzle)
	}
	for _, event := range events {
		if event.Description != VoiceRoomEventDescription {
			// Skip events not created by the bot
			continue
		} else if event.Name == VoiceRoomPlaceholderTitle {
			// Collect placeholder events
			placeholderEvents = append(placeholderEvents, event)
			continue
		} else if event.Status != discordgo.GuildScheduledEventStatusActive {
			// Skip completed and canceled events
			continue
		}
		if _, ok := puzzlesByChannel[event.ChannelID]; !ok {
			// Event has no more puzzles; delete
			log.Printf("deleting scheduled event %s in %s", event.ID, event.ChannelID)
			if err := c.discord.DeleteScheduledEvent(event); err != nil {
				return err
			}
		}
		eventsByChannel[event.ChannelID] = event
	}
	for channelID, puzzles := range puzzlesByChannel {
		ch, ok := c.discord.GetChannel(channelID)
		if !ok || ch.Type != discordgo.ChannelTypeGuildVoice {
			log.Printf("skipping sync of invalid voice room %q", channelID)
			for _, puzzle := range puzzles {
				c.CheckDiscordVoiceRoom(ctx, puzzle)
			}
			continue
		}

		var puzzleNames []string
		for _, puzzle := range puzzles {
			puzzleNames = append(puzzleNames, puzzle.Name)
		}
		eventTitle := strings.Join(sort.StringSlice(puzzleNames), " & ")

		if event, ok := eventsByChannel[channelID]; !ok {
			if len(placeholderEvents) > 0 {
				// ...take a placeholder event if available
				log.Printf("activating placeholder event in %q", channelID)
				event, placeholderEvents = placeholderEvents[0], placeholderEvents[1:]

				if event.ChannelID != channelID {
					// (changing the event's voice channel can't be done in the same call
					// as starting the event, apparently)
					event, err = c.discord.UpdateScheduledEvent(event,
						&discordgo.GuildScheduledEventParams{
							ChannelID: channelID,
						},
					)
					if err != nil {
						return err
					}
				}
			} else {
				// ...otherwise create a new event
				log.Printf("creating scheduled event in %q", channelID)
				start := time.Now().Add(eventDelay)
				event, err = c.discord.CreateScheduledEvent(&discordgo.GuildScheduledEventParams{
					ChannelID:          channelID,
					Name:               eventTitle,
					PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
					ScheduledStartTime: &start,
					Description:        VoiceRoomEventDescription,
					EntityType:         discordgo.GuildScheduledEventEntityTypeVoice,
				})
				if err != nil {
					return err
				}
			}

			// Then start the event
			event, err = c.discord.UpdateScheduledEvent(event,
				&discordgo.GuildScheduledEventParams{
					// FYI, we pass these fields again because Discord has (had?) a bug
					// where events are sometimes created with fields missing.
					ChannelID:   channelID,
					Name:        eventTitle,
					Description: VoiceRoomEventDescription,

					// Start the event!
					Status: discordgo.GuildScheduledEventStatusActive,
				},
			)
			if err != nil {
				return err
			} else if event.Status != discordgo.GuildScheduledEventStatusActive {
				return xerrors.Errorf("UpdateScheduledEvent failed to start event %q", eventTitle)
			}
		} else if eventTitle != event.Name {
			// Update event name
			log.Printf("updating scheduled event %s in %s", event.ID, event.ChannelID)
			_, err = c.discord.UpdateScheduledEvent(event,
				&discordgo.GuildScheduledEventParams{
					Name: eventTitle,
				},
			)
			if err != nil {
				return err
			}
		}
	}

	go func() {
		err := c.RestorePlaceholderEvent()
		if err != nil {
			log.Printf("error restoring placeholder event: %v", err)
		}
	}()
	return nil
}

func (c *Client) RestorePlaceholderEvent() error {
	events, err := c.discord.ListScheduledEvents()
	if err != nil {
		return err
	}

	var placeholderEvents []*discordgo.GuildScheduledEvent
	for _, event := range events {
		if event.Description != VoiceRoomEventDescription {
			// Skip events not created by the bot
			continue
		} else if event.Name == VoiceRoomPlaceholderTitle {
			// Collect placeholder events
			placeholderEvents = append(placeholderEvents, event)
			continue
		}
	}
	if len(placeholderEvents) > 0 {
		return nil
	}

	log.Printf("sync: restoring voice room placeholder event")
	var someVoiceChannelID string
	for channel := range c.discord.ListVoiceChannels() {
		someVoiceChannelID = channel
		break
	}
	if someVoiceChannelID == "" {
		return xerrors.Errorf("sync: at least one voice channel is required")
	}
	start := time.Now().Add(eventDelay)
	_, err = c.discord.CreateScheduledEvent(&discordgo.GuildScheduledEventParams{
		ChannelID:          someVoiceChannelID,
		Name:               VoiceRoomPlaceholderTitle,
		PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
		ScheduledStartTime: &start,
		Description:        VoiceRoomEventDescription,
		EntityType:         discordgo.GuildScheduledEventEntityTypeVoice,
	})
	return err
}
