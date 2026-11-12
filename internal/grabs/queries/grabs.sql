-- name: InsertGrab :exec
INSERT INTO grabs_grab (id, work_id, item_id, edition, source, manual_id, grabbed_by, grabbed_by_name, overrides, indexer_id,
    tracker_id, node_id, release_title, release_size, release_link_sealed, release_magnet, release_info_hash, fetch_query_id,
    min_ratio, min_seed_seconds, download_factor, upload_factor, stage, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(edition), sqlc.arg(source), sqlc.arg(manual_id),
    sqlc.arg(grabbed_by), sqlc.narg(grabbed_by_name), sqlc.arg(overrides), sqlc.arg(indexer_id), sqlc.arg(tracker_id),
    sqlc.arg(node_id),
    sqlc.arg(release_title), sqlc.arg(release_size), sqlc.arg(release_link_sealed), sqlc.narg(release_magnet),
    sqlc.narg(release_info_hash), sqlc.narg(fetch_query_id), sqlc.arg(min_ratio), sqlc.arg(min_seed_seconds),
    sqlc.arg(download_factor), sqlc.arg(upload_factor), 'fetching', sqlc.arg(created_at), sqlc.arg(created_at));

-- name: InsertGrabItem :exec
INSERT INTO grabs_grab_item (grab_id, item_id, position) VALUES (sqlc.arg(grab_id), sqlc.arg(item_id), sqlc.arg(position));

-- name: InsertTarget :exec
INSERT INTO grabs_target (grab_id, position, item_id, target_id, forced)
VALUES (sqlc.arg(grab_id), sqlc.arg(position), sqlc.arg(item_id), sqlc.arg(target_id), sqlc.arg(forced));

-- name: InsertStage :exec
INSERT INTO grabs_stage (grab_id, stage, entered_at) VALUES (sqlc.arg(grab_id), sqlc.arg(stage), sqlc.arg(entered_at))
ON CONFLICT (grab_id, stage) DO UPDATE SET entered_at = excluded.entered_at;

-- name: Advance :execrows
UPDATE grabs_grab SET
    stage = sqlc.arg(next),
    torrent_id = coalesce(sqlc.narg(torrent_id), torrent_id),
    fetch_node_id = coalesce(sqlc.narg(fetch_node_id), fetch_node_id),
    node_id = coalesce(sqlc.narg(node_id), node_id),
    failed_stage = sqlc.narg(failed_stage),
    failure_code = sqlc.narg(failure_code),
    failure_message = sqlc.narg(failure_message),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND stage = sqlc.arg(stage);

-- name: ListGrabs :many
SELECT * FROM grabs_grab ORDER BY created_at DESC, id DESC;

-- name: ListGrabsOfIndexer :many
SELECT * FROM grabs_grab WHERE indexer_id = sqlc.arg(indexer_id) ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: ListOpenGrabs :many
SELECT * FROM grabs_grab WHERE stage NOT IN ('imported', 'failed') ORDER BY created_at, id;

-- name: CountMoving :one
SELECT COUNT(*) FROM grabs_grab WHERE stage IN (sqlc.slice(stages));

-- name: GetGrab :one
SELECT * FROM grabs_grab WHERE id = sqlc.arg(id);

-- name: ListTargets :many
SELECT * FROM grabs_target ORDER BY grab_id, position;

-- name: ListGrabItems :many
SELECT * FROM grabs_grab_item ORDER BY grab_id, position;

-- name: ListStages :many
SELECT * FROM grabs_stage ORDER BY grab_id, entered_at, rowid;

-- name: ListGrabsOfTorrents :many
SELECT id, torrent_id, work_id, item_id, edition, release_title, stage, min_ratio, min_seed_seconds, created_at FROM grabs_grab
WHERE torrent_id IN (sqlc.slice(torrent_ids))
ORDER BY created_at, id;

-- name: ListItemsOfTorrents :many
SELECT i.grab_id, i.item_id FROM grabs_grab_item i JOIN grabs_grab g ON g.id = i.grab_id
WHERE g.torrent_id IN (sqlc.slice(torrent_ids))
ORDER BY i.grab_id, i.position;

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

-- name: AnswerSearchIndexer :execrows
UPDATE grabs_search_indexer SET outcome = sqlc.arg(outcome), releases = sqlc.narg(releases), until_at = sqlc.narg(until_at),
    since_at = sqlc.narg(since_at), node_id = sqlc.narg(node_id), lapsed = sqlc.narg(lapsed), cause = sqlc.narg(cause),
    text = sqlc.narg(text)
WHERE search_id = sqlc.arg(search_id) AND position = sqlc.arg(position) AND outcome IS NULL;

-- name: ListSearchIndexersOfWork :many
SELECT grabs_search_indexer.* FROM grabs_search_indexer
JOIN grabs_search ON grabs_search.id = grabs_search_indexer.search_id
WHERE grabs_search.work_id = sqlc.arg(work_id) AND grabs_search.manual_id IS NULL
ORDER BY grabs_search_indexer.search_id, grabs_search_indexer.position;

-- name: FinishSearch :exec
UPDATE grabs_search SET finished_at = sqlc.arg(finished_at), outcome = sqlc.arg(outcome), candidates = sqlc.arg(candidates),
    grab_id = sqlc.narg(grab_id), notes = sqlc.narg(notes)
WHERE id = sqlc.arg(id) AND finished_at IS NULL;

-- name: Redecide :exec
UPDATE grabs_search SET outcome = sqlc.arg(outcome), grab_id = sqlc.narg(grab_id), notes = sqlc.narg(notes)
WHERE id = sqlc.arg(id);

-- name: GetSearch :one
SELECT * FROM grabs_search WHERE id = sqlc.arg(id);

-- name: GetSearchOfGrab :one
SELECT * FROM grabs_search WHERE grab_id = sqlc.arg(grab_id);

-- name: InsertRelease :exec
INSERT INTO grabs_release (search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, rank, reasons, lost_step,
    lost_target_id, lost_own, lost_ref, scores, formats, covers, copy_of, release, link_sealed)
VALUES (sqlc.arg(search_id), sqlc.arg(position), sqlc.arg(title), sqlc.narg(indexer_id), sqlc.arg(indexer_name), sqlc.arg(size),
    sqlc.arg(seeders), sqlc.arg(outcome), sqlc.narg(rank), sqlc.arg(reasons), sqlc.narg(lost_step), sqlc.narg(lost_target_id),
    sqlc.narg(lost_own), sqlc.narg(lost_ref), sqlc.arg(scores), sqlc.arg(formats), sqlc.arg(covers), sqlc.narg(copy_of),
    sqlc.narg(release), sqlc.narg(link_sealed));

-- name: ListReleasesOfSearch :many
SELECT * FROM grabs_release WHERE search_id = sqlc.arg(search_id) ORDER BY rank IS NULL, rank, position;

-- name: ListKeptReleasesOfSearch :many
SELECT * FROM grabs_release WHERE search_id = sqlc.arg(search_id) AND release IS NOT NULL ORDER BY position;

-- name: EndReleasesOfGrab :exec
-- No rerun follows once a run's Grab leaves fetching other than by a drop.
UPDATE grabs_release SET release = NULL, link_sealed = NULL
WHERE search_id IN (SELECT id FROM grabs_search WHERE grab_id = sqlc.arg(grab_id));

-- name: DeleteReleasesOfSearch :exec
DELETE FROM grabs_release WHERE search_id = sqlc.arg(search_id);

-- name: DeleteOlderReleasesOfItem :exec
-- Finishing a run deletes the rows of its Item's other runs but those whose
-- Grab still fetches, which a rerun reads.
DELETE FROM grabs_release WHERE search_id IN (
    SELECT grabs_search.id FROM grabs_search
    LEFT JOIN grabs_grab ON grabs_grab.id = grabs_search.grab_id
    WHERE grabs_search.item_id = sqlc.arg(item_id) AND grabs_search.id <> sqlc.arg(search_id) AND grabs_search.manual_id IS NULL
        AND (grabs_grab.stage IS NULL OR grabs_grab.stage <> 'fetching'));

-- name: ListOpenSearches :many
SELECT * FROM grabs_search WHERE finished_at IS NULL ORDER BY started_at, id;

-- name: ListSearchesOfWork :many
SELECT * FROM grabs_search WHERE work_id = sqlc.arg(work_id) AND manual_id IS NULL ORDER BY started_at DESC, id DESC;

-- name: ListTargetsOfWorks :many
SELECT grabs_grab.id, grabs_target.item_id, grabs_grab.edition, grabs_grab.stage, grabs_grab.node_id, grabs_grab.failure_code,
    grabs_grab.failed_stage, grabs_grab.source, grabs_target.target_id, grabs_target.settled
FROM grabs_grab JOIN grabs_target ON grabs_target.grab_id = grabs_grab.id
WHERE grabs_grab.work_id IN (sqlc.slice(work_ids))
ORDER BY grabs_grab.created_at, grabs_grab.id, grabs_target.position;

-- name: ListEveryGrabTarget :many
SELECT grabs_grab.id, grabs_target.item_id, grabs_grab.edition, grabs_grab.stage, grabs_grab.node_id, grabs_grab.failure_code,
    grabs_grab.failed_stage, grabs_grab.source, grabs_target.target_id, grabs_target.settled
FROM grabs_grab JOIN grabs_target ON grabs_target.grab_id = grabs_grab.id
ORDER BY grabs_grab.created_at, grabs_grab.id, grabs_target.position;

-- name: StartMetadataSearch :one
UPDATE grabs_grab SET metadata_started_at = coalesce(metadata_started_at, sqlc.arg(at))
WHERE id = sqlc.arg(id) AND stage = 'fetching'
RETURNING metadata_started_at;

-- name: RepointTracker :exec
UPDATE grabs_grab SET tracker_id = sqlc.arg(into_id) WHERE tracker_id = sqlc.arg(from_id);

-- name: CountUnfinishedOfTracker :one
SELECT count(*) FROM grabs_grab WHERE tracker_id = sqlc.arg(tracker_id) AND stage NOT IN ('imported', 'failed');

-- name: ForgetTracker :exec
UPDATE grabs_grab SET tracker_id = NULL WHERE tracker_id = sqlc.arg(tracker_id);

-- name: ListDownloading :many
SELECT id, work_id FROM grabs_grab WHERE torrent_id = sqlc.arg(torrent_id) AND stage = 'downloading' ORDER BY id;

-- name: Repoint :exec
-- send-to moves every Grab downloading a Torrent with it.
UPDATE grabs_grab SET torrent_id = sqlc.arg(to_id), node_id = sqlc.arg(node_id), updated_at = sqlc.arg(updated_at)
WHERE torrent_id = sqlc.arg(from_id) AND stage = 'downloading';

-- name: ListReleaseTitles :many
SELECT id, release_title FROM grabs_grab ORDER BY id;

-- name: SetSent :exec
UPDATE grabs_grab SET sent_built_in_on = sqlc.narg(built_in_on), sent_client_id = sqlc.narg(client_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: SettleTarget :execrows
UPDATE grabs_target SET settled = sqlc.arg(settled), settled_at = sqlc.arg(settled_at)
WHERE grab_id = sqlc.arg(grab_id) AND item_id = sqlc.arg(item_id) AND target_id = sqlc.arg(target_id) AND settled IS NULL;

-- name: ListCarried :many
SELECT * FROM grabs_carry ORDER BY grab_id;

-- name: DeleteCarried :exec
DELETE FROM grabs_carry WHERE grab_id = sqlc.arg(grab_id);
