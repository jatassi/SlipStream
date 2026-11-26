-- name: ListPlayedFile :many
-- The file id names, unless gone, with each of its Items in position
-- order.
SELECT sqlc.embed(f), i.item_id FROM files_file f
JOIN files_file_item i ON i.file_id = f.id
WHERE f.id = sqlc.arg(id) AND f.gone_at IS NULL
ORDER BY i.position;

-- name: ListFilesOfItems :many
-- Every file not gone that holds one of item_ids, a JSON array, Kept and
-- moving files included, with each of its Items in position order.
SELECT sqlc.embed(f), i.item_id FROM files_file f
JOIN files_file_item i ON i.file_id = f.id
WHERE f.gone_at IS NULL AND f.id IN (
    SELECT h.file_id FROM files_file_item h
    WHERE h.item_id IN (SELECT unhex(replace(value, '-', '')) FROM json_each(CAST(sqlc.arg(item_ids) AS TEXT))))
ORDER BY f.id, i.position;
