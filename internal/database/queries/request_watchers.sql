-- name: GetRequestWatcher :one
SELECT * FROM request_watchers
WHERE request_id = ? AND user_id = ?
LIMIT 1;

-- name: ListRequestWatchers :many
SELECT * FROM request_watchers
WHERE request_id = ?
ORDER BY created_at;

-- name: ListUserWatchedRequests :many
SELECT r.* FROM requests r
JOIN request_watchers rw ON r.id = rw.request_id
WHERE rw.user_id = ?
ORDER BY r.created_at DESC;

-- name: CreateRequestWatcher :one
INSERT INTO request_watchers (request_id, user_id)
VALUES (?, ?)
RETURNING *;

-- name: DeleteRequestWatcher :exec
DELETE FROM request_watchers
WHERE request_id = ? AND user_id = ?;

-- name: DeleteRequestWatchersByRequest :exec
DELETE FROM request_watchers WHERE request_id = ?;

-- name: DeleteRequestWatchersByUser :exec
DELETE FROM request_watchers WHERE user_id = ?;

-- name: CountRequestWatchers :one
SELECT COUNT(*) FROM request_watchers WHERE request_id = ?;

-- name: IsWatchingRequest :one
SELECT EXISTS(
    SELECT 1 FROM request_watchers
    WHERE request_id = ? AND user_id = ?
) AS is_watching;
