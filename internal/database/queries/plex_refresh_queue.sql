-- name: EnqueuePlexRefresh :one
-- Enqueue a path for Plex library refresh, using INSERT OR REPLACE for deduplication
INSERT OR REPLACE INTO plex_refresh_queue (notification_id, server_id, section_key, path, queued_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetPendingPlexRefreshes :many
-- Get all pending refresh items for a specific notification and server
SELECT * FROM plex_refresh_queue
WHERE notification_id = ? AND server_id = ?
ORDER BY queued_at;

-- name: GetPendingPlexRefreshesBySection :many
-- Get pending refresh items for a specific notification, server, and section
SELECT * FROM plex_refresh_queue
WHERE notification_id = ? AND server_id = ? AND section_key = ?
ORDER BY queued_at;

-- name: ClearPlexRefreshes :exec
-- Clear all pending refresh items for a notification and server
DELETE FROM plex_refresh_queue
WHERE notification_id = ? AND server_id = ?;

-- name: ClearPlexRefreshesBySection :exec
-- Clear pending refresh items for a specific section
DELETE FROM plex_refresh_queue
WHERE notification_id = ? AND server_id = ? AND section_key = ?;

-- name: GetAllPendingPlexRefreshes :many
-- Get all pending refresh items grouped for scheduler processing
SELECT * FROM plex_refresh_queue
ORDER BY notification_id, server_id, section_key, queued_at;

-- name: CountPendingPlexRefreshesPerSection :many
-- Count pending refreshes per section (for deciding full vs partial refresh)
SELECT notification_id, server_id, section_key, COUNT(*) as count
FROM plex_refresh_queue
GROUP BY notification_id, server_id, section_key;

-- name: DeletePlexRefreshByID :exec
-- Delete a specific queue item by ID
DELETE FROM plex_refresh_queue WHERE id = ?;

-- name: DeletePlexRefreshesForNotification :exec
-- Delete all queue items for a notification (used when notification is deleted)
DELETE FROM plex_refresh_queue WHERE notification_id = ?;
