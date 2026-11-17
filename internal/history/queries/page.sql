-- name: FirstIDFrom :one
SELECT id FROM history_event INDEXED BY history_event_at
WHERE at >= sqlc.arg(at) ORDER BY at, id LIMIT 1;

-- name: ListIDs :many
SELECT id FROM history_event
WHERE id >= sqlc.arg(floor) AND id < sqlc.arg(before)
ORDER BY id DESC LIMIT sqlc.arg(limit);

-- name: ListKindIDs :many
SELECT id FROM history_event INDEXED BY history_event_kind
WHERE kind = sqlc.arg(kind) AND id >= sqlc.arg(floor) AND id < sqlc.arg(before)
ORDER BY id DESC LIMIT sqlc.arg(limit);

-- name: ListTypeKindIDs :many
SELECT id FROM history_event INDEXED BY history_event_type
WHERE content_type = sqlc.arg(content_type) AND kind = sqlc.arg(kind)
  AND id >= sqlc.arg(floor) AND id < sqlc.arg(before)
ORDER BY id DESC LIMIT sqlc.arg(limit);

-- name: ListWorkIDs :many
SELECT w.event_id FROM history_work AS w INDEXED BY history_work_work
JOIN history_event e ON e.id = w.event_id
WHERE w.work_id = sqlc.arg(work_id) AND w.event_id >= sqlc.arg(floor) AND w.event_id < sqlc.arg(before)
  AND e.kind IN (SELECT value FROM json_each(sqlc.arg(kinds)))
  AND (sqlc.narg(content_type) IS NULL OR e.content_type = sqlc.narg(content_type))
ORDER BY w.event_id DESC LIMIT sqlc.arg(limit);

-- name: CountKindFrom :one
SELECT COUNT(*) FROM history_event INDEXED BY history_event_kind
WHERE kind = sqlc.arg(kind) AND id >= sqlc.arg(floor);
