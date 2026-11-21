-- name: GetSettings :one
SELECT keep_scheduled_days, keep_each FROM backups_settings WHERE id = 1;
