-- name: ListModuleNamingSettings :many
SELECT setting_key, setting_value
FROM module_naming_settings
WHERE module_type = ?;

-- name: GetModuleNamingSetting :one
SELECT setting_value
FROM module_naming_settings
WHERE module_type = ? AND setting_key = ?;

-- name: UpsertModuleNamingSetting :exec
INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
VALUES (?, ?, ?)
ON CONFLICT(module_type, setting_key) DO UPDATE SET setting_value = excluded.setting_value;

-- name: DeleteModuleNamingSettings :exec
DELETE FROM module_naming_settings WHERE module_type = ?;
