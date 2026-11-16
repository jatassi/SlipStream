-- name: ListEveryDue :many
SELECT work_id, group_key, lane, trigger, due_at, search_id FROM grabs_due ORDER BY work_id, group_key;

-- name: ListDueOfWorks :many
SELECT work_id, group_key, lane, trigger, due_at, search_id FROM grabs_due WHERE work_id IN (sqlc.slice(work_ids))
ORDER BY work_id, group_key;

-- name: ListRunItems :many
-- Each Item a finished queue run searched, oldest run first. The run
-- bound keeps at most one such run per group and Item.
SELECT s.work_id, s.group_key, si.item_id, s.finished_at, s.outcome, s.candidates, s.grabs
FROM grabs_search s JOIN grabs_search_item si ON si.search_id = s.id
WHERE s.manual_id IS NULL AND s.group_key IS NOT NULL AND s.finished_at IS NOT NULL
ORDER BY s.finished_at, s.id, si.item_id;

-- name: ListRunItemsOfWorks :many
SELECT s.work_id, s.group_key, si.item_id, s.finished_at, s.outcome, s.candidates, s.grabs
FROM grabs_search s JOIN grabs_search_item si ON si.search_id = s.id
WHERE s.work_id IN (sqlc.slice(work_ids)) AND s.manual_id IS NULL AND s.group_key IS NOT NULL AND s.finished_at IS NOT NULL
ORDER BY s.finished_at, s.id, si.item_id;

-- name: ListWaits :many
SELECT w.work_id, w.item_id, w.edition, w.target_id, w.until_at,
    (SELECT count(*) FROM grabs_wait_candidate c WHERE c.item_id = w.item_id AND c.edition = w.edition AND c.target_id = w.target_id)
        AS candidates
FROM grabs_wait w
ORDER BY w.until_at, w.item_id, w.edition, w.target_id;

-- name: ListWaitsOfWorks :many
SELECT w.work_id, w.item_id, w.edition, w.target_id, w.until_at,
    (SELECT count(*) FROM grabs_wait_candidate c WHERE c.item_id = w.item_id AND c.edition = w.edition AND c.target_id = w.target_id)
        AS candidates
FROM grabs_wait w
WHERE w.work_id IN (sqlc.slice(work_ids))
ORDER BY w.until_at, w.item_id, w.edition, w.target_id;

-- name: ListStopsOfWork :many
-- Each counted failure of the Work's stopped cells up to its stop, and
-- whether its Grab is still kept.
SELECT f.item_id, f.edition, f.target_id, f.code, f.failed_at, f.grab_id,
    EXISTS (SELECT 1 FROM grabs_grab g WHERE g.id = f.grab_id) AS kept
FROM grabs_failure f
JOIN grabs_failure s ON s.item_id = f.item_id AND s.edition = f.edition AND s.target_id = f.target_id AND s.stopped_at IS NOT NULL
WHERE f.work_id = sqlc.arg(work_id) AND f.counted = 1 AND f.failed_at <= s.stopped_at
ORDER BY f.item_id, f.edition, f.target_id, f.failed_at, f.grab_id;

-- name: ListOpenRunsOfWork :many
SELECT id, group_key, lane, trigger FROM grabs_search
WHERE work_id = sqlc.arg(work_id) AND manual_id IS NULL AND group_key IS NOT NULL AND finished_at IS NULL
ORDER BY started_at, id;

-- name: ListPlannedOfWork :many
-- The Indexers the Work's open runs have lines planned and not sent on.
SELECT DISTINCT l.search_id, l.indexer_id FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id
WHERE s.work_id = sqlc.arg(work_id) AND l.indexer_id IS NOT NULL AND l.outcome IS NULL AND l.query_id IS NULL
    AND s.finished_at IS NULL AND s.manual_id IS NULL
ORDER BY l.search_id, l.indexer_id;

-- name: ListPlannedLanes :many
-- Each Indexer with lines planned and not sent, and the lanes of their
-- runs.
SELECT DISTINCT l.indexer_id, s.lane FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id
WHERE l.indexer_id IS NOT NULL AND l.outcome IS NULL AND l.query_id IS NULL AND s.finished_at IS NULL AND s.manual_id IS NULL
ORDER BY l.indexer_id, s.lane;

-- name: ListLane :many
-- A lane's rows that may start, in the lane's order, as ListQueued orders
-- them.
SELECT work_id, group_key, trigger FROM grabs_due WHERE lane = sqlc.arg(lane) AND search_id IS NULL
ORDER BY upgrade_only, fresh DESC, released_at DESC, queued_at, work_id, group_key;
