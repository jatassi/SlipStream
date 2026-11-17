-- name: GetSettings :one
SELECT * FROM history_settings;

-- name: SetKeep :exec
UPDATE history_settings SET keep_seconds = sqlc.narg(seconds);
