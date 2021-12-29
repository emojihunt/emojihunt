package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/emojiname"
)

func MakeEmojiNameHandler() client.DiscordMessageHandler {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) error {
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!name") {
			return nil
		}

		reply := func(msg string) error {
			_, err := s.ChannelMessageSend(m.ChannelID, msg)
			return err
		}

		emoji, err := emojiname.RandomEmoji(3)
		if err != nil {
			// Ignore error with this reply since we are already in an error case.
			reply(":grimacing: something went wrong, @tech can help")
			return fmt.Errorf("failed to get name: %v", err)
		}

		return reply(format(emoji))
	}
}

func format(emoji []*emojiname.Emoji) string {
	var chars, names []string
	for _, e := range emoji {
		for _, hex := range strings.Split(e.Unified, "-") {
			n, err := strconv.ParseInt(hex, 16, 32)
			if err != nil {
				log.Printf("bad unicode char %v in %v: %v", hex, e.Unified, err)
				n = 0xfffd // �
			}
			chars = append(chars, string(rune(n)))
		}
		names = append(names, e.Name)
	}
	return fmt.Sprintf(
		"Our team name is %v which you can pronounce like so: %v.",
		strings.Join(chars, ""),
		strings.Join(names, " — "),
	)
}
