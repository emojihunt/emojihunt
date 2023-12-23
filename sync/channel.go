package sync

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

const (
	roundCategoryPrefix  = "Round: "
	solvedCategoryPrefix = "Solved "
	solvedCategoryCount  = 3
	pinnedStatusHeader   = "Puzzle Information"
	locationDefaultMsg   = "Use `/puzzle voice` to assign a voice room"
	embedColor           = 0x7C39ED
)

// CreateDiscordChannel creates a new Discord channel and saves it to the
// Puzzle object.
func (c *Client) CreateDiscordChannel(ctx context.Context, puzzle state.Puzzle) (state.Puzzle, error) {
	log.Printf("sync: creating discord channel for %q", puzzle.Name)
	var position int
	puzzles, err := c.state.ListPuzzles(ctx)
	if err != nil {
		return puzzle, err
	}
	for i, item := range puzzles {
		if item.ID == puzzle.ID {
			position = 512 + 2*i - 1
		}
	}

	channel, err := c.discord.CreateChannel(puzzle.Name,
		puzzle.Round.DiscordCategory, position)
	if err != nil {
		return puzzle, err
	}
	puzzle, err = c.state.UpdatePuzzleAdvanced(ctx, puzzle.ID,
		func(puzzle *state.RawPuzzle) error {
			if puzzle.DiscordChannel != "" {
				return xerrors.Errorf("created duplicate Discord channel")
			}
			puzzle.DiscordChannel = channel.ID
			return nil
		}, false,
	)
	if err != nil {
		return puzzle, err
	}

	log.Printf("sync: sorting discord channels")
	var order []discord.ChannelOrder
	for i, puzzle := range puzzles {
		if !puzzle.HasDiscordChannel() {
			continue
		}
		order = append(order, discord.ChannelOrder{
			ID: puzzle.DiscordChannel, Position: 512 + 2*i,
		})
	}
	return puzzle, c.discord.SortChannels(order)
}

// CreateDiscordCategory creates a new Discord category and saves it to the
// Round object.
func (c *Client) CreateDiscordCategory(ctx context.Context, round state.Round) (state.Round, error) {
	log.Printf("sync: creating discord category for %q", round.Name)
	var position int
	rounds, err := c.state.ListRounds(ctx)
	if err != nil {
		return round, nil
	}
	for i, item := range rounds {
		if item.ID == round.ID {
			position = 64 + 2*i - 1
		}
	}

	category, err := c.discord.CreateCategory(roundCategoryPrefix+round.Name,
		position)
	if err != nil {
		return round, err
	}
	round, err = c.state.UpdateRoundAdvanced(ctx, round.ID,
		func(round *state.Round) error {
			if round.DiscordCategory != "" {
				return xerrors.Errorf("created duplicate Discord category")
			}
			round.DiscordCategory = category.ID
			return nil
		}, false,
	)
	if err != nil {
		return round, err
	}

	log.Printf("sync: sorting discord categories")
	var order []discord.ChannelOrder
	for i, round := range rounds {
		if round.HasDiscordCategory() {
			// Use a high offset so manually-managed channels are listed first
			order = append(order, discord.ChannelOrder{
				ID: round.DiscordCategory, Position: 64 + 2*i,
			})
		}
	}
	for i, category := range c.solvedCategories {
		order = append(order, discord.ChannelOrder{
			ID: category, Position: 256 + i,
		})
	}
	return round, c.discord.SortChannels(order)
}

func (c *Client) RestoreSolvedCategories() error {
	categories, err := c.discord.GetChannelCategories()
	if err != nil {
		return err
	}
	var solved []string
	for i := 0; i < solvedCategoryCount; i++ {
		name := solvedCategoryPrefix + string(rune(int('A')+i))
		if category, ok := categories[name]; ok {
			solved = append(solved, category.ID)
		} else {
			category, err := c.discord.CreateCategory(name, 256+i)
			if err != nil {
				return err
			}
			solved = append(solved, category.ID)
		}
	}
	c.solvedCategories = solved
	return nil
}

type DiscordChannelFields struct {
	PuzzleName    string
	PuzzleChannel string
	RoundName     string
	RoundCategory string
	IsSolved      bool
}

func NewDiscordChannelFields(puzzle state.Puzzle) DiscordChannelFields {
	return DiscordChannelFields{
		PuzzleName:    puzzle.Name,
		PuzzleChannel: puzzle.DiscordChannel,
		RoundName:     puzzle.Round.Name,
		RoundCategory: puzzle.Round.DiscordCategory,
		IsSolved:      puzzle.Status.IsSolved(),
	}
}

// UpdateDiscordChannel configures the name and category of the puzzle channel.
// Categories are either in a round-specific category (if unsolved) or one of a
// few "Solved" categories (for solved puzzles), and the channel name is
// prefixed with a check mark when the puzzle is solved.
func (c *Client) UpdateDiscordChannel(fields DiscordChannelFields) error {
	log.Printf("sync: updating discord channel for %q", fields.PuzzleName)

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
	err := c.discord.SetChannelCategory(fields.PuzzleChannel, category)
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
		ch <- c.discord.SetChannelName(fields.PuzzleChannel, title)
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
