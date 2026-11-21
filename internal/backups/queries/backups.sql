-- name: GetSettings :one
SELECT keep_scheduled_days, keep_each, copy_wait_limit_hours, check_copies_days FROM backups_settings WHERE id = 1;
