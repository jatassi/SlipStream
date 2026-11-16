-- name: InsertDelivery :exec
INSERT INTO notifications_delivery (event_id, connection_id, state, attempts, items, accepted)
VALUES (sqlc.arg(event_id), sqlc.arg(connection_id), 'due', 0, sqlc.narg(items), '[]');

-- name: OldestDue :one
SELECT * FROM notifications_delivery
WHERE connection_id = sqlc.arg(connection_id) AND state = 'due'
ORDER BY event_id
LIMIT 1;

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
SELECT d.event_id, d.connection_id FROM notifications_delivery d
JOIN notifications_event e ON e.id = d.event_id
WHERE d.state = 'due' AND e.at < sqlc.arg(at)
ORDER BY d.event_id, d.connection_id
LIMIT sqlc.arg(limit);

-- name: ListDueOff :many
SELECT d.event_id, d.connection_id FROM notifications_delivery d
JOIN notifications_connection c ON c.id = d.connection_id
JOIN notifications_event e ON e.id = d.event_id
WHERE d.state = 'due'
  AND (c.is_on = 0 OR e.kind NOT IN (SELECT value FROM json_each(c.events)))
ORDER BY d.event_id, d.connection_id
LIMIT sqlc.arg(limit);

-- name: CountDue :many
SELECT connection_id, COUNT(*) AS due FROM notifications_delivery
WHERE state = 'due'
GROUP BY connection_id
ORDER BY connection_id;

-- name: ConnectionsCarrying :many
SELECT DISTINCT i.key, d.connection_id FROM notifications_item i
JOIN notifications_delivery d ON d.event_id = i.event_id AND d.state = 'due'
WHERE i.key IN (sqlc.slice(keys))
ORDER BY i.key, d.connection_id;
