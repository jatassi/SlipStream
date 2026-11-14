-- name: GetSettings :one
SELECT * FROM mediaservers_settings;

-- name: SetWait :exec
UPDATE mediaservers_settings SET wait_seconds = sqlc.narg(seconds);

-- name: SetSessionsEvery :exec
UPDATE mediaservers_settings SET sessions_every_seconds = sqlc.narg(seconds);

-- name: SetUnhealthyAfter :exec
UPDATE mediaservers_settings SET unhealthy_after_seconds = sqlc.narg(seconds);

-- name: SetKeepPlays :exec
UPDATE mediaservers_settings SET keep_plays_seconds = sqlc.narg(seconds);

-- name: SetRenewMargin :exec
UPDATE mediaservers_settings SET renew_margin_seconds = sqlc.narg(seconds);
