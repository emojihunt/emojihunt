package state

import (
	"context"
	"net/url"

	"github.com/emojihunt/emojihunt/state/db"
	"golang.org/x/xerrors"
)

func ValidatePuzzle(p RawPuzzle) error {
	if p.Name == "" {
		return ValidationError{"name", "is required"}
	} else if p.Round == 0 {
		return ValidationError{"round", "is required"}
	} else if !p.Status.IsValid() {
		return ValidationError{"status", "is invalid"}
	} else if p.PuzzleURL == "" {
		return ValidationError{"puzzle_url", "is required"}
	} else if u, err := url.Parse(p.PuzzleURL); err != nil {
		return ValidationError{"puzzle_url", "is not a valid URL"}
	} else if u.Scheme != "http" && u.Scheme != "https" {
		return ValidationError{"puzzle_url", "is not a valid URL"}
	} else if !p.Status.IsSolved() && p.Answer != "" {
		return ValidationError{"status", "is unsolved but answer is not blank"}
	} else if p.Status.IsSolved() && p.Answer == "" {
		return ValidationError{"status", "is solved but answer is blank"}
	}
	return nil
}

func (c *Client) GetPuzzle(ctx context.Context, id int64) (Puzzle, error) {
	puzzle, err := c.queries.GetPuzzle(ctx, id)
	if err != nil {
		return Puzzle{}, xerrors.Errorf("GetPuzzle: %w", err)
	}
	return Puzzle(puzzle), nil
}

func (c *Client) GetPuzzleByChannel(ctx context.Context, channel string) (Puzzle, error) {
	puzzle, err := c.queries.GetPuzzleByChannel(ctx, channel)
	if err != nil {
		return Puzzle{}, xerrors.Errorf("GetPuzzleByChannel: %w", err)
	}
	return Puzzle(puzzle), nil
}

func (c *Client) ListPuzzles(ctx context.Context) ([]Puzzle, error) {
	results, err := c.queries.ListPuzzles(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzles: %w", err)
	}
	var puzzles = make([]Puzzle, len(results))
	for i, result := range results {
		puzzles[i] = Puzzle(result)
	}
	return puzzles, nil
}

func (c *Client) CreatePuzzle(ctx context.Context, puzzle RawPuzzle) (Puzzle, error) {
	var change PuzzleChange
	defer func() {
		if (change != PuzzleChange{}) {
			c.PuzzleChange <- change
		}
	}()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err := ValidatePuzzle(puzzle); err != nil {
		return Puzzle{}, err
	}
	id, err := c.queries.CreatePuzzle(ctx, db.CreatePuzzleParams{
		Name:           puzzle.Name,
		Answer:         puzzle.Answer,
		Round:          puzzle.Round,
		Status:         puzzle.Status,
		Note:           puzzle.Note,
		Location:       puzzle.Location,
		PuzzleURL:      puzzle.PuzzleURL,
		SpreadsheetID:  puzzle.SpreadsheetID,
		DiscordChannel: puzzle.DiscordChannel,
		Meta:           puzzle.Meta,
		VoiceRoom:      puzzle.VoiceRoom,
	})
	if err != nil {
		return Puzzle{}, xerrors.Errorf("CreatePuzzle: %w", err)
	}
	created, err := c.GetPuzzle(ctx, id)
	if err != nil {
		return Puzzle{}, err
	}
	change = PuzzleChange{nil, &created, true}
	return created, nil
}

func (c *Client) UpdatePuzzle(ctx context.Context, id int64,
	mutate func(puzzle *RawPuzzle) error) (Puzzle, error) {
	return c.UpdatePuzzleAdvanced(ctx, id, mutate, true)
}

func (c *Client) UpdatePuzzleAdvanced(
	ctx context.Context,
	id int64,
	mutate func(puzzle *RawPuzzle) error,
	sync bool,
) (Puzzle, error) {
	var change PuzzleChange
	defer func() {
		if (change != PuzzleChange{}) {
			c.PuzzleChange <- change
		}
	}()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	before, err := c.GetPuzzle(ctx, id)
	if err != nil {
		return Puzzle{}, err
	}
	var raw = before.RawPuzzle()
	if err := mutate(&raw); err != nil {
		return Puzzle{}, err
	} else if err := ValidatePuzzle(raw); err != nil {
		return Puzzle{}, err
	} else if raw.ID != id {
		return Puzzle{}, xerrors.Errorf("mutation must not change puzzle ID")
	} else if err := c.queries.UpdatePuzzle(ctx, db.UpdatePuzzleParams(raw)); err != nil {
		return Puzzle{}, xerrors.Errorf("UpdatePuzzle: %w", err)
	}

	after, err := c.GetPuzzle(ctx, id)
	if err != nil {
		return Puzzle{}, err
	}
	change = PuzzleChange{&before, &after, sync}
	return after, nil
}

func (c *Client) ClearPuzzleVoiceRoom(ctx context.Context, room string) error {
	// TODO: emit sync events
	return c.queries.ClearPuzzleVoiceRoom(ctx, room)
}

func (c *Client) DeletePuzzle(ctx context.Context, id int64) error {
	var change PuzzleChange
	defer func() {
		if (change != PuzzleChange{}) {
			c.PuzzleChange <- change
		}
	}()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	puzzle, err := c.GetPuzzle(ctx, id)
	if err != nil {
		return err
	}
	err = c.queries.DeletePuzzle(ctx, id)
	if err != nil {
		return xerrors.Errorf("DeletePuzzle: %w", err)
	}
	change = PuzzleChange{&puzzle, nil, true}
	return nil
}
