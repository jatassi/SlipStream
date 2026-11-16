-- name: InsertItem :exec
INSERT INTO notifications_item (event_id, key, level, subject, text, since_at, cleared_at)
VALUES (sqlc.arg(event_id), sqlc.arg(key), sqlc.arg(level), sqlc.arg(subject), sqlc.arg(text), sqlc.arg(since_at), sqlc.arg(cleared_at));

-- name: WaitingItem :one
SELECT i.* FROM notifications_item i
JOIN notifications_event e ON e.id = i.event_id
WHERE i.key = sqlc.arg(key) AND e.kind = sqlc.arg(kind) AND e.released_at IS NULL
ORDER BY i.event_id DESC
LIMIT 1;

-- name: NewestItem :one
SELECT * FROM notifications_item WHERE key = sqlc.arg(key) ORDER BY event_id DESC LIMIT 1;

-- name: ReplaceItem :exec
UPDATE notifications_item
SET level = sqlc.arg(level), subject = sqlc.arg(subject), text = sqlc.arg(text)
WHERE event_id = sqlc.arg(event_id) AND key = sqlc.arg(key);

-- name: DeleteItem :exec
DELETE FROM notifications_item WHERE event_id = sqlc.arg(event_id) AND key = sqlc.arg(key);

-- name: CountItems :one
SELECT COUNT(*) FROM notifications_item WHERE event_id = sqlc.arg(event_id);

-- name: ListItems :many
SELECT * FROM notifications_item WHERE event_id = sqlc.arg(event_id) ORDER BY key;
