-- name: GetCheck :one
SELECT value FROM secret_check WHERE id = 1;

-- name: InsertCheck :exec
INSERT INTO secret_check (id, value) VALUES (1, sqlc.arg(value));
