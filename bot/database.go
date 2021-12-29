package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/database"
)

func MakeDatabaseHandler(discord *client.Discord, poller *database.Poller) client.DiscordMessageHandler {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) error {
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!huntbot") {
			return nil
		}

		reply := ""
		info := ""
		switch m.Content {
		case "!huntbot kill":
			if poller.Enable(false) {
				reply = `Ok, I've disabled the bot for now.  Enable it with "!huntbot start".`
				info = fmt.Sprintf("**bot disabled by %v**", m.Author.Mention())
			} else {
				reply = `The bot was already disabled.  Enable it with "!huntbot start".`
			}
		case "!huntbot start":
			if poller.Enable(true) {
				reply = `Ok, I've enabled the bot for now.  Disable it with "!huntbot kill".`
				info = fmt.Sprintf("**bot enabled by %v**", m.Author.Mention())
			} else {
				reply = `The bot was already enabled.  Disable it with "!huntbot kill".`
			}
		default:
			reply = `I'm not sure what you mean.  Disable the bot with "!huntbot kill" ` +
				`or enable it with "!huntbot start".`
		}

		s.ChannelMessageSend(m.ChannelID, reply)
		if info != "" {
			discord.TechChannelSend(info)
			log.Print(info)
		}

		return nil
	}
}
