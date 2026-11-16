-- name: UpsertWait :exec
-- A cell already waiting keeps its start and waits until the earlier of
-- the two times.
INSERT INTO grabs_wait (work_id, item_id, edition, target_id, until_at, started_at)
VALUES (sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(edition), sqlc.arg(target_id), sqlc.arg(until_at), sqlc.arg(started_at))
ON CONFLICT (item_id, edition, target_id) DO UPDATE SET until_at = MIN(until_at, excluded.until_at);

-- name: InsertWaitCandidate :exec
INSERT INTO grabs_wait_candidate (item_id, edition, target_id, search_id, position)
VALUES (sqlc.arg(item_id), sqlc.arg(edition), sqlc.arg(target_id), sqlc.arg(search_id), sqlc.arg(position))
ON CONFLICT DO NOTHING;

-- name: DeleteWait :exec
DELETE FROM grabs_wait WHERE item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id);

-- name: ListWaitsOfWork :many
SELECT * FROM grabs_wait WHERE work_id = sqlc.arg(work_id) ORDER BY until_at, item_id, edition, target_id;

-- name: ListWaitCandidatesOfWork :many
-- Each candidate of work's waits, by the cell it waits at, with its
-- Release row.
SELECT c.item_id, c.edition, c.target_id, sqlc.embed(r)
FROM grabs_wait w
JOIN grabs_wait_candidate c ON c.item_id = w.item_id AND c.edition = w.edition AND c.target_id = w.target_id
JOIN grabs_release r ON r.search_id = c.search_id AND r.position = c.position
WHERE w.work_id = sqlc.arg(work_id)
ORDER BY r.search_id, r.position, c.item_id, c.edition, c.target_id;

-- name: ListWorksWithWaitsEnding :many
-- The Works with a wait that ends at or before now.
SELECT DISTINCT work_id FROM grabs_wait WHERE until_at <= sqlc.arg(now) ORDER BY work_id;

-- name: NextWaitEnd :one
SELECT until_at FROM grabs_wait WHERE until_at > sqlc.arg(now) ORDER BY until_at LIMIT 1;

-- name: GrabWaitingRelease :exec
-- A candidate a wait's end grabbed.
UPDATE grabs_release SET outcome = 'grabbed' WHERE search_id = sqlc.arg(search_id) AND position = sqlc.arg(position);

-- name: KeepEndedCandidates :exec
-- A waiting Release of runs that no wait holds any more ranked, and was
-- not grabbed.
UPDATE grabs_release SET outcome = 'kept'
WHERE grabs_release.search_id IN (sqlc.slice(search_ids)) AND grabs_release.outcome = 'waiting'
    AND NOT EXISTS (
        SELECT 1 FROM grabs_wait_candidate c WHERE c.search_id = grabs_release.search_id AND c.position = grabs_release.position);

-- name: AddWaitGrab :exec
-- A run whose Release a wait's end grabbed took one more Grab.
UPDATE grabs_search SET outcome = 'grabbed', grabs = grabs + 1, notes = coalesce(notes, sqlc.narg(notes)) WHERE id = sqlc.arg(id);
