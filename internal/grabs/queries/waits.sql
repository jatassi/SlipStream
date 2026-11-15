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
