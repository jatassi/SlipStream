-- name: NameTaken :one
SELECT CAST(EXISTS (
    SELECT 1 FROM downloadclients_client
    WHERE name = sqlc.arg(name) AND id IS NOT sqlc.arg(except) AND deleted_at IS NULL
) AS INTEGER);

-- name: AddressHolder :one
-- The client other than except that uses the address.
SELECT name, node_id FROM downloadclients_client
WHERE address_key = sqlc.arg(address_key) AND id IS NOT sqlc.arg(except) AND deleted_at IS NULL;

-- name: InsertClient :exec
INSERT INTO downloadclients_client (
    id, node_id, kind, name, address, address_key, username, secrets_sealed, label, folder, client_folders,
    daemon_host, daemon_port, check_every_seconds, import_delay_seconds, is_on, settings_version, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(node_id), sqlc.arg(kind), sqlc.arg(name), sqlc.arg(address), sqlc.arg(address_key),
    sqlc.narg(username), sqlc.arg(secrets_sealed), sqlc.arg(label), sqlc.narg(folder), sqlc.arg(client_folders),
    sqlc.narg(daemon_host), sqlc.narg(daemon_port), sqlc.narg(check_every_seconds), sqlc.narg(import_delay_seconds),
    sqlc.arg(is_on), 1, sqlc.arg(created_at), sqlc.arg(created_at)
);

-- name: UpdateClient :execrows
UPDATE downloadclients_client SET
    node_id = sqlc.arg(node_id), kind = sqlc.arg(kind), name = sqlc.arg(name), address = sqlc.arg(address),
    address_key = sqlc.arg(address_key), username = sqlc.narg(username), secrets_sealed = sqlc.arg(secrets_sealed),
    label = sqlc.arg(label), folder = sqlc.narg(folder), client_folders = sqlc.arg(client_folders),
    daemon_host = sqlc.narg(daemon_host), daemon_port = sqlc.narg(daemon_port),
    check_every_seconds = sqlc.narg(check_every_seconds), import_delay_seconds = sqlc.narg(import_delay_seconds),
    is_on = sqlc.arg(is_on), settings_version = sqlc.arg(settings_version), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: DeleteClient :execrows
UPDATE downloadclients_client SET deleted_at = sqlc.arg(deleted_at), updated_at = sqlc.arg(deleted_at), secrets_sealed = NULL
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: GetClientSealed :one
SELECT secrets_sealed FROM downloadclients_client WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ListClients :many
SELECT * FROM downloadclients_client WHERE deleted_at IS NULL ORDER BY name, id;

-- name: CountTracked :one
-- The Torrents a client holds until their Node parts from them.
SELECT count(*) FROM downloadclients_torrent WHERE client_id = sqlc.arg(client_id) AND parted_at IS NULL;

-- name: ListTracked :many
SELECT id FROM downloadclients_torrent WHERE client_id = sqlc.arg(client_id) AND parted_at IS NULL ORDER BY id;

-- name: StopTracking :exec
-- Deleting a client with Stop tracking parts every Torrent it holds at
-- once and leaves each in the client, a Removing one too, whose removal
-- the client never confirmed.
UPDATE downloadclients_torrent SET
    removed_at = coalesce(removed_at, sqlc.arg(at)),
    removed_reason = 'client_deleted',
    parted_at = sqlc.arg(at),
    freed = NULL
WHERE client_id = sqlc.arg(client_id) AND parted_at IS NULL;

-- name: CountLiveByClient :many
SELECT client_id, count(*) AS torrents FROM downloadclients_torrent
WHERE client_id IS NOT NULL AND removed_at IS NULL
GROUP BY client_id ORDER BY client_id;

-- name: GrabsGoTo :one
-- The Node whose new Grabs go to the client.
SELECT node_id FROM downloadclients_node WHERE grabs_to = sqlc.arg(client_id);

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
