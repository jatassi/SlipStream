-- name: ListServers :many
SELECT * FROM mediaservers_server ORDER BY name, id;

-- name: GetServer :one
SELECT * FROM mediaservers_server WHERE id = sqlc.arg(id);

-- name: InsertServer :exec
INSERT INTO mediaservers_server (
    id, kind, name, address, fallback_address, browser_address, identity, version, monitor_delay_ms,
    reaching_node_id, runs_on_node_id, device_id, secrets_sealed, fallback, checked_at, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(kind), sqlc.arg(name), sqlc.arg(address), sqlc.narg(fallback_address),
    sqlc.narg(browser_address), sqlc.arg(identity), sqlc.arg(version), sqlc.narg(monitor_delay_ms),
    sqlc.arg(reaching_node_id), sqlc.narg(runs_on_node_id), sqlc.narg(device_id), sqlc.arg(secrets_sealed),
    sqlc.arg(fallback), sqlc.arg(checked_at), sqlc.arg(at), sqlc.arg(at)
);

-- UpdateServer writes the configuration the API owns. The contact, sends
-- and unhealthy columns have their own writers.
-- name: UpdateServer :exec
UPDATE mediaservers_server
SET name = sqlc.arg(name),
    address = sqlc.arg(address),
    fallback_address = sqlc.narg(fallback_address),
    browser_address = sqlc.narg(browser_address),
    identity = sqlc.arg(identity),
    version = sqlc.arg(version),
    monitor_delay_ms = sqlc.narg(monitor_delay_ms),
    reaching_node_id = sqlc.arg(reaching_node_id),
    runs_on_node_id = sqlc.narg(runs_on_node_id),
    secrets_sealed = sqlc.arg(secrets_sealed),
    settings_version = sqlc.arg(settings_version),
    fallback = sqlc.arg(fallback),
    checked_at = sqlc.arg(checked_at),
    updated_at = sqlc.arg(at)
WHERE id = sqlc.arg(id);

-- name: SetTested :exec
UPDATE mediaservers_server
SET version = sqlc.arg(version),
    monitor_delay_ms = sqlc.narg(monitor_delay_ms),
    fallback = sqlc.arg(fallback),
    checked_at = sqlc.arg(checked_at),
    updated_at = sqlc.arg(checked_at)
WHERE id = sqlc.arg(id);

-- name: DeleteServer :execrows
DELETE FROM mediaservers_server WHERE id = sqlc.arg(id);

-- name: ListHolders :many
SELECT node_id, fetched_at FROM mediaservers_credential_fetch
WHERE server_id = sqlc.arg(server_id)
ORDER BY fetched_at DESC, node_id;
