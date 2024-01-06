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
	Before   *Puzzle
	After    *Puzzle
	ChangeID int64

	// Whether or not the change was triggered from a bot command.
	Bot bool
}

type RoundChange struct {
	Before   *Round
	After    *Round
	ChangeID int64
}

type DiscoveredPuzzle struct {
	Name      string
	RoundName string
	URL       string
}

type DiscoveredRound struct {
	MessageID  string
	Name       string
	NotifiedAt time.Time
	Puzzles    []DiscoveredPuzzle
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
