-- name: ClientTaken :one
SELECT
    CAST(EXISTS (SELECT 1 FROM downloadclients_client c WHERE c.name = sqlc.arg(name) AND c.deleted_at IS NULL) AS INTEGER) AS name_taken,
    CAST(EXISTS (SELECT 1 FROM downloadclients_client c WHERE c.address_key = sqlc.arg(address_key) AND c.deleted_at IS NULL) AS INTEGER) AS address_taken;

-- name: InsertClient :exec
INSERT INTO downloadclients_client (
    id, node_id, kind, name, address, address_key, username, secrets_sealed, label, import_delay_seconds,
    is_on, settings_version, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(node_id), sqlc.arg(kind), sqlc.arg(name), sqlc.arg(address), sqlc.arg(address_key),
    sqlc.narg(username), sqlc.arg(secrets_sealed), sqlc.arg(label), sqlc.arg(import_delay_seconds),
    1, 1, sqlc.arg(created_at), sqlc.arg(created_at)
);

-- name: GetClient :one
SELECT * FROM downloadclients_client WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: GetGrabsTo :one
-- The on client a Node's new Grabs go to. No row is its built-in client.
SELECT c.id FROM downloadclients_node n JOIN downloadclients_client c ON c.id = n.grabs_to
WHERE n.node_id = sqlc.arg(node_id) AND c.is_on = 1 AND c.deleted_at IS NULL;

-- name: ListNodeClients :many
SELECT * FROM downloadclients_client WHERE node_id = sqlc.arg(node_id) AND deleted_at IS NULL ORDER BY id;

-- name: GetClientSecrets :one
SELECT node_id, secrets_sealed, settings_version FROM downloadclients_client
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: SetPoll :execrows
-- A poll under settings older than the stored poll's changes nothing. A
-- poll that answers no Check now keeps the last one's token.
UPDATE downloadclients_client SET
    poll_state = sqlc.arg(poll_state), poll_since = sqlc.arg(poll_since), poll_problem = sqlc.narg(poll_problem),
    poll_settings_version = sqlc.arg(poll_settings_version), poll_check = coalesce(sqlc.narg(poll_check), poll_check),
    client_version = sqlc.narg(client_version)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND deleted_at IS NULL
  AND (poll_settings_version IS NULL OR poll_settings_version <= sqlc.arg(poll_settings_version));
