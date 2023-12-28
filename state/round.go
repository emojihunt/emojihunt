package state

import (
	"context"

	"github.com/emojihunt/emojihunt/state/db"
	"github.com/rivo/uniseg"
	"golang.org/x/xerrors"
)

func ValidateRound(r Round) error {
	if r.Name == "" {
		return ValidationError{"name", "is required"}
	} else if r.Emoji == "" {
		return ValidationError{"emoji", "is required"}
	} else if uniseg.GraphemeClusterCount(r.Emoji) != 1 {
		return ValidationError{"emoji", "must be a single grapheme cluster"}
	} else if uniseg.StringWidth(r.Emoji+"\ufe0f") != 2 {
		// This is *almost* correct. We add a U+FE0F to force emoji
		// presentation. https://github.com/rivo/uniseg/issues/27
		return ValidationError{"emoji", "must have emoji presentation"}
	} else if r.Hue < 0 || r.Hue >= 360 {
		return ValidationError{"hue", "must be in range [0, 360)"}
	}
	return nil
}

func (c *Client) GetRound(ctx context.Context, id int64) (Round, error) {
	round, err := c.queries.GetRound(ctx, id)
	if err != nil {
		return Round{}, xerrors.Errorf("GetRound: %w", err)
	}
	return Round(round), nil
}

func (c *Client) ListRounds(ctx context.Context) ([]Round, error) {
	results, err := c.queries.ListRounds(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListRounds: %w", err)
	}
	var rounds = make([]Round, len(results))
	for i, result := range results {
		rounds[i] = Round(result)
	}
	return rounds, nil
}

func (c *Client) CreateRound(ctx context.Context, round Round) (Round, error) {
	if err := ValidateRound(round); err != nil {
		return Round{}, err
	}
	result, err := c.queries.CreateRound(ctx, db.CreateRoundParams{
		Name:            round.Name,
		Emoji:           round.Emoji,
		Hue:             round.Hue,
		Sort:            round.Sort,
		Special:         round.Special,
		DriveFolder:     round.DriveFolder,
		DiscordCategory: round.DiscordCategory,
	})
	if err != nil {
		return Round{}, xerrors.Errorf("CreateRound: %w", err)
	}
	created := Round(result)
	c.RoundChange <- RoundChange{nil, &created}
	return created, nil
}

func (c *Client) UpdateRound(ctx context.Context, id int64,
	mutate func(round *Round) error) (Round, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	before, err := c.GetRound(ctx, id)
	if err != nil {
		return Round{}, err
	}
	var raw = before
	if err := mutate(&raw); err != nil {
		return Round{}, err
	} else if err := ValidateRound(raw); err != nil {
		return Round{}, err
	} else if raw.ID != id {
		return Round{}, xerrors.Errorf("mutation must not change round ID")
	} else if err := c.queries.UpdateRound(ctx, db.UpdateRoundParams(raw)); err != nil {
		return Round{}, xerrors.Errorf("UpdateRound: %w", err)
	}
	after, err := c.GetRound(ctx, id)
	if err != nil {
		return Round{}, err
	}
	c.RoundChange <- RoundChange{&before, &after}
	puzzles, err := c.ListPuzzlesByRound(ctx, id)
	if err != nil {
		return Round{}, err
	}
	for _, puzzle := range puzzles {
		var pre, post = puzzle, puzzle
		pre.Round = before
		c.PuzzleChange <- PuzzleChange{&pre, &post}
	}
	return after, nil
}

func (c *Client) DeleteRound(ctx context.Context, id int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	round, err := c.GetRound(ctx, id)
	if err != nil {
		return err
	}
	err = c.queries.DeleteRound(ctx, id)
	if err != nil {
		return xerrors.Errorf("DeleteRound: %w", err)
	}
	c.RoundChange <- RoundChange{&round, nil}
	return nil
}
