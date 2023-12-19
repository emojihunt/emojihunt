package db

import (
	"context"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

// ListPuzzles returns a list of all puzzles, including their contents.
func (c *Client) ListPuzzles(ctx context.Context) ([]Puzzle, error) {
	results, err := c.queries.ListPuzzles(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesFull: %w", err)
	}
	var puzzles = make([]Puzzle, len(results))
	for i, result := range results {
		puzzles[i] = Puzzle(result)
	}
	return puzzles, nil
}

// ListPuzzleFragmentsAndRounds returns a collection of all puzzle names and
// URLs, and a collection of all known rounds. It's used by the discovery script
// to deduplicate puzzles.
//
// Note that puzzle names and URLs are *uppercased* in the result map.
func (c *Client) ListPuzzleFragmentsAndRounds(ctx context.Context) (
	map[string]bool, map[string]int64, error) {

	var fragments = make(map[string]bool)
	puzzles, err := c.queries.ListPuzzles(ctx)
	if err != nil {
		return nil, nil, xerrors.Errorf("ListPuzzles: %w", err)
	}
	for _, puzzle := range puzzles {
		fragments[strings.ToUpper(puzzle.Name)] = true
		fragments[strings.ToUpper(puzzle.PuzzleURL)] = true
	}

	var rounds = make(map[string]int64)
	result, err := c.queries.ListRounds(ctx)
	if err != nil {
		return nil, nil, xerrors.Errorf("ListRounds: %w", err)
	}
	for _, round := range result {
		rounds[round.Name] = round.ID
	}

	return fragments, rounds, nil
}

type VoicePuzzle struct {
	ID        int64
	Name      string
	VoiceRoom string
}

// ListWithVoiceRoom returns all records with a voice room set. It returns a
// miniature struct containing just the puzzle name and voice room ID.
//
// The caller *must* acquire VoiceRoomMutex before calling this function.
func (c *Client) ListWithVoiceRoom(ctx context.Context) ([]VoicePuzzle, error) {
	puzzles, err := c.queries.ListPuzzlesWithVoiceRoom(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesWithVoiceRoom: %w", err)
	}

	var voicePuzzles []VoicePuzzle
	for _, puzzle := range puzzles {
		voicePuzzles = append(voicePuzzles, VoicePuzzle(puzzle))
	}
	return voicePuzzles, nil
}

type ReminderPuzzle struct {
	ID             int64
	Name           string
	DiscordChannel string
	Reminder       time.Time
}

type ReminderPuzzles []ReminderPuzzle

func (rps ReminderPuzzles) Len() int           { return len(rps) }
func (rps ReminderPuzzles) Less(i, j int) bool { return rps[i].Reminder.Before(rps[j].Reminder) }
func (rps ReminderPuzzles) Swap(i, j int)      { rps[i], rps[j] = rps[j], rps[i] }

// ListWithReminder returns all records with a reminder set. It returns a
// miniature struct containing just the puzzle name, channel ID, and reminder
// time.
//
// Results are returned in sorted order.
func (c *Client) ListWithReminder(ctx context.Context) ([]ReminderPuzzle, error) {
	puzzles, err := c.queries.ListPuzzlesWithReminder(ctx)
	if err != nil {
		return nil, xerrors.Errorf("ListPuzzlesWithReminder: %w", err)
	}

	var reminderPuzzles ReminderPuzzles
	for _, puzzle := range puzzles {
		reminderPuzzles = append(reminderPuzzles, ReminderPuzzle{
			ID:             puzzle.ID,
			Name:           puzzle.Name,
			DiscordChannel: puzzle.DiscordChannel,
			Reminder:       puzzle.Reminder.Time,
		})
	}
	return reminderPuzzles, nil
}

// LoadByID returns the given puzzle.
func (c *Client) LoadByID(ctx context.Context, id int64) (*Puzzle, error) {
	result, err := c.queries.GetPuzzle(ctx, id)
	if err != nil {
		return nil, xerrors.Errorf("GetPuzzle: %w", err)
	}
	var puzzle = Puzzle(result)
	return &puzzle, nil
}

func (c *Client) GetRawPuzzle(ctx context.Context, id int64) (RawPuzzle, error) {
	result, err := c.queries.GetRawPuzzle(ctx, id)
	if err != nil {
		return RawPuzzle{}, xerrors.Errorf("GetRawPuzzle: %w", err)
	}
	return result, nil
}

// LoadByDiscordChannel finds, locks and returns the matching record.
func (c *Client) LoadByDiscordChannel(ctx context.Context, channel string) (*Puzzle, error) {
	results, err := c.queries.GetPuzzlesByDiscordChannel(ctx, channel)
	if err != nil {
		return nil, err
	} else if len(results) < 1 {
		return nil, nil
	} else if len(results) > 1 {
		return nil, xerrors.Errorf("GetPuzzlesByDiscordChannel (%s)"+
			": expected 0 or 1 record, got %d", channel, len(results))
	}
	var puzzle = Puzzle(results[0])
	return &puzzle, nil
}
