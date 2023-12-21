-- name: GetPuzzle :one
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.archived, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.id = ?;

-- name: ListPuzzles :many
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.archived, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
ORDER BY p.id;

-- name: CreatePuzzle :one
INSERT INTO puzzles (
    name, answer, round, status, note, location, puzzle_url,
    spreadsheet_id, discord_channel, meta, archived, voice_room,
    reminder
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id;

-- name: UpdatePuzzle :exec
UPDATE puzzles
SET name = ?2, answer = ?3, round = ?4, status = ?5, note = ?6,
location = ?7, puzzle_url = ?8, spreadsheet_id = ?9, discord_channel = ?10,
meta = ?11, archived = ?12, voice_room = ?13, reminder = ?14
WHERE id = ?1;

-- name: DeletePuzzle :exec
DELETE FROM puzzles
WHERE id = ?;


-- name: GetRound :one
SELECT * FROM rounds
WHERE id = ? LIMIT 1;

-- name: ListRounds :many
SELECT * FROM rounds
ORDER BY id;

-- name: CreateRound :one
INSERT INTO rounds (name, emoji, hue, special)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateRound :exec
UPDATE rounds
SET name = ?2, emoji = ?3, hue = ?4, special = ?5
WHERE id = ?1;

-- name: DeleteRound :exec
DELETE FROM rounds
WHERE id = ?;


-- name: GetSetting :one
SELECT value from settings
WHERE key = ?;

-- name: UpdateSetting :exec
INSERT OR REPLACE INTO settings (key, value)
VALUES (?, ?);
