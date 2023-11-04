package syncer

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/schema"
	"golang.org/x/xerrors"
)

const (
	VoiceRoomEventDescription = "🤖 Event managed by Huntbot. Use `/voice` to modify!"
	VoiceRoomPlaceholderTitle = "🫥 Placeholder Event"

	eventDelay = 7 * 24 * time.Hour
)

// SyncVoiceRooms synchronizes all Discord scheduled events, creating and
// deleting events so that Discord matches the database state.
//
// The caller *must* acquire VoiceRoomMutex before calling this function.
func (s *Syncer) SyncVoiceRooms(ctx context.Context) error {
	log.Printf("syncer: syncing voice rooms")
	events, err := s.discord.ListScheduledEvents()
	if err != nil {
		return err
	}
	puzzles, err := s.db.ListWithVoiceRoom(ctx)
	if err != nil {
		return err
	}

	var placeholderEvents []*discordgo.GuildScheduledEvent
	var puzzlesByChannel = make(map[string][]schema.VoicePuzzle)
	var eventsByChannel = make(map[string]*discordgo.GuildScheduledEvent)
	for _, puzzle := range puzzles {
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
			if err := s.discord.DeleteScheduledEvent(event); err != nil {
				return err
			}
		}
		eventsByChannel[event.ChannelID] = event
	}
	for channelID, puzzles := range puzzlesByChannel {
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
					// (changing the event's voice channel can't be done in the
					// same call as starting the event, apparently)
					event, err = s.discord.UpdateScheduledEvent(event, map[string]interface{}{
						"channel_id": channelID,
					})
					if err != nil {
						return err
					}
				}
			} else {
				// ...otherwise create a new event
				log.Printf("creating scheduled event in %q", channelID)
				start := time.Now().Add(eventDelay)
				event, err = s.discord.CreateScheduledEvent(&discordgo.GuildScheduledEventParams{
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
			event, err = s.discord.UpdateScheduledEvent(event, map[string]interface{}{
				// FYI, we pass these fields again because Discord has (had?) a
				// bug where events are sometimes created with fields missing.
				"channel_id":  channelID,
				"name":        eventTitle,
				"description": VoiceRoomEventDescription,

				// Start the event!
				"status": discordgo.GuildScheduledEventStatusActive,
			})
			if err != nil {
				return err
			} else if event.Status != discordgo.GuildScheduledEventStatusActive {
				return xerrors.Errorf("UpdateScheduledEvent failed to start event: %v", event)
			}
		} else if eventTitle != event.Name {
			// Update event name
			log.Printf("updating scheduled event %s in %s", event.ID, event.ChannelID)
			_, err = s.discord.UpdateScheduledEvent(event, map[string]interface{}{
				"name": eventTitle,
			})
			if err != nil {
				return err
			}
		}
	}

	// Will block until the caller releases VoiceRoomMutex
	go s.RestorePlaceholderEvent()
	return nil
}

func (s *Syncer) RestorePlaceholderEvent() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RestorePlaceholderEvent: %v", err)
		}
	}()

	s.VoiceRoomMutex.Lock()
	defer s.VoiceRoomMutex.Unlock()

	events, err := s.discord.ListScheduledEvents()
	if err != nil {
		panic(err)
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
		return
	}

	start := time.Now().Add(eventDelay)
	_, err = s.discord.CreateScheduledEvent(&discordgo.GuildScheduledEventParams{
		ChannelID:          s.discord.DefaultVoiceChannel.ID,
		Name:               VoiceRoomPlaceholderTitle,
		PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
		ScheduledStartTime: &start,
		Description:        VoiceRoomEventDescription,
		EntityType:         discordgo.GuildScheduledEventEntityTypeVoice,
	})
	if err != nil {
		panic(err)
	}
}
