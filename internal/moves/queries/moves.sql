-- name: InsertBatch :exec
INSERT INTO moves_batch (id, origin, created_at, created_by, homed_works, include_torrents, repoint_routing, delete_emptied_roots, plan_token_hash, notes)
VALUES (sqlc.arg(id), sqlc.arg(origin), sqlc.arg(created_at), sqlc.arg(created_by), sqlc.arg(homed_works), sqlc.arg(include_torrents),
    sqlc.arg(repoint_routing), sqlc.arg(delete_emptied_roots), sqlc.arg(plan_token_hash), sqlc.arg(notes));

-- name: BatchOfToken :one
SELECT * FROM moves_batch WHERE plan_token_hash = sqlc.arg(hash);

-- name: GetBatch :one
SELECT * FROM moves_batch WHERE id = sqlc.arg(id);

-- name: ListBatches :many
SELECT * FROM moves_batch
WHERE CAST(sqlc.arg(first) AS INTEGER) = 1 OR (finished_at IS NOT NULL) > CAST(sqlc.arg(after_done) AS INTEGER)
    OR ((finished_at IS NOT NULL) = CAST(sqlc.arg(after_done) AS INTEGER)
        AND (created_at < sqlc.arg(after_created_at) OR (created_at = sqlc.arg(after_created_at) AND id > sqlc.arg(after_id))))
ORDER BY finished_at IS NOT NULL, created_at DESC, id LIMIT sqlc.arg(n);

-- name: FinishBatch :exec
UPDATE moves_batch SET finished_at = sqlc.arg(at) WHERE id = sqlc.arg(id) AND finished_at IS NULL;

-- name: CancelBatch :exec
UPDATE moves_batch SET canceled_at = sqlc.arg(at) WHERE id = sqlc.arg(id) AND canceled_at IS NULL;

-- name: CountBatchPhases :many
SELECT phase, count(*) AS n FROM moves_unit WHERE batch_id = sqlc.arg(batch_id) GROUP BY phase ORDER BY phase;

-- name: CountBatchesPhases :many
SELECT batch_id, phase, count(*) AS n FROM moves_unit WHERE batch_id IN (sqlc.slice(ids)) GROUP BY batch_id, phase ORDER BY batch_id, phase;

-- name: ListBatchTypes :many
SELECT DISTINCT batch_id, content_type FROM moves_unit WHERE batch_id IN (sqlc.slice(ids)) AND content_type IS NOT NULL ORDER BY batch_id, content_type;

-- name: OpenUnitsOfBatch :one
SELECT count(*) FROM moves_unit
WHERE batch_id = sqlc.arg(batch_id) AND (phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup') OR (phase IN ('failed', 'canceled') AND undone_at IS NULL));

-- name: DeleteFinishedBatches :execrows
DELETE FROM moves_batch WHERE id IN (
    SELECT b.id FROM moves_batch b WHERE b.finished_at IS NOT NULL AND b.finished_at < sqlc.arg(before)
    AND NOT EXISTS (SELECT 1 FROM moves_unit u WHERE u.batch_id = b.id
        AND (u.phase NOT IN ('done', 'failed', 'canceled') OR (u.phase IN ('failed', 'canceled') AND u.undone_at IS NULL)))
    ORDER BY b.finished_at LIMIT sqlc.arg(n)
);

-- name: NextSeq :one
SELECT CAST(coalesce(max(seq), 0) + 1 AS INTEGER) FROM moves_unit;

-- name: InsertUnit :exec
INSERT INTO moves_unit (id, batch_id, seq, position, kind, work_id, content_type, torrent_id, from_root_id, from_node_id, from_client_id,
    to_root_id, to_node_id, to_client_id, lane, phase, waiting, waiting_at, due_at, pairs_with_unit_id, planned_bytes, planned_at, notes)
VALUES (sqlc.arg(id), sqlc.arg(batch_id), sqlc.arg(seq), sqlc.arg(position), sqlc.arg(kind), sqlc.narg(work_id), sqlc.narg(content_type),
    sqlc.narg(torrent_id), sqlc.narg(from_root_id), sqlc.arg(from_node_id), sqlc.narg(from_client_id), sqlc.narg(to_root_id), sqlc.arg(to_node_id),
    sqlc.narg(to_client_id), sqlc.arg(lane), 'queued', sqlc.narg(waiting), sqlc.arg(waiting_at), sqlc.arg(due_at), sqlc.narg(pairs_with_unit_id),
    sqlc.arg(planned_bytes), sqlc.arg(planned_at), sqlc.arg(notes));

-- name: GetUnit :one
SELECT * FROM moves_unit WHERE id = sqlc.arg(id);

-- name: ListUnits :many
SELECT * FROM moves_unit WHERE id IN (sqlc.slice(ids)) ORDER BY seq;

-- name: ListUnitsOfBatch :many
SELECT * FROM moves_unit WHERE batch_id = sqlc.arg(batch_id) AND position > sqlc.arg(after) ORDER BY position LIMIT sqlc.arg(n);

-- name: ListBatchDestinations :many
SELECT DISTINCT to_node_id FROM moves_unit WHERE batch_id = sqlc.arg(batch_id) ORDER BY to_node_id;

-- name: ListDoneUnitsOfBatch :many
SELECT * FROM moves_unit WHERE batch_id = sqlc.arg(batch_id) AND phase = 'done' ORDER BY position;

-- name: ListOpenUnitsOfBatch :many
SELECT * FROM moves_unit WHERE batch_id = sqlc.arg(batch_id) AND phase IN ('queued', 'transfer', 'place', 'switch') ORDER BY position;

-- name: ListOpenUnitsOfWorks :many
SELECT * FROM moves_unit WHERE work_id IN (sqlc.slice(works)) AND phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup') ORDER BY seq;

-- name: ListLatestUnitsOfWorks :many
SELECT u.* FROM moves_unit u
WHERE u.work_id IN (sqlc.slice(works)) AND u.seq = (SELECT max(v.seq) FROM moves_unit v WHERE v.work_id = u.work_id)
ORDER BY u.seq;

-- name: UpdateUnit :exec
UPDATE moves_unit SET phase = sqlc.arg(phase), waiting = sqlc.narg(waiting), waiting_at = sqlc.arg(waiting_at), due_at = sqlc.arg(due_at),
    pass = sqlc.arg(pass), switch_now_at = sqlc.arg(switch_now_at), failure_code = sqlc.narg(failure_code), failure_detail = sqlc.narg(failure_detail),
    failed_phase = sqlc.narg(failed_phase), undone_at = sqlc.arg(undone_at), reconcile = sqlc.arg(reconcile), transfer_id = sqlc.narg(transfer_id),
    planned_bytes = sqlc.arg(planned_bytes), planned_at = sqlc.arg(planned_at), switched_at = sqlc.arg(switched_at), cleaned_at = sqlc.arg(cleaned_at),
    notes = sqlc.arg(notes), handoff = sqlc.narg(handoff), handoff_at = sqlc.arg(handoff_at), keeps_copy = sqlc.arg(keeps_copy),
    dest_torrent_id = sqlc.narg(dest_torrent_id), to_folder = sqlc.narg(to_folder)
WHERE id = sqlc.arg(id);

-- name: SetDue :exec
UPDATE moves_unit SET due_at = sqlc.arg(due_at) WHERE id = sqlc.arg(id);

-- name: ListHeldLanes :many
SELECT lane FROM moves_unit WHERE phase IN ('transfer', 'place') ORDER BY lane;

-- name: NextHead :one
SELECT * FROM moves_unit
WHERE phase = 'queued' AND (waiting IS NULL OR waiting IN ('pair_busy', 'node_offline')) AND lane > sqlc.arg(after)
ORDER BY lane, seq LIMIT 1;

-- name: HeadOf :one
SELECT * FROM moves_unit
WHERE phase = 'queued' AND (waiting IS NULL OR waiting IN ('pair_busy', 'node_offline')) AND lane = sqlc.arg(lane)
ORDER BY lane, seq LIMIT 1;

-- name: ListDue :many
SELECT * FROM moves_unit WHERE due_at IS NOT NULL AND due_at <= sqlc.arg(now) ORDER BY due_at, seq LIMIT sqlc.arg(n);

-- name: NextDue :one
SELECT CAST(coalesce(min(due_at), 0) AS INTEGER) FROM moves_unit WHERE due_at IS NOT NULL;

-- name: MarkOpenDue :exec
UPDATE moves_unit SET due_at = sqlc.arg(now)
WHERE phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup') OR (phase IN ('failed', 'canceled') AND undone_at IS NULL);

-- name: Wake :exec
UPDATE moves_unit SET due_at = sqlc.arg(now) WHERE waiting = sqlc.arg(waiting) AND (due_at IS NULL OR due_at > sqlc.arg(now));

-- name: WakeWorks :exec
UPDATE moves_unit SET due_at = sqlc.arg(now)
WHERE waiting = sqlc.arg(waiting) AND (due_at IS NULL OR due_at > sqlc.arg(now)) AND work_id IN (sqlc.slice(works));

-- name: WakeTorrents :exec
UPDATE moves_unit SET due_at = sqlc.arg(now)
WHERE waiting = sqlc.arg(waiting) AND (due_at IS NULL OR due_at > sqlc.arg(now))
    AND (torrent_id IN (sqlc.slice(sources)) OR dest_torrent_id IN (sqlc.slice(dests)));

-- name: ListMovingTorrents :many
-- A torrent moves from its unit's insert until its switch, and while a
-- failed unit keeps its copy.
SELECT * FROM moves_unit
WHERE torrent_id IN (sqlc.slice(torrents)) AND (phase IN ('queued', 'transfer', 'place', 'switch') OR keeps_copy = 1)
ORDER BY seq;

-- name: CountClientUnits :one
-- A unit uses its clients until it ends, and a failed one while it keeps
-- its copy.
SELECT count(*) FROM moves_unit
WHERE kind = 'torrent' AND (from_client_id = sqlc.arg(client) OR to_client_id = sqlc.arg(client))
    AND (phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup') OR keeps_copy = 1
        OR (phase IN ('failed', 'canceled') AND undone_at IS NULL));

-- name: ListOpenUnitsOfTorrents :many
SELECT * FROM moves_unit
WHERE torrent_id IN (sqlc.slice(torrents)) AND (phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup') OR keeps_copy = 1)
ORDER BY seq;

-- name: WakeNodes :exec
UPDATE moves_unit SET due_at = sqlc.arg(now)
WHERE waiting = sqlc.arg(waiting) AND (due_at IS NULL OR due_at > sqlc.arg(now))
    AND (from_node_id IN (sqlc.slice(froms)) OR to_node_id IN (sqlc.slice(tos)));

-- name: WakeFromRoots :exec
UPDATE moves_unit SET due_at = sqlc.arg(now)
WHERE waiting = sqlc.arg(waiting) AND (due_at IS NULL OR due_at > sqlc.arg(now)) AND from_root_id IN (sqlc.slice(roots));

-- name: WakeTransfers :exec
UPDATE moves_unit SET due_at = sqlc.arg(now) WHERE phase = 'transfer' AND (due_at IS NULL OR due_at > sqlc.arg(now));

-- name: MarkReconcile :exec
UPDATE moves_unit SET reconcile = 1, due_at = NULL WHERE phase IN ('queued', 'transfer', 'place', 'switch');

-- name: WakeReconcile :exec
UPDATE moves_unit SET due_at = sqlc.arg(now) WHERE reconcile = 1;

-- name: CountBusyRoot :one
SELECT count(*) FROM moves_unit
WHERE (from_root_id = sqlc.arg(root) OR to_root_id = sqlc.arg(root))
    AND (phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup') OR (phase IN ('failed', 'canceled') AND undone_at IS NULL));

-- name: HeldWorks :many
SELECT work_id FROM moves_unit WHERE work_id IN (sqlc.slice(works)) AND phase IN ('queued', 'transfer', 'place', 'switch') ORDER BY work_id;

-- name: ListNeeds :many
SELECT * FROM moves_unit
WHERE phase = 'failed' OR (phase = 'switch' AND waiting IS NOT NULL AND waiting_at <= sqlc.arg(before))
ORDER BY seq DESC LIMIT sqlc.arg(n);

-- name: InsertUnitFile :exec
INSERT INTO moves_unit_file (unit_id, position, kind, via, file_id, sits_rel, payload_index, size, modified_at, node_file_id, node_volume_id, ledger)
VALUES (sqlc.arg(unit_id), sqlc.arg(position), sqlc.arg(kind), sqlc.arg(via), sqlc.narg(file_id), sqlc.arg(sits_rel), sqlc.narg(payload_index),
    sqlc.arg(size), sqlc.arg(modified_at), sqlc.narg(node_file_id), sqlc.narg(node_volume_id), sqlc.arg(ledger));

-- name: ReleasePair :exec
-- A torrent unit's Place ended: the Work unit placing from its payload
-- takes its lane next.
UPDATE moves_unit SET waiting = NULL, waiting_at = NULL, due_at = sqlc.arg(now)
WHERE pairs_with_unit_id = sqlc.arg(torrent_unit) AND waiting = 'linked_torrent';

-- name: UnpairFiles :exec
-- A torrent unit failed or was canceled: each file its Work unit hasn't
-- placed from the payload comes from the library instead.
UPDATE moves_unit_file SET via = 'library', payload_index = NULL
WHERE unit_id IN (SELECT id FROM moves_unit WHERE pairs_with_unit_id = sqlc.arg(torrent_unit)) AND via = 'arriving' AND placed = 0;

-- name: Unpair :exec
-- A Work unit past its Place placed every file it takes from the payload
-- and keeps its pair.
UPDATE moves_unit SET pairs_with_unit_id = NULL, due_at = sqlc.arg(now),
    waiting = CASE WHEN waiting = 'linked_torrent' THEN NULL ELSE waiting END,
    waiting_at = CASE WHEN waiting = 'linked_torrent' THEN NULL ELSE waiting_at END
WHERE pairs_with_unit_id = sqlc.arg(torrent_unit) AND phase IN ('queued', 'transfer', 'place');

-- name: ListUnitFiles :many
SELECT * FROM moves_unit_file WHERE unit_id IN (sqlc.slice(units)) ORDER BY unit_id, position;

-- name: DeleteUnitFiles :exec
DELETE FROM moves_unit_file WHERE unit_id = sqlc.arg(unit_id);

-- name: DeleteUnitFile :exec
DELETE FROM moves_unit_file WHERE unit_id = sqlc.arg(unit_id) AND position = sqlc.arg(position);

-- name: UpdateUnitFile :exec
UPDATE moves_unit_file SET size = sqlc.arg(size), modified_at = sqlc.arg(modified_at), node_file_id = sqlc.narg(node_file_id),
    node_volume_id = sqlc.narg(node_volume_id), placed = sqlc.arg(placed), placed_modified_at = sqlc.arg(placed_modified_at),
    placed_node_file_id = sqlc.narg(placed_node_file_id), placed_node_volume_id = sqlc.narg(placed_node_volume_id),
    copy_bytes = sqlc.arg(copy_bytes), dropped_at = sqlc.arg(dropped_at), cleanup = sqlc.narg(cleanup)
WHERE unit_id = sqlc.arg(unit_id) AND position = sqlc.arg(position);

-- name: ListReservedTo :many
SELECT f.unit_id, f.position, f.sits_rel, u.to_root_id FROM moves_unit_file f JOIN moves_unit u ON u.id = f.unit_id
WHERE u.to_root_id IN (sqlc.slice(roots))
    AND (u.phase IN ('queued', 'transfer', 'place', 'switch') OR (u.phase IN ('failed', 'canceled') AND u.undone_at IS NULL))
ORDER BY f.unit_id, f.position;

-- name: ListReservedFrom :many
SELECT f.sits_rel, u.from_root_id FROM moves_unit_file f JOIN moves_unit u ON u.id = f.unit_id
WHERE u.from_root_id IN (sqlc.slice(roots)) AND f.cleanup = 'pending'
ORDER BY f.unit_id, f.position;

-- name: ListCopyGrants :many
SELECT u.to_node_id, u.lane, CAST(sum(f.copy_bytes) AS INTEGER) AS bytes FROM moves_unit_file f JOIN moves_unit u ON u.id = f.unit_id
WHERE f.copy_bytes > 0 AND u.to_node_id = sqlc.arg(node)
GROUP BY u.to_node_id, u.lane ORDER BY u.lane;

-- name: GetSettings :one
SELECT needs_you_after_minutes, plex_bound_minutes FROM moves_settings WHERE id = 1;

-- name: SetSettings :exec
UPDATE moves_settings SET needs_you_after_minutes = sqlc.narg(needs_you_after_minutes), plex_bound_minutes = sqlc.narg(plex_bound_minutes) WHERE id = 1;

-- name: LaneHeld :one
SELECT EXISTS (SELECT 1 FROM moves_unit WHERE lane = sqlc.arg(lane) AND phase IN ('transfer', 'place'));

-- name: ListOpenUnitsBeforeSwitch :many
SELECT * FROM moves_unit WHERE work_id IN (sqlc.slice(works)) AND phase IN ('queued', 'transfer', 'place', 'switch') ORDER BY seq;

-- name: ListEmptyingBatches :many
-- The unfinished batches that delete the roots they empty, once every
-- unit has ended.
SELECT b.id FROM moves_batch b
WHERE b.finished_at IS NULL AND b.delete_emptied_roots = 1 AND NOT EXISTS (
    SELECT 1 FROM moves_unit u WHERE u.batch_id = b.id AND u.phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup')
)
ORDER BY b.created_at;

-- name: FinishBatchKeeping :exec
UPDATE moves_batch SET finished_at = sqlc.arg(at), notes = json_set(notes, '$.kept', json(CAST(sqlc.arg(kept) AS TEXT)))
WHERE id = sqlc.arg(id) AND finished_at IS NULL;
