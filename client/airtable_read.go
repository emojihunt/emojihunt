package client

import (
	"fmt"
	"log"
	"sync"

	"github.com/gauravjsingh/emojihunt/schema"
)

func (air *Airtable) ListRecords() ([]schema.Puzzle, error) {
	var puzzles []schema.Puzzle
	var offset = ""
	for {
		response, err := air.table.GetRecords().
			PageSize(pageSize).
			WithOffset(offset).
			Do()
		if err != nil {
			return nil, err
		}

		air.mutex.Lock()
		for _, record := range response.Records {
			if record.Deleted {
				// Skip deleted records? I think this field is only used in
				// response to DELETE requests, but let's check it just in case.
				continue
			}
			puzzle, err := air.parseRecord(record)
			if err != nil {
				air.mutex.Unlock()
				return nil, err
			} else if puzzle.DiscordChannel != "" {
				// Keep Discord Channel -> Airtable ID cache up-to-date; this is
				// why we need to be holding air.mu
				air.channelToRecord[puzzle.DiscordChannel] = puzzle.AirtableRecord.ID
			}
			puzzles = append(puzzles, *puzzle)
		}
		air.mutex.Unlock()

		if response.Offset != "" {
			// More records exist, continue to next request
			offset = response.Offset
		} else {
			// All done, return all records
			return puzzles, nil
		}
	}
}

func (air *Airtable) ListWithVoiceRoom() ([]*schema.Puzzle, error) {
	response, err := air.table.GetRecords().
		WithFilterFormula("{Voice Room}!=''").
		Do()
	if err != nil {
		return nil, err
	} else if response.Offset != "" {
		// This shouldn't happen, but if it does we fail instead of spending too
		// much of our rate limit on paginated requests.
		return nil, fmt.Errorf("airtable query failed: too many records have a voice room")
	}

	var puzzles []*schema.Puzzle
	air.mutex.Lock()
	for _, record := range response.Records {
		if record.Deleted {
			continue
		}
		puzzle, err := air.parseRecord(record)
		if err != nil {
			air.mutex.Unlock()
			return nil, err
		} else if puzzle.DiscordChannel != "" {
			// Keep Discord Channel -> Airtable ID cache up-to-date; this is
			// why we need to be holding air.mu
			air.channelToRecord[puzzle.DiscordChannel] = puzzle.AirtableRecord.ID
		}
		puzzles = append(puzzles, puzzle)
	}
	air.mutex.Unlock()
	return puzzles, nil
}

// LockByID locks the given Airtable record ID and loads the corresponding
// puzzle under the lock. If no error is returned, the caller is responsible for
// calling Unlock() on the puzzle.
func (air *Airtable) LockByID(id string) (*schema.Puzzle, error) {
	unlock := air.lockPuzzle(id)

	record, err := air.table.GetRecord(id)
	if err != nil {
		unlock()
		return nil, err
	}
	puzzle, err := air.parseRecord(record)
	if err != nil {
		unlock()
		return nil, err
	} else if puzzle.DiscordChannel != "" {
		// Keep Discord Channel -> Airtable ID cache up-to-date
		air.mutex.Lock()
		air.channelToRecord[puzzle.DiscordChannel] = puzzle.AirtableRecord.ID
		air.mutex.Unlock()
	}

	puzzle.Unlock = unlock
	return puzzle, nil
}

// LockByDiscordChannel maps the given Discord channel ID using our cache, then
// locks the given Airtable record ID and loads the corresponding puzzle under
// the lock. If no error is returned, the caller is responsible for calling
// Unlock() on the puzzle.
func (air *Airtable) LockByDiscordChannel(channel string) (*schema.Puzzle, error) {
	air.mutex.Lock()
	expectedRecordID := air.channelToRecord[channel]
	air.mutex.Unlock()

	unlock := air.lockPuzzle(expectedRecordID)

	response, err := air.table.GetRecords().
		WithFilterFormula(fmt.Sprintf("{Discord Channel}='%s'", channel)).
		Do()
	if err != nil {
		unlock()
		return nil, err
	}
	if len(response.Records) < 1 {
		unlock()
		return nil, nil
	} else if len(response.Records) > 1 {
		unlock()
		return nil, fmt.Errorf("expected 0 or 1 record, got: %#v", response.Records)
	}
	puzzle, err := air.parseRecord(response.Records[0])
	if err != nil {
		unlock()
		return nil, err
	} else if puzzle.DiscordChannel != channel {
		// Airtable returned an incorrect response to our query?!
		unlock()
		return nil, fmt.Errorf("expected puzzle %q to have discord channel %q, got %q",
			puzzle.AirtableRecord.ID, channel, puzzle.DiscordChannel,
		)
	} else if puzzle.AirtableRecord.ID != expectedRecordID {
		// Our cache is out of date, oops...update it and retry.
		unlock()
		air.mutex.Lock()
		air.channelToRecord[puzzle.DiscordChannel] = puzzle.AirtableRecord.ID
		air.mutex.Unlock()
		return air.LockByDiscordChannel(channel)
	}

	puzzle.Unlock = unlock
	return puzzle, nil
}

func (air *Airtable) lockPuzzle(id string) func() {
	// https://stackoverflow.com/a/64612611
	value, _ := air.mutexes.LoadOrStore(id, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	log.Printf("lock: acquiring %q", id)
	mu.Lock()
	log.Printf("lock: acquired %q", id)
	return func() { mu.Unlock(); log.Printf("lock: released %q", id) }
}
