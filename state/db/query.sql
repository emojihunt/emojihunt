-- name: GetPuzzle :one
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.id = ?;

-- name: GetPuzzleByChannel :one
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.discord_channel = ?;

-- name: GetPuzzlesByVoiceRoom :many
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.voice_room = ?;

-- name: ListPuzzles :many
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
ORDER BY rounds.special, rounds.sort, rounds.id, p.meta, p.name
COLLATE nocase;

-- name: ListPuzzlesByRound :many
SELECT
    p.id, p.name, p.answer, sqlc.embed(rounds), p.status, p.note,
    p.location, p.puzzle_url, p.spreadsheet_id, p.discord_channel,
    p.meta, p.voice_room, p.reminder
FROM puzzles AS p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.round = ?
ORDER BY rounds.special, rounds.sort, rounds.id, p.meta, p.name
COLLATE nocase;

-- name: ListPuzzlesByVoiceRoom :many
SELECT p.id, p.name, p.voice_room
FROM puzzles as p
INNER JOIN rounds ON p.round = rounds.id
WHERE p.voice_room != ""
ORDER BY p.voice_room, rounds.special, rounds.sort, rounds.id, p.meta, p.name
COLLATE nocase;

-- name: CreatePuzzle :one
INSERT INTO puzzles (
    name, answer, round, status, note, location, puzzle_url,
    spreadsheet_id, discord_channel, meta, voice_room, reminder
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id;

-- name: UpdatePuzzle :exec
UPDATE puzzles
SET name = ?2, answer = ?3, round = ?4, status = ?5, note = ?6,
location = ?7, puzzle_url = ?8, spreadsheet_id = ?9, discord_channel = ?10,
meta = ?11, voice_room = ?12, reminder = ?13
WHERE id = ?1;

-- name: ClearPuzzleVoiceRoom :exec
UPDATE puzzles
SET voice_room = ""
WHERE voice_room = ?;

-- name: DeletePuzzle :exec
DELETE FROM puzzles
WHERE id = ?;


-- name: GetRound :one
SELECT * FROM rounds
WHERE id = ? LIMIT 1;

-- name: ListRounds :many
SELECT * FROM rounds
ORDER BY special, sort, id
COLLATE nocase;

-- name: CreateRound :one
INSERT INTO rounds (
    name, emoji, hue, sort, special, drive_folder, discord_category
) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateRound :exec
UPDATE rounds
SET name = ?2, emoji = ?3, hue = ?4, sort = ?5, special = ?6,
    drive_folder = ?7, discord_category = ?8
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
