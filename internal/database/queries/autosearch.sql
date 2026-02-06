-- Autosearch status queries for tracking search failures and backoff

-- name: GetAutosearchStatus :one
SELECT * FROM autosearch_status WHERE item_type = ? AND item_id = ? AND search_type = ? LIMIT 1;

-- name: UpsertAutosearchStatus :one
INSERT INTO autosearch_status (item_type, item_id, search_type, failure_count, last_searched_at, last_meta_change_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(item_type, item_id, search_type) DO UPDATE SET
    failure_count = excluded.failure_count,
    last_searched_at = excluded.last_searched_at,
    last_meta_change_at = COALESCE(excluded.last_meta_change_at, autosearch_status.last_meta_change_at)
RETURNING *;

-- name: IncrementAutosearchFailure :exec
INSERT INTO autosearch_status (item_type, item_id, search_type, failure_count, last_searched_at)
VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP)
ON CONFLICT(item_type, item_id, search_type) DO UPDATE SET
    failure_count = autosearch_status.failure_count + 1,
    last_searched_at = CURRENT_TIMESTAMP;

-- name: ResetAutosearchFailure :exec
UPDATE autosearch_status
SET failure_count = 0, last_meta_change_at = CURRENT_TIMESTAMP
WHERE item_type = ? AND item_id = ? AND search_type = ?;

-- name: ResetAllAutosearchFailuresForItem :exec
UPDATE autosearch_status
SET failure_count = 0, last_meta_change_at = CURRENT_TIMESTAMP
WHERE item_type = ? AND item_id = ?;

-- name: MarkAutosearchSearched :exec
UPDATE autosearch_status
SET last_searched_at = CURRENT_TIMESTAMP
WHERE item_type = ? AND item_id = ? AND search_type = ?;

-- name: DeleteAutosearchStatus :exec
DELETE FROM autosearch_status WHERE item_type = ? AND item_id = ?;

-- name: ListItemsExceedingBackoffThreshold :many
SELECT * FROM autosearch_status
WHERE failure_count >= ? AND search_type = ?
ORDER BY last_searched_at DESC;

-- name: CountItemsExceedingBackoffThreshold :one
SELECT COUNT(*) FROM autosearch_status WHERE failure_count >= ? AND search_type = ?;

-- name: ClearAllAutosearchStatus :exec
DELETE FROM autosearch_status;
