// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0
// source: query.sql

package db

import (
	"context"
	"database/sql"

	"github.com/emojihunt/emojihunt/db/field"
)

const createPuzzle = `-- name: CreatePuzzle :one
INSERT INTO puzzles (
    name, answer, round, status, description, location, puzzle_url,
    spreadsheet_id, discord_channel, original_url, name_override,
    archived, voice_room
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
`

type CreatePuzzleParams struct {
	Name           string       `json:"name"`
	Answer         string       `json:"answer"`
	Round          int64        `json:"round"`
	Status         field.Status `json:"status"`
	Description    string       `json:"description"`
	Location       string       `json:"location"`
	PuzzleURL      string       `json:"puzzle_url"`
	SpreadsheetID  string       `json:"spreadsheet_id"`
	DiscordChannel string       `json:"discord_channel"`
	OriginalURL    string       `json:"original_url"`
	NameOverride   string       `json:"name_override"`
	Archived       bool         `json:"archived"`
	VoiceRoom      string       `json:"voice_room"`
}

func (q *Queries) CreatePuzzle(ctx context.Context, arg CreatePuzzleParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, createPuzzle,
		arg.Name,
		arg.Answer,
		arg.Round,
		arg.Status,
		arg.Description,
		arg.Location,
		arg.PuzzleURL,
		arg.SpreadsheetID,
		arg.DiscordChannel,
		arg.OriginalURL,
		arg.NameOverride,
		arg.Archived,
		arg.VoiceRoom,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const createRound = `-- name: CreateRound :one
INSERT INTO rounds (name, emoji)
VALUES (?, ?)
RETURNING id, name, emoji
`

type CreateRoundParams struct {
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

func (q *Queries) CreateRound(ctx context.Context, arg CreateRoundParams) (Round, error) {
	row := q.db.QueryRowContext(ctx, createRound, arg.Name, arg.Emoji)
	var i Round
	err := row.Scan(&i.ID, &i.Name, &i.Emoji)
	return i, err
}

const getPuzzle = `-- name: GetPuzzle :one
SELECT
    p.id, p.name, p.answer, rounds.id, rounds.name, rounds.emoji, p.status, p.description,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.original_url, p.name_override, p.archived, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE puzzles.id = ? LIMIT 1
`

type GetPuzzleRow struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	Answer         string       `json:"answer"`
	Round          Round        `json:"round"`
	Status         field.Status `json:"status"`
	Description    string       `json:"description"`
	Location       string       `json:"location"`
	PuzzleURL      string       `json:"puzzle_url"`
	SpreadsheetID  string       `json:"spreadsheet_id"`
	DiscordChannel string       `json:"discord_channel"`
	OriginalURL    string       `json:"original_url"`
	NameOverride   string       `json:"name_override"`
	Archived       bool         `json:"archived"`
	VoiceRoom      string       `json:"voice_room"`
	Reminder       sql.NullTime `json:"reminder"`
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
		&i.Status,
		&i.Description,
		&i.Location,
		&i.PuzzleURL,
		&i.SpreadsheetID,
		&i.DiscordChannel,
		&i.OriginalURL,
		&i.NameOverride,
		&i.Archived,
		&i.VoiceRoom,
		&i.Reminder,
	)
	return i, err
}

const getPuzzlesByDiscordChannel = `-- name: GetPuzzlesByDiscordChannel :many
SELECT
    p.id, p.name, p.answer, rounds.id, rounds.name, rounds.emoji, p.status, p.description,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.original_url, p.name_override, p.archived, p.voice_room, p.reminder
FROM puzzles as p
INNER JOIN rounds on p.round = rounds.id
WHERE discord_channel = ?
`

type GetPuzzlesByDiscordChannelRow struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	Answer         string       `json:"answer"`
	Round          Round        `json:"round"`
	Status         field.Status `json:"status"`
	Description    string       `json:"description"`
	Location       string       `json:"location"`
	PuzzleURL      string       `json:"puzzle_url"`
	SpreadsheetID  string       `json:"spreadsheet_id"`
	DiscordChannel string       `json:"discord_channel"`
	OriginalURL    string       `json:"original_url"`
	NameOverride   string       `json:"name_override"`
	Archived       bool         `json:"archived"`
	VoiceRoom      string       `json:"voice_room"`
	Reminder       sql.NullTime `json:"reminder"`
}

func (q *Queries) GetPuzzlesByDiscordChannel(ctx context.Context, discordChannel string) ([]GetPuzzlesByDiscordChannelRow, error) {
	rows, err := q.db.QueryContext(ctx, getPuzzlesByDiscordChannel, discordChannel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetPuzzlesByDiscordChannelRow
	for rows.Next() {
		var i GetPuzzlesByDiscordChannelRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Answer,
			&i.Round.ID,
			&i.Round.Name,
			&i.Round.Emoji,
			&i.Status,
			&i.Description,
			&i.Location,
			&i.PuzzleURL,
			&i.SpreadsheetID,
			&i.DiscordChannel,
			&i.OriginalURL,
			&i.NameOverride,
			&i.Archived,
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

const getRound = `-- name: GetRound :one
SELECT id, name, emoji FROM rounds
WHERE id = ? LIMIT 1
`

func (q *Queries) GetRound(ctx context.Context, id int64) (Round, error) {
	row := q.db.QueryRowContext(ctx, getRound, id)
	var i Round
	err := row.Scan(&i.ID, &i.Name, &i.Emoji)
	return i, err
}

const getState = `-- name: GetState :many
SELECT id, data from state
ORDER BY id
`

func (q *Queries) GetState(ctx context.Context) ([]State, error) {
	rows, err := q.db.QueryContext(ctx, getState)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []State
	for rows.Next() {
		var i State
		if err := rows.Scan(&i.ID, &i.Data); err != nil {
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

const listPuzzleDiscoveryFragments = `-- name: ListPuzzleDiscoveryFragments :many
SELECT id, name, puzzle_url, original_url FROM puzzles
ORDER BY id
`

type ListPuzzleDiscoveryFragmentsRow struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	PuzzleURL   string `json:"puzzle_url"`
	OriginalURL string `json:"original_url"`
}

func (q *Queries) ListPuzzleDiscoveryFragments(ctx context.Context) ([]ListPuzzleDiscoveryFragmentsRow, error) {
	rows, err := q.db.QueryContext(ctx, listPuzzleDiscoveryFragments)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPuzzleDiscoveryFragmentsRow
	for rows.Next() {
		var i ListPuzzleDiscoveryFragmentsRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.PuzzleURL,
			&i.OriginalURL,
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

const listPuzzles = `-- name: ListPuzzles :many
SELECT
    p.id, p.name, p.answer, rounds.id, rounds.name, rounds.emoji, p.status, p.description,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.original_url, p.name_override, p.archived, p.voice_room, p.reminder
FROM puzzles as p
INNER JOIN rounds on p.round = rounds.id
ORDER BY p.id
`

type ListPuzzlesRow struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	Answer         string       `json:"answer"`
	Round          Round        `json:"round"`
	Status         field.Status `json:"status"`
	Description    string       `json:"description"`
	Location       string       `json:"location"`
	PuzzleURL      string       `json:"puzzle_url"`
	SpreadsheetID  string       `json:"spreadsheet_id"`
	DiscordChannel string       `json:"discord_channel"`
	OriginalURL    string       `json:"original_url"`
	NameOverride   string       `json:"name_override"`
	Archived       bool         `json:"archived"`
	VoiceRoom      string       `json:"voice_room"`
	Reminder       sql.NullTime `json:"reminder"`
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
			&i.Status,
			&i.Description,
			&i.Location,
			&i.PuzzleURL,
			&i.SpreadsheetID,
			&i.DiscordChannel,
			&i.OriginalURL,
			&i.NameOverride,
			&i.Archived,
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

const listPuzzlesWithReminder = `-- name: ListPuzzlesWithReminder :many
SELECT id, name, discord_channel, reminder FROM puzzles
WHERE reminder IS NOT NULL
ORDER BY reminder
`

type ListPuzzlesWithReminderRow struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	DiscordChannel string       `json:"discord_channel"`
	Reminder       sql.NullTime `json:"reminder"`
}

func (q *Queries) ListPuzzlesWithReminder(ctx context.Context) ([]ListPuzzlesWithReminderRow, error) {
	rows, err := q.db.QueryContext(ctx, listPuzzlesWithReminder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPuzzlesWithReminderRow
	for rows.Next() {
		var i ListPuzzlesWithReminderRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.DiscordChannel,
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

const listPuzzlesWithVoiceRoom = `-- name: ListPuzzlesWithVoiceRoom :many
SELECT id, name, voice_room FROM puzzles
WHERE voice_room != ""
ORDER BY id
`

type ListPuzzlesWithVoiceRoomRow struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	VoiceRoom string `json:"voice_room"`
}

func (q *Queries) ListPuzzlesWithVoiceRoom(ctx context.Context) ([]ListPuzzlesWithVoiceRoomRow, error) {
	rows, err := q.db.QueryContext(ctx, listPuzzlesWithVoiceRoom)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPuzzlesWithVoiceRoomRow
	for rows.Next() {
		var i ListPuzzlesWithVoiceRoomRow
		if err := rows.Scan(&i.ID, &i.Name, &i.VoiceRoom); err != nil {
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
SELECT id, name, emoji FROM rounds
ORDER BY id
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
		if err := rows.Scan(&i.ID, &i.Name, &i.Emoji); err != nil {
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

const updateArchived = `-- name: UpdateArchived :exec
UPDATE puzzles SET archived = ?2
WHERE id = ?1
`

type UpdateArchivedParams struct {
	ID       int64 `json:"id"`
	Archived bool  `json:"archived"`
}

func (q *Queries) UpdateArchived(ctx context.Context, arg UpdateArchivedParams) error {
	_, err := q.db.ExecContext(ctx, updateArchived, arg.ID, arg.Archived)
	return err
}

const updateDescription = `-- name: UpdateDescription :exec
UPDATE puzzles SET description = ?2
WHERE id = ?1
`

type UpdateDescriptionParams struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
}

func (q *Queries) UpdateDescription(ctx context.Context, arg UpdateDescriptionParams) error {
	_, err := q.db.ExecContext(ctx, updateDescription, arg.ID, arg.Description)
	return err
}

const updateDiscordChannel = `-- name: UpdateDiscordChannel :exec
UPDATE puzzles SET discord_channel = ?2
WHERE id = ?1
`

type UpdateDiscordChannelParams struct {
	ID             int64  `json:"id"`
	DiscordChannel string `json:"discord_channel"`
}

func (q *Queries) UpdateDiscordChannel(ctx context.Context, arg UpdateDiscordChannelParams) error {
	_, err := q.db.ExecContext(ctx, updateDiscordChannel, arg.ID, arg.DiscordChannel)
	return err
}

const updateLocation = `-- name: UpdateLocation :exec
UPDATE puzzles SET location = ?2
WHERE id = ?1
`

type UpdateLocationParams struct {
	ID       int64  `json:"id"`
	Location string `json:"location"`
}

func (q *Queries) UpdateLocation(ctx context.Context, arg UpdateLocationParams) error {
	_, err := q.db.ExecContext(ctx, updateLocation, arg.ID, arg.Location)
	return err
}

const updateRound = `-- name: UpdateRound :exec
UPDATE rounds
SET name = ?2, emoji = ?3
WHERE id = ?1
`

type UpdateRoundParams struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

func (q *Queries) UpdateRound(ctx context.Context, arg UpdateRoundParams) error {
	_, err := q.db.ExecContext(ctx, updateRound, arg.ID, arg.Name, arg.Emoji)
	return err
}

const updateSpreadsheetID = `-- name: UpdateSpreadsheetID :exec
UPDATE puzzles SET spreadsheet_id = ?2
WHERE id = ?1
`

type UpdateSpreadsheetIDParams struct {
	ID            int64  `json:"id"`
	SpreadsheetID string `json:"spreadsheet_id"`
}

func (q *Queries) UpdateSpreadsheetID(ctx context.Context, arg UpdateSpreadsheetIDParams) error {
	_, err := q.db.ExecContext(ctx, updateSpreadsheetID, arg.ID, arg.SpreadsheetID)
	return err
}

const updateState = `-- name: UpdateState :exec
INSERT OR REPLACE INTO state (id, data)
VALUES (1, ?)
`

func (q *Queries) UpdateState(ctx context.Context, data []byte) error {
	_, err := q.db.ExecContext(ctx, updateState, data)
	return err
}

const updateStatusAndAnswer = `-- name: UpdateStatusAndAnswer :exec
UPDATE puzzles SET status = ?2, answer = ?3, archived = ?4
WHERE id = ?1
`

type UpdateStatusAndAnswerParams struct {
	ID       int64        `json:"id"`
	Status   field.Status `json:"status"`
	Answer   string       `json:"answer"`
	Archived bool         `json:"archived"`
}

func (q *Queries) UpdateStatusAndAnswer(ctx context.Context, arg UpdateStatusAndAnswerParams) error {
	_, err := q.db.ExecContext(ctx, updateStatusAndAnswer,
		arg.ID,
		arg.Status,
		arg.Answer,
		arg.Archived,
	)
	return err
}

const updateVoiceRoom = `-- name: UpdateVoiceRoom :exec
UPDATE puzzles SET voice_room = ?2, location = ?3
WHERE id = ?1
`

type UpdateVoiceRoomParams struct {
	ID        int64  `json:"id"`
	VoiceRoom string `json:"voice_room"`
	Location  string `json:"location"`
}

func (q *Queries) UpdateVoiceRoom(ctx context.Context, arg UpdateVoiceRoomParams) error {
	_, err := q.db.ExecContext(ctx, updateVoiceRoom, arg.ID, arg.VoiceRoom, arg.Location)
	return err
}
