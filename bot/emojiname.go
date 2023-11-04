package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/emojiname"
)

type EmojiNameBot struct{}

func NewEmojiNameBot() discord.Bot {
	return &EmojiNameBot{}
}

func (b *EmojiNameBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "name",
		Description: "Generate a team name 🌊🎨🎡",
	}, false
}

func (b *EmojiNameBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	var chars, names []string
	emoji, err := emojiname.RandomEmoji(3)
	if err != nil {
		return "", err
	}

	for _, e := range emoji {
		for _, hex := range strings.Split(e.Unified, "-") {
			n, err := strconv.ParseInt(hex, 16, 32)
			if err != nil {
				return "", fmt.Errorf("could not parse %x: %w", e.Unified, err)
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
}
