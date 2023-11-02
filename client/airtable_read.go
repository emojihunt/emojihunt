package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/schema"
)

// ListPuzzles returns a list of all known record IDs.
func (air *Airtable) ListPuzzles() ([]string, error) {
	var ids []string
	puzzles, err := air.database.ListPuzzleIDs(context.TODO())
	if err != nil {
		return nil, err
	}
	for _, puzzle := range puzzles {
		ids = append(ids, fmt.Sprintf("%d", puzzle))
	}
	return ids, nil
}

// ListPuzzleFragmentsAndRounds returns a collection of all puzzle names and
// URLs, and a collection of all known rounds. It's used by the discovery script
// to deduplicate puzzles. No lock is held, but puzzle names, URLs and Original
// URLs are ~immutable (the bot will never write to them, and humans rarely do),
// so it's safe.
//
// Note that puzzle names and URLs are *uppercased* in the result map.
func (air *Airtable) ListPuzzleFragmentsAndRounds() (map[string]bool, map[string]db.Round, error) {
	var fragments = make(map[string]bool)
	puzzles, err := air.database.ListPuzzleDiscoveryFragments(context.TODO())
	if err != nil {
		return nil, nil, err
	}
	for _, puzzle := range puzzles {
		fragments[strings.ToUpper(puzzle.Name)] = true
		fragments[strings.ToUpper(puzzle.PuzzleUrl)] = true
		fragments[strings.ToUpper(puzzle.OriginalUrl)] = true
	}

	var rounds = make(map[string]db.Round)
	result, err := air.database.ListRounds(context.TODO())
	if err != nil {
		return nil, nil, err
	}
	for _, round := range result {
		rounds[round.Name] = round
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
	puzzles, err := air.database.ListPuzzlesWithVoiceRoom(context.TODO())
	if err != nil {
		return nil, err
	}

	var voicePuzzles []schema.VoicePuzzle
	for _, puzzle := range puzzles {
		voicePuzzles = append(voicePuzzles, schema.VoicePuzzle{
			RecordID:  fmt.Sprintf("%d", puzzle.ID),
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
	puzzles, err := air.database.ListPuzzlesWithReminder(context.TODO())
	if err != nil {
		return nil, err
	}

	var reminderPuzzles schema.ReminderPuzzles
	for _, puzzle := range puzzles {
		reminderPuzzles = append(reminderPuzzles, schema.ReminderPuzzle{
			RecordID:       fmt.Sprintf("%d", puzzle.ID),
			Name:           puzzle.Name,
			DiscordChannel: puzzle.DiscordChannel,
			Reminder:       puzzle.Reminder.Time,
		})
	}
	return reminderPuzzles, nil
}

// LockByID locks the given Airtable record ID and loads the corresponding
// puzzle under the lock. If no error is returned, the caller is responsible for
// calling Unlock() on the puzzle.
func (air *Airtable) LockByID(id string) (*schema.Puzzle, error) {
	unlock := air.lockPuzzle(id)

	i, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	record, err := air.database.GetPuzzle(context.TODO(), int64(i))
	if err != nil {
		unlock()
		return nil, err
	}
	puzzle := air.parseDatabaseResult(&record, unlock)
	if puzzle.DiscordChannel != "" {
		// Keep Discord Channel -> Airtable ID cache up-to-date
		air.mutex.Lock()
		air.channelToRecord[puzzle.DiscordChannel] = fmt.Sprintf("%d", puzzle.AirtableRecord.ID)
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

	response, err := air.database.GetPuzzlesByDiscordChannel(context.TODO(), channel)
	if err != nil {
		unlock()
		return nil, err
	}
	if len(response) < 1 {
		unlock()
		return nil, nil
	} else if len(response) > 1 {
		unlock()
		return nil, fmt.Errorf("expected 0 or 1 record, got %d", len(response))
	}
	puzzle := air.parseDatabaseResult(&response[0], unlock)
	if puzzle.DiscordChannel != channel {
		// Airtable returned an incorrect response to our query?!
		unlock()
		return nil, fmt.Errorf("expected puzzle %q to have discord channel %q, got %q",
			puzzle.AirtableRecord.ID, channel, puzzle.DiscordChannel,
		)
	} else if fmt.Sprintf("%d", puzzle.AirtableRecord.ID) != expectedRecordID {
		// Our cache is out of date, oops...update it and retry.
		unlock()
		air.mutex.Lock()
		air.channelToRecord[puzzle.DiscordChannel] = fmt.Sprintf("%d", puzzle.AirtableRecord.ID)
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

func (air *Airtable) parseDatabaseResult(record *db.Puzzle, unlock func()) *schema.Puzzle {
	return &schema.Puzzle{
		Name:         record.Name,
		Answer:       record.Answer,
		Rounds:       schema.Rounds{},   // TODO
		Status:       schema.NotStarted, // TODO
		Description:  record.Description,
		Location:     record.Location,
		NameOverride: record.NameOverride,

		AirtableRecord: record,
		PuzzleURL:      record.PuzzleUrl,
		SpreadsheetID:  record.SpreadsheetID,
		DiscordChannel: record.DiscordChannel,

		Archived:    record.Archived,
		OriginalURL: record.OriginalUrl,
		VoiceRoom:   record.VoiceRoom,

		Unlock: unlock,
	}
}
