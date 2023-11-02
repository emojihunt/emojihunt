// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0

package db

import (
	"database/sql"

	"github.com/emojihunt/emojihunt/schema"
)

type Puzzle struct {
	ID             int64
	Name           string
	Answer         string
	Rounds         schema.Rounds
	Status         schema.Status
	Description    string
	Location       string
	PuzzleURL      string
	SpreadsheetID  string
	DiscordChannel string
	OriginalURL    string
	NameOverride   string
	Archived       bool
	VoiceRoom      string
	Reminder       sql.NullTime
}

type Round struct {
	ID    int64
	Name  string
	Emoji string
}

type State struct {
	ID   int64
	Data []byte
}
