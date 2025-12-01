package state

import (
	"time"

	"github.com/emojihunt/emojihunt/state/db"
	"github.com/emojihunt/emojihunt/state/status"
)

type (
	Round     = db.Round
	RawPuzzle = db.Puzzle
	VoiceInfo = db.ListPuzzlesByVoiceRoomRow
)

// Must match db.GetPuzzleRow and db.ListPuzzlesRow
type Puzzle struct {
	ID             int64         `json:"id"`
	Name           string        `json:"name"`
	Answer         string        `json:"answer"`
	Round          Round         `json:"round"`
	Status         status.Status `json:"status"`
	Note           string        `json:"note"`
	Location       string        `json:"location"`
	PuzzleURL      string        `json:"puzzle_url"`
	SpreadsheetID  string        `json:"spreadsheet_id"`
	DiscordChannel string        `json:"discord_channel"`
	Meta           bool          `json:"meta"`
	VoiceRoom      string        `json:"voice_room"`
	Reminder       time.Time     `json:"reminder"`
}

type PuzzleChange struct {
	Before *Puzzle
	After  *Puzzle

	// If zero, don't broadcast the change to Ably.
	ChangeID int64

	// An optional channel to notify on completion. Only set when called from a
	// bot command.
	BotComplete chan error
}

func (change *PuzzleChange) SyncMessage() AblySyncMessage {
	var msg = AblySyncMessage{ChangeID: change.ChangeID}
	if change.After == nil {
		msg.Kind = status.AblyKindDelete
		msg.Puzzle = &AblyPuzzle{ID: change.Before.ID}
	} else {
		var encoded = change.After.AblyPuzzle()
		msg.Kind = status.AblyKindUpsert
		msg.Puzzle = &encoded
	}
	return msg
}

type RoundChange struct {
	Before   *Round
	After    *Round
	ChangeID int64
}

func (change *RoundChange) SyncMessage() AblySyncMessage {
	var msg = AblySyncMessage{ChangeID: change.ChangeID}
	if change.After == nil {
		msg.Kind = status.AblyKindDelete
		msg.Round = &Round{ID: change.Before.ID}
	} else {
		msg.Kind = status.AblyKindUpsert
		msg.Round = change.After
	}
	return msg
}

type LiveMessage struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type ScrapedPuzzle struct {
	Name      string `json:"name"`
	RoundName string `json:"round_name"`
	PuzzleURL string `json:"puzzle_url"`
}

type ScrapedRound struct {
	MessageID  string
	Name       string
	NotifiedAt time.Time
	Puzzles    []ScrapedPuzzle
}

func (p Puzzle) HasReminder() bool {
	return p.Reminder.Year() >= 2000
}

func (p Puzzle) RawPuzzle() RawPuzzle {
	return RawPuzzle{
		ID:             p.ID,
		Name:           p.Name,
		Answer:         p.Answer,
		Round:          p.Round.ID,
		Status:         p.Status,
		Note:           p.Note,
		Location:       p.Location,
		PuzzleURL:      p.PuzzleURL,
		SpreadsheetID:  p.SpreadsheetID,
		DiscordChannel: p.DiscordChannel,
		Meta:           p.Meta,
		VoiceRoom:      p.VoiceRoom,
		Reminder:       p.Reminder,
	}
}

// Works around an encoding bug involving time.Time
type AblyPuzzle struct {
	ID             int64         `json:"id"`
	Name           string        `json:"name"`
	Answer         string        `json:"answer"`
	Round          int64         `json:"round"`
	Status         status.Status `json:"status"`
	Note           string        `json:"note"`
	Location       string        `json:"location"`
	PuzzleURL      string        `json:"puzzle_url"`
	SpreadsheetID  string        `json:"spreadsheet_id"`
	DiscordChannel string        `json:"discord_channel"`
	Meta           bool          `json:"meta"`
	VoiceRoom      string        `json:"voice_room"`
	Reminder       string        `json:"reminder"`
}

type AblySyncMessage struct {
	ChangeID int64           `json:"change_id"`
	Kind     status.AblyKind `json:"kind"`
	Puzzle   *AblyPuzzle     `json:"puzzle,omitempty"`
	Round    *Round          `json:"round,omitempty"`
}

func (p Puzzle) AblyPuzzle() AblyPuzzle {
	return AblyPuzzle{
		ID:             p.ID,
		Name:           p.Name,
		Answer:         p.Answer,
		Round:          p.Round.ID,
		Status:         p.Status,
		Note:           p.Note,
		Location:       p.Location,
		PuzzleURL:      p.PuzzleURL,
		SpreadsheetID:  p.SpreadsheetID,
		DiscordChannel: p.DiscordChannel,
		Meta:           p.Meta,
		VoiceRoom:      p.VoiceRoom,
		Reminder:       p.Reminder.Format(time.RFC3339),
	}
}
