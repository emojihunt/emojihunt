package client

import (
	"fmt"
	"sync"

	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/schema"
)

type Airtable struct {
	database *db.Queries

	// A map of Airtable Record ID -> puzzle mutex. The puzzle mutex should be
	// held while reading or writing the puzzle, and should be acquired before
	// the voice room mutex (if needed).
	mutexes *sync.Map
}

// FYI the Airtable library has a built-in rate limiter that will block if we
// exceed 4 requests per second. This will keep us under Airtable's 5
// requests-per-second limit, which is important because if we break that limit
// we get suspended for 30 seconds.

func NewAirtable(database *db.Queries) *Airtable {
	return &Airtable{
		database: database,
		mutexes:  &sync.Map{},
	}
}

func (air *Airtable) EditURL(puzzle *schema.Puzzle) string {
	return fmt.Sprintf(
		"https://airtable.com/TODO/TODO/%d", puzzle.AirtableRecord.ID,
	)
}
