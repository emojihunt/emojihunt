package state

import (
	"crypto/sha256"
	"encoding/binary"
	"time"

	"github.com/emojihunt/emojihunt/db/field"
)

// Must match db.Round
type Round struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Emoji   string `json:"emoji"`
	Hue     int64  `json:"hue"`
	Special bool   `json:"special"`
}

// Must match db.GetPuzzleRow and db.ListPuzzlesRow
type Puzzle struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	Answer         string       `json:"answer"`
	Round          Round        `json:"round"`
	Status         field.Status `json:"status"`
	Note           string       `json:"note"`
	Location       string       `json:"location"`
	PuzzleURL      string       `json:"puzzle_url"`
	SpreadsheetID  string       `json:"spreadsheet_id"`
	DiscordChannel string       `json:"discord_channel"`
	Meta           bool         `json:"meta"`
	Archived       bool         `json:"archived"`
	VoiceRoom      string       `json:"voice_room"`
	Reminder       time.Time    `json:"reminder"`
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

var categories = []string{"A", "B", "C"}

func (p Puzzle) ArchiveCategory() string {
	// Hash the Discord channel ID, since it's not totally random
	h := sha256.New()
	if _, err := h.Write([]byte(p.DiscordChannel)); err != nil {
		panic(err)
	}
	i := binary.BigEndian.Uint64(h.Sum(nil)[:8])

	return categories[i%uint64(len(categories))]
}

func (p Puzzle) HasReminder() bool {
	return p.Reminder.Year() < 2000
}
