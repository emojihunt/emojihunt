package db

import (
	"context"
	"fmt"

	"github.com/emojihunt/emojihunt/schema"
)

// AddPuzzles creates the given puzzles and returns the created records as a
// list of schema.Puzzle objects. It acquires the lock for each created puzzle;
// if the error is nil, the caller must call Unlock() on each puzzle.
func (c *Client) AddPuzzles(puzzles []schema.NewPuzzle, newRound bool) ([]schema.Puzzle, error) {
	if newRound {
		return nil, fmt.Errorf("TODO: insert-round logic")
	}

	var created []schema.Puzzle
	for _, puzzle := range puzzles {
		record, err := c.queries.CreatePuzzle(context.TODO(), CreatePuzzleParams{
			Name:        puzzle.Name,
			Rounds:      schema.Rounds{puzzle.Round},
			PuzzleURL:   puzzle.PuzzleURL,
			OriginalURL: puzzle.PuzzleURL,
		})
		if err != nil {
			return created, err
		}
		unlock := c.lockPuzzle(record.ID)
		parsed := c.parseDatabaseResult(&record, unlock)
		created = append(created, *parsed)
	}
	return created, nil
}