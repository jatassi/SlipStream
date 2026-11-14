-- name: FileAtPath :one
SELECT file_id FROM mediaservers_item
WHERE server_id = sqlc.arg(server_id) AND path_key = sqlc.arg(path_key);

-- name: MarkItemsStale :exec
UPDATE mediaservers_item
SET stale = 1
WHERE server_id = sqlc.arg(server_id)
  AND item_key IN (sqlc.slice(item_keys));
