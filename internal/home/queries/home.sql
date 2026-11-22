-- name: GetSettings :one
SELECT * FROM home_settings;

-- name: SetRows :exec
UPDATE home_settings SET rows_shown = sqlc.narg(rows_shown);
