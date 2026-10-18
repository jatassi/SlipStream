-- name: InsertFile :exec
INSERT INTO files_file (id, work_id, item_id, cut, root_id, path, size, probe, grab_id, quality, label, extra)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(cut), sqlc.arg(root_id), sqlc.arg(path),
    sqlc.arg(size), sqlc.arg(probe), sqlc.arg(grab_id), sqlc.arg(quality), sqlc.arg(label), sqlc.arg(extra));

-- name: RenameFile :exec
UPDATE files_file SET path = sqlc.arg(path), label = sqlc.arg(label), extra = sqlc.arg(extra) WHERE id = sqlc.arg(id);

-- name: DeleteFile :exec
DELETE FROM files_file WHERE id = sqlc.arg(id);

-- name: ListFilesOfWorks :many
SELECT * FROM files_file WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY id;

-- name: ListFileIDs :many
SELECT id FROM files_file ORDER BY id;

-- name: InsertFill :exec
INSERT INTO files_fill (file_id, target_id, item_id, edition)
VALUES (sqlc.arg(file_id), sqlc.arg(target_id), sqlc.arg(item_id), sqlc.arg(edition));

-- name: DeleteFill :exec
DELETE FROM files_fill WHERE item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id);

-- name: ListFillsOfWorks :many
SELECT files_fill.* FROM files_fill
JOIN files_file ON files_file.id = files_fill.file_id
WHERE files_file.work_id IN (sqlc.slice(work_ids))
ORDER BY files_fill.item_id, files_fill.edition, files_fill.target_id;

-- name: ListFillsOfTargets :many
SELECT files_fill.target_id, files_file.work_id, files_file.item_id, files_file.size FROM files_fill
JOIN files_file ON files_file.id = files_fill.file_id
WHERE files_fill.target_id IN (sqlc.slice(target_ids))
ORDER BY files_fill.target_id, files_fill.file_id;

-- name: GetMove :one
SELECT sits FROM files_move WHERE file_id = sqlc.arg(file_id);

-- name: InsertMove :exec
INSERT INTO files_move (file_id, sits) VALUES (sqlc.arg(file_id), sqlc.arg(sits));

-- name: MoveReached :execrows
UPDATE files_move SET sits = sqlc.arg(reached) WHERE file_id = sqlc.arg(file_id) AND sits = sqlc.arg(sits);

-- name: DeleteMove :execrows
DELETE FROM files_move WHERE file_id = sqlc.arg(file_id) AND sits = sqlc.arg(sits);

-- name: FinishMove :execrows
DELETE FROM files_move WHERE file_id = sqlc.arg(file_id) AND sits = sqlc.arg(sits)
    AND EXISTS (SELECT 1 FROM files_file WHERE files_file.id = files_move.file_id AND files_file.path = sqlc.arg(path));

-- name: ListMoves :many
SELECT files_move.file_id, files_move.sits, files_file.root_id, files_file.path FROM files_move
JOIN files_file ON files_file.id = files_move.file_id
ORDER BY files_move.file_id;

-- name: InsertDoomed :exec
INSERT INTO files_doomed (file_id, root_id, path, sits) VALUES (sqlc.arg(file_id), sqlc.arg(root_id), sqlc.arg(path), sqlc.arg(sits));

-- name: ListDoomed :many
SELECT * FROM files_doomed ORDER BY file_id;

-- name: DeleteDoomed :exec
DELETE FROM files_doomed WHERE file_id = sqlc.arg(file_id);

-- name: ListWorksOfCuts :many
SELECT DISTINCT work_id FROM files_file WHERE cut IN (sqlc.slice(cuts)) ORDER BY work_id;

-- name: CountFilesOfCut :one
SELECT count(*) FROM files_file WHERE cut = sqlc.arg(cut);

-- name: HasFiles :one
SELECT EXISTS (SELECT 1 FROM files_file WHERE work_id = sqlc.arg(work_id));

-- name: UseOfRoots :many
SELECT root_id, count(*) AS files, CAST(total(size) AS INTEGER) AS bytes FROM files_file
WHERE root_id IN (sqlc.slice(root_ids)) GROUP BY root_id;

-- name: DoomedOfRoots :many
SELECT root_id, count(*) AS files FROM files_doomed
WHERE root_id IN (sqlc.slice(root_ids)) GROUP BY root_id;
