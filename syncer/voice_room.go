package syncer

import (
	"context"
	"log"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

const (
	VoiceRoomEventDescription = "🤖 Event managed by Huntbot. Use `/voice` to modify!"
)

// SyncVoiceRooms synchronizes all Discord scheduled events with Airtable. If
// any Airtable puzzles reference an event that's been deleted or completed in
// Discord, the field will be cleared in Airtable. Otherwise, the scheduled
// event in Discord will be updated to match Airtable.
//
// The caller *must* acquire VoiceRoomMutex before calling this function.
//
func (s *Syncer) SyncVoiceRooms(ctx context.Context) error {
	events, err := s.discord.ListScheduledEvents()
	if err != nil {
		return err
	}

	puzzles, err := s.airtable.ListWithVoiceRoom()
	if err != nil {
		return err
	}
	var eventsByChannel = make(map[string]*discordgo.GuildScheduledEvent)
	var puzzlesByChannel = make(map[string][]*schema.Puzzle)
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

		if event, ok := events[channelID]; !ok {
			// Someone must have stopped the event manually (or
			// Discord stopped it because the voice room emptied for
			// more than a few minutes). Un-assign all of the stale
			// puzzles from the room.
			for _, puzzle := range puzzles {
				_, err = s.airtable.UpdateVoiceRoom(puzzle, nil)
				if err != nil {
					return err
				}
				if err = s.DiscordCreateUpdatePin(puzzle); err != nil {
					return err
				}
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
