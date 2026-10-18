-- name: ListNodes :many
SELECT * FROM nodes_node WHERE removed_at IS NULL ORDER BY id;

-- name: GetNode :one
SELECT * FROM nodes_node WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: GetNodeByKey :one
SELECT * FROM nodes_node WHERE public_key = sqlc.arg(public_key);

-- name: NameTaken :one
SELECT EXISTS (SELECT 1 FROM nodes_node WHERE name = sqlc.arg(name) AND removed_at IS NULL);

-- name: InsertNode :exec
INSERT INTO nodes_node (id, public_key, name, paired_at)
VALUES (sqlc.arg(id), sqlc.arg(public_key), sqlc.arg(name), sqlc.arg(paired_at));

-- name: SetPairedAt :exec
UPDATE nodes_node SET paired_at = sqlc.arg(paired_at) WHERE id = sqlc.arg(id);

-- name: SetSeenAt :exec
UPDATE nodes_node SET seen_at = sqlc.arg(seen_at) WHERE id = sqlc.arg(id);

-- name: ListAddresses :many
SELECT * FROM nodes_address ORDER BY node_id, position;

-- name: NodeAddresses :many
SELECT address FROM nodes_address WHERE node_id = sqlc.arg(node_id) ORDER BY position;

-- name: DeleteAddresses :exec
DELETE FROM nodes_address WHERE node_id = sqlc.arg(node_id);

-- name: InsertAddress :exec
INSERT INTO nodes_address (node_id, position, address)
VALUES (sqlc.arg(node_id), sqlc.arg(position), sqlc.arg(address));

-- name: ListRoles :many
SELECT * FROM nodes_role ORDER BY node_id, role;

-- name: NodeRoles :many
SELECT role FROM nodes_role WHERE node_id = sqlc.arg(node_id) ORDER BY role;

-- name: InsertRole :exec
INSERT INTO nodes_role (node_id, role, assigned_at)
VALUES (sqlc.arg(node_id), sqlc.arg(role), sqlc.arg(assigned_at))
ON CONFLICT (node_id, role) DO NOTHING;

-- name: DeleteRole :exec
DELETE FROM nodes_role WHERE node_id = sqlc.arg(node_id) AND role = sqlc.arg(role);

-- name: InsertJoinCode :exec
INSERT INTO nodes_join_code (id, secret_hash, created_at, expires_at)
VALUES (sqlc.arg(id), sqlc.arg(secret_hash), sqlc.arg(created_at), sqlc.arg(expires_at));

-- name: GetJoinCode :one
SELECT * FROM nodes_join_code WHERE secret_hash = sqlc.arg(secret_hash);

-- name: GetJoinCodeByID :one
SELECT * FROM nodes_join_code WHERE id = sqlc.arg(id);

-- name: ListOpenJoinCodes :many
SELECT * FROM nodes_join_code
WHERE used_by IS NULL AND expires_at > sqlc.arg(now)
ORDER BY created_at, id;

-- name: UseJoinCode :exec
UPDATE nodes_join_code SET used_by = sqlc.arg(used_by), used_at = sqlc.arg(used_at), dial_address = NULL, dial_at = NULL
WHERE secret_hash = sqlc.arg(secret_hash);

-- name: SetJoinCodeDial :exec
UPDATE nodes_join_code SET dial_address = sqlc.arg(dial_address), dial_at = sqlc.arg(dial_at) WHERE id = sqlc.arg(id);

-- name: DeleteJoinCode :execrows
DELETE FROM nodes_join_code WHERE id = sqlc.arg(id);

-- name: DeleteJoinCodesExpiredBefore :exec
DELETE FROM nodes_join_code WHERE expires_at < sqlc.arg(expires_at);

-- name: GetJoinCodeLifetime :one
SELECT join_code_lifetime_seconds FROM nodes_settings WHERE id = 1;

-- name: SetJoinCodeLifetime :exec
UPDATE nodes_settings SET join_code_lifetime_seconds = sqlc.arg(join_code_lifetime_seconds) WHERE id = 1;

-- name: ListForfeits :many
SELECT node_id, role FROM nodes_forfeit ORDER BY node_id, role;

-- name: GetOfflineGrace :one
SELECT offline_grace_seconds FROM nodes_settings WHERE id = 1;

-- name: SetOfflineGrace :exec
UPDATE nodes_settings SET offline_grace_seconds = sqlc.arg(offline_grace_seconds) WHERE id = 1;

-- name: InsertForfeit :exec
INSERT INTO nodes_forfeit (node_id, role, declared_at)
VALUES (sqlc.arg(node_id), sqlc.arg(role), sqlc.arg(declared_at));

-- name: DeleteForfeit :exec
DELETE FROM nodes_forfeit WHERE node_id = sqlc.arg(node_id) AND role = sqlc.arg(role);

-- name: ListDials :many
SELECT * FROM nodes_dial ORDER BY node_id, position;

-- name: NodeDials :many
SELECT address FROM nodes_dial WHERE node_id = sqlc.arg(node_id) ORDER BY position;

-- name: DeleteDials :exec
DELETE FROM nodes_dial WHERE node_id = sqlc.arg(node_id);

-- name: InsertDial :exec
INSERT INTO nodes_dial (node_id, position, address)
VALUES (sqlc.arg(node_id), sqlc.arg(position), sqlc.arg(address));
