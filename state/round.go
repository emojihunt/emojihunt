package state

import (
	"context"

	"github.com/emojihunt/emojihunt/db"
	"github.com/rivo/uniseg"
	"golang.org/x/xerrors"
)

func ValidateRound(r db.Round) error {
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
		Name:    round.Name,
		Emoji:   round.Emoji,
		Hue:     round.Hue,
		Special: round.Special,
	})
	if err != nil {
		return Round{}, xerrors.Errorf("CreateRound: %w", err)
	}
	return Round(result), nil
}

func (c *Client) UpdateRound(ctx context.Context, round Round) error {
	if err := ValidateRound(round); err != nil {
		return err
	}
	err := c.queries.UpdateRound(ctx, db.UpdateRoundParams(round))
	if err != nil {
		return xerrors.Errorf("UpdateRound: %w", err)
	}
	return nil
}

func (c *Client) DeleteRound(ctx context.Context, id int64) error {
	err := c.queries.DeleteRound(ctx, id)
	if err != nil {
		return xerrors.Errorf("DeleteRound: %w", err)
	}
	return nil
}
