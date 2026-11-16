-- name: UpsertTold :exec
INSERT INTO notifications_told (target, key, subject, since_at, told_at)
VALUES (sqlc.arg(target), sqlc.arg(key), sqlc.arg(subject), sqlc.arg(since_at), sqlc.arg(told_at))
ON CONFLICT (target, key) DO UPDATE SET subject = excluded.subject, since_at = excluded.since_at, told_at = excluded.told_at;

-- name: ListToldFor :many
SELECT * FROM notifications_told WHERE key IN (sqlc.slice(keys)) ORDER BY key, target;

-- name: DeleteTold :exec
DELETE FROM notifications_told WHERE target = sqlc.arg(target) AND key = sqlc.arg(key);

-- name: DeleteToldOf :exec
DELETE FROM notifications_told WHERE target = sqlc.arg(target);
