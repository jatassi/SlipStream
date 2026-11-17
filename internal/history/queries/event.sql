-- name: InsertEvent :one
INSERT INTO history_event (kind, at, content_type, work_id, grab_id, import_id, subject)
VALUES (sqlc.arg(kind), sqlc.arg(at), sqlc.narg(content_type), sqlc.narg(work_id), sqlc.narg(grab_id), sqlc.narg(import_id), sqlc.arg(subject))
RETURNING id;

-- name: LastIDBefore :one
SELECT id FROM history_event INDEXED BY history_event_at
WHERE at < sqlc.arg(at) ORDER BY at DESC, id DESC LIMIT 1;

-- name: NewestID :one
SELECT id FROM history_event ORDER BY id DESC LIMIT 1;

-- name: ListByIDs :many
SELECT * FROM history_event WHERE id IN (sqlc.slice(ids)) ORDER BY id DESC;

-- name: ListSweep :many
SELECT id FROM history_event
WHERE id > sqlc.arg(after) AND id <= sqlc.arg(ceiling)
ORDER BY id LIMIT sqlc.arg(limit);

-- name: DeleteEvents :exec
DELETE FROM history_event WHERE id IN (sqlc.slice(ids));

-- name: ListAll :many
SELECT * FROM history_event ORDER BY id;
