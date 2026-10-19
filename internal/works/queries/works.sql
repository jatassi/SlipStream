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
UPDATE works_work SET label = sqlc.arg(label), sort_label = sqlc.arg(sort_label), refreshed_at = sqlc.arg(refreshed_at),
    gone_upstream_at = NULL
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

-- name: MarkGoneUpstream :exec
UPDATE works_work SET gone_upstream_at = sqlc.arg(gone_upstream_at) WHERE id = sqlc.arg(id);

-- name: SetRatingsRead :exec
UPDATE works_work SET ratings_read_at = sqlc.arg(ratings_read_at) WHERE id = sqlc.arg(id);

-- name: WorksToMonitor :many
SELECT DISTINCT work_id FROM works_item
WHERE monitored <> sqlc.arg(monitored) AND id IN (sqlc.slice(item_ids))
ORDER BY work_id;

-- name: SetMonitored :exec
UPDATE works_item SET monitored = sqlc.arg(monitored) WHERE id IN (sqlc.slice(item_ids));

-- name: ListRatings :many
SELECT * FROM works_rating WHERE work_id = sqlc.arg(work_id) ORDER BY item_id, source;

-- name: InsertRating :exec
INSERT INTO works_rating (work_id, item_id, source, value, scale, votes, fetched_at)
VALUES (sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(source), sqlc.arg(value), sqlc.arg(scale), sqlc.arg(votes), sqlc.arg(fetched_at));

-- name: UpdateRating :exec
UPDATE works_rating SET value = sqlc.arg(value), scale = sqlc.arg(scale), votes = sqlc.arg(votes), fetched_at = sqlc.arg(fetched_at)
WHERE work_id = sqlc.arg(work_id) AND item_id IS sqlc.arg(item_id) AND source = sqlc.arg(source);

-- name: ListArtwork :many
SELECT * FROM works_artwork WHERE work_id = sqlc.arg(work_id) ORDER BY item_id, kind;

-- name: InsertArtwork :exec
INSERT INTO works_artwork (work_id, item_id, kind, source, path)
VALUES (sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(kind), sqlc.arg(source), sqlc.arg(path));

-- name: UpdateArtwork :exec
UPDATE works_artwork SET source = sqlc.arg(source), path = sqlc.arg(path)
WHERE work_id = sqlc.arg(work_id) AND item_id IS sqlc.arg(item_id) AND kind = sqlc.arg(kind);
