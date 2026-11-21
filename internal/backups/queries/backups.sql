-- name: GetSettings :one
SELECT keep_scheduled_days, keep_each, warn_after, copy_wait_limit_hours, check_copies_days FROM backups_settings WHERE id = 1;

-- name: SetSettings :exec
UPDATE backups_settings
SET keep_scheduled_days = ?, keep_each = ?, warn_after = ?, copy_wait_limit_hours = ?, check_copies_days = ?
WHERE id = 1;
