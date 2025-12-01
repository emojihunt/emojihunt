package state

import (
	"context"
	"encoding/json"

	"github.com/emojihunt/emojihunt/state/db"
	"golang.org/x/xerrors"
)

func (c *Client) LogPuzzleChange(ctx context.Context, before *Puzzle,
	after *Puzzle, complete chan error) (PuzzleChange, error) {

	c.changeID += 1

	var change = PuzzleChange{before, after, c.changeID, complete}
	var msg = change.SyncMessage()

	encoded, err := json.Marshal(msg.Puzzle)
	if err != nil {
		return PuzzleChange{}, xerrors.Errorf("Marshal: %w", err)
	}
	err = c.queries.CreateChangelog(ctx, db.CreateChangelogParams{
		ID:     change.ChangeID,
		Kind:   msg.Kind,
		Puzzle: encoded,
	})
	if err != nil {
		return PuzzleChange{}, xerrors.Errorf("CreateChangelog: %w", err)
	}
	err = c.queries.PruneChangelog(ctx)
	if err != nil {
		return PuzzleChange{}, xerrors.Errorf("PruneChangelog: %w", err)
	}

	c.PuzzleChange <- change
	return change, nil
}

func (c *Client) LogRoundChange(ctx context.Context, before *Round,
	after *Round) (RoundChange, error) {

	c.changeID += 1

	var change = RoundChange{before, after, c.changeID}
	var msg = change.SyncMessage()

	encoded, err := json.Marshal(msg.Puzzle)
	if err != nil {
		return RoundChange{}, xerrors.Errorf("Marshal: %w", err)
	}
	err = c.queries.CreateChangelog(ctx, db.CreateChangelogParams{
		ID:    change.ChangeID,
		Kind:  msg.Kind,
		Round: encoded,
	})
	if err != nil {
		return RoundChange{}, xerrors.Errorf("CreateChangelog: %w", err)
	}
	err = c.queries.PruneChangelog(ctx)
	if err != nil {
		return RoundChange{}, xerrors.Errorf("PruneChangelog: %w", err)
	}

	c.RoundChange <- change
	return change, nil
}
