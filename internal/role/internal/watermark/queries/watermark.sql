-- name: GetWatermark :one
SELECT seq FROM role_watermark WHERE node_id = sqlc.arg(node_id) AND epoch = sqlc.arg(epoch);

-- name: SetWatermark :exec
INSERT INTO role_watermark (node_id, epoch, seq) VALUES (sqlc.arg(node_id), sqlc.arg(epoch), sqlc.arg(seq))
ON CONFLICT (node_id, epoch) DO UPDATE SET seq = excluded.seq;

-- name: InsertRejected :exec
INSERT INTO role_rejected (node_id, epoch, seq, scope, reason, rejected_at)
VALUES (sqlc.arg(node_id), sqlc.arg(epoch), sqlc.arg(seq), sqlc.arg(scope), sqlc.arg(reason), sqlc.arg(rejected_at));

-- name: ListRejected :many
SELECT node_id, rejected_at FROM role_rejected ORDER BY node_id, rejected_at;

-- name: DeleteRejected :exec
DELETE FROM role_rejected WHERE node_id = sqlc.arg(node_id);
