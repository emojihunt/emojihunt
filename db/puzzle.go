package db

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/emojihunt/emojihunt/db/field"
)

// Fields must match GetPuzzleRow and friends
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

func (p Puzzle) SpreadsheetURL() string {
	if p.SpreadsheetID == "" {
		panic("called SpreadsheetURL() on a puzzle with no spreadsheet")
	}
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", p.SpreadsheetID)
}

func (p Puzzle) ShouldArchive() bool {
	// We shouldn't archive the channel until the answer has been filled in on
	// Airtable
	return p.Status.IsSolved() && p.Answer != ""
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
