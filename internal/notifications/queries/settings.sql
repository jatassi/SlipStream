-- name: GetSettings :one
SELECT * FROM notifications_settings;

-- name: SetHealthWait :exec
UPDATE notifications_settings SET health_wait_seconds = sqlc.narg(seconds);

-- name: SetGiveUp :exec
UPDATE notifications_settings SET give_up_seconds = sqlc.narg(seconds);

-- name: SetWarn :exec
UPDATE notifications_settings SET warn_seconds = sqlc.narg(seconds);
