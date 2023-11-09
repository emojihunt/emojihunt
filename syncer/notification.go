package syncer

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/schema"
)

// notifyNewPuzzle sends the "New puzzle!" message to #more-eyes.
func (s *Syncer) notifyNewPuzzle(puzzle *schema.Puzzle) error {
	msg := fmt.Sprintf("%s **New puzzle!** <#%s>",
		puzzle.Round.Emoji, puzzle.DiscordChannel)
	_, err := s.discord.ChannelSend(s.discord.MoreEyesChannel, msg)
	return err
}

// notifyPuzzleWorking sends the "Work started on puzzle" message to #more-eyes.
func (s *Syncer) notifyPuzzleWorking(puzzle *schema.Puzzle) error {
	msg := fmt.Sprintf("%s Work started on puzzle <#%s>",
		puzzle.Round.Emoji, puzzle.DiscordChannel)
	_, err := s.discord.ChannelSend(s.discord.MoreEyesChannel, msg)
	return err
}

// notifyPuzzleFullySolved sends the two "Puzzle solved!" (or "Puzzle
// backsolved!") messages: one to the puzzle channel, and another to
// #hanging-out.
func (s *Syncer) notifyPuzzleFullySolved(puzzle *schema.Puzzle, suppressSolveNotif bool) error {
	if !suppressSolveNotif {
		msg := fmt.Sprintf(
			"Puzzle %s! The answer was `%v`. I'll archive this channel.",
			puzzle.Status.SolvedVerb(), puzzle.Answer,
		)
		if err := s.discord.ChannelSendRawID(puzzle.DiscordChannel, msg); err != nil {
			return err
		}
	}

	msg := fmt.Sprintf("%s Puzzle <#%s> was **%s!** Answer: `%s`.",
		puzzle.Round.Emoji, puzzle.DiscordChannel, puzzle.Status.SolvedVerb(), puzzle.Answer)
	_, err := s.discord.ChannelSend(s.discord.HangingOutChannel, msg)
	return err
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
			URL:   fmt.Sprintf("https://TODO/edit/%d", puzzle.ID),
		},
	}
	_, err := s.discord.ChannelSendComponents(s.discord.QMChannel, msg, components)
	return err
}
