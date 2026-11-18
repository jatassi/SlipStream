-- name: ListSubjects :many
-- Up to n library files after the file id after, from the first when it is
-- NULL, in id order, and whether each one's bytes sit at its path: no move
-- is open.
SELECT sqlc.embed(f), CAST(m.file_id IS NULL AS INTEGER) AS at_name FROM files_file f
LEFT JOIN files_move m ON m.file_id = f.id
WHERE f.id > coalesce(sqlc.arg(after), x'')
ORDER BY f.id
LIMIT sqlc.arg(n);

-- name: ListSubjectsOfWorks :many
-- ListSubjects narrowed to the files of work_ids.
SELECT sqlc.embed(f), CAST(m.file_id IS NULL AS INTEGER) AS at_name FROM files_file f
LEFT JOIN files_move m ON m.file_id = f.id
WHERE f.work_id IN (sqlc.slice(work_ids)) AND f.id > coalesce(sqlc.arg(after), x'')
ORDER BY f.id
LIMIT sqlc.arg(n);

-- name: ListSubjectsOfFiles :many
-- ListSubjects of the files ids names.
SELECT sqlc.embed(f), CAST(m.file_id IS NULL AS INTEGER) AS at_name FROM files_file f
LEFT JOIN files_move m ON m.file_id = f.id
WHERE f.id IN (sqlc.slice(ids))
ORDER BY f.id;
