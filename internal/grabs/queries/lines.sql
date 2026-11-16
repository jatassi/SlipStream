-- name: ListPlannedIndexers :many
-- The Indexers with lines planned and not sent, NULL for a deleted one.
SELECT DISTINCT l.indexer_id FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id
WHERE l.outcome IS NULL AND l.query_id IS NULL AND s.finished_at IS NULL AND s.manual_id IS NULL
ORDER BY l.indexer_id
LIMIT sqlc.arg(n);

-- name: ListPendingLines :many
-- An Indexer's planned lines, its FIFO: presses first, then by when the
-- run started.
SELECT l.search_id, l.position, s.work_id, s.lane FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id
WHERE l.indexer_id IS sqlc.narg(indexer_id) AND l.outcome IS NULL AND l.query_id IS NULL AND s.finished_at IS NULL
    AND s.manual_id IS NULL
ORDER BY s.lane, s.started_at, l.search_id, l.position;

-- name: CountPlanned :many
SELECT l.indexer_id, count(*) AS planned FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id
WHERE l.outcome IS NULL AND l.query_id IS NULL AND s.finished_at IS NULL AND s.manual_id IS NULL
GROUP BY l.indexer_id
ORDER BY l.indexer_id;

-- name: DispatchLine :execrows
UPDATE grabs_search_indexer SET query_id = sqlc.arg(query_id), dispatched_at = sqlc.arg(dispatched_at), deadline_at = sqlc.arg(deadline_at)
WHERE search_id = sqlc.arg(search_id) AND position = sqlc.arg(position) AND outcome IS NULL AND query_id IS NULL;

-- name: DeletePlannedLine :exec
DELETE FROM grabs_search_indexer
WHERE search_id = sqlc.arg(search_id) AND position = sqlc.arg(position) AND outcome IS NULL AND query_id IS NULL;

-- name: ListSentLines :many
-- Every line sent and not answered, of the open queue runs.
SELECT l.search_id, l.position, l.indexer_id, l.indexer_name, l.item_id, l.query_id, l.deadline_at, s.work_id
FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id
WHERE l.outcome IS NULL AND l.query_id IS NOT NULL AND s.finished_at IS NULL AND s.manual_id IS NULL
ORDER BY l.search_id, l.position;
