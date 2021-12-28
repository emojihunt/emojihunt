package voiceroom

import (
	"fmt"
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

		puzzle, err := air.FindByDiscordChannel(m.ChannelID)
		if err != nil {
			reply = fmt.Sprintf("Unable to get puzzle name for channel ID %q. Contact @tech.", m.ChannelID)
			return fmt.Errorf("unable to get puzzle name for channel ID %q: %v", m.ChannelID, err)
		}

		var rID string
		if matches[2] != "" {
			_, ok := dis.ClosestRoomID(matches[2])
			if !ok {
				reply = fmt.Sprintf("Unable to find room %q. Available rooms are: %v", matches[2], strings.Join(dis.AvailableRooms(), ", "))
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
			setPinnedVoiceInfo(dis, m.ChannelID, &rID)
			reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, dis.ChannelMention(rID))
		case "stop":
			setPinnedVoiceInfo(dis, m.ChannelID, nil)
			reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
		default:
			reply = fmt.Sprintf("Unrecognized voice bot action %q. Valid commands are \"!room start $RoomName\" or \"!room start $RoomName\"", m.Content)
			return fmt.Errorf("impossible voice bot action %q: %q", matches[1], m.Content)
		}

		return nil
	}
}

func setPinnedVoiceInfo(dis *client.Discord, puzzleChannelID string, voiceChannelID *string) (didUpdate bool, err error) {
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
