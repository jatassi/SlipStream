-- name: GetDue :one
SELECT * FROM grabs_due WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key);

-- name: PutDue :exec
-- Every field but the open run, which only a run's start and finish set.
INSERT INTO grabs_due (work_id, group_key, lane, queued_at, trigger, include_gone, upgrade_only, fresh, released_at, due_at,
    gap_seconds, changed_at)
VALUES (sqlc.arg(work_id), sqlc.arg(group_key), sqlc.narg(lane), sqlc.narg(queued_at), sqlc.narg(trigger), sqlc.arg(include_gone),
    sqlc.arg(upgrade_only), sqlc.arg(fresh), sqlc.narg(released_at), sqlc.arg(due_at), sqlc.arg(gap_seconds), sqlc.arg(changed_at))
ON CONFLICT (work_id, group_key) DO UPDATE SET lane = excluded.lane, queued_at = excluded.queued_at, trigger = excluded.trigger,
    include_gone = excluded.include_gone, upgrade_only = excluded.upgrade_only, fresh = excluded.fresh,
    released_at = excluded.released_at, due_at = excluded.due_at, gap_seconds = excluded.gap_seconds,
    changed_at = excluded.changed_at;

-- name: DeleteDueItems :exec
DELETE FROM grabs_due_item WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key);

-- name: InsertDueItem :exec
INSERT INTO grabs_due_item (work_id, group_key, item_id) VALUES (sqlc.arg(work_id), sqlc.arg(group_key), sqlc.arg(item_id));

-- name: ListQueued :many
-- A lane's rows that may start, in the lane's order.
SELECT * FROM grabs_due WHERE lane = sqlc.arg(lane) AND search_id IS NULL
ORDER BY upgrade_only, fresh DESC, released_at DESC, queued_at, work_id, group_key
LIMIT sqlc.arg(n);

-- name: UnqueueDue :exec
-- The queued entry leaves its lane, onto the run when one starts.
UPDATE grabs_due SET lane = NULL, queued_at = NULL, trigger = NULL, include_gone = 0, search_id = sqlc.narg(search_id),
    changed_at = sqlc.arg(changed_at)
WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key);

-- name: LastSearchOfGroup :one
SELECT finished_at FROM grabs_search
WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key) AND manual_id IS NULL AND finished_at IS NOT NULL
ORDER BY finished_at DESC, id DESC
LIMIT 1;

-- name: PaceDue :exec
-- A finished run's group: its next gap and due time, and no run open.
UPDATE grabs_due SET gap_seconds = sqlc.arg(gap_seconds), due_at = sqlc.arg(due_at), search_id = NULL, changed_at = sqlc.arg(changed_at)
WHERE search_id = sqlc.arg(search_id);
