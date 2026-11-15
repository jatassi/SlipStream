-- name: ListDue :many
-- Every queue row, queued ones first in queued order: grabstest's
-- snapshot of the queue.
SELECT * FROM grabs_due ORDER BY queued_at IS NULL, queued_at, work_id, group_key;

-- name: ListDueItems :many
SELECT item_id FROM grabs_due_item WHERE work_id = sqlc.arg(work_id) AND group_key = sqlc.arg(group_key) ORDER BY item_id;
