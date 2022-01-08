package syncer

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

const (
	VoiceRoomStatusHeader     = "Working Room"
	VoiceRoomEventDescription = "🤖 Event managed by Huntbot. Use `/voice` to modify!"
)

func (s *Syncer) SetVoiceRoomNoSync(puzzle *schema.Puzzle, event *discordgo.GuildScheduledEvent) (*schema.Puzzle, error) {
	// Update pinned message in channel
	msg := "No voice room set. Use `/voice start` to start working in $room."
	eventDesc := "unset"
	if event != nil {
		msg = fmt.Sprintf("Join us in <#%s>!", *event.ChannelID)
		eventDesc = event.ID
	}
	log.Printf("updating airtable and pinned message for %q: event %s", puzzle.Name, eventDesc)
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: VoiceRoomStatusHeader},
		Description: msg,
	}
	err := s.discord.CreateUpdatePin(puzzle.DiscordChannel, VoiceRoomStatusHeader, embed)
	if err != nil {
		return nil, err
	}

	// Update Airtable with new event
	var eventID string
	if event != nil {
		eventID = event.ID
	}
	return s.airtable.UpdateVoiceRoomEvent(puzzle, eventID)
}

func (s *Syncer) SyncVoiceRooms() error {
	events, err := s.discord.ListScheduledEvents()
	if err != nil {
		return err
	}

	puzzles, err := s.airtable.FindWithVoiceRoomEvent()
	if err != nil {
		return err
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
		if event.Description != VoiceRoomEventDescription {
			// Skip events not created by the bot
			continue
		}
		if _, ok := puzzlesByEvent[event.ID]; !ok {
			// Event has no more puzzles; delete
			log.Printf("deleting scheduled event %s in %s", event.ID, *event.ChannelID)
			if err := s.discord.DeleteScheduledEvent(event); err != nil {
				return err
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
				_, err = s.SetVoiceRoomNoSync(puzzle, nil)
				if err != nil {
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
