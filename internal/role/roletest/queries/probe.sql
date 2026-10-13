-- name: GetIntent :one
SELECT value FROM probe_intent WHERE node_id = sqlc.arg(node_id);

-- name: SetIntent :exec
INSERT INTO probe_intent (node_id, value) VALUES (sqlc.arg(node_id), sqlc.arg(value))
ON CONFLICT (node_id) DO UPDATE SET value = excluded.value;

-- name: InsertApplied :exec
INSERT INTO probe_applied (node_id, n) VALUES (sqlc.arg(node_id), sqlc.arg(n));

-- name: ListApplied :many
SELECT n FROM probe_applied WHERE node_id = sqlc.arg(node_id) ORDER BY id;
