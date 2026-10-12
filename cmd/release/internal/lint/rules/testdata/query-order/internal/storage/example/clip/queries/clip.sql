-- name: InsertClip :one
INSERT INTO clip (shelf_id, title) VALUES (sqlc.arg(shelf_id), sqlc.arg(title))
RETURNING *;

-- name: ListClipsOnShelf :many
SELECT * FROM clip WHERE shelf_id = sqlc.arg(shelf_id) ORDER BY public_id;

-- name: MarkClipWatched :exec
UPDATE clip SET watched_at = sqlc.arg(watched_at) WHERE id = sqlc.arg(id);
