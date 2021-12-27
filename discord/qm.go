package discord

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (c *Client) QMHandler(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!qm") || m.ChannelID != c.qmChannelID {
		return nil
	}

	var reply string
	var err error
	switch m.Content {
	case "!qm start":
		if err = s.GuildMemberRoleAdd(c.guildID, m.Author.ID, c.qmRoleID); err != nil {
			reply = fmt.Sprintf("unable to make %s a QM: %v", m.Author.Mention(), err)
			break
		}
		reply = fmt.Sprintf("%s is now a QM", m.Author.Mention())
	case "!qm stop":
		if err = s.GuildMemberRoleRemove(c.guildID, m.Author.ID, c.qmRoleID); err != nil {
			reply = fmt.Sprintf("unable to remove %s from QM role: %v", m.Author.Mention(), err)
			break
		}
		reply = fmt.Sprintf("%s is no longer a QM", m.Author.Mention())
	default:
		err = fmt.Errorf("unexpected QM command: %q", m.Content)
		reply = fmt.Sprintf("unexpected command: %q\nsupported qm commands are \"!qm start\" and \"!qm stop\"", m.Content)
	}
	if err != nil {
		log.Printf("error setting QM: %v", err)
	}
	_, err = s.ChannelMessageSend(m.ChannelID, reply)
	return err
}
