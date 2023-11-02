-- name: GetPuzzle :one
SELECT * FROM puzzles
WHERE id = ? LIMIT 1;

-- name: ListPuzzles :many
SELECT * FROM puzzles
ORDER BY id;
