package db

import (
	"context"
	"strings"

	"github.com/emojihunt/emojihunt/schema"
	"golang.org/x/xerrors"
)

// ListPuzzles returns a list of all known record IDs.
func (c *Client) ListPuzzles(ctx context.Context) ([]int64, error) {
	return c.queries.ListPuzzleIDs(ctx)
}

// ListPuzzlesFull returns a list of all puzzles, including their contents.
func (c *Client) ListPuzzlesFull(ctx context.Context) ([]*schema.Puzzle, error) {
	records, err := c.queries.ListPuzzlesFull(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesFull: %w", err)
	}

	var puzzles []*schema.Puzzle
	for _, record := range records {
		puzzles = append(puzzles, c.parseDatabaseResult(&record))
	}
	return puzzles, nil
}

// ListPuzzleFragmentsAndRounds returns a collection of all puzzle names and
// URLs, and a collection of all known rounds. It's used by the discovery script
// to deduplicate puzzles.
//
// Note that puzzle names and URLs are *uppercased* in the result map.
func (c *Client) ListPuzzleFragmentsAndRounds(ctx context.Context) (
	map[string]bool, map[string]schema.Round, error) {

	var fragments = make(map[string]bool)
	puzzles, err := c.queries.ListPuzzleDiscoveryFragments(ctx)
	if err != nil {
		return nil, nil, xerrors.Errorf("ListPuzzleDiscoveryFragments: %w", err)
	}
	for _, puzzle := range puzzles {
		fragments[strings.ToUpper(puzzle.Name)] = true
		fragments[strings.ToUpper(puzzle.PuzzleURL)] = true
		fragments[strings.ToUpper(puzzle.OriginalURL)] = true
	}

	var rounds = make(map[string]schema.Round)
	result, err := c.queries.ListRounds(ctx)
	if err != nil {
		return nil, nil, xerrors.Errorf("ListRounds: %w", err)
	}
	for _, round := range result {
		rounds[round.Name] = schema.Round{
			Name:  round.Name,
			Emoji: round.Emoji,
		}
	}

	return fragments, rounds, nil
}

// ListWithVoiceRoom returns all records with a voice room set. It returns a
// miniature struct containing just the puzzle name and voice room ID.
//
// The caller *must* acquire VoiceRoomMutex before calling this function.
func (c *Client) ListWithVoiceRoom(ctx context.Context) ([]schema.VoicePuzzle, error) {
	puzzles, err := c.queries.ListPuzzlesWithVoiceRoom(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesWithVoiceRoom: %w", err)
	}

	var voicePuzzles []schema.VoicePuzzle
	for _, puzzle := range puzzles {
		voicePuzzles = append(voicePuzzles, schema.VoicePuzzle{
			ID:        puzzle.ID,
			Name:      puzzle.Name,
			VoiceRoom: puzzle.VoiceRoom,
		})
	}
	return voicePuzzles, nil
}

// ListWithReminder returns all records with a reminder set. It returns a
// miniature struct containing just the puzzle name, channel ID, and reminder
// time.
//
// Results are returned in sorted order.
func (c *Client) ListWithReminder(ctx context.Context) ([]schema.ReminderPuzzle, error) {
	puzzles, err := c.queries.ListPuzzlesWithReminder(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesWithReminder: %w", err)
	}

	var reminderPuzzles schema.ReminderPuzzles
	for _, puzzle := range puzzles {
		reminderPuzzles = append(reminderPuzzles, schema.ReminderPuzzle{
			ID:             puzzle.ID,
			Name:           puzzle.Name,
			DiscordChannel: puzzle.DiscordChannel,
			Reminder:       puzzle.Reminder.Time,
		})
	}
	return reminderPuzzles, nil
}

// LoadByID returns the given puzzle.
func (c *Client) LoadByID(ctx context.Context, id int64) (*schema.Puzzle, error) {
	record, err := c.queries.GetPuzzle(ctx, id)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	puzzle := c.parseDatabaseResult(&record)
	return puzzle, nil
}

// LoadByDiscordChannel finds, locks and returns the matching record.
func (c *Client) LoadByDiscordChannel(ctx context.Context, channel string) (*schema.Puzzle, error) {
	response, err := c.queries.GetPuzzlesByDiscordChannel(ctx, channel)
	if err != nil {
		return nil, err
	} else if len(response) < 1 {
		return nil, nil
	} else if len(response) > 1 {
		return nil, xerrors.Errorf("GetPuzzlesByDiscordChannel (%s)"+
			": expected 0 or 1 record, got %d", channel, len(response))
	}
	puzzle := c.parseDatabaseResult(&response[0])
	return puzzle, nil
}

func (c *Client) parseDatabaseResult(record *Puzzle) *schema.Puzzle {
	return &schema.Puzzle{
		ID:           record.ID,
		Name:         record.Name,
		Answer:       record.Answer,
		Round:        schema.Round{},    // TODO
		Status:       schema.NotStarted, // TODO
		Description:  record.Description,
		Location:     record.Location,
		NameOverride: record.NameOverride,

		PuzzleURL:      record.PuzzleURL,
		SpreadsheetID:  record.SpreadsheetID,
		DiscordChannel: record.DiscordChannel,

		Archived:    record.Archived,
		OriginalURL: record.OriginalURL,
		VoiceRoom:   record.VoiceRoom,
	}
}
