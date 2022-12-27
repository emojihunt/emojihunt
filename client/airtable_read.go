package client

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/emojihunt/emojihunt/schema"
)

// ListApprovedPuzzles returns a list of all known record IDs.
func (air *Airtable) ListApprovedPuzzles() ([]string, error) {
	var ids []string
	puzzles, err := air.listRecordsWithFilter("")
	if err != nil {
		return nil, err
	}
	for _, puzzle := range puzzles {
		if !puzzle.Pending {
			ids = append(ids, puzzle.AirtableRecord.ID)
		}
	}
	return ids, nil
}

// ListPuzzlesToAction loads all puzzles from the Airtable API and returns two
// lists: a list of schema.InvalidPuzzle objects representing puzzles that
// failed basic validation (we can't even create the Discord channel and
// spreadsheet because they're missing basic information), and a list of record
// IDs for puzzles that need to be actioned by the syncer. No lock is held, so
// the caller needs to re-load each puzzle in the latter list with LockByID and
// make sure it still needs actioning.
func (air *Airtable) ListPuzzlesToAction() ([]schema.InvalidPuzzle, []string, error) {
	timestamp := time.Now()

	puzzles, err := air.listRecordsWithFilter("")
	if err != nil {
		return nil, nil, err
	}

	var invalid []schema.InvalidPuzzle // invalid, notify the QM
	var needsAction []string           // needs some kind of re-sync
	for _, puzzle := range puzzles {
		if puzzle.Pending {
			// Skip auto-added records that haven't been confirmed by a human
			continue
		} else if puzzle.LastModified == nil || timestamp.Sub(*puzzle.LastModified) < air.ModifyGracePeriod {
			// Skip puzzles that are being actively edited by a human
			continue
		} else if puzzle.DiscordChannel == "-" {
			// Skip puzzles that a QM has set to ignore
			continue
		} else if len(puzzle.Problems()) > 0 {
			invalid = append(invalid, schema.InvalidPuzzle{
				RecordID: puzzle.AirtableRecord.ID,
				Name:     puzzle.Name,
				Problems: puzzle.Problems(),
				EditURL:  air.EditURL(&puzzle),
			})
		} else if puzzle.SpreadsheetID == "" || puzzle.DiscordChannel == "" {
			needsAction = append(needsAction, puzzle.AirtableRecord.ID)
		} else if puzzle.Status != puzzle.LastBotStatus || puzzle.ShouldArchive() != puzzle.Archived {
			needsAction = append(needsAction, puzzle.AirtableRecord.ID)
		} else if puzzle.LastModifiedBy != air.BotUserID && puzzle.LastModifiedBy != "" {
			needsAction = append(needsAction, puzzle.AirtableRecord.ID)
		} else {
			// no-op
		}
	}
	return invalid, needsAction, nil
}

// ListPuzzleFragmentsAndRounds returns a collection of all puzzle names and
// URLs, and a collection of all known rounds. It's used by the discovery script
// to deduplicate puzzles. No lock is held, but puzzle names, URLs and Original
// URLs are ~immutable (the bot will never write to them, and humans rarely do),
// so it's safe.
//
// Note that puzzle names and URLs are *uppercased* in the result map.
func (air *Airtable) ListPuzzleFragmentsAndRounds() (map[string]bool, map[string]schema.Round, error) {
	puzzles, err := air.listRecordsWithFilter("")
	if err != nil {
		return nil, nil, err
	}

	var fragments = make(map[string]bool)
	var rounds = make(map[string]schema.Round)
	for _, puzzle := range puzzles {
		fragments[strings.ToUpper(puzzle.Name)] = true
		fragments[strings.ToUpper(puzzle.PuzzleURL)] = true
		fragments[strings.ToUpper(puzzle.OriginalURL)] = true

		for _, round := range puzzle.Rounds {
			rounds[round.Name] = round
		}
	}
	return fragments, rounds, nil
}

// ListWithVoiceRoom returns all records in Airtable with a voice room set.
// Instead of locking all of the matching puzzles, we return a miniature struct
// containing just the puzzle name and voice room ID. The puzzle name is safe to
// access because it's ~immutable, and the voice room ID is safe to access
// because it's only written when holding VoiceRoomMutex.
//
// The caller *must* acquire VoiceRoomMutex before calling this function.
func (air *Airtable) ListWithVoiceRoom() ([]schema.VoicePuzzle, error) {
	puzzles, err := air.listRecordsWithFilter("{Voice Room}!=''")
	if err != nil {
		return nil, err
	}

	var voicePuzzles []schema.VoicePuzzle
	for _, puzzle := range puzzles {
		voicePuzzles = append(voicePuzzles, schema.VoicePuzzle{
			RecordID:  puzzle.AirtableRecord.ID,
			Name:      puzzle.Name,
			VoiceRoom: puzzle.VoiceRoom,
		})
	}
	return voicePuzzles, nil
}

// ListWithReminder returns all records in Airtable with a reminder set. Instead
// of locking all of the matching puzzles, we return a miniature struct
// containing just the puzzle name, channel ID, and reminder time. The puzzle
// name and channel ID are safe to access because they're ~immutable; the
// reminder time is loaded at the current timestamp without coordination, which
// is less ideal but close enough.
//
// Results are returned in sorted order.
func (air *Airtable) ListWithReminder() ([]schema.ReminderPuzzle, error) {
	puzzles, err := air.listRecordsWithFilter("{Reminder}!=''")
	if err != nil {
		return nil, err
	}

	var reminderPuzzles schema.ReminderPuzzles
	for _, puzzle := range puzzles {
		if puzzle.Reminder == nil {
			return nil, fmt.Errorf("Airtable returned puzzle with nil reminder: %#v", puzzle)
		}
		reminderPuzzles = append(reminderPuzzles, schema.ReminderPuzzle{
			RecordID:       puzzle.AirtableRecord.ID,
			Name:           puzzle.Name,
			DiscordChannel: puzzle.DiscordChannel,
			Reminder:       *puzzle.Reminder,
		})
	}
	sort.Sort(reminderPuzzles)
	return reminderPuzzles, nil
}

// listRecordsWithFilter queries Airtable for all records matching the given
// filter. To list all records, pass the empty string as the filter.
//
// This function is for internal use only: no locks are acquired and callers are
// responsible for avoiding race conditions.
func (air *Airtable) listRecordsWithFilter(filter string) ([]schema.Puzzle, error) {
	var puzzles []schema.Puzzle
	var offset = ""
	for {
		request := air.table.GetRecords().
			PageSize(pageSize).
			WithOffset(offset)
		if filter != "" {
			request = request.WithFilterFormula(filter)
		}
		response, err := request.Do()
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
			puzzle, err := air.parseRecord(record, nil)
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
	puzzle, err := air.parseRecord(record, unlock)
	if err != nil {
		unlock()
		return nil, err
	} else if puzzle.DiscordChannel != "" {
		// Keep Discord Channel -> Airtable ID cache up-to-date
		air.mutex.Lock()
		air.channelToRecord[puzzle.DiscordChannel] = puzzle.AirtableRecord.ID
		air.mutex.Unlock()
	}
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
		return nil, fmt.Errorf("expected 0 or 1 record, got %d", len(response.Records))
	}
	puzzle, err := air.parseRecord(response.Records[0], unlock)
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
	return puzzle, nil
}

func (air *Airtable) lockPuzzle(id string) func() {
	// https://stackoverflow.com/a/64612611
	value, _ := air.mutexes.LoadOrStore(id, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return func() { mu.Unlock() }
}
