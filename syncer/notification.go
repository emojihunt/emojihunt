package syncer

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

// notifyNewPuzzle sends the "A new puzzle is available!" message to #general.
func (s *Syncer) notifyNewPuzzle(puzzle *schema.Puzzle) error {
	log.Printf("Posting information about new puzzle %q", puzzle.Name)

	// Pin a message with the spreadsheet URL to the channel
	if err := s.discordCreateUpdatePin(puzzle); err != nil {
		return fmt.Errorf("error pinning puzzle info: %v", err)
	}

	// Post a message in the general channel with a link to the puzzle.
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "A new puzzle is available!",
			IconURL: puzzle.Round.TwemojiURL(),
		},
		Title: puzzle.Name,
		URL:   puzzle.PuzzleURL,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%s>", puzzle.DiscordChannel),
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
	if err := s.discord.GeneralChannelSendEmbed(embed); err != nil {
		return fmt.Errorf("error posting new puzzle announcement: %v", err)
	}

	return nil
}

// notifyPuzzleSolved sends the two "Puzzle solved!" (or "Puzzle backsolved!")
// messages: one to the puzzle channel, and another to #general.
func (s *Syncer) notifyPuzzleSolved(puzzle *schema.Puzzle) error {
	msg := fmt.Sprintf(
		"Puzzle %s! The answer was `%v`. I'll archive this channel.",
		puzzle.Status.SolvedVerb(), puzzle.Answer,
	)
	if err := s.discord.ChannelSend(puzzle.DiscordChannel, msg); err != nil {
		return fmt.Errorf("error posting solved puzzle announcement: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("Puzzle %s!", puzzle.Status.SolvedVerb()),
			IconURL: puzzle.Round.TwemojiURL(),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%s>", puzzle.DiscordChannel),
				Inline: true,
			},
			{
				Name:   "Answer",
				Value:  fmt.Sprintf("`%s`", puzzle.Answer),
				Inline: true,
			},
		},
	}
	if err := s.discord.GeneralChannelSendEmbed(embed); err != nil {
		return fmt.Errorf("error posting solved puzzle announcement: %v", err)
	}
	return nil
}
