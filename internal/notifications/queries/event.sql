-- name: InsertEvent :one
INSERT INTO notifications_event (kind, at, due_at, subject)
VALUES (sqlc.arg(kind), sqlc.arg(at), sqlc.arg(due_at), sqlc.arg(subject))
RETURNING id;

-- name: GetEvent :one
SELECT * FROM notifications_event WHERE id = sqlc.arg(id);

-- name: ListDueUnreleased :many
SELECT * FROM notifications_event
WHERE released_at IS NULL AND due_at <= sqlc.arg(now)
ORDER BY due_at, id
LIMIT sqlc.arg(limit);

-- name: NextUnreleased :one
SELECT due_at FROM notifications_event WHERE released_at IS NULL ORDER BY due_at LIMIT 1;

-- name: SetReleased :exec
UPDATE notifications_event SET released_at = sqlc.arg(released_at) WHERE id = sqlc.arg(id);

-- name: DeleteEvent :exec
DELETE FROM notifications_event WHERE id = sqlc.arg(id);

-- name: ListPrunable :many
SELECT id FROM notifications_event WHERE at < sqlc.arg(before) ORDER BY at, id LIMIT sqlc.arg(limit);
