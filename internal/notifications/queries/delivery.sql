-- name: InsertDelivery :exec
INSERT INTO notifications_delivery (event_id, connection_id, kind, at, state, attempts, items, accepted)
VALUES (sqlc.arg(event_id), sqlc.arg(connection_id), sqlc.arg(kind), sqlc.arg(at), 'due', 0, sqlc.narg(items), '[]');

-- name: OldestDue :one
SELECT * FROM notifications_delivery
WHERE connection_id = sqlc.arg(connection_id) AND state = 'due'
ORDER BY event_id
LIMIT 1;

-- name: GetDelivery :one
SELECT * FROM notifications_delivery WHERE event_id = sqlc.arg(event_id) AND connection_id = sqlc.arg(connection_id);

-- name: UpdateDue :exec
UPDATE notifications_delivery
SET attempts = sqlc.arg(attempts), accepted = sqlc.arg(accepted)
WHERE event_id = sqlc.arg(event_id) AND connection_id = sqlc.arg(connection_id) AND state = 'due';

-- name: SetItems :exec
UPDATE notifications_delivery SET items = sqlc.narg(items)
WHERE event_id = sqlc.arg(event_id) AND connection_id = sqlc.arg(connection_id);

-- name: DeleteDelivery :exec
DELETE FROM notifications_delivery WHERE event_id = sqlc.arg(event_id) AND connection_id = sqlc.arg(connection_id);

-- name: GiveUp :execrows
UPDATE notifications_delivery
SET state = 'given_up', ended_at = sqlc.arg(at), reason = sqlc.arg(reason), response = sqlc.narg(response)
WHERE event_id = sqlc.arg(event_id) AND connection_id = sqlc.arg(connection_id) AND state = 'due';

-- name: ListDueOlderThan :many
SELECT event_id, connection_id FROM notifications_delivery
WHERE state = 'due' AND at < sqlc.arg(at)
ORDER BY event_id, connection_id
LIMIT sqlc.arg(limit);

-- name: ListDueOff :many
SELECT d.event_id, d.connection_id FROM notifications_delivery d
JOIN notifications_connection c ON c.id = d.connection_id
WHERE d.state = 'due'
  AND (c.is_on = 0 OR d.kind NOT IN (SELECT value FROM json_each(c.events)))
ORDER BY d.event_id, d.connection_id
LIMIT sqlc.arg(limit);

-- name: CountDue :many
SELECT connection_id, COUNT(*) AS due FROM notifications_delivery
WHERE state = 'due'
GROUP BY connection_id
ORDER BY connection_id;

-- name: ConnectionsCarrying :many
-- The connections with a due delivery that carries key, bounded by Give up.
-- The key's quoted form matches one element exactly, as in WaitingRaise.
SELECT DISTINCT connection_id FROM notifications_delivery
WHERE state = 'due' AND instr(items, json_quote(CAST(sqlc.arg(key) AS TEXT))) > 0
ORDER BY connection_id;

-- name: OldestDueAt :one
SELECT at FROM notifications_delivery WHERE state = 'due' ORDER BY at LIMIT 1;
