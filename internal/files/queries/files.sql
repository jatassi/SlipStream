-- name: InsertFile :exec
INSERT INTO files_file (id, work_id, item_id, edition, root_id, path, size, probe, grab_id, quality)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(edition), sqlc.arg(root_id), sqlc.arg(path),
    sqlc.arg(size), sqlc.arg(probe), sqlc.arg(grab_id), sqlc.arg(quality));

-- name: InsertFill :exec
INSERT INTO files_fill (file_id, target_id, item_id, edition)
VALUES (sqlc.arg(file_id), sqlc.arg(target_id), sqlc.arg(item_id), sqlc.arg(edition));

-- name: ListFillsOfWork :many
SELECT files_fill.target_id, sqlc.embed(files_file) FROM files_fill
JOIN files_file ON files_file.id = files_fill.file_id
WHERE files_file.work_id = sqlc.arg(work_id)
ORDER BY files_fill.item_id, files_fill.edition, files_fill.target_id;
