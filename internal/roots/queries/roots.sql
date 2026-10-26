-- name: CountRootsOf :one
SELECT count(*) FROM roots_root WHERE content_type = sqlc.arg(content_type);

-- name: InsertRoot :exec
INSERT INTO roots_root (id, content_type, node_id, path, is_default)
VALUES (sqlc.arg(id), sqlc.arg(content_type), sqlc.arg(node_id), sqlc.arg(path), sqlc.arg(is_default));

-- name: ListRoots :many
SELECT * FROM roots_root ORDER BY content_type, id;

-- name: GetRoot :one
SELECT * FROM roots_root WHERE id = sqlc.arg(id);

-- name: GetDefaultRoot :one
SELECT * FROM roots_root WHERE content_type = sqlc.arg(content_type) AND is_default = 1;

-- name: ClearDefaultRoot :exec
UPDATE roots_root SET is_default = 0 WHERE content_type = sqlc.arg(content_type) AND is_default = 1;

-- name: SetDefaultRoot :exec
UPDATE roots_root SET is_default = 1 WHERE id = sqlc.arg(id);

-- name: GetRootOfWork :one
SELECT r.* FROM roots_work w JOIN roots_root r ON r.id = w.root_id WHERE w.work_id = sqlc.arg(work_id);

-- name: ListHomedWorks :many
SELECT work_id FROM roots_work ORDER BY work_id;

-- name: HomeWork :exec
INSERT INTO roots_work (work_id, root_id) VALUES (sqlc.arg(work_id), sqlc.arg(root_id))
ON CONFLICT (work_id) DO UPDATE SET root_id = excluded.root_id;

-- name: CountWorksByRoot :many
SELECT root_id, count(*) AS works FROM roots_work GROUP BY root_id ORDER BY root_id;

-- name: ListRootsOnNode :many
SELECT * FROM roots_root WHERE node_id = sqlc.arg(node_id) ORDER BY id;

-- name: ListIntentRoots :many
SELECT r.id, r.path, r.place, s.place AS seen_place
FROM roots_root r LEFT JOIN roots_seen s ON s.root_id = r.id
WHERE r.node_id = sqlc.arg(node_id) ORDER BY r.id;

-- name: ListSeen :many
SELECT * FROM roots_seen ORDER BY root_id;

-- name: SeeRoot :exec
INSERT INTO roots_seen (root_id, place, volume_id) VALUES (sqlc.arg(root_id), sqlc.arg(place), sqlc.arg(volume_id))
ON CONFLICT (root_id) DO UPDATE SET place = excluded.place, volume_id = excluded.volume_id;

-- name: DeleteVolumesOf :exec
DELETE FROM roots_volume WHERE node_id = sqlc.arg(node_id);

-- name: InsertVolume :exec
INSERT INTO roots_volume (node_id, id, name, mount, kind, total, free, sampled_at)
VALUES (sqlc.arg(node_id), sqlc.arg(id), sqlc.arg(name), sqlc.arg(mount), sqlc.arg(kind), sqlc.arg(total), sqlc.arg(free), sqlc.arg(sampled_at));

-- name: MoveRoot :exec
UPDATE roots_root SET node_id = sqlc.arg(node_id), path = sqlc.arg(path), place = place + 1 WHERE id = sqlc.arg(id);

-- name: DeleteRoot :exec
DELETE FROM roots_root WHERE id = sqlc.arg(id);

-- name: ListWorksOfRoot :many
SELECT work_id FROM roots_work WHERE root_id = sqlc.arg(root_id) ORDER BY work_id;

-- name: ListVolumesOf :many
SELECT * FROM roots_volume WHERE node_id = sqlc.arg(node_id) ORDER BY id;

-- name: ListVolumes :many
SELECT * FROM roots_volume ORDER BY node_id, name, id;

-- name: ListRootNodes :many
SELECT DISTINCT node_id FROM roots_root ORDER BY node_id;

-- name: GetSampleEvery :one
SELECT sample_every FROM roots_settings WHERE id = 1;

-- name: SetSampleEvery :exec
UPDATE roots_settings SET sample_every = sqlc.arg(sample_every) WHERE id = 1;

-- name: ListHomesOf :many
SELECT work_id, root_id FROM roots_work WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;
