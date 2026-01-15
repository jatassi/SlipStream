-- name: CreateQueueMedia :one
INSERT INTO queue_media (
    download_mapping_id, episode_id, movie_id, file_path, file_status, target_slot_id
) VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetQueueMedia :one
SELECT * FROM queue_media WHERE id = ?;

-- name: GetQueueMediaByDownloadMapping :many
SELECT * FROM queue_media WHERE download_mapping_id = ?;

-- name: GetQueueMediaByEpisode :one
SELECT * FROM queue_media WHERE episode_id = ? LIMIT 1;

-- name: GetQueueMediaByMovie :one
SELECT * FROM queue_media WHERE movie_id = ? LIMIT 1;

-- name: ListQueueMediaByStatus :many
SELECT * FROM queue_media WHERE file_status = ? ORDER BY created_at;

-- name: ListPendingImports :many
SELECT * FROM queue_media
WHERE file_status IN ('ready', 'failed')
ORDER BY created_at;

-- name: UpdateQueueMediaStatus :one
UPDATE queue_media SET
    file_status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateQueueMediaStatusWithError :one
UPDATE queue_media SET
    file_status = ?,
    error_message = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateQueueMediaFilePath :one
UPDATE queue_media SET
    file_path = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: IncrementQueueMediaImportAttempts :one
UPDATE queue_media SET
    import_attempts = import_attempts + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: ResetQueueMediaForRetry :one
UPDATE queue_media SET
    file_status = 'ready',
    error_message = NULL,
    import_attempts = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteQueueMedia :exec
DELETE FROM queue_media WHERE id = ?;

-- name: DeleteQueueMediaByDownloadMapping :exec
DELETE FROM queue_media WHERE download_mapping_id = ?;

-- name: CountQueueMediaByStatus :one
SELECT COUNT(*) FROM queue_media WHERE file_status = ?;

-- name: GetReadyQueueMediaWithMapping :many
SELECT
    qm.*,
    dm.client_id,
    dm.download_id,
    dm.series_id,
    dm.season_number,
    dm.is_season_pack,
    dm.is_complete_series
FROM queue_media qm
JOIN download_mappings dm ON dm.id = qm.download_mapping_id
WHERE qm.file_status = 'ready'
ORDER BY qm.created_at;

-- name: GetFailedQueueMediaWithMapping :many
SELECT
    qm.*,
    dm.client_id,
    dm.download_id,
    dm.series_id,
    dm.season_number,
    dm.is_season_pack,
    dm.is_complete_series
FROM queue_media qm
JOIN download_mappings dm ON dm.id = qm.download_mapping_id
WHERE qm.file_status = 'failed'
ORDER BY qm.created_at;

-- name: ListPendingQueueMedia :many
SELECT * FROM queue_media
WHERE file_status IN ('pending', 'ready', 'importing', 'failed')
ORDER BY created_at DESC;

-- name: UpdateQueueMediaSlot :one
-- Req 16.2.3: Per-episode slot assignment for season packs
UPDATE queue_media SET
    target_slot_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;
