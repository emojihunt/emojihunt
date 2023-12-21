package state

import (
	"context"
	"net/url"

	"github.com/emojihunt/emojihunt/db"
	"golang.org/x/xerrors"
)

func ValidatePuzzle(p db.RawPuzzle) error {
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

func (c *Client) ListPuzzles(ctx context.Context) ([]Puzzle, error) {
	results, err := c.queries.ListPuzzles(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesFull: %w", err)
	}
	var puzzles = make([]Puzzle, len(results))
	for i, result := range results {
		puzzles[i] = Puzzle(result)
	}
	return puzzles, nil
}

func (c *Client) CreatePuzzle(ctx context.Context, puzzle db.RawPuzzle) (Puzzle, error) {
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
		Archived:       puzzle.Archived,
		VoiceRoom:      puzzle.VoiceRoom,
	})
	if err != nil {
		return Puzzle{}, xerrors.Errorf("CreatePuzzle: %w", err)
	}
	return c.GetPuzzle(ctx, id)
}

func (c *Client) UpdatePuzzle(ctx context.Context, puzzle db.RawPuzzle) error {
	if err := ValidatePuzzle(puzzle); err != nil {
		return err
	}
	err := c.queries.UpdatePuzzle(ctx, db.UpdatePuzzleParams(puzzle))
	if err != nil {
		return xerrors.Errorf("UpdatePuzzle: %w", err)
	}
	return nil
}

func (c *Client) DeletePuzzle(ctx context.Context, id int64) error {
	err := c.queries.DeletePuzzle(ctx, id)
	if err != nil {
		return xerrors.Errorf("DeletePuzzle: %w", err)
	}
	return nil
}
