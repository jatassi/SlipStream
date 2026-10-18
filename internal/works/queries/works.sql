-- name: GetWork :one
SELECT * FROM works_work WHERE id = sqlc.arg(id);

-- name: GetWorkByKey :one
SELECT * FROM works_work WHERE content_type = sqlc.arg(content_type) AND key = sqlc.arg(key);

-- name: ListWorks :many
SELECT * FROM works_work ORDER BY sort_label, id;

-- name: ListWorksOfType :many
SELECT * FROM works_work WHERE content_type = sqlc.arg(content_type) ORDER BY sort_label, id;

-- name: HeldTypes :many
SELECT DISTINCT content_type FROM works_work ORDER BY content_type;

-- name: HoldsType :one
SELECT EXISTS (SELECT 1 FROM works_work WHERE content_type = sqlc.arg(content_type));

-- name: InsertWork :exec
INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at)
VALUES (sqlc.arg(id), sqlc.arg(content_type), sqlc.arg(key), sqlc.arg(label), sqlc.arg(sort_label), sqlc.arg(added_at), sqlc.arg(refreshed_at));

-- name: UpdateWork :exec
UPDATE works_work SET label = sqlc.arg(label), sort_label = sqlc.arg(sort_label), refreshed_at = sqlc.arg(refreshed_at)
WHERE id = sqlc.arg(id);

-- name: ListLevels :many
SELECT * FROM works_level WHERE work_id = sqlc.arg(work_id) ORDER BY depth, withdrawn_at IS NOT NULL, position, id;

-- name: InsertLevel :exec
INSERT INTO works_level (id, work_id, parent_id, depth, key, label, position, withdrawn_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(parent_id), sqlc.arg(depth), sqlc.arg(key), sqlc.arg(label), sqlc.arg(position), sqlc.arg(withdrawn_at));

-- name: UpdateLevel :exec
UPDATE works_level SET parent_id = sqlc.arg(parent_id), depth = sqlc.arg(depth), label = sqlc.arg(label),
    position = sqlc.arg(position), withdrawn_at = sqlc.arg(withdrawn_at)
WHERE id = sqlc.arg(id);

-- name: GetItem :one
SELECT * FROM works_item WHERE id = sqlc.arg(id);

-- name: ListItems :many
SELECT * FROM works_item WHERE work_id = sqlc.arg(work_id) ORDER BY withdrawn_at IS NOT NULL, position, id;

-- name: ListItemsOfWorks :many
SELECT * FROM works_item WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id, withdrawn_at IS NOT NULL, position, id;

-- name: InsertItem :exec
INSERT INTO works_item (id, work_id, level_id, key, label, position, withdrawn_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(level_id), sqlc.arg(key), sqlc.arg(label), sqlc.arg(position), sqlc.arg(withdrawn_at));

-- name: UpdateItem :exec
UPDATE works_item SET level_id = sqlc.arg(level_id), label = sqlc.arg(label), position = sqlc.arg(position),
    withdrawn_at = sqlc.arg(withdrawn_at)
WHERE id = sqlc.arg(id);

-- name: ListDisabledTypes :many
SELECT content_type FROM works_disabled_type ORDER BY content_type;

-- name: IsDisabled :one
SELECT EXISTS (SELECT 1 FROM works_disabled_type WHERE content_type = sqlc.arg(content_type));

-- name: DisableType :execrows
INSERT INTO works_disabled_type (content_type, disabled_at) VALUES (sqlc.arg(content_type), sqlc.arg(disabled_at))
ON CONFLICT (content_type) DO NOTHING;

-- name: EnableType :execrows
DELETE FROM works_disabled_type WHERE content_type = sqlc.arg(content_type);

-- name: MonitorItems :exec
UPDATE works_item SET monitored = 1 WHERE work_id = sqlc.arg(work_id);
