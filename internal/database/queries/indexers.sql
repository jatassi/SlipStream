-- name: GetIndexer :one
SELECT * FROM indexers WHERE id = ? LIMIT 1;

-- name: ListIndexers :many
SELECT * FROM indexers ORDER BY priority, name;

-- name: ListEnabledIndexers :many
SELECT * FROM indexers WHERE enabled = 1 ORDER BY priority, name;

-- name: CreateIndexer :one
INSERT INTO indexers (
    name, type, url, api_key, categories, supports_movies, supports_tv, priority, enabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateIndexer :one
UPDATE indexers SET
    name = ?,
    type = ?,
    url = ?,
    api_key = ?,
    categories = ?,
    supports_movies = ?,
    supports_tv = ?,
    priority = ?,
    enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteIndexer :exec
DELETE FROM indexers WHERE id = ?;

-- name: CountIndexers :one
SELECT COUNT(*) FROM indexers;
