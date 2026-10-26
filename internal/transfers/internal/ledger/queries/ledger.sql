-- name: InsertTransfer :exec
INSERT INTO transfers_transfer (id, purpose, job, source_node, destination_kind, destination_root, created_at)
VALUES (sqlc.arg(id), sqlc.arg(purpose), sqlc.arg(job), sqlc.arg(source_node), sqlc.arg(destination_kind), sqlc.arg(destination_root), sqlc.arg(created_at));

-- name: InsertFile :exec
INSERT INTO transfers_file (transfer_id, position, item, name, source_kind, info_hash, file_index, size)
VALUES (sqlc.arg(transfer_id), sqlc.arg(position), sqlc.arg(item), sqlc.arg(name), sqlc.arg(source_kind), sqlc.arg(info_hash), sqlc.arg(file_index), sqlc.arg(size));

-- name: TransferOfJob :one
SELECT id FROM transfers_transfer WHERE purpose = sqlc.arg(purpose) AND job = sqlc.arg(job);

-- name: GetTransfer :one
SELECT * FROM transfers_transfer WHERE id = sqlc.arg(id);

-- name: ListTransfers :many
SELECT * FROM transfers_transfer ORDER BY created_at, id;

-- name: ListTransferIDs :many
SELECT id FROM transfers_transfer ORDER BY id;

-- name: ListFiles :many
SELECT * FROM transfers_file ORDER BY transfer_id, position;

-- name: FilesOf :many
SELECT * FROM transfers_file WHERE transfer_id = sqlc.arg(transfer_id) ORDER BY position;

-- name: SetVerified :exec
UPDATE transfers_file SET verified_place = sqlc.arg(verified_place)
WHERE transfer_id = sqlc.arg(transfer_id) AND position = sqlc.arg(position);

-- name: SetPlaced :exec
UPDATE transfers_file SET placed = 1 WHERE transfer_id = sqlc.arg(transfer_id) AND position = sqlc.arg(position);

-- name: CountUnplaced :one
SELECT count(*) FROM transfers_file WHERE transfer_id = sqlc.arg(transfer_id) AND placed = 0;

-- name: SetFailure :exec
UPDATE transfers_transfer
SET failure_code = sqlc.arg(failure_code), failure_position = sqlc.arg(failure_position), failure_detail = sqlc.arg(failure_detail)
WHERE id = sqlc.arg(id) AND failure_code IS NULL;

-- name: DeleteTransfer :exec
DELETE FROM transfers_transfer WHERE id = sqlc.arg(id);

-- name: ListPairs :many
SELECT * FROM transfers_pair ORDER BY source_node, destination_node;

-- name: PutPair :exec
INSERT INTO transfers_pair (source_node, destination_node, rate, rate_at, route, route_at)
VALUES (sqlc.arg(source_node), sqlc.arg(destination_node), sqlc.arg(rate), sqlc.arg(rate_at), sqlc.arg(route), sqlc.arg(route_at))
ON CONFLICT (source_node, destination_node) DO UPDATE SET
    rate = excluded.rate, rate_at = excluded.rate_at, route = excluded.route, route_at = excluded.route_at;

-- name: GetNodeSettings :one
SELECT * FROM transfers_node WHERE node_id = sqlc.arg(node_id);

-- name: ListNodeSettings :many
SELECT * FROM transfers_node ORDER BY node_id;

-- name: SetNodeSettings :exec
INSERT INTO transfers_node (node_id, send_limit) VALUES (sqlc.arg(node_id), sqlc.arg(send_limit))
ON CONFLICT (node_id) DO UPDATE SET send_limit = excluded.send_limit;
