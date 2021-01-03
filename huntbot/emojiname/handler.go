package emojiname

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func format(emoji []*Emoji) string {
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
			fmt.Sprintf(":%v:", e.ShortNames[rng.Intn(len(e.ShortNames))]))
		names = append(names, e.Name)
	}
	return fmt.Sprintf(
		"Our team name is %v which you can type like so: `%v` or pronounce like so: %v.",
		strings.Join(chars, ""),
		strings.Join(shortcodes, " "),
		strings.Join(names, " — "),
	)
}

func Handler(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!name") {
		return nil
	}

	reply := func(msg string) error {
		_, err := s.ChannelMessageSend(m.ChannelID, msg)
		return err
	}

	emoji, err := RandomEmoji(3)
	if err != nil {
		// Ignore error with this reply since we are already in an error case.
		reply(":grimacing: something went wrong, @tech can help")
		return fmt.Errorf("failed to get name: %v", err)
	}

	return reply(format(emoji))
}
