package client

import (
	"github.com/emojihunt/emojihunt/schema"
	"github.com/mehanizm/airtable"
)

// AddPuzzles creates the given puzzles in Airtable and returns the created
// records as a list of schema.Puzzle objects. It acquires the lock for each
// created puzzle; if the error is nil, the caller must call Unlock() on each
// puzzle.
func (air *Airtable) AddPuzzles(puzzles []schema.NewPuzzle) ([]schema.Puzzle, error) {
	var created []schema.Puzzle
	for i := 0; i < len(puzzles); i += 10 {
		records := airtable.Records{}
		limit := i + 10
		if limit > len(puzzles) {
			limit = len(puzzles)
		}
		for _, puzzle := range puzzles[i:limit] {
			fields := map[string]interface{}{
				"Name":         puzzle.Name + pendingSuffix,
				"Round":        puzzle.Round.Serialize(),
				"Puzzle URL":   puzzle.PuzzleURL,
				"Original URL": puzzle.PuzzleURL,
			}
			records.Records = append(records.Records,
				&airtable.Record{
					Fields: fields,
				},
			)
		}
		response, err := air.table.AddRecords(&records)
		if err != nil {
			return nil, err
		}
		for _, record := range response.Records {
			unlock := air.lockPuzzle(record.ID)
			parsed, err := air.parseRecord(record, unlock)
			if err != nil {
				return nil, err
			}
			created = append(created, *parsed)
		}
	}
	return created, nil
}
