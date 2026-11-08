-- name: MergeTotals :exec
UPDATE downloadclients_torrent SET
    uploaded = MAX(uploaded, CAST(sqlc.arg(uploaded) AS INTEGER)),
    downloaded = MAX(downloaded, CAST(sqlc.arg(downloaded) AS INTEGER)),
    active_seconds = MAX(active_seconds, CAST(sqlc.arg(active_seconds) AS INTEGER)),
    seeding_seconds = MAX(seeding_seconds, CAST(sqlc.arg(seeding_seconds) AS INTEGER)),
    announced_seconds = MAX(announced_seconds, CAST(sqlc.arg(announced_seconds) AS INTEGER)),
    totals_at = sqlc.arg(totals_at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id);

-- name: GetNodeSettings :one
SELECT * FROM downloadclients_node WHERE node_id = ?;

-- name: SetNodeSettings :exec
INSERT INTO downloadclients_node (
    node_id, download_limit, upload_limit, active_downloads, active_checks, connections, utp, stalled_after_seconds, dht, grabs_to
) VALUES (
    sqlc.arg(node_id), sqlc.narg(download_limit), sqlc.narg(upload_limit), sqlc.narg(active_downloads),
    sqlc.narg(active_checks), sqlc.narg(connections), sqlc.narg(utp), sqlc.narg(stalled_after_seconds), sqlc.narg(dht),
    sqlc.narg(grabs_to)
)
ON CONFLICT (node_id) DO UPDATE SET
    download_limit = excluded.download_limit,
    upload_limit = excluded.upload_limit,
    active_downloads = excluded.active_downloads,
    active_checks = excluded.active_checks,
    connections = excluded.connections,
    utp = excluded.utp,
    stalled_after_seconds = excluded.stalled_after_seconds,
    dht = excluded.dht,
    grabs_to = excluded.grabs_to;

-- name: GetThresholds :one
SELECT fault_stall_seconds, unregistered_gap_seconds, metadata_wait_seconds FROM downloadclients_settings WHERE id = 1;

-- name: SetThresholds :exec
UPDATE downloadclients_settings SET
    fault_stall_seconds = sqlc.narg(fault_stall_seconds),
    unregistered_gap_seconds = sqlc.narg(unregistered_gap_seconds),
    metadata_wait_seconds = sqlc.narg(metadata_wait_seconds)
WHERE id = 1;

-- name: GetMetainfoOfHash :one
-- The metainfo of the newest Torrent of a hash on any Node, live or
-- removed.
SELECT metainfo FROM downloadclients_torrent WHERE info_hash = sqlc.arg(info_hash) ORDER BY added_order DESC LIMIT 1;

-- name: ListHeldOfHash :many
SELECT id, node_id, removed_at FROM downloadclients_torrent
WHERE info_hash = sqlc.arg(info_hash) AND parted_at IS NULL
ORDER BY added_order;

-- name: GetInheritedTotals :one
SELECT uploaded, downloaded, active_seconds, seeding_seconds, announced_seconds, totals_at FROM downloadclients_torrent
WHERE info_hash = sqlc.arg(info_hash) AND removed_at IS NOT NULL
ORDER BY removed_at DESC, added_order DESC
LIMIT 1;

-- name: NextOrder :one
SELECT CAST(coalesce(max(added_order), 0) + 1 AS INTEGER) FROM downloadclients_torrent;

-- name: InsertTorrent :exec
INSERT INTO downloadclients_torrent (
    id, node_id, client_id, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id, wanted, added_order, added_at,
    uploaded, downloaded, active_seconds, seeding_seconds, announced_seconds, totals_at
) VALUES (
    sqlc.arg(id), sqlc.arg(node_id), sqlc.narg(client_id), sqlc.arg(info_hash), sqlc.arg(metainfo), sqlc.arg(name), sqlc.arg(size),
    sqlc.arg(private), sqlc.arg(trackers), sqlc.arg(content_type), sqlc.narg(tracker_id), 'started', sqlc.arg(added_order), sqlc.arg(added_at),
    sqlc.arg(uploaded), sqlc.arg(downloaded), sqlc.arg(active_seconds), sqlc.arg(seeding_seconds), sqlc.arg(announced_seconds), sqlc.arg(totals_at)
);

-- name: ListAttached :many
SELECT t.id, t.info_hash, t.size, t.removed_reason, t.report, t.completion, t.fault, t.fault_at,
       t.client_id, coalesce(c.kind, '') AS client_kind, coalesce(c.name, '') AS client_name
FROM downloadclients_torrent t LEFT JOIN downloadclients_client c ON c.id = t.client_id
WHERE t.id IN (sqlc.slice(ids))
ORDER BY t.id;

-- name: ListSources :many
SELECT info_hash, name, removed_reason, report FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND info_hash IN (sqlc.slice(hashes))
ORDER BY added_order;

-- name: ListIntent :many
SELECT id, client_id, info_hash, wanted, added_order, removed_at, delete_files, retry FROM downloadclients_torrent
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
SELECT count(*) AS torrents, CAST(coalesce(sum(size), 0) AS INTEGER) AS bytes
FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND parted_at IS NULL;

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
SELECT node_id, metainfo, content_type, save_folder, removed_at, attached,
       uploaded, downloaded, active_seconds, seeding_seconds, announced_seconds
FROM downloadclients_torrent WHERE id = sqlc.arg(id);

-- name: ListHoldingNodes :many
SELECT DISTINCT node_id FROM downloadclients_torrent WHERE parted_at IS NULL ORDER BY node_id;

-- name: ListUnparted :many
SELECT id FROM downloadclients_torrent WHERE node_id = sqlc.arg(node_id) AND parted_at IS NULL ORDER BY id;

-- name: WriteOff :exec
UPDATE downloadclients_torrent SET
    removed_at = coalesce(removed_at, sqlc.arg(at)),
    removed_reason = 'node_written_off',
    parted_at = sqlc.arg(at)
WHERE node_id = sqlc.arg(node_id) AND parted_at IS NULL;

-- name: ListActed :many
SELECT t.id, t.node_id, t.info_hash, t.removed_at, t.wanted, t.report, t.completion, t.client_id, t.attached,
       coalesce(c.name, '') AS client_name, coalesce(c.is_on, 1) = 1 AS client_on, c.poll_state
FROM downloadclients_torrent t LEFT JOIN downloadclients_client c ON c.id = t.client_id
WHERE t.id IN (sqlc.slice(ids))
ORDER BY t.id;

-- name: SetWanted :exec
UPDATE downloadclients_torrent SET wanted = sqlc.arg(wanted)
WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: GetRetry :one
SELECT retry, size FROM downloadclients_torrent
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id);

-- name: SetRetry :exec
UPDATE downloadclients_torrent SET retry = sqlc.arg(retry)
WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: Remove :exec
UPDATE downloadclients_torrent SET
    removed_at = sqlc.arg(removed_at), removed_reason = sqlc.arg(removed_reason), delete_files = sqlc.arg(delete_files)
WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: RemoveParted :exec
-- A removal its client takes no part in parts at once and frees nothing.
UPDATE downloadclients_torrent SET
    removed_at = sqlc.arg(at), removed_reason = sqlc.arg(removed_reason), parted_at = sqlc.arg(at), freed = 0
WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: ListHeld :many
-- Each by_ flag narrows to its argument. An empty info_hashes lists none.
SELECT t.id, t.node_id, t.info_hash, t.name, t.size, t.private, t.trackers, t.content_type, t.tracker_id, t.wanted,
       t.added_order, t.added_at, t.removed_at, t.removed_reason, t.parted_at, t.uploaded, t.downloaded, t.active_seconds,
       t.seeding_seconds, t.announced_seconds, t.totals_at, t.report, t.fault, t.fault_at, t.client_id, t.attached,
       t.completed_at, t.retry, coalesce(c.kind, '') AS client_kind, coalesce(c.name, '') AS client_name, coalesce(c.is_on, 1) = 1 AS client_on
FROM downloadclients_torrent t LEFT JOIN downloadclients_client c ON c.id = t.client_id
WHERE t.parted_at IS NULL
  AND (NOT CAST(sqlc.arg(by_tracker) AS INTEGER) OR t.tracker_id = sqlc.narg(tracker_id))
  AND (NOT CAST(sqlc.arg(no_tracker) AS INTEGER) OR t.tracker_id IS NULL)
  AND (NOT CAST(sqlc.arg(by_hash) AS INTEGER) OR t.info_hash IN (sqlc.slice(info_hashes)))
ORDER BY t.added_order;

-- name: ListHashFacts :many
-- Every built-in Torrent of the hashes its Node holds, Removing ones too,
-- and each hash's latest one its Node parted from. It selects ListHeld's
-- columns.
SELECT t.id, t.node_id, t.info_hash, t.name, t.size, t.private, t.trackers, t.content_type, t.tracker_id, t.wanted,
       t.added_order, t.added_at, t.removed_at, t.removed_reason, t.parted_at, t.uploaded, t.downloaded, t.active_seconds,
       t.seeding_seconds, t.announced_seconds, t.totals_at, t.report, t.fault, t.fault_at, t.client_id, t.attached,
       t.completed_at, t.retry, coalesce(c.kind, '') AS client_kind, coalesce(c.name, '') AS client_name, coalesce(c.is_on, 1) = 1 AS client_on
FROM downloadclients_torrent t LEFT JOIN downloadclients_client c ON c.id = t.client_id
WHERE t.info_hash IN (sqlc.slice(info_hashes)) AND t.client_id IS NULL
  AND (t.parted_at IS NULL OR t.id = (
      SELECT p.id FROM downloadclients_torrent AS p
      WHERE p.info_hash = t.info_hash AND p.client_id IS NULL AND p.parted_at IS NOT NULL
      ORDER BY p.removed_at DESC, p.added_order DESC
      LIMIT 1))
ORDER BY t.added_order;

-- name: ListExternalAdds :many
-- When each live Torrent of the hashes that an external client holds was
-- added.
SELECT info_hash, added_at FROM downloadclients_torrent
WHERE info_hash IN (sqlc.slice(info_hashes)) AND client_id IS NOT NULL AND removed_at IS NULL
ORDER BY info_hash, added_at;

-- name: ListTrackerHashes :many
SELECT DISTINCT info_hash FROM downloadclients_torrent WHERE tracker_id = sqlc.arg(tracker_id) AND client_id IS NULL ORDER BY info_hash;

-- name: GetDetails :one
SELECT metainfo, save_folder FROM downloadclients_torrent
WHERE id = sqlc.arg(id) AND parted_at IS NULL;

-- name: GetNodeTotals :one
SELECT count(*) AS torrents,
       CAST(coalesce(sum(uploaded), 0) AS INTEGER) AS uploaded,
       CAST(coalesce(sum(downloaded), 0) AS INTEGER) AS downloaded,
       CAST(coalesce(sum(active_seconds), 0) AS INTEGER) AS active_seconds,
       CAST(coalesce(sum(seeding_seconds), 0) AS INTEGER) AS seeding_seconds
FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND client_id IS NULL AND removed_at IS NULL;

-- name: SetTrackerIfNone :exec
UPDATE downloadclients_torrent SET tracker_id = sqlc.arg(tracker_id) WHERE id = sqlc.arg(id) AND tracker_id IS NULL;

-- name: RepointTracker :exec
UPDATE downloadclients_torrent SET tracker_id = sqlc.arg(into_id) WHERE tracker_id = sqlc.arg(from_id);

-- name: ListTorrentsOfTracker :many
SELECT id, removed_at IS NULL AS live FROM downloadclients_torrent WHERE tracker_id = sqlc.arg(tracker_id) ORDER BY id;

-- name: ForgetTracker :exec
UPDATE downloadclients_torrent SET tracker_id = NULL WHERE tracker_id = sqlc.arg(tracker_id) AND removed_at IS NOT NULL;

-- name: ListTrackerless :many
-- Every Torrent with no Tracker, live or removed. Its metainfo comes only
-- when its trackers were never stored, as for one added before S28.
SELECT id, info_hash, trackers, CAST(CASE WHEN trackers = '[]' THEN metainfo END AS BLOB) AS metainfo
FROM downloadclients_torrent
WHERE tracker_id IS NULL
ORDER BY added_order;

-- name: SetOrigin :exec
-- The Node's first word on an external torrent's origin stands.
UPDATE downloadclients_torrent SET attached = sqlc.arg(attached)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND attached IS NULL;

-- name: ChangeWanted :execrows
-- A pause or resume made in the client changes only the wanted state it
-- was made from, so it never undoes an Admin's change it didn't see.
UPDATE downloadclients_torrent SET wanted = sqlc.arg(to_wanted)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND removed_at IS NULL AND wanted = sqlc.arg(from_wanted);

-- name: Vanish :execrows
-- A live Torrent parts as removed by its client, freeing nothing; a
-- Removing one parts with its own reason, its removal done by the client.
UPDATE downloadclients_torrent SET
    removed_at = coalesce(removed_at, sqlc.arg(at)),
    removed_reason = coalesce(removed_reason, 'client'),
    parted_at = sqlc.arg(at),
    freed = CASE WHEN removed_at IS NULL THEN NULL ELSE 0 END
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND parted_at IS NULL;

-- name: GetRaw :one
SELECT raw_uploaded, raw_downloaded, raw_active_seconds, raw_seeding_seconds FROM downloadclients_torrent
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND client_id IS NOT NULL;

-- name: AddCounted :exec
UPDATE downloadclients_torrent SET
    uploaded = uploaded + sqlc.arg(uploaded), downloaded = downloaded + sqlc.arg(downloaded),
    active_seconds = active_seconds + sqlc.arg(active_seconds), seeding_seconds = seeding_seconds + sqlc.arg(seeding_seconds),
    raw_uploaded = sqlc.arg(raw_uploaded), raw_downloaded = sqlc.arg(raw_downloaded),
    raw_active_seconds = sqlc.arg(raw_active_seconds), raw_seeding_seconds = sqlc.arg(raw_seeding_seconds),
    totals_at = sqlc.arg(totals_at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id);

-- name: GetRetarget :one
SELECT node_id, client_id, info_hash, metainfo, content_type, tracker_id, reported_at, removed_at FROM downloadclients_torrent
WHERE id = sqlc.arg(id);

-- name: Rehome :exec
-- Before its Node first reports a Torrent, it goes to another client in
-- place: no client holds it yet.
UPDATE downloadclients_torrent SET node_id = sqlc.arg(node_id), client_id = sqlc.narg(client_id)
WHERE id = sqlc.arg(id) AND removed_at IS NULL AND reported_at IS NULL;
