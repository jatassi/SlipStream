-- name: ListUnsettledTargets :many
SELECT item_id, target_id FROM grabs_target WHERE grab_id = sqlc.arg(grab_id) AND settled IS NULL ORDER BY position;

-- name: ListPlacedTargets :many
SELECT item_id, target_id FROM grabs_target WHERE grab_id = sqlc.arg(grab_id) AND settled = 'placed' ORDER BY position;

-- name: InsertFailure :execrows
INSERT INTO grabs_failure (grab_id, work_id, item_id, edition, target_id, code, info_hash, indexer_id, guid, failed_at, counted, refuses)
VALUES (sqlc.arg(grab_id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(edition), sqlc.arg(target_id), sqlc.arg(code),
    sqlc.narg(info_hash), sqlc.narg(indexer_id), sqlc.narg(guid), sqlc.arg(failed_at), sqlc.arg(counted), sqlc.arg(refuses))
ON CONFLICT DO NOTHING;

-- name: CountFailuresSince :one
SELECT count(*) FROM grabs_failure
WHERE item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id)
    AND counted = 1 AND failed_at >= sqlc.arg(since);

-- name: StopAt :exec
UPDATE grabs_failure SET stopped_at = sqlc.arg(stopped_at)
WHERE grab_id = sqlc.arg(grab_id) AND item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id);

-- name: CellStopped :one
SELECT EXISTS (
    SELECT 1 FROM grabs_failure
    WHERE item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id) AND stopped_at IS NOT NULL
);

-- name: Uncount :exec
UPDATE grabs_failure SET counted = 0, stopped_at = NULL
WHERE item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id) AND counted = 1;

-- name: DeleteFailuresOf :exec
DELETE FROM grabs_failure WHERE item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id);

-- name: ListTried :many
-- The Releases the Work's failure rows refuse.
SELECT DISTINCT info_hash, indexer_id, guid FROM grabs_failure WHERE work_id = sqlc.arg(work_id) AND refuses = 1
ORDER BY info_hash, indexer_id, guid;

-- name: ListStopped :many
SELECT DISTINCT item_id, edition, target_id FROM grabs_failure WHERE stopped_at IS NOT NULL ORDER BY item_id, edition, target_id;

-- name: ListStoppedOfWorks :many
SELECT DISTINCT item_id, edition, target_id FROM grabs_failure WHERE stopped_at IS NOT NULL AND work_id IN (sqlc.slice(work_ids))
ORDER BY item_id, edition, target_id;
