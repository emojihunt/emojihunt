package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/emojihunt/emojihunt/state/db"
	"golang.org/x/xerrors"
)

type DiscoveryConfig struct {
	// URL of the "All Puzzles" page on the hunt website
	PuzzlesURL  string `json:"puzzles_url"`
	CookieName  string `json:"cookie_name"`
	CookieValue string `json:"cookie_value"`

	// Group Mode: in many years (2021, 2020, etc.), the puzzle list is grouped
	// by round, and there is some grouping element (e.g. a <section>) for each
	// round that contains both the round name and the list of puzzles.
	//
	// In other years (2022), the puzzle list is presented as a sequence of
	// alternating round names (e.g. <h2>) and puzzle lists (e.g. <table>) with
	// no grouping element. If this is the case, set `groupedMode=false` and use
	// the group selector to select the overall container. Note that the round
	// name element must be an *immediate* child of the container, and the
	// puzzle list element must be its immediate sibling.
	//
	// EXAMPLES
	//
	// 2022 (https://puzzles.mit.edu/2022/puzzles/)
	// - Group:       `section#main-content` (group mode off)
	// - Round Name:  `h2`
	// - Puzzle List: `table`
	//
	// 2021 (https://puzzles.mit.edu/2021/puzzles.html)
	// - Group:       `.info div section` (group mode on)
	// - Round Name:  `a h3`
	// - Puzzle List: `table`
	//
	// 2020 (https://puzzles.mit.edu/2020/puzzles/)
	// - Group:       `#loplist > li:not(:first-child)` (group mode on)
	// - Round Name:  `a`
	// - Puzzle List: `ul li a`
	//
	// 2019 (https://puzzles.mit.edu/2019/puzzle.html)
	// - Group:       `.puzzle-list-section:nth-child(2) .round-list-item` (group mode on)
	// - Round Name:  `.round-list-header`
	// - Puzzle List: `.round-list-item`
	// - Puzzle Item: `.puzzle-list-item a`
	//
	GroupMode          bool   `json:"group_mode"`
	GroupSelector      string `json:"group_selector"`
	RoundNameSelector  string `json:"round_name_selector"`
	PuzzleListSelector string `json:"puzzle_list_selector"`

	// Optional: defaults to "a" (this is probably what you want)
	PuzzleItemSelector string `json:"puzzle_item_selector"`

	// URL of the websocket endpoint (optional)
	WebsocketURL string `json:"websocket_url"`

	// Token to send in the AUTH message (optional)
	WebsocketToken string `json:"websocket_token"`
}

func (c *Client) DiscoveryConfig(ctx context.Context) (DiscoveryConfig, error) {
	data, err := c.queries.GetSetting(ctx, discoveryConfigSetting)
	if errors.Is(err, sql.ErrNoRows) {
		return DiscoveryConfig{}, nil
	} else if err != nil {
		return DiscoveryConfig{}, xerrors.Errorf("GetSetting: %w", err)
	}
	var result DiscoveryConfig
	err = json.Unmarshal(data, &result)
	if err != nil {
		return DiscoveryConfig{}, xerrors.Errorf("setting unmarshal: %w", err)
	}
	return result, nil
}

func (c *Client) UpdateDiscoveryConfig(ctx context.Context,
	mutate func(config *DiscoveryConfig) error) (DiscoveryConfig, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if config, err := c.DiscoveryConfig(ctx); err != nil {
		return DiscoveryConfig{}, err
	} else if err := mutate(&config); err != nil {
		return DiscoveryConfig{}, err
	} else if err := c.writeSetting(ctx, discoveryConfigSetting, config); err != nil {
		return DiscoveryConfig{}, xerrors.Errorf("writeSetting: %w", err)
	}
	c.DiscoveryChange <- true
	return c.DiscoveryConfig(ctx)
}

func (c *Client) ShouldCreatePuzzle(ctx context.Context, puzzle ScrapedPuzzle) (bool, error) {
	count, err := c.queries.CheckPuzzleIsCreated(ctx, db.CheckPuzzleIsCreatedParams{
		Name: puzzle.Name, PuzzleURL: puzzle.PuzzleURL,
	})
	if err != nil {
		return false, xerrors.Errorf("CheckPuzzleIsCreated: %w", err)
	} else if count > 0 {
		return false, nil
	}

	count, err = c.queries.CheckPuzzleIsDiscovered(ctx, db.CheckPuzzleIsDiscoveredParams{
		Name: puzzle.Name, PuzzleURL: puzzle.PuzzleURL,
	})
	if err != nil {
		return false, xerrors.Errorf("CheckPuzzleIsDiscovered: %w", err)
	} else if count > 0 {
		return false, nil
	}
	return true, nil
}

func (c *Client) GetCreatedRound(ctx context.Context, name string) (Round, error) {
	round, err := c.queries.GetCreatedRound(ctx, name)
	if err != nil {
		return Round{}, xerrors.Errorf("GetCreatedRound: %w", err)
	}
	return round, nil
}

func (c *Client) GetDiscoveredRound(ctx context.Context, name string) (db.DiscoveredRound, error) {
	discovered, err := c.queries.GetDiscoveredRound(ctx, name)
	if err != nil {
		return db.DiscoveredRound{}, xerrors.Errorf("GetDiscoveredRound: %w", err)
	}
	return discovered, nil
}

func (c *Client) CreateDiscoveredPuzzle(ctx context.Context, puzzle db.CreateDiscoveredPuzzleParams) error {
	err := c.queries.CreateDiscoveredPuzzle(ctx, puzzle)
	if err != nil {
		return xerrors.Errorf("CreateDiscoveredPuzzle: %w", err)
	}
	return nil
}

func (c *Client) CreateDiscoveredRound(ctx context.Context, round string) (int64, error) {
	id, err := c.queries.CreateDiscoveredRound(ctx, db.CreateDiscoveredRoundParams{
		Name: round,
	})
	if err != nil {
		return 0, xerrors.Errorf("CreateDiscoveredRound: %w", err)
	}
	return id, nil
}

func (c *Client) UpdateDiscoveredRound(ctx context.Context, round db.DiscoveredRound) error {
	err := c.queries.UpdateDiscoveredRound(ctx, db.UpdateDiscoveredRoundParams(round))
	if err != nil {
		return xerrors.Errorf("UpdateDiscoveredRound: %w", err)
	}
	return nil
}

func (c *Client) ListPendingDiscoveredRounds(ctx context.Context) ([]db.DiscoveredRound, error) {
	discovered, err := c.queries.ListPendingDiscoveredRounds(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPendingDiscoveredRounds: %w", err)
	}
	return discovered, nil
}

func (c *Client) ListDiscoveredPuzzlesForRound(ctx context.Context, id int64) ([]db.DiscoveredPuzzle, error) {
	discovered, err := c.queries.ListDiscoveredPuzzlesForRound(
		ctx, sql.NullInt64{Int64: id, Valid: true},
	)
	if err != nil {
		return nil, xerrors.Errorf("ListDiscoveredPuzzlesForRound: %w", err)
	}
	return discovered, nil
}

func (c *Client) CompleteDiscoveredRound(ctx context.Context, id int64, round Round) error {
	err := c.queries.CompleteDiscoveredRound(ctx, db.CompleteDiscoveredRoundParams{
		ID: id, CreatedAs: round.ID,
	})
	if err != nil {
		return xerrors.Errorf("CompleteDiscoveredRound: %w", err)
	}
	return nil
}

func (c *Client) ListCreatablePuzzles(ctx context.Context) ([]db.ListCreatablePuzzlesRow, error) {
	puzzles, err := c.queries.ListCreatablePuzzles(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListCreatablePuzzles: %w", err)
	}
	return puzzles, nil
}

func (c *Client) CompleteDiscoveredPuzzle(ctx context.Context, id int64) error {
	err := c.queries.CompleteDiscoveredPuzzle(ctx, id)
	if err != nil {
		return xerrors.Errorf("CompleteDiscoveredPuzzle: %w", err)
	}
	return nil
}
