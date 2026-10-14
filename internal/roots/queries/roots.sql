-- name: HasRoot :one
SELECT EXISTS (SELECT 1 FROM roots_root WHERE content_type = sqlc.arg(content_type));

-- name: InsertRoot :exec
INSERT INTO roots_root (id, content_type, node_id, path)
VALUES (sqlc.arg(id), sqlc.arg(content_type), sqlc.arg(node_id), sqlc.arg(path));

-- name: ListRoots :many
SELECT * FROM roots_root ORDER BY content_type;

-- name: GetRootOf :one
SELECT * FROM roots_root WHERE content_type = sqlc.arg(content_type);

-- name: GetRoot :one
SELECT * FROM roots_root WHERE id = sqlc.arg(id);
