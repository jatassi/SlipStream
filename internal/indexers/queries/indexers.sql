-- name: NameTaken :one
SELECT EXISTS (SELECT 1 FROM indexers_indexer WHERE name = sqlc.arg(name));

-- name: InsertIndexer :exec
INSERT INTO indexers_indexer (id, name, protocol, url, api_key, home_node_id, caps, created_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(protocol), sqlc.arg(url), sqlc.arg(api_key), sqlc.arg(home_node_id), sqlc.arg(caps), sqlc.arg(created_at));

-- name: ListIndexers :many
SELECT * FROM indexers_indexer ORDER BY name;
