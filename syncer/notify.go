package syncer

import (
	"fmt"
	"log"

	"github.com/emojihunt/emojihunt/state"
)

// NotifyNewPuzzle sends the "New puzzle!" message to #more-eyes.
func (c *Client) NotifyNewPuzzle(puzzle state.Puzzle) error {
	log.Printf("sync: notifying for new puzzle %q", puzzle.Name)
	_, err := c.discord.ChannelSend(
		c.discord.MoreEyesChannel,
		fmt.Sprintf(
			"%s **New puzzle!** %s", puzzle.Round.Emoji, puzzle.Mention(),
		),
	)
	return err
}

// NotifyPuzzleWorking sends the "Work started on puzzle" message to #more-eyes.
func (c *Client) NotifyPuzzleWorking(puzzle state.Puzzle) error {
	log.Printf("sync: notifying for working puzzle %q", puzzle.Name)
	_, err := c.discord.ChannelSend(
		c.discord.MoreEyesChannel, fmt.Sprintf(
			"%s Work started on puzzle %s", puzzle.Round.Emoji, puzzle.Mention(),
		),
	)
	return err
}

// NotifySolveInPuzzleChannel sends the "Puzzle solved!" (or "...backsolved!",
// etc.) message to the puzzle channel.
func (c *Client) NotifySolveInPuzzleChannel(puzzle state.Puzzle) error {
	log.Printf("sync: notifying for solved puzzle %q in puzzle channel", puzzle.Name)
	return c.discord.ChannelSendRawID(
		puzzle.DiscordChannel,
		fmt.Sprintf(
			"%s Puzzle %s The answer was `%v`. I'll archive this channel.",
			puzzle.Status.Emoji(), puzzle.Status.SolvedVerb(), puzzle.Answer,
		),
	)
}

// NotifySolveInHangingOut sends the same message as above to #hanging-out.
// Unlike all of the other methods in this file, it does *not* require a puzzle
// channel to exist.
func (c *Client) NotifySolveInHangingOut(puzzle state.Puzzle) error {
	log.Printf("sync: notifying for solved puzzle %q in #hanging-out", puzzle.Name)
	kind := "Puzzle"
	if puzzle.Meta {
		kind = "Meta"
	}
	_, err := c.discord.ChannelSend(
		c.discord.HangingOutChannel,
		fmt.Sprintf(
			"%s %s %s was **%s** Answer: `%s`.",
			puzzle.Round.Emoji, kind, puzzle.Mention(), puzzle.Status.SolvedVerb(), puzzle.Answer,
		),
	)
	return err
}
