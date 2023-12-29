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
)

const (
	roundCategoryPrefix  = "Round: "
	solvedCategoryPrefix = "Solved "
	solvedCategoryCount  = 3
	pinnedStatusHeader   = "Puzzle Information"
	locationDefaultMsg   = "Use `/puzzle voice` to assign a voice room"
	embedColor           = 0x7C39ED
)

// CreateDiscordChannel creates a new Discord channel and returns its ID.
func (c *Client) CreateDiscordChannel(ctx context.Context, puzzle state.RawPuzzle, round state.Round) (string, error) {
	log.Printf("sync: creating discord channel for %q", puzzle.Name)
	channel, err := c.discord.CreateChannel(puzzle.Name, round.DiscordCategory)
	if err != nil {
		return "", err
	}
	return channel.ID, nil
}

// CreateDiscordCategory creates a new Discord category and returns its ID.
func (c *Client) CreateDiscordCategory(ctx context.Context, round state.Round) (string, error) {
	log.Printf("sync: creating discord category for %q", round.Name)
	category, err := c.discord.CreateCategory(roundCategoryPrefix + round.Name)
	if err != nil {
		return "", err
	}
	return category.ID, nil
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
			log.Printf("sync: restoring category %q", name)
			category, err := c.discord.CreateCategory(name)
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

type DiscordCategoryFields struct {
	RoundName     string
	RoundCategory string
}

func NewDiscordCategoryFields(round state.Round) DiscordCategoryFields {
	return DiscordCategoryFields{
		RoundName:     round.Name,
		RoundCategory: round.DiscordCategory,
	}
}

// UpdateDiscordCategory configures the name of the round category.
func (c *Client) UpdateDiscordCategory(fields DiscordCategoryFields) error {
	log.Printf("sync: updating discord category for %q", fields.RoundName)

	// The Discord rate limit on channel renames is fairly restrictive (2 per 10
	// minutes per channel), so finish renaming the category asynchronously if we
	// get rate-limited.
	var name = roundCategoryPrefix + fields.RoundName
	ch := make(chan error)
	go func() {
		ch <- c.discord.SetChannelName(fields.RoundCategory, name)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		rateLimit := c.discord.CheckRateLimit(discordgo.EndpointChannel(fields.RoundCategory))
		if rateLimit == nil {
			// No rate limiting detected; maybe the Discord request is just
			// slow? Wait for it to finish.
			return <-ch
		}
		// Being rate limited; goroutine will finish later.
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on category renaming. Category will be "+
			"renamed to %q in %s.", name, time.Until(*rateLimit).Round(time.Second))
		_, err := c.discord.ChannelSend(c.discord.QMChannel, msg)
		return err
	}
}

func (c *Client) HandleDiscordPuzzleError(ctx context.Context, puzzle state.Puzzle, orig error) {
	var code = discord.ErrCode(orig)
	if code != discordgo.ErrCodeUnknownChannel && code != discordgo.ErrCodeInvalidFormBody {
		return
	} else if puzzle.DiscordChannel == "" {
		return
	}
	log.Printf("sync: checking on puzzle channel for %q", puzzle.Name)

	// Check that puzzle channel exists
	var channel = puzzle.DiscordChannel
	_, err := c.discord.GetChannel(channel)
	if discord.ErrCode(err) == discordgo.ErrCodeUnknownChannel {
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

	c.HandleDiscordRoundError(ctx, puzzle.Round, orig)

	// Check that solved categories exist. Caveat: unlike the fixups above, this
	// won't fire a fresh change notification for the puzzle...
	go c.RestoreSolvedCategories()
}

func (c *Client) HandleDiscordRoundError(ctx context.Context, round state.Round, orig error) {
	var code = discord.ErrCode(orig)
	if code != discordgo.ErrCodeUnknownChannel && code != discordgo.ErrCodeInvalidFormBody {
		return
	}
	log.Printf("sync: checking on round category for %q", round.Name)

	// Check that round category exists
	var original = round.DiscordCategory
	_, err := c.discord.GetChannel(original)
	if discord.ErrCode(err) == discordgo.ErrCodeUnknownChannel {
		created, err := c.CreateDiscordCategory(ctx, round)
		if err != nil {
			return
		}
		go c.state.UpdateRound(ctx, round.ID,
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
	}
}
