-- name: GetPuzzle :one
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.archived, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.id = ?;

-- name: GetPuzzlesByDiscordChannel :many

SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.archived, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.discord_channel = ?;

-- name: GetRawPuzzle :one
SELECT * FROM puzzles
WHERE id = ? LIMIT 1;

-- name: ListPuzzles :many
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.archived, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
ORDER BY p.id;

-- name: ListPuzzlesWithVoiceRoom :many
SELECT id, name, voice_room FROM puzzles
WHERE voice_room != ""
ORDER BY id;

-- name: ListPuzzlesWithReminder :many
SELECT id, name, discord_channel, reminder FROM puzzles
WHERE reminder > 0
ORDER BY reminder;

-- name: CreatePuzzle :one
INSERT INTO puzzles (
    name, answer, round, status, note, location, puzzle_url,
    spreadsheet_id, discord_channel, meta, archived, voice_room,
    reminder
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id;

-- name: DeletePuzzle :exec
DELETE FROM puzzles
WHERE id = ?;

-- name: UpdatePuzzle :exec
UPDATE puzzles
SET name = ?2, answer = ?3, round = ?4, status = ?5, note = ?6,
location = ?7, puzzle_url = ?8, spreadsheet_id = ?9, discord_channel = ?10,
meta = ?11, archived = ?12, voice_room = ?13, reminder = ?14
WHERE id = ?1;

-- name: UpdateDiscordChannel :exec
UPDATE puzzles SET discord_channel = ?2
WHERE id = ?1;

-- name: UpdateSpreadsheetID :exec
UPDATE puzzles SET spreadsheet_id = ?2
WHERE id = ?1;

-- name: UpdateStatusAndAnswer :exec
UPDATE puzzles SET status = ?2, answer = ?3, archived = ?4
WHERE id = ?1;

-- name: UpdateNote :exec
UPDATE puzzles SET note = ?2
WHERE id = ?1;

-- name: UpdateLocation :exec
UPDATE puzzles SET location = ?2
WHERE id = ?1;

-- name: UpdateArchived :exec
UPDATE puzzles SET archived = ?2
WHERE id = ?1;

-- name: UpdateVoiceRoom :exec
UPDATE puzzles SET voice_room = ?2, location = ?3
WHERE id = ?1;


-- name: CreateRound :one
INSERT INTO rounds (name, emoji, hue, special)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetRound :one
SELECT * FROM rounds
WHERE id = ? LIMIT 1;

-- name: UpdateRound :exec
UPDATE rounds
SET name = ?2, emoji = ?3, hue = ?4, special = ?5
WHERE id = ?1;

-- name: DeleteRound :exec
DELETE FROM rounds
WHERE id = ?;

-- name: ListRounds :many
SELECT * FROM rounds
ORDER BY id;


-- name: GetState :many
SELECT * from state
ORDER BY id;

-- name: UpdateState :exec
INSERT OR REPLACE INTO state (id, data)
VALUES (1, ?);
