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
