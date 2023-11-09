package db

import "context"

func (c *Client) ListRounds(ctx context.Context) ([]Round, error) {
	return c.queries.ListRounds(ctx)
}

func (c *Client) CreateRound(ctx context.Context, name string, emoji string) (Round, error) {
	return c.queries.CreateRound(ctx, CreateRoundParams{
		Name:  name,
		Emoji: emoji,
	})
}
