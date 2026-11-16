-- name: ListDueKeys :many
SELECT work_id, group_key FROM grabs_due ORDER BY work_id, group_key;

-- name: InsertBacklogDue :exec
-- A wanted group's first row. A group that has one keeps it.
INSERT INTO grabs_due (work_id, group_key, upgrade_only, fresh, released_at, due_at, gap_seconds, changed_at)
VALUES (sqlc.arg(work_id), sqlc.arg(group_key), sqlc.arg(upgrade_only), sqlc.arg(fresh), sqlc.narg(released_at), sqlc.arg(due_at),
    sqlc.arg(gap_seconds), sqlc.arg(changed_at))
ON CONFLICT (work_id, group_key) DO NOTHING;

-- name: RefreshDueOrder :exec
-- The row's order keys from a snapshot taken at seen, unless a writer
-- touched the row since.
UPDATE grabs_due SET upgrade_only = sqlc.arg(upgrade_only), fresh = sqlc.arg(fresh), released_at = sqlc.narg(released_at)
WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key) AND changed_at < sqlc.arg(seen);

-- name: QueueBacklogDue :execrows
-- An idle row that is due joins Backlog search's lane, whole. Upgrade-only
-- rows join only while Backlog search looks for upgrades.
UPDATE grabs_due SET lane = sqlc.arg(lane), queued_at = sqlc.arg(queued_at), trigger = sqlc.arg(trigger), changed_at = sqlc.arg(queued_at)
WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key) AND lane IS NULL AND search_id IS NULL
    AND due_at <= sqlc.arg(queued_at) AND (upgrade_only = 0 OR upgrade_only = sqlc.arg(upgrades));

-- name: DeleteIdleDue :exec
-- A row a snapshot taken at seen found no Paced cell for, unless it is
-- queued, searching or touched since.
DELETE FROM grabs_due
WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key) AND lane IS NULL AND search_id IS NULL
    AND changed_at < sqlc.arg(seen);

-- name: ListFailureCells :many
SELECT DISTINCT work_id, item_id, edition, target_id FROM grabs_failure ORDER BY work_id, item_id, edition, target_id;

-- name: FailuresKept :one
-- Whether a cell's failure rows stay though a snapshot taken at seen found
-- it not Paced: it failed since, or a Grab still holds it unsettled.
SELECT EXISTS (
    SELECT 1 FROM grabs_failure f
    WHERE f.item_id = sqlc.arg(item_id) AND f.edition = sqlc.arg(edition) AND f.target_id = sqlc.arg(target_id)
        AND f.failed_at >= sqlc.arg(seen)
    UNION ALL
    SELECT 1 FROM grabs_target t JOIN grabs_grab g ON g.id = t.grab_id
    WHERE t.item_id = sqlc.arg(item_id) AND t.target_id = sqlc.arg(target_id) AND g.edition = sqlc.arg(edition)
        AND t.settled IS NULL AND g.stage NOT IN ('imported', 'failed')
);

-- name: ListItemSearches :many
SELECT item_id FROM grabs_item_search ORDER BY item_id;

-- name: DeleteItemSearch :exec
DELETE FROM grabs_item_search WHERE item_id = sqlc.arg(item_id);
