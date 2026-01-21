-- name: GetProwlarrIndexerSettings :one
SELECT * FROM prowlarr_indexer_settings WHERE prowlarr_indexer_id = ? LIMIT 1;

-- name: GetAllProwlarrIndexerSettings :many
SELECT * FROM prowlarr_indexer_settings ORDER BY prowlarr_indexer_id;

-- name: UpsertProwlarrIndexerSettings :one
INSERT INTO prowlarr_indexer_settings (
    prowlarr_indexer_id,
    priority,
    content_type,
    movie_categories,
    tv_categories
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(prowlarr_indexer_id) DO UPDATE SET
    priority = excluded.priority,
    content_type = excluded.content_type,
    movie_categories = excluded.movie_categories,
    tv_categories = excluded.tv_categories,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteProwlarrIndexerSettings :exec
DELETE FROM prowlarr_indexer_settings WHERE prowlarr_indexer_id = ?;

-- name: DeleteAllProwlarrIndexerSettings :exec
DELETE FROM prowlarr_indexer_settings;

-- name: RecordProwlarrIndexerSuccess :exec
UPDATE prowlarr_indexer_settings SET
    success_count = success_count + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE prowlarr_indexer_id = ?;

-- name: RecordProwlarrIndexerFailure :exec
UPDATE prowlarr_indexer_settings SET
    failure_count = failure_count + 1,
    last_failure_at = CURRENT_TIMESTAMP,
    last_failure_reason = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE prowlarr_indexer_id = ?;

-- name: ResetProwlarrIndexerStats :exec
UPDATE prowlarr_indexer_settings SET
    success_count = 0,
    failure_count = 0,
    last_failure_at = NULL,
    last_failure_reason = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE prowlarr_indexer_id = ?;

-- name: GetProwlarrIndexerSettingsByContentType :many
SELECT * FROM prowlarr_indexer_settings
WHERE content_type = ? OR content_type = 'both'
ORDER BY priority ASC;
