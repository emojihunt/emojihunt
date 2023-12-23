// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0
// source: query.sql

package db

import (
	"context"
	"time"

	"github.com/emojihunt/emojihunt/state/status"
)

const clearPuzzleVoiceRoom = `-- name: ClearPuzzleVoiceRoom :exec
UPDATE puzzles
SET voice_room = ""
WHERE voice_room = ?
`

func (q *Queries) ClearPuzzleVoiceRoom(ctx context.Context, voiceRoom string) error {
	_, err := q.db.ExecContext(ctx, clearPuzzleVoiceRoom, voiceRoom)
	return err
}

const createPuzzle = `-- name: CreatePuzzle :one
INSERT INTO puzzles (
    name, answer, round, status, note, location, puzzle_url,
    spreadsheet_id, discord_channel, meta, voice_room, reminder
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
`

type CreatePuzzleParams struct {
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

func (q *Queries) CreatePuzzle(ctx context.Context, arg CreatePuzzleParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, createPuzzle,
		arg.Name,
		arg.Answer,
		arg.Round,
		arg.Status,
		arg.Note,
		arg.Location,
		arg.PuzzleURL,
		arg.SpreadsheetID,
		arg.DiscordChannel,
		arg.Meta,
		arg.VoiceRoom,
		arg.Reminder,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const createRound = `-- name: CreateRound :one
INSERT INTO rounds (name, emoji, hue, special, drive_folder, discord_category)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, name, emoji, hue, special, drive_folder, discord_category
`

type CreateRoundParams struct {
	Name            string `json:"name"`
	Emoji           string `json:"emoji"`
	Hue             int64  `json:"hue"`
	Special         bool   `json:"special"`
	DriveFolder     string `json:"drive_folder"`
	DiscordCategory string `json:"discord_category"`
}

func (q *Queries) CreateRound(ctx context.Context, arg CreateRoundParams) (Round, error) {
	row := q.db.QueryRowContext(ctx, createRound,
		arg.Name,
		arg.Emoji,
		arg.Hue,
		arg.Special,
		arg.DriveFolder,
		arg.DiscordCategory,
	)
	var i Round
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Emoji,
		&i.Hue,
		&i.Special,
		&i.DriveFolder,
		&i.DiscordCategory,
	)
	return i, err
}

const deletePuzzle = `-- name: DeletePuzzle :exec
DELETE FROM puzzles
WHERE id = ?
`

func (q *Queries) DeletePuzzle(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, deletePuzzle, id)
	return err
}

const deleteRound = `-- name: DeleteRound :exec
DELETE FROM rounds
WHERE id = ?
`

func (q *Queries) DeleteRound(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, deleteRound, id)
	return err
}

const getPuzzle = `-- name: GetPuzzle :one
SELECT
    p.id, p.name, p.answer, rounds.id, rounds.name, rounds.emoji, rounds.hue, rounds.special, rounds.drive_folder, rounds.discord_category, p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.id = ?
`

type GetPuzzleRow struct {
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

func (q *Queries) GetPuzzle(ctx context.Context, id int64) (GetPuzzleRow, error) {
	row := q.db.QueryRowContext(ctx, getPuzzle, id)
	var i GetPuzzleRow
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Answer,
		&i.Round.ID,
		&i.Round.Name,
		&i.Round.Emoji,
		&i.Round.Hue,
		&i.Round.Special,
		&i.Round.DriveFolder,
		&i.Round.DiscordCategory,
		&i.Status,
		&i.Note,
		&i.Location,
		&i.PuzzleURL,
		&i.SpreadsheetID,
		&i.DiscordChannel,
		&i.Meta,
		&i.VoiceRoom,
		&i.Reminder,
	)
	return i, err
}

const getPuzzleByChannel = `-- name: GetPuzzleByChannel :one
SELECT
    p.id, p.name, p.answer, rounds.id, rounds.name, rounds.emoji, rounds.hue, rounds.special, rounds.drive_folder, rounds.discord_category, p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.discord_channel = ?
`

type GetPuzzleByChannelRow struct {
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

func (q *Queries) GetPuzzleByChannel(ctx context.Context, discordChannel string) (GetPuzzleByChannelRow, error) {
	row := q.db.QueryRowContext(ctx, getPuzzleByChannel, discordChannel)
	var i GetPuzzleByChannelRow
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Answer,
		&i.Round.ID,
		&i.Round.Name,
		&i.Round.Emoji,
		&i.Round.Hue,
		&i.Round.Special,
		&i.Round.DriveFolder,
		&i.Round.DiscordCategory,
		&i.Status,
		&i.Note,
		&i.Location,
		&i.PuzzleURL,
		&i.SpreadsheetID,
		&i.DiscordChannel,
		&i.Meta,
		&i.VoiceRoom,
		&i.Reminder,
	)
	return i, err
}

const getRound = `-- name: GetRound :one
SELECT id, name, emoji, hue, special, drive_folder, discord_category FROM rounds
WHERE id = ? LIMIT 1
`

func (q *Queries) GetRound(ctx context.Context, id int64) (Round, error) {
	row := q.db.QueryRowContext(ctx, getRound, id)
	var i Round
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Emoji,
		&i.Hue,
		&i.Special,
		&i.DriveFolder,
		&i.DiscordCategory,
	)
	return i, err
}

const getSetting = `-- name: GetSetting :one
SELECT value from settings
WHERE key = ?
`

func (q *Queries) GetSetting(ctx context.Context, key interface{}) ([]byte, error) {
	row := q.db.QueryRowContext(ctx, getSetting, key)
	var value []byte
	err := row.Scan(&value)
	return value, err
}

const listPuzzles = `-- name: ListPuzzles :many
SELECT
    p.id, p.name, p.answer, rounds.id, rounds.name, rounds.emoji, rounds.hue, rounds.special, rounds.drive_folder, rounds.discord_category, p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
ORDER BY rounds.special, rounds.id, p.meta, p.name
`

type ListPuzzlesRow struct {
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

func (q *Queries) ListPuzzles(ctx context.Context) ([]ListPuzzlesRow, error) {
	rows, err := q.db.QueryContext(ctx, listPuzzles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPuzzlesRow
	for rows.Next() {
		var i ListPuzzlesRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Answer,
			&i.Round.ID,
			&i.Round.Name,
			&i.Round.Emoji,
			&i.Round.Hue,
			&i.Round.Special,
			&i.Round.DriveFolder,
			&i.Round.DiscordCategory,
			&i.Status,
			&i.Note,
			&i.Location,
			&i.PuzzleURL,
			&i.SpreadsheetID,
			&i.DiscordChannel,
			&i.Meta,
			&i.VoiceRoom,
			&i.Reminder,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listRounds = `-- name: ListRounds :many
SELECT id, name, emoji, hue, special, drive_folder, discord_category FROM rounds
ORDER BY special, id
`

func (q *Queries) ListRounds(ctx context.Context) ([]Round, error) {
	rows, err := q.db.QueryContext(ctx, listRounds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Round
	for rows.Next() {
		var i Round
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Emoji,
			&i.Hue,
			&i.Special,
			&i.DriveFolder,
			&i.DiscordCategory,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updatePuzzle = `-- name: UpdatePuzzle :exec
UPDATE puzzles
SET name = ?2, answer = ?3, round = ?4, status = ?5, note = ?6,
location = ?7, puzzle_url = ?8, spreadsheet_id = ?9, discord_channel = ?10,
meta = ?11, voice_room = ?12, reminder = ?13
WHERE id = ?1
`

type UpdatePuzzleParams struct {
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

func (q *Queries) UpdatePuzzle(ctx context.Context, arg UpdatePuzzleParams) error {
	_, err := q.db.ExecContext(ctx, updatePuzzle,
		arg.ID,
		arg.Name,
		arg.Answer,
		arg.Round,
		arg.Status,
		arg.Note,
		arg.Location,
		arg.PuzzleURL,
		arg.SpreadsheetID,
		arg.DiscordChannel,
		arg.Meta,
		arg.VoiceRoom,
		arg.Reminder,
	)
	return err
}

const updateRound = `-- name: UpdateRound :exec
UPDATE rounds
SET name = ?2, emoji = ?3, hue = ?4, special = ?5, drive_folder = ?6,
    discord_category = ?7
WHERE id = ?1
`

type UpdateRoundParams struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Emoji           string `json:"emoji"`
	Hue             int64  `json:"hue"`
	Special         bool   `json:"special"`
	DriveFolder     string `json:"drive_folder"`
	DiscordCategory string `json:"discord_category"`
}

func (q *Queries) UpdateRound(ctx context.Context, arg UpdateRoundParams) error {
	_, err := q.db.ExecContext(ctx, updateRound,
		arg.ID,
		arg.Name,
		arg.Emoji,
		arg.Hue,
		arg.Special,
		arg.DriveFolder,
		arg.DiscordCategory,
	)
	return err
}

const updateSetting = `-- name: UpdateSetting :exec
INSERT OR REPLACE INTO settings (key, value)
VALUES (?, ?)
`

type UpdateSettingParams struct {
	Key   interface{} `json:"key"`
	Value []byte      `json:"value"`
}

func (q *Queries) UpdateSetting(ctx context.Context, arg UpdateSettingParams) error {
	_, err := q.db.ExecContext(ctx, updateSetting, arg.Key, arg.Value)
	return err
}
