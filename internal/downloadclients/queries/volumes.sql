-- name: PutVolume :exec
INSERT INTO downloadclients_volume (node_id, mount, name, total, free, sampled_at)
VALUES (sqlc.arg(node_id), sqlc.arg(mount), sqlc.arg(name), sqlc.arg(total), sqlc.arg(free), sqlc.arg(sampled_at))
ON CONFLICT (node_id) DO UPDATE SET
    mount = excluded.mount, name = excluded.name, total = excluded.total, free = excluded.free, sampled_at = excluded.sampled_at
WHERE excluded.sampled_at >= downloadclients_volume.sampled_at;

-- name: ListVolumes :many
SELECT * FROM downloadclients_volume ORDER BY node_id;

-- name: GetVolume :one
SELECT * FROM downloadclients_volume WHERE node_id = sqlc.arg(node_id);
