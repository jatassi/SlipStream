-- name: CreateDownloadMapping :one
INSERT INTO download_mappings (
    client_id, download_id, module_type, entity_type, entity_id, season_number,
    is_season_pack, is_complete_series, target_slot_id, source
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT (client_id, download_id) DO UPDATE SET
    module_type = excluded.module_type,
    entity_type = excluded.entity_type,
    entity_id = excluded.entity_id,
    season_number = excluded.season_number,
    is_season_pack = excluded.is_season_pack,
    is_complete_series = excluded.is_complete_series,
    target_slot_id = excluded.target_slot_id,
    source = excluded.source,
    import_attempts = 0,
    last_import_error = NULL,
    next_import_retry_at = NULL
RETURNING *;

-- name: GetDownloadMapping :one
SELECT * FROM download_mappings
WHERE client_id = ? AND download_id = ?;

-- name: GetDownloadMappingsByClientDownloadIDs :many
SELECT * FROM download_mappings
WHERE (client_id, download_id) IN (/*SLICE:client_download_ids*/sqlc.slice('client_download_ids'));

-- name: ListActiveDownloadMappings :many
SELECT * FROM download_mappings
ORDER BY created_at DESC;

-- name: DeleteDownloadMapping :exec
DELETE FROM download_mappings
WHERE client_id = ? AND download_id = ?;

-- name: DeleteOldDownloadMappings :exec
DELETE FROM download_mappings
WHERE created_at < datetime('now', '-7 days');

-- name: GetDownloadingEntityIDs :many
SELECT DISTINCT entity_id FROM download_mappings
WHERE module_type = ? AND entity_type = ?;

-- name: GetDownloadingEntityData :many
SELECT DISTINCT entity_type, entity_id, season_number, is_season_pack, is_complete_series
FROM download_mappings
WHERE module_type = ?;

-- name: IsEntityDownloading :one
SELECT EXISTS(
    SELECT 1 FROM download_mappings
    WHERE module_type = ? AND entity_type = ? AND entity_id = ?
) AS downloading;

-- name: ClearDownloadMappingSlot :exec
-- Req 10.2.1: Clear slot when download fails or is rejected
UPDATE download_mappings SET target_slot_id = NULL
WHERE client_id = ? AND download_id = ?;

-- name: GetDownloadMappingsBySlot :many
-- Get all mappings targeting a specific slot
SELECT * FROM download_mappings
WHERE target_slot_id = ?
ORDER BY created_at DESC;

-- name: DeleteDownloadMappingsByEntity :exec
-- Clean up download mappings when an entity is deleted
DELETE FROM download_mappings
WHERE module_type = ? AND entity_type = ? AND entity_id = ?;

-- name: IncrementDownloadMappingAttempts :one
UPDATE download_mappings
SET import_attempts = import_attempts + 1,
    last_import_error = ?,
    next_import_retry_at = ?
WHERE client_id = ? AND download_id = ?
RETURNING import_attempts;
