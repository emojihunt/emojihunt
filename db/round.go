package db

import (
	"context"

	"github.com/rivo/uniseg"
	"golang.org/x/xerrors"
)

func (c *Client) GetRound(ctx context.Context, id int64) (Round, error) {
	round, err := c.queries.GetRound(ctx, id)
	if err != nil {
		return Round{}, xerrors.Errorf("GetRound: %w", err)
	}
	return round, nil
}

func (c *Client) ListRounds(ctx context.Context) ([]Round, error) {
	rounds, err := c.queries.ListRounds(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListRounds: %w", err)
	}
	return rounds, nil
}

func (c *Client) CreateRound(ctx context.Context, round Round) (Round, error) {
	if err := round.Validate(); err != nil {
		return Round{}, err
	}
	round, err := c.queries.CreateRound(ctx, CreateRoundParams{
		Name:  round.Name,
		Emoji: round.Emoji,
	})
	if err != nil {
		return Round{}, xerrors.Errorf("CreateRound: %w", err)
	}
	return round, nil
}

func (c *Client) UpdateRound(ctx context.Context, round Round) error {
	if err := round.Validate(); err != nil {
		return err
	}
	err := c.queries.UpdateRound(ctx, UpdateRoundParams(round))
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

func (r Round) Validate() error {
	if r.Name == "" {
		return ValidationError{"missing required field", "name"}
	} else if r.Emoji == "" {
		return ValidationError{"missing required field", "emoji"}
	} else if uniseg.GraphemeClusterCount(r.Emoji) != 1 {
		return ValidationError{"value must be an emoji", "emoji"}
	} else if uniseg.StringWidth(r.Emoji) != 2 {
		// *almost* correct, see https://github.com/rivo/uniseg/issues/27
		return ValidationError{"value must be an emoji", "emoji"}
	}
	return nil
}
