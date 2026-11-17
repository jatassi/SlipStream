-- name: InsertPending :exec
INSERT INTO notifications_pending (event_id, kind, at, due_at, items)
VALUES (sqlc.arg(event_id), sqlc.arg(kind), sqlc.arg(at), sqlc.arg(due_at), sqlc.narg(items));

-- name: ListDuePending :many
SELECT * FROM notifications_pending
WHERE due_at <= sqlc.arg(now)
ORDER BY due_at, event_id
LIMIT sqlc.arg(limit);

-- name: NextPending :one
SELECT due_at FROM notifications_pending ORDER BY due_at LIMIT 1;

-- name: DeletePending :exec
DELETE FROM notifications_pending WHERE event_id = sqlc.arg(event_id);

-- name: WaitingRaise :one
-- A Health key holds only URL-unreserved characters and ".", so its
-- quoted form is the same in Go's JSON and SQLite's, and matches one
-- element exactly.
SELECT event_id, items FROM notifications_pending
WHERE kind = 'new_health_item' AND instr(items, json_quote(CAST(sqlc.arg(key) AS TEXT))) > 0
ORDER BY event_id DESC
LIMIT 1;

-- name: SetPendingItems :exec
UPDATE notifications_pending SET items = sqlc.narg(items) WHERE event_id = sqlc.arg(event_id);

-- name: ListOwed :many
-- A pending row, a due delivery, or the newer entry a pending or due
-- message renders a key from.
SELECT p.event_id AS id FROM notifications_pending p WHERE p.event_id IN (sqlc.slice(ids))
UNION
SELECT d.event_id FROM notifications_delivery d WHERE d.state = 'due' AND d.event_id IN (sqlc.slice(due_ids))
UNION
SELECT w.worse_id FROM notifications_worse w
WHERE w.worse_id IN (sqlc.slice(worse_ids))
  AND (w.event_id IN (SELECT q.event_id FROM notifications_pending q)
       OR w.event_id IN (SELECT r.event_id FROM notifications_delivery r WHERE r.state = 'due'))
ORDER BY 1;
