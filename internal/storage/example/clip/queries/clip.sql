-- name: InsertClip :one
INSERT INTO clip (shelf_id, public_id, title, created_at)
VALUES (sqlc.arg(shelf_id), sqlc.arg(public_id), sqlc.arg(title), sqlc.arg(created_at))
RETURNING *;

-- name: GetClip :one
SELECT * FROM clip WHERE id = sqlc.arg(id);

-- name: ListClipsOnShelf :many
SELECT * FROM clip WHERE shelf_id = sqlc.arg(shelf_id) ORDER BY public_id;

-- name: ListUnwatchedClips :many
SELECT * FROM clip WHERE shelf_id = sqlc.arg(shelf_id) AND watched_at IS NULL ORDER BY public_id;

-- name: MarkClipWatched :exec
UPDATE clip SET watched_at = sqlc.arg(watched_at) WHERE id = sqlc.arg(id);
