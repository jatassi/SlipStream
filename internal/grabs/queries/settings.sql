-- name: GetSettings :one
SELECT * FROM grabs_settings;

-- name: SetSettings :exec
UPDATE grabs_settings
SET backlog_on = sqlc.narg(backlog_on), backlog_upgrades = sqlc.narg(backlog_upgrades), rss_on = sqlc.narg(rss_on),
    longest_gap_seconds = sqlc.narg(longest_gap_seconds), full_pace_seconds = sqlc.narg(full_pace_seconds),
    stop_count = sqlc.narg(stop_count), stop_window_seconds = sqlc.narg(stop_window_seconds);
