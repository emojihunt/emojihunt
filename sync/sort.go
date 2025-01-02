package sync

import (
	"cmp"
	"context"
	"log"
	"slices"
	"strings"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
)

// When setting the position of Discord channels and categories, start at a high
// offset so manually-managed items are listed first.
const baseSortOffset = 64

type PuzzleSortFields struct {
	ID             int64
	DiscordChannel string
	IsSolved       bool

	Name string
	Meta bool
	RoundSortFields
}

func NewPuzzleSortFields(puzzle state.RawPuzzle, round state.Round) PuzzleSortFields {
	return PuzzleSortFields{
		ID:              puzzle.ID,
		DiscordChannel:  puzzle.DiscordChannel,
		IsSolved:        puzzle.Status.IsSolved(),
		Name:            puzzle.Name,
		Meta:            puzzle.Meta,
		RoundSortFields: NewRoundSortFields(round),
	}
}

func PuzzleSort(a, b PuzzleSortFields) int {
	if round := RoundSort(a.RoundSortFields, b.RoundSortFields); round != 0 {
		return round
	} else if a.Meta != b.Meta {
		if a.Meta {
			return 1
		} else {
			return -1
		}
	} else {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}
}

func (c *Client) SortDiscordChannels(ctx context.Context, puzzle PuzzleSortFields) (int, error) {
	c.sortLock.Lock()
	defer c.sortLock.Unlock()

	var puzzles []PuzzleSortFields
	var existing bool
	results, err := c.state.ListPuzzles(ctx)
	if err != nil {
		return 0, err
	}
	for _, result := range results {
		if result.ID == puzzle.ID {
			puzzles = append(puzzles, puzzle)
			existing = true
		} else {
			puzzles = append(puzzles, NewPuzzleSortFields(result.RawPuzzle(), result.Round))
		}
	}
	if !existing {
		puzzles = append(puzzles, puzzle)
	}
	slices.SortFunc(puzzles, PuzzleSort)

	var found int
	var position int = baseSortOffset
	var round int64
	var order []discord.ChannelOrder
	var channels = c.discord.ListChannelsByID()
	for _, p := range puzzles {
		// Scope position to the round (category). Adding new puzzles to a round
		// shouldn't shift the position of *all* later puzzles.
		if round != p.RoundSortFields.ID {
			position = baseSortOffset
			round = p.RoundSortFields.ID
		}
		position += 1
		if p.ID == puzzle.ID {
			found = position
		}
		if p.IsSolved {
			// skip solved puzzles for speed
		} else if channel, ok := channels[p.DiscordChannel]; !ok {
			// no such channel, skip
		} else if channel.Position != position {
			// update position only if it's not already correct
			order = append(order, discord.ChannelOrder{
				ID: p.DiscordChannel, Position: position,
			})
		}
	}

	if len(order) > 0 {
		log.Printf("sync: sorting discord channels")
		// Note: this may error if we try to sort more than 100 channels. Hopefully
		// we won't have that many puzzles open at once (for speed, we only sort
		// unsolved puzzles' channels).
		err = c.discord.SortChannels(order)
		if err != nil {
			return 0, err
		}
	}
	return found, nil
}

type RoundSortFields struct {
	ID              int64
	DiscordCategory string

	Name    string
	Special bool
	Sort    int64
}

func NewRoundSortFields(round state.Round) RoundSortFields {
	return RoundSortFields{
		ID:              round.ID,
		DiscordCategory: round.DiscordCategory,
		Name:            round.Name,
		Special:         round.Special,
		Sort:            round.Sort,
	}
}

func RoundSort(a, b RoundSortFields) int {
	if a.Special != b.Special {
		if a.Special {
			return -1
		} else {
			return 1
		}
	} else if a.Sort != b.Sort {
		return cmp.Compare(a.Sort, b.Sort)
	} else {
		return cmp.Compare(a.ID, b.ID)
	}
}

func (c *Client) SortDiscordCategories(ctx context.Context, round RoundSortFields) (int, error) {
	c.sortLock.Lock()
	defer c.sortLock.Unlock()

	var rounds []RoundSortFields
	var existing bool
	results, err := c.state.ListRounds(ctx)
	if err != nil {
		return 0, err
	}
	for _, result := range results {
		if result.ID == round.ID {
			rounds = append(rounds, round)
			existing = true
		} else {
			rounds = append(rounds, NewRoundSortFields(result))
		}
	}
	if !existing {
		rounds = append(rounds, round)
	}
	slices.SortFunc(rounds, RoundSort)

	var found int
	var order []discord.ChannelOrder
	for i, r := range rounds {
		var position = baseSortOffset + i
		if r.ID == round.ID {
			found = position
		}
		if r.DiscordCategory != "" {
			order = append(order, discord.ChannelOrder{
				ID: r.DiscordCategory, Position: position,
			})
		}
	}
	for i, solved := range c.solvedCategories {
		order = append(order, discord.ChannelOrder{
			ID: solved, Position: baseSortOffset*4 + i,
		})
	}

	log.Printf("sync: sorting discord categories")
	err = c.discord.SortChannels(order)
	if err != nil {
		return 0, err
	}
	return found, nil
}
