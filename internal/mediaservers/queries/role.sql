-- name: ListReached :many
SELECT id, kind, name, address, identity, settings_version, device_id, monitor_delay_ms
FROM mediaservers_server
WHERE reaching_node_id = sqlc.arg(node_id)
ORDER BY id;

-- name: ListReachedNames :many
SELECT name FROM mediaservers_server WHERE reaching_node_id = sqlc.arg(node_id) ORDER BY name;

-- name: GetSecrets :one
SELECT reaching_node_id, secrets_sealed, settings_version FROM mediaservers_server WHERE id = sqlc.arg(id);

-- name: RecordFetch :exec
INSERT INTO mediaservers_credential_fetch (server_id, node_id, settings_version, first_fetched_at, fetched_at)
VALUES (sqlc.arg(server_id), sqlc.arg(node_id), sqlc.arg(settings_version), sqlc.arg(fetched_at), sqlc.arg(fetched_at))
ON CONFLICT (server_id, node_id) DO UPDATE SET settings_version = excluded.settings_version, fetched_at = excluded.fetched_at;

-- name: ListReachingNodes :many
SELECT DISTINCT reaching_node_id FROM mediaservers_server ORDER BY reaching_node_id;

-- Repoint names the Controller as the reaching Node of every server node
-- reached. The Controller calls the fallback address, and the contact was
-- node's word, so it is unknown until the Controller reports.
-- name: Repoint :execrows
UPDATE mediaservers_server
SET reaching_node_id = sqlc.arg(controller),
    address = coalesce(fallback_address, address),
    fallback_address = NULL,
    contact = 'unknown',
    contact_since = NULL,
    contact_found = NULL,
    updated_at = sqlc.arg(at)
WHERE reaching_node_id = sqlc.arg(node_id);
