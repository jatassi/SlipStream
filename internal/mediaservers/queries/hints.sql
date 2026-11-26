-- name: FileAtPath :one
SELECT file_id FROM mediaservers_item
WHERE server_id = sqlc.arg(server_id) AND path_key = sqlc.arg(path_key);

-- name: MarkItemsStale :exec
UPDATE mediaservers_item
SET stale = 1
WHERE server_id = sqlc.arg(server_id)
  AND item_key IN (sqlc.slice(item_keys));

-- name: GetShownFolder :one
SELECT server_folder FROM mediaservers_mapping
WHERE server_id = sqlc.arg(server_id) AND root_id = sqlc.arg(root_id) AND shown = 1;

-- name: MarkPathStale :exec
UPDATE mediaservers_item
SET stale = 1
WHERE server_id = sqlc.arg(server_id) AND path_key = sqlc.arg(path_key);
