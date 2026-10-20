-- name: MergeTotals :exec
UPDATE downloadclients_torrent SET
    uploaded = MAX(uploaded, CAST(sqlc.arg(uploaded) AS INTEGER)),
    downloaded = MAX(downloaded, CAST(sqlc.arg(downloaded) AS INTEGER)),
    active_seconds = MAX(active_seconds, CAST(sqlc.arg(active_seconds) AS INTEGER)),
    seeding_seconds = MAX(seeding_seconds, CAST(sqlc.arg(seeding_seconds) AS INTEGER)),
    totals_at = sqlc.arg(totals_at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id);

-- name: GetNodeSettings :one
SELECT * FROM downloadclients_node WHERE node_id = ?;

-- name: SetNodeSettings :exec
INSERT INTO downloadclients_node (
    node_id, download_limit, upload_limit, active_downloads, active_checks, connections, utp, stalled_after_seconds
) VALUES (
    sqlc.arg(node_id), sqlc.narg(download_limit), sqlc.narg(upload_limit), sqlc.narg(active_downloads),
    sqlc.narg(active_checks), sqlc.narg(connections), sqlc.narg(utp), sqlc.narg(stalled_after_seconds)
)
ON CONFLICT (node_id) DO UPDATE SET
    download_limit = excluded.download_limit,
    upload_limit = excluded.upload_limit,
    active_downloads = excluded.active_downloads,
    active_checks = excluded.active_checks,
    connections = excluded.connections,
    utp = excluded.utp,
    stalled_after_seconds = excluded.stalled_after_seconds;

-- name: GetThresholds :one
SELECT fault_stall_seconds, unregistered_gap_seconds FROM downloadclients_settings WHERE id = 1;

-- name: SetThresholds :exec
UPDATE downloadclients_settings SET
    fault_stall_seconds = sqlc.narg(fault_stall_seconds),
    unregistered_gap_seconds = sqlc.narg(unregistered_gap_seconds)
WHERE id = 1;

-- name: ListHeldOfHash :many
SELECT id, node_id, removed_at FROM downloadclients_torrent
WHERE info_hash = sqlc.arg(info_hash) AND parted_at IS NULL
ORDER BY added_order;

-- name: GetInheritedTotals :one
SELECT uploaded, downloaded, active_seconds, seeding_seconds, totals_at FROM downloadclients_torrent
WHERE info_hash = sqlc.arg(info_hash) AND removed_at IS NOT NULL
ORDER BY removed_at DESC, added_order DESC
LIMIT 1;

-- name: NextOrder :one
SELECT CAST(coalesce(max(added_order), 0) + 1 AS INTEGER) FROM downloadclients_torrent;

-- name: InsertTorrent :exec
INSERT INTO downloadclients_torrent (
    id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted, added_order, added_at,
    uploaded, downloaded, active_seconds, seeding_seconds, totals_at
) VALUES (
    sqlc.arg(id), sqlc.arg(node_id), sqlc.arg(info_hash), sqlc.arg(metainfo), sqlc.arg(name), sqlc.arg(size),
    sqlc.arg(private), sqlc.arg(trackers), sqlc.arg(content_type), 'started', sqlc.arg(added_order), sqlc.arg(added_at),
    sqlc.arg(uploaded), sqlc.arg(downloaded), sqlc.arg(active_seconds), sqlc.arg(seeding_seconds), sqlc.arg(totals_at)
);

-- name: ListAttached :many
SELECT id, info_hash, size, removed_reason, report, completion, fault, fault_at FROM downloadclients_torrent
WHERE id IN (sqlc.slice(ids))
ORDER BY id;

-- name: ListIntent :many
SELECT id, info_hash, wanted, added_order, removed_at, delete_files FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND parted_at IS NULL
ORDER BY added_order;

-- name: ListKept :many
SELECT id, info_hash, name, size, report FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND removed_at IS NULL
ORDER BY added_order;

-- name: ListDownloads :many
SELECT id, size, wanted, report FROM downloadclients_torrent
WHERE removed_at IS NULL AND completion IS NULL
ORDER BY added_order;

-- name: CountHeld :one
SELECT count(*) AS torrents, CAST(coalesce(sum(size), 0) AS INTEGER) AS bytes FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND removed_at IS NULL;

-- name: SetCompletion :exec
UPDATE downloadclients_torrent SET completion = sqlc.arg(completion), completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND completion IS NULL;

-- name: SetReport :execrows
UPDATE downloadclients_torrent SET report = sqlc.arg(report), reported_at = sqlc.arg(reported_at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id);

-- name: SetFault :exec
UPDATE downloadclients_torrent SET fault = sqlc.arg(fault), fault_at = sqlc.arg(fault_at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND fault IS NULL;

-- name: SetSaveFolder :exec
UPDATE downloadclients_torrent SET save_folder = sqlc.arg(save_folder)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND save_folder IS NULL;

-- name: SetDone :exec
UPDATE downloadclients_torrent SET report = json_set(report, '$.done', CAST(sqlc.arg(done) AS INTEGER))
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND report IS NOT NULL;

-- name: SetParted :execrows
UPDATE downloadclients_torrent SET parted_at = sqlc.arg(parted_at), freed = sqlc.arg(freed)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND removed_at IS NOT NULL AND parted_at IS NULL;

-- name: GetFetched :one
SELECT node_id, metainfo, content_type, save_folder, removed_at,
       uploaded, downloaded, active_seconds, seeding_seconds
FROM downloadclients_torrent WHERE id = sqlc.arg(id);

-- name: ListHoldingNodes :many
SELECT DISTINCT node_id FROM downloadclients_torrent WHERE parted_at IS NULL ORDER BY node_id;

-- name: WriteOff :execrows
UPDATE downloadclients_torrent SET
    removed_at = coalesce(removed_at, sqlc.arg(at)),
    removed_reason = 'node_written_off',
    parted_at = sqlc.arg(at)
WHERE node_id = sqlc.arg(node_id) AND parted_at IS NULL;
