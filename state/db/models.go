// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0

package db

import (
	"time"

	"github.com/emojihunt/emojihunt/state/status"
)

type Puzzle struct {
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
	Reminder       time.Time     `json:"reminder"`
}

type Round struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Emoji   string `json:"emoji"`
	Hue     int64  `json:"hue"`
	Special bool   `json:"special"`
}

type Setting struct {
	Key   interface{} `json:"key"`
	Value []byte      `json:"value"`
}
