package syncer

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

const (
	pinnedStatusHeader  = "Puzzle Information"
	voiceRoomDefaultMsg = "Use `/voice start` to assign a voice room"
)

// DiscordCreateUpdatePin creates or updates the pinned message at the top of
// the puzzle channel. This message contains information about the puzzle status
// as well as links to the puzzle and the spreadsheet.
//
// This function is called by BasicUpdate. Other packages need to call it when
// updating non-status fields, such as the voice room.
//
func (s *Syncer) DiscordCreateUpdatePin(puzzle *schema.Puzzle) error {
	voiceRoomMsg := voiceRoomDefaultMsg
	if puzzle.VoiceRoomEvent != "" {
		var err error
		event, err := s.discord.GetScheduledEvent(puzzle.VoiceRoomEvent)
		if err != nil {
			return err
		}
		voiceRoomMsg = fmt.Sprintf("Join us in <#%s>!", *event.ChannelID)
	}
	roundHeader := "Round"
	if len(puzzle.Rounds) > 1 {
		roundHeader = "Rounds"
	}
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  puzzle.Name,
		URL:    puzzle.PuzzleURL,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   roundHeader,
				Value:  strings.Join(puzzle.Rounds.EmojisAndNames(), ", "),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  puzzle.Status.Human(),
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
			{
				Name:   "Voice Room",
				Value:  voiceRoomMsg,
				Inline: false,
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
	var category *discordgo.Channel
	if puzzle.ShouldArchive() {
		category = s.discord.SolvedCategory
	} else {
		category = s.discord.PuzzleCategory
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
