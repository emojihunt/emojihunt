package bot

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
)

var roomRE = regexp.MustCompile(`!room (start|stop)(?: (.*))?$`)

const roomStatusHeader = "Working Room"

func MakeVoiceRoomHandler(air *client.Airtable, dis *client.Discord) client.DiscordMessageHandler {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) error {
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!room") {
			return nil
		}

		msg := handleVoiceRoomCommand(air, dis, m)
		_, err := s.ChannelMessageSend(m.ChannelID, msg)
		if err != nil {
			log.Printf("bot/voiceroom: error sending message %q: %v", msg, err)
		}
		return nil
	}
}

func handleVoiceRoomCommand(air *client.Airtable, dis *client.Discord, m *discordgo.MessageCreate) string {
	matches := roomRE.FindStringSubmatch(m.Content)
	if len(matches) != 3 {
		// Not a command
		return fmt.Sprintf("Invalid command %q. Voice command must be of the form \"!room start $room\" or \"!room stop $room\" where $room is a voice channel", m.Content)
	}

	puzzle, err := air.FindByDiscordChannel(m.ChannelID)
	if err != nil {
		return fmt.Sprintf("Unable to get puzzle name for channel ID %q. Contact @tech.", m.ChannelID)
	}

	var rID string
	if matches[2] != "" {
		var ok bool
		rID, ok = dis.ClosestRoomID(matches[2])
		if !ok {
			return fmt.Sprintf("Unable to find room %q. Available rooms are: %v", matches[2], strings.Join(dis.AvailableRooms(), ", "))
		}
	}

	// Note that discord only allows updating a channel name twice per 10 minutes, so this will often take 10+ minutes.
	switch matches[1] {
	case "start":
		if rID == "" {
			return "!room start requires a room"
		}
		setPinnedVoiceInfo(dis, m.ChannelID, &rID)
		return fmt.Sprintf("Set the room for puzzle %q to <#%s>", puzzle.Name, rID)
	case "stop":
		setPinnedVoiceInfo(dis, m.ChannelID, nil)
		return fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
	default:
		return fmt.Sprintf("Unrecognized voice bot action %q. Valid commands are \"!room start $RoomName\" or \"!room start $RoomName\"", m.Content)
	}
}

func setPinnedVoiceInfo(dis *client.Discord, puzzleChannelID string, voiceChannelID *string) error {
	room := "No voice room set. \"!room start $room\" to start working in $room."
	if voiceChannelID != nil {
		room = fmt.Sprintf("Join us in <#%s>!", *voiceChannelID)
	}
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: roomStatusHeader},
		Description: room,
	}

	return dis.CreateUpdatePin(puzzleChannelID, roomStatusHeader, embed)
}
