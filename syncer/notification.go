package syncer

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

// notifyNewPuzzle sends the "A new puzzle is available!" message to #general.
func (s *Syncer) notifyNewPuzzle(puzzle *schema.Puzzle) error {
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
	return s.discord.ChannelSendEmbed(s.discord.GeneralChannel, embed)
}

// notifyPuzzleFullySolved sends the two "Puzzle solved!" (or "Puzzle
// backsolved!") messages: one to the puzzle channel, and another to #general.
func (s *Syncer) notifyPuzzleFullySolved(puzzle *schema.Puzzle) error {
	msg := fmt.Sprintf(
		"Puzzle %s! The answer was `%v`. I'll archive this channel.",
		puzzle.Status.SolvedVerb(), puzzle.Answer,
	)
	if err := s.discord.ChannelSendRawID(puzzle.DiscordChannel, msg); err != nil {
		return err
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
	return s.discord.ChannelSendEmbed(s.discord.GeneralChannel, embed)
}

// notifyPuzzleSolvedMissingAnswer sends messages to the puzzle channel and to
// #qm asking for the answer to be entered into Airtable.
func (s *Syncer) notifyPuzzleSolvedMissingAnswer(puzzle *schema.Puzzle) error {
	puzMsg := fmt.Sprintf(
		"Puzzle %s! Please add the answer to Airtable.",
		puzzle.Status.SolvedVerb(),
	)
	if err := s.discord.ChannelSendRawID(puzzle.DiscordChannel, puzMsg); err != nil {
		return err
	}

	msg := fmt.Sprintf(
		"**:woman_shrugging: Help!** Puzzle %q is marked as %s, but no answer was "+
			"entered in Airtable.",
		puzzle.Name, puzzle.Status.SolvedVerb(),
	)
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Edit in Airtable",
			Style: discordgo.LinkButton,
			Emoji: discordgo.ComponentEmoji{Name: "📝"},
			URL:   s.airtable.EditURL(puzzle),
		},
	}
	return s.discord.ChannelSendComponents(s.discord.QMChannel, msg, components)
}

// notifyPuzzleStatusChange sends messages about ordinary puzzle status changes
// (i.e. everything except when a puzzle is solved).
func (s *Syncer) notifyPuzzleStatusChange(puzzle *schema.Puzzle) error {
	if s.discord.StatusUpdateChannel == nil {
		// status update channel is optional
		return nil
	}
	msg := fmt.Sprintf(
		"%s Puzzle <#%s> is now %v.",
		puzzle.Round.Emoji, puzzle.DiscordChannel, puzzle.Status.Pretty(),
	)
	return s.discord.ChannelSend(s.discord.StatusUpdateChannel, msg)
}
