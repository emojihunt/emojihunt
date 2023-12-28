package state

import (
	"context"
	"database/sql"
	"errors"
	"net/url"

	"github.com/emojihunt/emojihunt/state/db"
	"golang.org/x/xerrors"
)

func (c *Client) ValidatePuzzle(ctx context.Context, p RawPuzzle) error {
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

	if p.DiscordChannel != "" {
		puzzle, err := c.queries.GetPuzzleByChannel(ctx, p.DiscordChannel)
		if err == nil && puzzle.ID != p.ID {
			return ValidationError{"discord_channel", "is not unique"}
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ValidationError{"discord_channel", "is not unique"}
		}
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

func (c *Client) ListPuzzlesByRound(ctx context.Context, round int64) ([]Puzzle, error) {
	results, err := c.queries.ListPuzzlesByRound(ctx, round)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesByRound: %w", err)
	}
	var puzzles = make([]Puzzle, len(results))
	for i, result := range results {
		puzzles[i] = Puzzle(result)
	}
	return puzzles, nil
}

func (c *Client) CreatePuzzle(ctx context.Context, puzzle RawPuzzle) (Puzzle, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err := c.ValidatePuzzle(ctx, puzzle); err != nil {
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
	c.PuzzleChange <- PuzzleChange{nil, &created, false}
	return created, nil
}

func (c *Client) UpdatePuzzle(ctx context.Context, id int64,
	mutate func(puzzle *RawPuzzle) error) (Puzzle, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	before, err := c.GetPuzzle(ctx, id)
	if err != nil {
		return Puzzle{}, err
	}
	var raw = before.RawPuzzle()
	if err := mutate(&raw); err != nil {
		return Puzzle{}, err
	} else if err := c.ValidatePuzzle(ctx, raw); err != nil {
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
	c.PuzzleChange <- PuzzleChange{&before, &after, false}
	return after, nil
}

func (c *Client) UpdatePuzzleByDiscordChannel(ctx context.Context, channel string,
	mutate func(puzzle *RawPuzzle) error) (Puzzle, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	before, err := c.GetPuzzleByChannel(ctx, channel)
	if err != nil {
		return Puzzle{}, err
	}
	var raw = before.RawPuzzle()
	if err := mutate(&raw); err != nil {
		return Puzzle{}, err
	} else if err := c.ValidatePuzzle(ctx, raw); err != nil {
		return Puzzle{}, err
	} else if raw.ID != before.ID {
		return Puzzle{}, xerrors.Errorf("mutation must not change puzzle ID")
	} else if err := c.queries.UpdatePuzzle(ctx, db.UpdatePuzzleParams(raw)); err != nil {
		return Puzzle{}, xerrors.Errorf("UpdatePuzzle: %w", err)
	}

	after, err := c.GetPuzzle(ctx, before.ID)
	if err != nil {
		return Puzzle{}, err
	}
	c.PuzzleChange <- PuzzleChange{&before, &after, true}
	return after, nil
}

func (c *Client) ClearPuzzleVoiceRoom(ctx context.Context, room string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	rows, err := c.queries.GetPuzzlesByVoiceRoom(ctx, room)
	if err != nil {
		return xerrors.Errorf("GetPuzzlesByVoiceRoom: %w", err)
	}
	err = c.queries.ClearPuzzleVoiceRoom(ctx, room)
	if err != nil {
		return xerrors.Errorf("ClearPuzzleVoiceRoom: %w", err)
	}
	for _, row := range rows {
		var before, after = Puzzle(row), Puzzle(row)
		after.VoiceRoom = ""
		c.PuzzleChange <- PuzzleChange{&before, &after, false}
	}
	return nil
}

func (c *Client) DeletePuzzle(ctx context.Context, id int64) error {
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
	c.PuzzleChange <- PuzzleChange{&puzzle, nil, false}
	return nil
}
