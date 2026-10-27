-- name: InsertGrab :exec
INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, tracker_id, node_id, release_title, release_size,
    release_link_sealed, release_magnet, release_info_hash, fetch_query_id, min_ratio, min_seed_seconds, download_factor,
    upload_factor, stage, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(edition), 'search', sqlc.arg(indexer_id), sqlc.arg(tracker_id),
    sqlc.arg(node_id),
    sqlc.arg(release_title), sqlc.arg(release_size), sqlc.arg(release_link_sealed), sqlc.narg(release_magnet),
    sqlc.narg(release_info_hash), sqlc.narg(fetch_query_id), sqlc.arg(min_ratio), sqlc.arg(min_seed_seconds),
    sqlc.arg(download_factor), sqlc.arg(upload_factor), 'fetching', sqlc.arg(created_at), sqlc.arg(created_at));

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
INSERT INTO grabs_search (id, work_id, item_id, started_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(started_at));

-- name: InsertSearchIndexer :exec
INSERT INTO grabs_search_indexer (search_id, position, indexer_id, indexer_name, query_id, outcome, releases, until_at, since_at,
    node_id, lapsed, cause, text)
VALUES (sqlc.arg(search_id), sqlc.arg(position), sqlc.arg(indexer_id), sqlc.arg(indexer_name), sqlc.narg(query_id), sqlc.narg(outcome),
    sqlc.narg(releases), sqlc.narg(until_at), sqlc.narg(since_at), sqlc.narg(node_id), sqlc.narg(lapsed), sqlc.narg(cause), sqlc.narg(text));

-- name: ListSearchIndexersOfSearch :many
SELECT * FROM grabs_search_indexer WHERE search_id = sqlc.arg(search_id) ORDER BY position;

-- name: AnswerSearchIndexer :exec
UPDATE grabs_search_indexer SET outcome = sqlc.arg(outcome), releases = sqlc.narg(releases), until_at = sqlc.narg(until_at),
    since_at = sqlc.narg(since_at), node_id = sqlc.narg(node_id), lapsed = sqlc.narg(lapsed), cause = sqlc.narg(cause),
    text = sqlc.narg(text)
WHERE search_id = sqlc.arg(search_id) AND position = sqlc.arg(position) AND outcome IS NULL;

-- name: ListSearchIndexersOfWork :many
SELECT grabs_search_indexer.* FROM grabs_search_indexer
JOIN grabs_search ON grabs_search.id = grabs_search_indexer.search_id
WHERE grabs_search.work_id = sqlc.arg(work_id)
ORDER BY grabs_search_indexer.search_id, grabs_search_indexer.position;

-- name: FinishSearch :exec
UPDATE grabs_search SET finished_at = sqlc.arg(finished_at), outcome = sqlc.arg(outcome), candidates = sqlc.arg(candidates),
    grab_id = sqlc.narg(grab_id), notes = sqlc.narg(notes)
WHERE id = sqlc.arg(id) AND finished_at IS NULL;

-- name: Redecide :exec
UPDATE grabs_search SET outcome = sqlc.arg(outcome), grab_id = sqlc.narg(grab_id), notes = sqlc.narg(notes)
WHERE id = sqlc.arg(id);

-- name: GetSearchOfGrab :one
SELECT * FROM grabs_search WHERE grab_id = sqlc.arg(grab_id);

-- name: InsertRelease :exec
INSERT INTO grabs_release (search_id, position, indexer_id, release, link_sealed)
VALUES (sqlc.arg(search_id), sqlc.arg(position), sqlc.arg(indexer_id), sqlc.arg(release), sqlc.arg(link_sealed));

-- name: ListReleasesOfSearch :many
SELECT * FROM grabs_release WHERE search_id = sqlc.arg(search_id) ORDER BY position;

-- name: DeleteRelease :exec
DELETE FROM grabs_release WHERE search_id = sqlc.arg(search_id) AND position = sqlc.arg(position);

-- name: DeleteReleasesOfGrab :exec
DELETE FROM grabs_release WHERE search_id IN (SELECT id FROM grabs_search WHERE grab_id = sqlc.arg(grab_id));

-- name: DeleteReleasesOfSearch :exec
DELETE FROM grabs_release WHERE search_id = sqlc.arg(search_id);

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

-- name: RepointTracker :exec
UPDATE grabs_grab SET tracker_id = sqlc.arg(into_id) WHERE tracker_id = sqlc.arg(from_id);
