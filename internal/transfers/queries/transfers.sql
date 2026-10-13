-- name: InsertTransfer :exec
INSERT INTO transfers_transfer (id, source_node, info_hash, file_index, staging_path, created_at)
VALUES (sqlc.arg(id), sqlc.arg(source_node), sqlc.arg(info_hash), sqlc.arg(file_index), sqlc.arg(staging_path), sqlc.arg(created_at))
ON CONFLICT (id) DO NOTHING;

-- name: Transfer :one
SELECT * FROM transfers_transfer WHERE id = sqlc.arg(id);

-- name: ListTransfers :many
SELECT id, staging_path FROM transfers_transfer ORDER BY id;

-- name: SetManifest :exec
UPDATE transfers_transfer
SET size = sqlc.arg(size), chunk_size = sqlc.arg(chunk_size), chunk_hashes = sqlc.arg(chunk_hashes)
WHERE id = sqlc.arg(id);

-- name: DeleteTransfer :exec
DELETE FROM transfers_transfer WHERE id = sqlc.arg(id);

-- name: Chunks :many
SELECT idx FROM transfers_chunk WHERE transfer_id = sqlc.arg(transfer_id) ORDER BY idx;

-- name: InsertChunk :exec
INSERT INTO transfers_chunk (transfer_id, idx) VALUES (sqlc.arg(transfer_id), sqlc.arg(idx));

-- name: DeleteChunks :exec
DELETE FROM transfers_chunk WHERE transfer_id = sqlc.arg(transfer_id);
