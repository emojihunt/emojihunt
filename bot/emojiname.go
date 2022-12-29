package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/emojiname"
)

func RegisterEmojiNameBot(discord *client.Discord) {
	var bot = emojiNameBot{discord}
	discord.AddCommand(bot.makeSlashCommand())
}

type emojiNameBot struct {
	discord *client.Discord
}

func (bot *emojiNameBot) makeSlashCommand() *client.DiscordCommand {
	return &client.DiscordCommand{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "name",
			Description: "Generate a team name 🌊🎨🎡",
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			var chars, names []string
			emoji, err := emojiname.RandomEmoji(3)
			if err != nil {
				return "", err
			}
			for _, e := range emoji {
				for _, hex := range strings.Split(e.Unified, "-") {
					n, err := strconv.ParseInt(hex, 16, 32)
					if err != nil {
						return "", fmt.Errorf("bad unicode char %v in %v: %v", hex, e.Unified, err)
					}
					chars = append(chars, string(rune(n)))
				}
				names = append(names, e.Name)
			}
			return fmt.Sprintf(
				"Our team name is %s which you can pronounce like so: `%s`.",
				strings.Join(chars, ""),
				strings.Join(names, " — "),
			), nil
		},
	}
}
