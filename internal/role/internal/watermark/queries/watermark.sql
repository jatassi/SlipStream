-- name: GetWatermark :one
SELECT seq FROM role_watermark WHERE node_id = sqlc.arg(node_id) AND epoch = sqlc.arg(epoch);

-- name: SetWatermark :exec
INSERT INTO role_watermark (node_id, epoch, seq) VALUES (sqlc.arg(node_id), sqlc.arg(epoch), sqlc.arg(seq))
ON CONFLICT (node_id, epoch) DO UPDATE SET seq = excluded.seq;
