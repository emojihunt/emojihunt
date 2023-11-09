package db

import (
	"context"

	"github.com/emojihunt/emojihunt/schema"
	"golang.org/x/xerrors"
)

// AddPuzzles creates the given puzzles and returns the created records as a
// list of schema.Puzzle objects.
func (c *Client) AddPuzzles(ctx context.Context, puzzles []schema.NewPuzzle, newRound bool) ([]schema.Puzzle, error) {
	if newRound {
		return nil, xerrors.Errorf("TODO: insert-round logic")
	}

	var created []schema.Puzzle
	for _, puzzle := range puzzles {
		record, err := c.queries.CreatePuzzle(ctx, CreatePuzzleParams{
			Name:        puzzle.Name,
			Round:       0, // TODO
			PuzzleURL:   puzzle.PuzzleURL,
			OriginalURL: puzzle.OriginalURL,
		})
		if err != nil {
			return created, err
		}
		parsed := c.parseDatabaseResult(&record)
		created = append(created, *parsed)
	}
	return created, nil
}
