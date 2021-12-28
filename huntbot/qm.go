package huntbot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
)

func GetQMHandler(c *client.Discord) func(*discordgo.Session, *discordgo.MessageCreate) error {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) error {
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!qm") || m.ChannelID != c.QMChannelID {
			return nil
		}

		var reply string
		var err error
		switch m.Content {
		case "!qm start":
			if err = s.GuildMemberRoleAdd(c.GuildID, m.Author.ID, c.QMRoleID); err != nil {
				reply = fmt.Sprintf("unable to make %s a QM: %v", m.Author.Mention(), err)
				break
			}
			reply = fmt.Sprintf("%s is now a QM", m.Author.Mention())
		case "!qm stop":
			if err = s.GuildMemberRoleRemove(c.GuildID, m.Author.ID, c.QMRoleID); err != nil {
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
}
