package db

import (
	"context"
)

type NewPuzzle struct {
	Name        string
	Round       string
	PuzzleURL   string
	OriginalURL string
}

// AddPuzzles creates the given puzzles and returns the created records as a
// list of Puzzle objects.
func (c *Client) AddPuzzles(ctx context.Context, puzzles []NewPuzzle) ([]Puzzle, error) {
	var created []Puzzle
	for _, puzzle := range puzzles {
		record, err := c.queries.CreatePuzzle(ctx, CreatePuzzleParams{
			Name:        puzzle.Name,
			Round:       1, // TODO
			PuzzleURL:   puzzle.PuzzleURL,
			OriginalURL: puzzle.OriginalURL,
		})
		if err != nil {
			return created, err
		}
		created = append(created, record)
	}
	return created, nil
}
