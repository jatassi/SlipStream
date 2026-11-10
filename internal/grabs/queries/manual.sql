-- name: InsertManual :exec
INSERT INTO grabs_manual (id, work_id, scope, query, started_by, started_by_name, started_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(scope), sqlc.narg(query), sqlc.arg(started_by), sqlc.narg(started_by_name),
    sqlc.arg(started_at));

-- name: GetManual :one
SELECT * FROM grabs_manual WHERE id = sqlc.arg(id);

-- name: GetManualOfWork :one
SELECT * FROM grabs_manual WHERE work_id = sqlc.arg(work_id);

-- name: ListRunsOfManual :many
SELECT id FROM grabs_search WHERE manual_id = sqlc.arg(manual_id) ORDER BY started_at, id;

-- name: DeleteManual :exec
-- Its runs, their lines and their Releases go by cascade; its Grabs keep
-- a null manual_id.
DELETE FROM grabs_manual WHERE id = sqlc.arg(id);

-- name: InsertManualRun :exec
INSERT INTO grabs_search (id, work_id, item_id, manual_id, started_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(manual_id), sqlc.arg(started_at));

-- name: NextReleasePosition :one
SELECT CAST(coalesce(max(position) + 1, 0) AS INTEGER) FROM grabs_release WHERE search_id = sqlc.arg(search_id);

-- name: InsertListed :exec
INSERT INTO grabs_release (search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, reasons, scores, formats,
    covers, release, link_sealed, read)
VALUES (sqlc.arg(search_id), sqlc.arg(position), sqlc.arg(title), sqlc.narg(indexer_id), sqlc.arg(indexer_name), sqlc.arg(size),
    sqlc.arg(seeders), 'listed', '[]', '[]', sqlc.arg(formats), '[]', sqlc.arg(release), sqlc.arg(link_sealed), sqlc.arg(read));

-- name: FinishHeardSearch :exec
-- A Manual run ends once every line it planned has an outcome.
UPDATE grabs_search SET finished_at = sqlc.arg(finished_at), outcome = 'listed',
    candidates = (SELECT count(*) FROM grabs_release WHERE grabs_release.search_id = grabs_search.id)
WHERE id = sqlc.arg(id) AND finished_at IS NULL
    AND NOT EXISTS (SELECT 1 FROM grabs_search_indexer
        WHERE grabs_search_indexer.search_id = grabs_search.id AND grabs_search_indexer.outcome IS NULL);

-- name: ListManualRuns :many
SELECT * FROM grabs_search WHERE manual_id = sqlc.arg(manual_id) ORDER BY started_at, id;

-- name: ListLinesOfManual :many
SELECT grabs_search_indexer.* FROM grabs_search_indexer JOIN grabs_search ON grabs_search.id = grabs_search_indexer.search_id
WHERE grabs_search.manual_id = sqlc.arg(manual_id)
ORDER BY grabs_search_indexer.search_id, grabs_search_indexer.position;

-- name: CountKeptOfManual :many
-- Each run's count of kept Releases, which says whether the view's decoded
-- copy of the run is current: a run's rows are only ever appended.
SELECT grabs_release.search_id, count(*) AS kept FROM grabs_release JOIN grabs_search ON grabs_search.id = grabs_release.search_id
WHERE grabs_search.manual_id = sqlc.arg(manual_id) AND grabs_release.release IS NOT NULL
GROUP BY grabs_release.search_id;

-- name: ListCellGrabsOfWork :many
-- Each Grab of the Work at each of its Targets, oldest first, so the
-- last of a cell is its latest.
SELECT grabs_grab.id, grabs_grab.item_id, grabs_grab.edition, grabs_target.target_id, grabs_grab.stage, grabs_grab.source,
    grabs_grab.release_title, grabs_grab.failure_code, grabs_grab.failure_message
FROM grabs_grab JOIN grabs_target ON grabs_target.grab_id = grabs_grab.id
WHERE grabs_grab.work_id = sqlc.arg(work_id)
ORDER BY grabs_grab.created_at, grabs_grab.id, grabs_target.position;
