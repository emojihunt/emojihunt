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

// discordUpdateChannel sets or updates the name and category of the puzzle
// channel. Categories are either "Puzzles" (for open puzzles) or "Solved" (for
// solved puzzles), and the puzzle name includes a check mark when the puzzle is
// solved. It needs to be called when the puzzle status changes.
func (s *Syncer) discordUpdateChannel(puzzle *schema.Puzzle) error {
	var category string
	if puzzle.ShouldArchive() {
		category = s.discord.SolvedCategoryID
	} else {
		category = s.discord.PuzzleCategoryID
	}

	err := s.discord.SetChannelCategory(puzzle.DiscordChannel, category)
	if err != nil {
		return err
	}

	var name = puzzle.Name
	if puzzle.Status.IsSolved() {
		name = "✅ " + name
	}
	return s.discord.SetChannelName(puzzle.DiscordChannel, name)
}
