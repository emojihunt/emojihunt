// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0

package db

import (
	"database/sql"
)

type Puzzle struct {
	ID             int64
	Name           string
	Answer         string
	Round          sql.NullInt64
	Status         string
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
