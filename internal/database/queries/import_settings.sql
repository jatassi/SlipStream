-- name: GetImportSettings :one
SELECT * FROM import_settings WHERE id = 1;

-- name: UpdateImportSettings :one
UPDATE import_settings SET
    validation_level = ?,
    minimum_file_size_mb = ?,
    video_extensions = ?,
    match_conflict_behavior = ?,
    unknown_media_behavior = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: UpdateImportValidationSettings :one
UPDATE import_settings SET
    validation_level = ?,
    minimum_file_size_mb = ?,
    video_extensions = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: UpdateImportMatchingSettings :one
UPDATE import_settings SET
    match_conflict_behavior = ?,
    unknown_media_behavior = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: EnsureImportSettingsExist :exec
INSERT OR IGNORE INTO import_settings (id) VALUES (1);
