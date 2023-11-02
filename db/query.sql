-- name: GetPuzzle :one
SELECT * FROM puzzles
WHERE id = ? LIMIT 1;

-- name: GetPuzzlesByDiscordChannel :many
SELECT * from puzzles
WHERE discord_channel = ?;

-- name: ListPuzzleIDs :many
SELECT id FROM puzzles
ORDER BY id;

-- name: ListPuzzleDiscoveryFragments :many
SELECT id, name, puzzle_url, original_url FROM puzzles
ORDER BY id;

-- name: ListPuzzlesWithVoiceRoom :many
SELECT id, name, voice_room FROM puzzles
WHERE voice_room != ""
ORDER BY id;

-- name: ListPuzzlesWithReminder :many
SELECT id, name, discord_channel, reminder FROM puzzles
WHERE reminder IS NOT NULL
ORDER BY reminder;

-- name: ListRounds :many
SELECT * FROM rounds
ORDER BY id;

-- name: CreatePuzzle :one
INSERT INTO puzzles (name, round, puzzle_url, original_url) VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateDiscordChannel :one
UPDATE puzzles SET discord_channel = ?2
WHERE id = ?1
RETURNING *;

-- name: UpdateSpreadsheetID :one
UPDATE puzzles SET spreadsheet_id = ?2
WHERE id = ?1
RETURNING *;

-- name: UpdateStatusAndAnswer :one
UPDATE puzzles SET status = ?2, answer = ?3, archived = ?4
WHERE id = ?1
RETURNING *;

-- name: UpdateDescription :one
UPDATE puzzles SET description = ?2
WHERE id = ?1
RETURNING *;

-- name: UpdateLocation :one
UPDATE puzzles SET location = ?2
WHERE id = ?1
RETURNING *;

-- name: UpdateArchived :one
UPDATE puzzles SET archived = ?2
WHERE id = ?1
RETURNING *;

-- name: UpdateVoiceRoom :one
UPDATE puzzles SET voice_room = ?2, location = ?3
WHERE id = ?1
RETURNING *;
