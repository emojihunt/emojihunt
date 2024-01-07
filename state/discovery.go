package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

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
