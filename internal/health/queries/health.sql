-- name: ListItems :many
SELECT * FROM health_item ORDER BY category, key;

-- name: InsertItem :exec
INSERT INTO health_item (category, key, level, since, changed_at)
VALUES (sqlc.arg(category), sqlc.arg(key), sqlc.arg(level), sqlc.arg(since), sqlc.arg(changed_at));

-- name: UpdateItemLevel :exec
UPDATE health_item SET level = sqlc.arg(level), changed_at = sqlc.arg(changed_at)
WHERE category = sqlc.arg(category) AND key = sqlc.arg(key);

-- name: DeleteItem :exec
DELETE FROM health_item WHERE category = sqlc.arg(category) AND key = sqlc.arg(key);

-- name: DeleteCategory :exec
DELETE FROM health_item WHERE category = sqlc.arg(category);

-- name: GetSettings :one
SELECT warn_amount, warn_unit, error_amount, error_unit FROM health_settings WHERE id = 1;

-- name: SetSettings :exec
UPDATE health_settings SET warn_amount = sqlc.arg(warn_amount), warn_unit = sqlc.arg(warn_unit),
    error_amount = sqlc.arg(error_amount), error_unit = sqlc.arg(error_unit)
WHERE id = 1;

-- name: ListVolumeLevels :many
SELECT * FROM health_volume ORDER BY node_id, mount;

-- name: GetVolumeLevels :one
SELECT * FROM health_volume WHERE node_id = sqlc.arg(node_id) AND mount = sqlc.arg(mount);

-- name: SetVolumeLevels :exec
INSERT INTO health_volume (node_id, mount, warn_amount, warn_unit, error_amount, error_unit)
VALUES (sqlc.arg(node_id), sqlc.arg(mount), sqlc.arg(warn_amount), sqlc.arg(warn_unit), sqlc.arg(error_amount), sqlc.arg(error_unit))
ON CONFLICT (node_id, mount) DO UPDATE SET
    warn_amount = excluded.warn_amount, warn_unit = excluded.warn_unit,
    error_amount = excluded.error_amount, error_unit = excluded.error_unit;

-- name: DeleteVolumeLevels :exec
DELETE FROM health_volume WHERE node_id = sqlc.arg(node_id) AND mount = sqlc.arg(mount);
