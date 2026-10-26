-- name: InsertGrab :exec
INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size, release_link,
    release_info_hash, stage, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(edition), 'search', sqlc.arg(indexer_id), sqlc.arg(node_id),
    sqlc.arg(release_title), sqlc.arg(release_size), sqlc.arg(release_link), sqlc.narg(release_info_hash), 'fetching',
    sqlc.arg(created_at), sqlc.arg(created_at));

-- name: InsertTarget :exec
INSERT INTO grabs_target (grab_id, position, target_id) VALUES (sqlc.arg(grab_id), sqlc.arg(position), sqlc.arg(target_id));

-- name: InsertFill :exec
INSERT INTO grabs_fill (grab_id, position, target_id) VALUES (sqlc.arg(grab_id), sqlc.arg(position), sqlc.arg(target_id));

-- name: InsertStage :exec
INSERT INTO grabs_stage (grab_id, stage, entered_at) VALUES (sqlc.arg(grab_id), sqlc.arg(stage), sqlc.arg(entered_at));

-- name: Advance :execrows
UPDATE grabs_grab SET
    stage = sqlc.arg(next),
    torrent_id = coalesce(sqlc.narg(torrent_id), torrent_id),
    fetch_node_id = coalesce(sqlc.narg(fetch_node_id), fetch_node_id),
    node_id = coalesce(sqlc.narg(node_id), node_id),
    file_index = coalesce(sqlc.narg(file_index), file_index),
    probe = coalesce(sqlc.narg(probe), probe),
    root_id = coalesce(sqlc.narg(root_id), root_id),
    path = coalesce(sqlc.narg(path), path),
    failed_stage = sqlc.narg(failed_stage),
    failure_code = sqlc.narg(failure_code),
    failure_message = sqlc.narg(failure_message),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND stage = sqlc.arg(stage);

-- name: ListGrabs :many
SELECT * FROM grabs_grab ORDER BY created_at DESC, id DESC;

-- name: ListOpenGrabs :many
SELECT * FROM grabs_grab WHERE stage NOT IN ('imported', 'failed') ORDER BY created_at, id;

-- name: CountMoving :one
SELECT COUNT(*) FROM grabs_grab WHERE stage NOT IN ('imported', 'failed');

-- name: GetGrab :one
SELECT * FROM grabs_grab WHERE id = sqlc.arg(id);

-- name: ListTargets :many
SELECT * FROM grabs_target ORDER BY grab_id, position;

-- name: ListFills :many
SELECT * FROM grabs_fill ORDER BY grab_id, position;

-- name: ListStages :many
SELECT * FROM grabs_stage ORDER BY grab_id, entered_at, rowid;

-- name: ListGrabsOfTorrents :many
SELECT id, torrent_id, work_id, item_id, edition, release_title, stage, file_index, created_at FROM grabs_grab
WHERE torrent_id IN (sqlc.slice(torrent_ids))
ORDER BY created_at, id;

-- name: InsertSearch :exec
INSERT INTO grabs_search (id, work_id, item_id, indexer_id, started_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(indexer_id), sqlc.arg(started_at));

-- name: FinishSearch :exec
UPDATE grabs_search SET finished_at = sqlc.arg(finished_at), outcome = sqlc.arg(outcome), candidates = sqlc.arg(candidates),
    grab_id = sqlc.narg(grab_id), notes = sqlc.narg(notes)
WHERE id = sqlc.arg(id) AND finished_at IS NULL;

-- name: ListOpenSearches :many
SELECT * FROM grabs_search WHERE finished_at IS NULL ORDER BY started_at, id;

-- name: ListSearchesOfWork :many
SELECT * FROM grabs_search WHERE work_id = sqlc.arg(work_id) ORDER BY started_at DESC, id DESC;

-- name: InsertRefusal :exec
INSERT INTO grabs_refusal (search_id, position, title, reasons)
VALUES (sqlc.arg(search_id), sqlc.arg(position), sqlc.arg(title), sqlc.arg(reasons));

-- name: ListRefusalsOfWork :many
SELECT grabs_refusal.* FROM grabs_refusal
JOIN grabs_search ON grabs_search.id = grabs_refusal.search_id
WHERE grabs_search.work_id = sqlc.arg(work_id)
ORDER BY grabs_refusal.search_id, grabs_refusal.position;

-- name: ListTargetsOfWork :many
SELECT grabs_grab.id, grabs_grab.item_id, grabs_grab.edition, grabs_grab.stage, grabs_target.target_id
FROM grabs_grab JOIN grabs_target ON grabs_target.grab_id = grabs_grab.id
WHERE grabs_grab.work_id = sqlc.arg(work_id)
ORDER BY grabs_grab.created_at, grabs_grab.id, grabs_target.position;

-- name: ListTargetsOfWorks :many
SELECT grabs_grab.id, grabs_grab.item_id, grabs_grab.edition, grabs_grab.stage, grabs_target.target_id
FROM grabs_grab JOIN grabs_target ON grabs_target.grab_id = grabs_grab.id
WHERE grabs_grab.work_id IN (sqlc.slice(work_ids))
ORDER BY grabs_grab.created_at, grabs_grab.id, grabs_target.position;

-- name: SettleGrab :exec
UPDATE grabs_grab SET unwanted = sqlc.arg(unwanted) WHERE id = sqlc.arg(id);

-- name: GetUnwanted :one
SELECT unwanted FROM grabs_grab WHERE id = sqlc.arg(id);

-- name: CountGrabsInto :one
SELECT COUNT(*) FROM grabs_grab WHERE stage NOT IN ('imported', 'failed')
    AND (root_id = sqlc.arg(root_id) OR work_id IN (sqlc.slice(work_ids)));

-- name: ReleaseRoot :exec
UPDATE grabs_grab SET root_id = NULL WHERE root_id = sqlc.arg(root_id) AND stage IN ('imported', 'failed');

-- name: StartMetadataSearch :one
UPDATE grabs_grab SET metadata_started_at = coalesce(metadata_started_at, sqlc.arg(at))
WHERE id = sqlc.arg(id) AND stage = 'fetching'
RETURNING metadata_started_at;
