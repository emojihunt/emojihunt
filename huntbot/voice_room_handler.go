package huntbot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var roomRE = regexp.MustCompile(`!room (start|stop)(?: (.*))?$`)

func (h *HuntBot) VoiceRoomHandler(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!room") {
		return nil
	}

	// TODO: reply errors are not caught.
	var reply string
	defer func(reply *string) {
		if *reply == "" {
			return
		}
		s.ChannelMessageSend(m.ChannelID, *reply)
	}(&reply)

	matches := roomRE.FindStringSubmatch(m.Content)
	if len(matches) != 3 {
		// Not a command
		reply = fmt.Sprintf("Invalid command %q. Voice command must be of the form \"!room start $room\" or \"!room stop $room\" where $room is a voice channel", m.Content)
		return nil
	}

	h.mu.Lock()
	puzzle, ok := h.channelToPuzzle[m.ChannelID]
	h.mu.Unlock()
	if !ok {
		reply = fmt.Sprintf("Unable to get puzzle name for channel ID %q. Contact @tech.", m.ChannelID)
		return fmt.Errorf("unable to get puzzle name for channel ID %q", m.ChannelID)
	}

	var rID string
	if matches[2] != "" {
		rID, ok = h.discord.ClosestRoomID(matches[2])
		if !ok {
			reply = fmt.Sprintf("Unable to find room %q. Available rooms are: %v", matches[2], strings.Join(h.discord.AvailableRooms(), ", "))
			return nil
		}
	}

	// Note that discord only allows updating a channel name twice per 10 minutes, so this will often take 10+ minutes.
	switch matches[1] {
	case "start":
		if rID == "" {
			reply = "!room start requires a room"
			return fmt.Errorf("missing room ID from command: %s", m.Content)
		}
		if h.cfg.UpdateRooms {
			if rID == "" {
				reply = "!room start requires a room"
				return fmt.Errorf("missing room ID from command: %s", m.Content)
			}
			updated, err := h.discord.AddPuzzleToRoom(puzzle, rID)
			if err != nil {
				reply = "error updating room name, contact @tech."
				return err
			}
			if !updated {
				reply = fmt.Sprintf("Puzzle %q is already in room %s", puzzle, h.discord.ChannelMention(rID))
				return nil
			}
		}
		h.setPinnedVoiceInfo(m.ChannelID, &rID)
		reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle, h.discord.ChannelMention(rID))
	case "stop":
		if h.cfg.UpdateRooms {
			if rID == "" {
				reply = "!room stop requires a room to update room names"
				return fmt.Errorf("missing room ID from command: %s", m.Content)
			}
			updated, err := h.discord.RemovePuzzleFromRoom(puzzle, rID)
			if err != nil {
				reply = "error updating room name, contact @tech."
				return err
			}
			if !updated {
				reply = fmt.Sprintf("Puzzle %q was already not in room %s", puzzle, h.discord.ChannelMention(rID))
				return nil
			}
		}
		h.setPinnedVoiceInfo(m.ChannelID, nil)
		reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle)
	default:
		reply = fmt.Sprintf("Unrecognized voice bot action %q. Valid commands are \"!room start $RoomName\" or \"!room start $RoomName\"", m.Content)
		return fmt.Errorf("impossible voice bot action %q: %q", matches[1], m.Content)
	}

	return nil
}

const roomStatusHeader = "Working Room"

func (h *HuntBot) setPinnedVoiceInfo(puzzleChannelID string, voiceChannelID *string) (didUpdate bool, err error) {
	room := "No voice room set. \"!room start $room\" to start working in $room."
	if voiceChannelID != nil {
		room = fmt.Sprintf("Join us in <#%s>!", *voiceChannelID)
	}
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: roomStatusHeader},
		Description: room,
	}

	return h.discord.CreateUpdatePin(puzzleChannelID, roomStatusHeader, embed)
}
