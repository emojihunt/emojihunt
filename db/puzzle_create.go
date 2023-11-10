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
