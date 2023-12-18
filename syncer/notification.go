package syncer

import (
	"fmt"

	"github.com/emojihunt/emojihunt/db"
)

// notifyNewPuzzle sends the "New puzzle!" message to #more-eyes.
func (s *Syncer) notifyNewPuzzle(puzzle *db.Puzzle) error {
	msg := fmt.Sprintf("%s **New puzzle!** <#%s>",
		puzzle.Round.Emoji, puzzle.DiscordChannel)
	_, err := s.discord.ChannelSend(s.discord.MoreEyesChannel, msg)
	return err
}

// notifyPuzzleWorking sends the "Work started on puzzle" message to #more-eyes.
func (s *Syncer) notifyPuzzleWorking(puzzle *db.Puzzle) error {
	msg := fmt.Sprintf("%s Work started on puzzle <#%s>",
		puzzle.Round.Emoji, puzzle.DiscordChannel)
	_, err := s.discord.ChannelSend(s.discord.MoreEyesChannel, msg)
	return err
}

// notifyPuzzleFullySolved sends the two "Puzzle solved!" (or "Puzzle
// backsolved!") messages: one to the puzzle channel, and another to
// #hanging-out.
func (s *Syncer) notifyPuzzleFullySolved(puzzle *db.Puzzle, suppressSolveNotif bool) error {
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
