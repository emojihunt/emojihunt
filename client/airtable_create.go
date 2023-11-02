package client

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/schema"
)

// AddPuzzles creates the given puzzles in Airtable and returns the created
// records as a list of schema.Puzzle objects. It acquires the lock for each
// created puzzle; if the error is nil, the caller must call Unlock() on each
// puzzle.
func (air *Airtable) AddPuzzles(puzzles []schema.NewPuzzle, newRound bool) ([]schema.Puzzle, error) {
	if newRound {
		return nil, fmt.Errorf("TODO: insert-round logic")
	}

	var created []schema.Puzzle
	for _, puzzle := range puzzles {
		record, err := air.database.CreatePuzzle(context.TODO(), db.CreatePuzzleParams{
			Name:        puzzle.Name,
			Round:       sql.NullInt64{}, // TODO
			PuzzleUrl:   puzzle.PuzzleURL,
			OriginalUrl: puzzle.PuzzleURL,
		})
		if err != nil {
			return created, err
		}
		unlock := air.lockPuzzle(fmt.Sprintf("%d", record.ID))
		parsed := air.parseDatabaseResult(&record, unlock)
		created = append(created, *parsed)
	}
	return created, nil
}
