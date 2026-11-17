-- name: InsertItem :exec
INSERT INTO history_item (event_id, key, level, subject, text, since_at, cleared_at)
VALUES (sqlc.arg(event_id), sqlc.arg(key), sqlc.arg(level), sqlc.arg(subject), sqlc.arg(text), sqlc.arg(since_at), sqlc.arg(cleared_at));

-- name: ListItemsOf :many
SELECT * FROM history_item WHERE event_id IN (sqlc.slice(event_ids)) ORDER BY event_id, key;

-- name: NewestItem :one
SELECT * FROM history_item INDEXED BY history_item_key
WHERE key = sqlc.arg(key) ORDER BY event_id DESC LIMIT 1;
