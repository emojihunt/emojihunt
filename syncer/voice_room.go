package syncer

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/schema"
)

const (
	VoiceRoomEventDescription = "🤖 Event managed by Huntbot. Use `/voice` to modify!"
)

// SyncVoiceRooms synchronizes all Discord scheduled events with Airtable,
// creating and deleting events so that Discord matches the state in Airtable.
//
// The caller *must* acquire VoiceRoomMutex before calling this function.
//
func (s *Syncer) SyncVoiceRooms(ctx context.Context) error {
	log.Printf("syncer: syncing voice rooms")
	events, err := s.discord.ListScheduledEvents()
	if err != nil {
		return err
	}
	puzzles, err := s.airtable.ListWithVoiceRoom()
	if err != nil {
		return err
	}

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
		} else if event.Status != discordgo.GuildScheduledEventStatusActive {
			// Skip completed and canceled events
			continue
		}
		if _, ok := puzzlesByChannel[*event.ChannelID]; !ok {
			// Event has no more puzzles; delete
			log.Printf("deleting scheduled event %s in %s", event.ID, *event.ChannelID)
			if err := s.discord.DeleteScheduledEvent(event); err != nil {
				return err
			}
		}
		eventsByChannel[*event.ChannelID] = event
	}
	for channelID, puzzles := range puzzlesByChannel {
		var puzzleNames []string
		for _, puzzle := range puzzles {
			puzzleNames = append(puzzleNames, puzzle.Name)
		}
		eventTitle := strings.Join(sort.StringSlice(puzzleNames), " & ")

		if event, ok := eventsByChannel[channelID]; !ok {
			// If there's no existing event for this voice room, create one
			log.Printf("creating scheduled event in %q", channelID)
			start := time.Now().Add(5 * time.Minute)
			event, err = s.discord.CreateScheduledEvent(&discordgo.GuildScheduledEvent{
				ChannelID:          &channelID,
				Name:               eventTitle,
				PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
				ScheduledStartTime: &start,
				Description:        VoiceRoomEventDescription,
				EntityType:         discordgo.GuildScheduledEventEntityTypeVoice,
			})
			if err != nil {
				return err
			}
			event, err = s.discord.UpdateScheduledEvent(event, map[string]interface{}{
				// These fields are duplicative, but Discord occasionally
				// appears to create the event with some fields missing
				// (maybe a Discord bug?) so let's try and set them again to
				// be sure.
				"channel_id":  channelID,
				"name":        eventTitle,
				"description": VoiceRoomEventDescription,

				// Start the event!
				"status": discordgo.GuildScheduledEventStatusActive,
			})
			if event.Status != discordgo.GuildScheduledEventStatusActive {
				log.Printf("Warning! UpdateScheduledEvent failed to start event: %v", event)
			}
			if err != nil {
				return err
			}
		} else if eventTitle != event.Name {
			// Update event name
			log.Printf("updating scheduled event %s in %s", event.ID, *event.ChannelID)
			_, err = s.discord.UpdateScheduledEvent(event, map[string]interface{}{
				"name": eventTitle,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
