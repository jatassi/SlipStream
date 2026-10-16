-- name: NameTaken :one
SELECT EXISTS (SELECT 1 FROM indexers_indexer WHERE name = sqlc.arg(name));

-- name: InsertIndexer :exec
INSERT INTO indexers_indexer (id, name, protocol, url, api_key_sealed, home_node_id, caps, created_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(protocol), sqlc.arg(url), sqlc.arg(api_key_sealed), sqlc.arg(home_node_id), sqlc.arg(caps), sqlc.arg(created_at));

-- name: ListIndexers :many
SELECT * FROM indexers_indexer ORDER BY name;

-- name: GetIndexer :one
SELECT * FROM indexers_indexer WHERE id = sqlc.arg(id);

-- name: SetIndexerKey :execrows
UPDATE indexers_indexer SET api_key_sealed = sqlc.arg(api_key_sealed), caps = sqlc.arg(caps) WHERE id = sqlc.arg(id);
