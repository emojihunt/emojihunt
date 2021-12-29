package syncer

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

const pinnedStatusHeader = "Puzzle Information"

// discordCreateUpdatePin creates or updates the pinned message at the top of
// the puzzle channel. This message contains information about the puzzle status
// as well as links to the puzzle and the spreadsheet. It needs to be updated
// when the puzzle status changes.
func (s *Syncer) discordCreateUpdatePin(puzzle *schema.Puzzle) error {
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

// discordUpdateChannelCategory sets or updates the category of the puzzle
// channel, either "Puzzles" (for open puzzles) or "Solved" (for solved
// puzzles). It needs to be called when the puzzle status changes.
func (s *Syncer) discordUpdateChannelCategory(puzzle *schema.Puzzle) error {
	var category string
	if puzzle.ShouldArchive() {
		category = s.discord.SolvedCategoryID
	} else {
		category = s.discord.PuzzleCategoryID
	}

	return s.discord.SetChannelCategory(puzzle.DiscordChannel, category)
}
