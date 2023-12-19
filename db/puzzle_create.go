package db

import (
	"context"

	"golang.org/x/xerrors"
)

type NewPuzzle struct {
	Name  string
	Round int64
	URL   string
}

// AddPuzzles creates the given puzzles and returns the created records as a
// list of Puzzle objects.
func (c *Client) AddPuzzles(ctx context.Context, puzzles []NewPuzzle) ([]Puzzle, error) {
	var created []Puzzle
	for _, puzzle := range puzzles {
		id, err := c.queries.CreatePuzzle(ctx, CreatePuzzleParams{
			Name:        puzzle.Name,
			Round:       puzzle.Round,
			PuzzleURL:   puzzle.URL,
			OriginalURL: puzzle.URL,
		})
		if err != nil {
			return created, xerrors.Errorf("CreatePuzzle: %w", err)
		}
		record, err := c.queries.GetPuzzle(ctx, id)
		if err != nil {
			return created, xerrors.Errorf("GetPuzzle: %w", err)
		}
		created = append(created, Puzzle(record))
	}
	return created, nil
}

func (c *Client) CreatePuzzle(ctx context.Context, puzzle RawPuzzle) (*Puzzle, error) {
	if err := puzzle.Validate(); err != nil {
		return nil, err
	}
	id, err := c.queries.CreatePuzzle(ctx, CreatePuzzleParams{
		Name:           puzzle.Name,
		Answer:         puzzle.Answer,
		Round:          puzzle.Round,
		Status:         puzzle.Status,
		Note:           puzzle.Note,
		Location:       puzzle.Location,
		PuzzleURL:      puzzle.PuzzleURL,
		SpreadsheetID:  puzzle.SpreadsheetID,
		DiscordChannel: puzzle.DiscordChannel,
		OriginalURL:    puzzle.OriginalURL,
		NameOverride:   puzzle.NameOverride,
		Archived:       puzzle.Archived,
		VoiceRoom:      puzzle.VoiceRoom,
	})
	if err != nil {
		return nil, xerrors.Errorf("CreatePuzzle: %w", err)
	}
	return c.LoadByID(ctx, id)
}

func (c *Client) DeletePuzzle(ctx context.Context, id int64) error {
	err := c.queries.DeletePuzzle(ctx, id)
	if err != nil {
		return xerrors.Errorf("DeletePuzzle: %w", err)
	}
	return nil
}

func (p RawPuzzle) Validate() error {
	if p.Name == "" {
		return ValidationError{"name", "is required"}
	} else if p.Round == 0 {
		return ValidationError{"round", "is required"}
	} else if !p.Status.IsValid() {
		return ValidationError{"status", "is invalid"}
	} else if p.PuzzleURL == "" {
		return ValidationError{"puzzle_url", "is required"}
	} else if !p.Status.IsSolved() && p.Answer != "" {
		return ValidationError{"status", "is unsolved but answer is not blank"}
	} else if p.Status.IsSolved() && p.Answer == "" {
		return ValidationError{"status", "is solved but answer is blank"}
	}
	return nil
}
