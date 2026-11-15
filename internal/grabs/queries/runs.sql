-- name: PromoteSearch :exec
-- A press promotes the open run it outranks.
UPDATE grabs_search SET lane = sqlc.arg(lane), trigger = sqlc.arg(trigger), include_gone = sqlc.arg(include_gone)
WHERE id = sqlc.arg(id) AND finished_at IS NULL;

-- name: ListReadyRuns :many
-- The open queue runs every line of which answered, was skipped or went
-- late.
SELECT * FROM grabs_search s
WHERE s.finished_at IS NULL AND s.manual_id IS NULL
    AND NOT EXISTS (SELECT 1 FROM grabs_search_indexer l WHERE l.search_id = s.id AND l.outcome IS NULL)
ORDER BY s.started_at, s.id;

-- name: InsertHeard :exec
INSERT INTO grabs_release (search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, reasons, scores, formats,
    covers, release, link_sealed, heard_at)
VALUES (sqlc.arg(search_id), sqlc.arg(position), sqlc.arg(title), sqlc.narg(indexer_id), sqlc.arg(indexer_name), sqlc.arg(size),
    sqlc.arg(seeders), 'heard', '[]', '[]', '[]', '[]', sqlc.arg(release), sqlc.arg(link_sealed), sqlc.arg(heard_at));

-- name: ListHeard :many
SELECT * FROM grabs_release WHERE search_id = sqlc.arg(search_id) AND outcome = 'heard' ORDER BY position;

-- name: DeleteHeard :exec
DELETE FROM grabs_release WHERE search_id = sqlc.arg(search_id) AND outcome = 'heard';

-- name: InsertSingleLine :exec
-- A line of phase two: one Item's own query.
INSERT INTO grabs_search_indexer (search_id, position, indexer_id, indexer_name, item_id)
VALUES (sqlc.arg(search_id), sqlc.arg(position), sqlc.arg(indexer_id), sqlc.arg(indexer_name), sqlc.arg(item_id));

-- name: StartPhaseTwo :exec
UPDATE grabs_search SET phase = 2 WHERE id = sqlc.arg(id) AND finished_at IS NULL;

-- name: ListSearchedItems :many
-- Of items, those an automatic run already searched.
SELECT item_id FROM grabs_item_search WHERE item_id IN (sqlc.slice(item_ids)) ORDER BY item_id;
