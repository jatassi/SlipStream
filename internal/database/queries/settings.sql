-- name: GetSetting :one
SELECT * FROM settings WHERE key = ? LIMIT 1;

-- name: ListSettings :many
SELECT * FROM settings ORDER BY key;

-- name: SetSetting :one
INSERT INTO settings (key, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key) DO UPDATE SET
    value = excluded.value,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteSetting :exec
DELETE FROM settings WHERE key = ?;

-- Quality Profiles
-- name: GetQualityProfile :one
SELECT * FROM quality_profiles WHERE id = ? LIMIT 1;

-- name: ListQualityProfiles :many
SELECT * FROM quality_profiles ORDER BY name;

-- name: CreateQualityProfile :one
INSERT INTO quality_profiles (name, cutoff, items, hdr_settings, video_codec_settings, audio_codec_settings, audio_channel_settings, upgrades_enabled)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateQualityProfile :one
UPDATE quality_profiles SET
    name = ?,
    cutoff = ?,
    items = ?,
    hdr_settings = ?,
    video_codec_settings = ?,
    audio_codec_settings = ?,
    audio_channel_settings = ?,
    upgrades_enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteQualityProfile :exec
DELETE FROM quality_profiles WHERE id = ?;

-- name: GetQualityProfileByName :one
SELECT * FROM quality_profiles WHERE name = ? LIMIT 1;

-- name: CountMoviesUsingProfile :one
SELECT COUNT(*) FROM movies WHERE quality_profile_id = ?;

-- name: CountSeriesUsingProfile :one
SELECT COUNT(*) FROM series WHERE quality_profile_id = ?;

-- Root Folders
-- name: GetRootFolder :one
SELECT * FROM root_folders WHERE id = ? LIMIT 1;

-- name: ListRootFolders :many
SELECT * FROM root_folders ORDER BY name;

-- name: ListRootFoldersByMediaType :many
SELECT * FROM root_folders WHERE media_type = ? ORDER BY name;

-- name: CreateRootFolder :one
INSERT INTO root_folders (path, name, media_type, free_space)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateRootFolder :one
UPDATE root_folders SET
    path = ?,
    name = ?,
    free_space = ?
WHERE id = ?
RETURNING *;

-- name: DeleteRootFolder :exec
DELETE FROM root_folders WHERE id = ?;

-- name: GetRootFolderByPath :one
SELECT * FROM root_folders WHERE path = ? LIMIT 1;

-- name: ListRootFoldersByType :many
SELECT * FROM root_folders WHERE media_type = ? ORDER BY name;

-- name: UpdateRootFolderFreeSpace :exec
UPDATE root_folders SET free_space = ? WHERE id = ?;

-- Downloads Queue
-- name: GetDownload :one
SELECT * FROM downloads WHERE id = ? LIMIT 1;

-- name: ListDownloads :many
SELECT * FROM downloads ORDER BY added_at DESC;

-- name: ListActiveDownloads :many
SELECT * FROM downloads WHERE status IN ('queued', 'downloading', 'paused') ORDER BY added_at;

-- name: CreateDownload :one
INSERT INTO downloads (
    client_id, external_id, title, media_type, media_id, status, progress, size, download_url, output_path
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateDownloadStatus :one
UPDATE downloads SET
    status = ?,
    progress = ?,
    completed_at = CASE WHEN ? = 'completed' THEN CURRENT_TIMESTAMP ELSE completed_at END
WHERE id = ?
RETURNING *;

-- name: DeleteDownload :exec
DELETE FROM downloads WHERE id = ?;

-- History
-- name: ListHistory :many
SELECT * FROM history ORDER BY created_at DESC LIMIT ?;

-- name: ListHistoryPaginated :many
SELECT * FROM history
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListHistoryByEventType :many
SELECT * FROM history
WHERE event_type = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListHistoryByMediaType :many
SELECT * FROM history
WHERE media_type = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListHistoryFiltered :many
SELECT * FROM history
WHERE (? = '' OR event_type = ?)
  AND (? = '' OR media_type = ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountHistory :one
SELECT COUNT(*) FROM history;

-- name: CountHistoryByEventType :one
SELECT COUNT(*) FROM history WHERE event_type = ?;

-- name: CountHistoryByMediaType :one
SELECT COUNT(*) FROM history WHERE media_type = ?;

-- name: CountHistoryFiltered :one
SELECT COUNT(*) FROM history
WHERE (? = '' OR event_type = ?)
  AND (? = '' OR media_type = ?);

-- name: ListHistoryByMedia :many
SELECT * FROM history WHERE media_type = ? AND media_id = ? ORDER BY created_at DESC;

-- name: CreateHistoryEntry :one
INSERT INTO history (event_type, media_type, media_id, source, quality, data)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteAllHistory :exec
DELETE FROM history;

-- name: DeleteOldHistory :exec
DELETE FROM history WHERE created_at < ?;

-- Default Settings
-- name: GetDefaultForEntityTypeAndMediaType :one
SELECT * FROM settings WHERE key = ? LIMIT 1;

-- name: SetDefaultForEntityTypeAndMediaType :one
INSERT INTO settings (key, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key) DO UPDATE SET
    value = excluded.value,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetDefaultsForEntityType :many
SELECT * FROM settings WHERE key LIKE ? ORDER BY key;

-- name: GetAllDefaults :many
SELECT * FROM settings WHERE key LIKE 'default_%' ORDER BY key;
