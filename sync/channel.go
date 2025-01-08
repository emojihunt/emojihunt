package sync

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
)

const (
	locationDefaultMsg = "Use `/puzzle voice` to assign a voice room."
)

// CreateDiscordChannel creates a new Discord channel and returns its ID.
func (c *Client) CreateDiscordChannel(ctx context.Context, puzzle state.RawPuzzle,
	round state.Round) (string, error) {
	log.Printf("sync: creating discord channel for %q", puzzle.Name)
	var original = round.DiscordCategory
	_, ok := c.discord.GetChannel(original)
	if !ok {
		created, err := c.CreateDiscordCategory(ctx, round)
		if err != nil {
			return "", err
		}
		round, _, err = c.state.UpdateRound(ctx, round.ID,
			func(round *state.Round) error {
				if round.DiscordCategory == original {
					log.Printf("sync: replacing deleted discord category for %q", round.Name)
					round.DiscordCategory = created
				} else {
					log.Printf("sync: created duplicate discord category for %q", round.Name)
				}
				return nil
			},
		)
		if err != nil {
			return "", err
		}
	}
	position, err := c.SortDiscordChannels(ctx, NewPuzzleSortFields(puzzle, round))
	if err != nil {
		return "", err
	}
	channel, err := c.discord.CreateChannel(puzzle.Name, round.DiscordCategory, position)
	if err != nil {
		return "", err
	}
	return channel.ID, nil
}

type DiscordChannelFields struct {
	PuzzleName    string
	PuzzleChannel string
	RoundName     string
	RoundCategory string
	IsSolved      bool
	PuzzleSortFields
}

func NewDiscordChannelFields(puzzle state.Puzzle) DiscordChannelFields {
	return DiscordChannelFields{
		PuzzleName:       puzzle.Name,
		PuzzleChannel:    puzzle.DiscordChannel,
		RoundName:        puzzle.Round.Name,
		RoundCategory:    puzzle.Round.DiscordCategory,
		IsSolved:         puzzle.Status.IsSolved(),
		PuzzleSortFields: NewPuzzleSortFields(puzzle.RawPuzzle(), puzzle.Round),
	}
}

// UpdateDiscordChannel configures the name and category of the puzzle channel.
// Categories are either in a round-specific category (if unsolved) or one of a
// few "Solved" categories (for solved puzzles), and the channel name is
// prefixed with a check mark when the puzzle is solved.
func (c *Client) UpdateDiscordChannel(ctx context.Context, fields DiscordChannelFields) error {
	log.Printf("sync: updating discord channel for %q", fields.PuzzleName)
	position, err := c.SortDiscordChannels(ctx, fields.PuzzleSortFields)
	if err != nil {
		return err
	}

	// Move puzzle channel to the correct category
	var category = fields.RoundCategory
	if fields.IsSolved {
		h := sha256.New()
		if _, err := h.Write([]byte(fields.PuzzleChannel)); err != nil {
			return err
		}
		i := binary.BigEndian.Uint64(h.Sum(nil)[:8]) % solvedCategoryCount
		category = c.solvedCategories[i]
	}
	err = c.discord.SetChannelCategory(fields.PuzzleChannel, category, position)
	if err != nil {
		return err
	}

	// The Discord rate limit on channel renames is fairly restrictive (2 per 10
	// minutes per channel), so finish renaming the channel asynchronously if we
	// get rate-limited.
	var title = fields.PuzzleName
	if fields.IsSolved {
		title = "✅ " + title
	}
	ch := make(chan error)
	go func() {
		ch <- c.discord.SetChannelName(fields.PuzzleChannel, title, position)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		rateLimit := c.discord.CheckRateLimit(discordgo.EndpointChannel(fields.PuzzleChannel))
		if rateLimit == nil {
			// No rate limiting detected; maybe the Discord request is just
			// slow? Wait for it to finish.
			return <-ch
		}
		// Being rate limited; goroutine will finish later.
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on channel renaming. Channel will be "+
			"renamed to %q in %s.", title, time.Until(*rateLimit).Round(time.Second))
		return c.discord.ChannelSendRawID(fields.PuzzleChannel, msg)
	}
}

func (c *Client) CheckDiscordPuzzle(ctx context.Context, puzzle state.Puzzle) {
	log.Printf("sync: checking puzzle channel for %q", puzzle.Name)
	var channel = puzzle.DiscordChannel
	_, ok := c.discord.GetChannel(channel)
	if !ok {
		go c.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				if puzzle.DiscordChannel == channel {
					log.Printf("sync: clearing nonexistent discord channel %q on %q", channel, puzzle.Name)
					puzzle.DiscordChannel = ""
				}
				return nil
			},
		)
	}

	c.CheckDiscordRound(ctx, puzzle.Round)

	// Check that solved categories exist. Caveat: unlike the fixups above, this
	// won't fire a fresh change notification for the puzzle...
	go c.RestoreSolvedCategories()
}
