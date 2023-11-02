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

	// Mutex mutex protects channelToRecord. It should be held briefly when
	// updating the map. We should never perform an operation that could block,
	// like acquiring another lock or making an API call, while holding mutex.
	mutex           *sync.Mutex
	channelToRecord map[string]string
}

// FYI the Airtable library has a built-in rate limiter that will block if we
// exceed 4 requests per second. This will keep us under Airtable's 5
// requests-per-second limit, which is important because if we break that limit
// we get suspended for 30 seconds.

func NewAirtable(database *db.Queries) *Airtable {
	return &Airtable{
		database:        database,
		mutexes:         &sync.Map{},
		mutex:           &sync.Mutex{},
		channelToRecord: make(map[string]string),
	}
}

func (air *Airtable) EditURL(puzzle *schema.Puzzle) string {
	return fmt.Sprintf(
		"https://airtable.com/TODO/TODO/%d", puzzle.AirtableRecord.ID,
	)
}
