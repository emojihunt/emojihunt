package syncer

import (
	"fmt"
	"log"

	"github.com/emojihunt/emojihunt/state"
)

// NotifyNewPuzzle sends the "New puzzle!" message to #progress.
func (c *Client) NotifyNewPuzzle(puzzle state.Puzzle) error {
	log.Printf("sync: notifying for new puzzle %q", puzzle.Name)
	_, err := c.discord.ChannelSend(
		c.discord.ProgressChannel,
		fmt.Sprintf(
			"%s **New puzzle!** %s", puzzle.Round.Emoji, puzzle.Mention(),
		),
	)
	return err
}

// NotifyPuzzleWorking sends the "Work started on puzzle" message to #progress.
func (c *Client) NotifyPuzzleWorking(puzzle state.Puzzle) error {
	log.Printf("sync: notifying for working puzzle %q", puzzle.Name)
	_, err := c.discord.ChannelSend(
		c.discord.ProgressChannel, fmt.Sprintf(
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

// NotifySolveInProgress sends the same message as above to #progress.
func (c *Client) NotifySolveInProgress(puzzle state.Puzzle) error {
	log.Printf("sync: notifying for solved puzzle %q in #progress", puzzle.Name)
	kind := "Puzzle"
	if puzzle.Meta {
		kind = "Meta"
	}
	_, err := c.discord.ChannelSend(
		c.discord.ProgressChannel,
		fmt.Sprintf(
			"%s %s %s was **%s** Answer: `%s`.",
			puzzle.Round.Emoji, kind, puzzle.Mention(), puzzle.Status.SolvedVerb(), puzzle.Answer,
		),
	)
	return err
}
