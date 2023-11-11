package db

import (
	"context"

	"golang.org/x/xerrors"
)

func (c *Client) CreateRound(ctx context.Context, name string, emoji string) (Round, error) {
	round, err := c.queries.CreateRound(ctx, CreateRoundParams{
		Name:  name,
		Emoji: emoji,
	})
	if err != nil {
		return Round{}, xerrors.Errorf("CreateRound: %w", err)
	}
	return round, nil
}

func (c *Client) GetRound(ctx context.Context, id int64) (Round, error) {
	round, err := c.queries.GetRound(ctx, id)
	if err != nil {
		return Round{}, xerrors.Errorf("GetRound: %w", err)
	}
	return round, nil
}

func (c *Client) UpdateRound(ctx context.Context, round Round) error {
	err := c.queries.UpdateRound(ctx, UpdateRoundParams(round))
	if err != nil {
		return xerrors.Errorf("UpdateRound: %w", err)
	}
	return nil
}

func (c *Client) ListRounds(ctx context.Context) ([]Round, error) {
	rounds, err := c.queries.ListRounds(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListRounds: %w", err)
	}
	return rounds, nil
}
