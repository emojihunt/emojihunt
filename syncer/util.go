package syncer

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

const pinnedStatusHeader = "Puzzle Information"

func (s *Syncer) SetPinnedStatusInfo(puzzle *schema.Puzzle) (didUpdate bool, err error) {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  puzzle.Name,
		URL:    puzzle.PuzzleURL,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Round",
				Value:  fmt.Sprintf("%v %v", puzzle.Round.Emoji, puzzle.Round.Name),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  puzzle.Status.Pretty(),
				Inline: true,
			},
			{
				Name:   "Puzzle",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.PuzzleURL),
				Inline: true,
			},
			{
				Name:   "Sheet",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.SpreadsheetURL()),
				Inline: true,
			},
		},
	}

	return s.discord.CreateUpdatePin(puzzle.DiscordChannel, pinnedStatusHeader, embed)
}