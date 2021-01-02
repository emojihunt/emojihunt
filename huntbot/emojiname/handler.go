package emojiname

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func format(emoji []Emoji) string {
	var chars, shortcodes, names []string
	for _, e := range emoji {
		for _, hex := range strings.Split(e.Unified, "-") {
			n, err := strconv.ParseInt(hex, 16, 32)
			if err != nil {
				log.Printf("bad unicode char %v in %v: %v", hex, e.Unified, err)
				n = 0xfffd // �
			}
			chars = append(chars, string(rune(n)))
		}
		shortcodes = append(shortcodes,
			fmt.Sprintf(":%v:", e.ShortNames[rand.Intn(len(e.ShortNames))]))
		names = append(names, e.Name)
	}
	return fmt.Sprintf(
		"Our team name is %v which you can type like so: `%v` or pronounce like so: %v.",
		strings.Join(chars, ""),
		strings.Join(shortcodes, " "),
		strings.Join(names, " — "),
	)
}

func Handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!name") {
		return
	}

	reply := func(msg string) { s.ChannelMessageSend(m.ChannelID, msg) }

	emoji, err := RandomEmoji(3)
	if err != nil {
		log.Printf("failed to get name: %v", err)
		reply(":grimacing: something went wrong, @tech can help")
		return
	}

	reply(format(emoji))
}
