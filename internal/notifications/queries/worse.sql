-- name: UpsertWorse :exec
INSERT INTO notifications_worse (event_id, key, worse_id)
VALUES (sqlc.arg(event_id), sqlc.arg(key), sqlc.arg(worse_id))
ON CONFLICT (event_id, key) DO UPDATE SET worse_id = excluded.worse_id;

-- name: DeleteWorse :exec
DELETE FROM notifications_worse WHERE event_id = sqlc.arg(event_id) AND key = sqlc.arg(key);

-- name: ListWorseOf :many
SELECT * FROM notifications_worse WHERE event_id IN (sqlc.slice(event_ids)) ORDER BY event_id, key;
