package db

import (
	"context"
	"fmt"
)

func (c *Client) LoadState(ctx context.Context) ([]byte, error) {
	result, err := c.queries.GetState(ctx)
	if err != nil {
		return nil, err
	}
	if len(result) < 1 {
		if err := c.queries.UpdateState(ctx, []byte("{}")); err != nil {
			return nil, err
		}
		return c.LoadState(ctx)
	} else if len(result) > 1 {
		return nil, fmt.Errorf("found multiple state rows")
	}
	return result[0].Data, nil
}

func (c *Client) WriteState(ctx context.Context, data []byte) error {
	return c.queries.UpdateState(ctx, data)
}
