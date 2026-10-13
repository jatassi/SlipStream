-- name: ListNodes :many
SELECT * FROM nodes_node ORDER BY id;

-- name: GetNode :one
SELECT * FROM nodes_node WHERE id = sqlc.arg(id);

-- name: GetNodeByKey :one
SELECT * FROM nodes_node WHERE public_key = sqlc.arg(public_key);

-- name: NameTaken :one
SELECT EXISTS (SELECT 1 FROM nodes_node WHERE name = sqlc.arg(name));

-- name: InsertNode :exec
INSERT INTO nodes_node (id, public_key, name, paired_at)
VALUES (sqlc.arg(id), sqlc.arg(public_key), sqlc.arg(name), sqlc.arg(paired_at));

-- name: SetPairedAt :exec
UPDATE nodes_node SET paired_at = sqlc.arg(paired_at) WHERE id = sqlc.arg(id);

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
INSERT INTO nodes_join_code (secret_hash, created_at, expires_at)
VALUES (sqlc.arg(secret_hash), sqlc.arg(created_at), sqlc.arg(expires_at));

-- name: GetJoinCode :one
SELECT * FROM nodes_join_code WHERE secret_hash = sqlc.arg(secret_hash);

-- name: UseJoinCode :exec
UPDATE nodes_join_code SET used_by = sqlc.arg(used_by), used_at = sqlc.arg(used_at)
WHERE secret_hash = sqlc.arg(secret_hash);

-- name: GetJoinCodeLifetime :one
SELECT join_code_lifetime_seconds FROM nodes_settings WHERE id = 1;

-- name: SetJoinCodeLifetime :exec
UPDATE nodes_settings SET join_code_lifetime_seconds = sqlc.arg(join_code_lifetime_seconds) WHERE id = 1;
