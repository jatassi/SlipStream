-- name: InsertArriving :exec
-- A Move's destination: a new Torrent of the source's metainfo on the
-- destination's Node and client, paused, with the source's origin and
-- totals.
INSERT INTO downloadclients_torrent (
    id, node_id, client_id, info_hash, metainfo, name, size, private, trackers, content_type, adopted, tracker_id, save_folder,
    wanted, added_order, added_at, uploaded, downloaded, active_seconds, seeding_seconds, announced_seconds, totals_at, arriving_unit
)
SELECT sqlc.arg(id), sqlc.arg(node_id), sqlc.narg(client_id), info_hash, metainfo, name, size, private, trackers, content_type, adopted,
       tracker_id, sqlc.narg(save_folder), 'paused', sqlc.arg(added_order), sqlc.arg(added_at), uploaded, downloaded, active_seconds,
       seeding_seconds, announced_seconds, totals_at, sqlc.arg(arriving_unit)
FROM downloadclients_torrent AS src WHERE src.id = sqlc.arg(from_id);

-- name: Watch :exec
-- A watch that ends forgets what it saw, so the next starts afresh.
UPDATE downloadclients_torrent SET watch_announce = 1 WHERE id = sqlc.arg(id) AND watch_announce = 0;

-- name: Unwatch :exec
UPDATE downloadclients_torrent SET watch_announce = 0, announced_at = NULL, announce_due_at = NULL, paused_at = NULL
WHERE id = sqlc.arg(id);

-- name: AskStart :execrows
-- The arriving Torrent takes the source's totals after its pause and is
-- asked to start inside its window. A result of an earlier ask clears.
UPDATE downloadclients_torrent SET
    uploaded = max(uploaded, (SELECT s.uploaded FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    downloaded = max(downloaded, (SELECT s.downloaded FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    active_seconds = max(active_seconds, (SELECT s.active_seconds FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    seeding_seconds = max(seeding_seconds, (SELECT s.seeding_seconds FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    announced_seconds = max(announced_seconds, (SELECT s.announced_seconds FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    start_asked_at = sqlc.arg(asked_at), start_until_at = sqlc.arg(until_at), wanted = 'started',
    arrival = NULL, arrival_reason = NULL, arrived_at = NULL
WHERE downloadclients_torrent.id = sqlc.arg(id) AND arriving_unit IS NOT NULL AND removed_at IS NULL;

-- name: RollBackArriving :exec
UPDATE downloadclients_torrent SET wanted = 'paused', start_asked_at = NULL, start_until_at = NULL,
    arrival = NULL, arrival_reason = NULL, arrived_at = NULL, fault = NULL, fault_at = NULL
WHERE downloadclients_torrent.id = sqlc.arg(id) AND arriving_unit IS NOT NULL AND removed_at IS NULL;

-- name: ResumeSource :exec
UPDATE downloadclients_torrent SET wanted = 'started', paused_at = NULL WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: RemoveMoved :execrows
UPDATE downloadclients_torrent SET removed_at = sqlc.arg(at), removed_reason = 'moved', delete_files = 1,
    watch_announce = 0, announced_at = NULL, announce_due_at = NULL, paused_at = NULL
WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: Arrive :execrows
-- The arriving Torrent becomes the torrent's live one, with the larger of
-- its own totals and the source's final ones.
UPDATE downloadclients_torrent SET
    uploaded = max(uploaded, (SELECT s.uploaded FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    downloaded = max(downloaded, (SELECT s.downloaded FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    active_seconds = max(active_seconds, (SELECT s.active_seconds FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    seeding_seconds = max(seeding_seconds, (SELECT s.seeding_seconds FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    announced_seconds = max(announced_seconds, (SELECT s.announced_seconds FROM downloadclients_torrent s WHERE s.id = sqlc.arg(from_id))),
    arriving_unit = NULL, start_asked_at = NULL, start_until_at = NULL, arrival = NULL, arrival_reason = NULL, arrived_at = NULL
WHERE downloadclients_torrent.id = sqlc.arg(id) AND arriving_unit IS NOT NULL AND removed_at IS NULL;

-- name: FoldAnnounces :exec
-- The latest answer never moves back; the next due is the report's.
UPDATE downloadclients_torrent SET
    announced_at = CASE WHEN announced_at IS NULL OR announced_at < sqlc.narg(answered_at) THEN coalesce(sqlc.narg(answered_at), announced_at)
        ELSE announced_at END,
    announce_due_at = sqlc.narg(due_at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND watch_announce = 1;

-- name: SetPausedAt :exec
-- A paused Torrent a Move watches is seen paused once each tracker
-- answered its stopped, or at once in a client that can't tell.
UPDATE downloadclients_torrent SET paused_at = sqlc.arg(at)
WHERE downloadclients_torrent.id = sqlc.arg(id) AND downloadclients_torrent.node_id = sqlc.arg(node_id)
  AND watch_announce = 1 AND wanted = 'paused' AND paused_at IS NULL AND removed_at IS NULL
  AND (CAST(sqlc.arg(stopped) AS INTEGER) = 1
    OR downloadclients_torrent.client_id IN (SELECT c.id FROM downloadclients_client c WHERE c.kind IN (sqlc.slice(blind))));

-- name: SetArrival :exec
-- A started or refused counts only for the ask it echoes, a complete add's
-- result only before any ask; the first result stands.
UPDATE downloadclients_torrent SET arrival = sqlc.arg(arrival), arrival_reason = sqlc.narg(arrival_reason), arrived_at = sqlc.arg(at)
WHERE id = sqlc.arg(id) AND node_id = sqlc.arg(node_id) AND arriving_unit IS NOT NULL AND removed_at IS NULL AND arrival IS NULL
  AND (CAST(sqlc.arg(added) AS INTEGER) = 1 AND start_asked_at IS NULL OR start_asked_at = sqlc.narg(asked_at));

-- name: ListHandoffs :many
SELECT id, node_id, client_id, wanted, announced_at, announce_due_at, paused_at, start_asked_at, arrival, arrival_reason,
       removed_at, removed_reason, parted_at, completion IS NOT NULL AS completed,
       uploaded, downloaded, active_seconds, seeding_seconds, announced_seconds, totals_at
FROM downloadclients_torrent WHERE id IN (sqlc.slice(ids)) ORDER BY id;

-- name: ListMoveSources :many
SELECT t.id, t.node_id, t.client_id, t.info_hash, t.metainfo IS NOT NULL AS has_metainfo, t.content_type, t.size, t.name, t.wanted,
       t.private = 1 AS private, t.withheld_at, t.removed_at, t.arriving_unit IS NOT NULL AS arriving, t.completion,
       coalesce(c.kind, '') AS client_kind, coalesce(c.name, '') AS client_name, c.client_version
FROM downloadclients_torrent t LEFT JOIN downloadclients_client c ON c.id = t.client_id
WHERE t.id IN (sqlc.slice(ids))
ORDER BY t.id;

-- name: ListHeldOnNode :many
SELECT info_hash, client_id FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND info_hash IN (sqlc.slice(hashes)) AND removed_at IS NULL AND arriving_unit IS NULL
ORDER BY added_order;

-- name: ListTorrentsInClient :many
-- A client's live Torrents a Move can name, a Move's arriving ones left
-- out, in added order. A NULL client is node's built-in one.
SELECT id FROM downloadclients_torrent
WHERE node_id = sqlc.arg(node_id) AND client_id IS sqlc.narg(client_id) AND removed_at IS NULL AND arriving_unit IS NULL
ORDER BY added_order;
