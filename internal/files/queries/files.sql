-- name: InsertFile :exec
INSERT INTO files_file (id, work_id, item_id, cut, root_id, path, size, probe, grab_id, quality, label, extra,
    release_name, title_read, modified_at, node_file_id, node_volume_id, probed_at)
VALUES (sqlc.arg(id), sqlc.arg(work_id), sqlc.arg(item_id), sqlc.arg(cut), sqlc.arg(root_id), sqlc.arg(path),
    sqlc.arg(size), sqlc.arg(probe), sqlc.arg(grab_id), sqlc.arg(quality), sqlc.arg(label), sqlc.arg(extra),
    sqlc.arg(release_name), sqlc.arg(title_read), sqlc.arg(modified_at), sqlc.arg(node_file_id), sqlc.arg(node_volume_id),
    sqlc.arg(probed_at));

-- name: UpdateScanned :exec
UPDATE files_file SET path = sqlc.arg(path), size = sqlc.arg(size), probe = sqlc.arg(probe), grab_id = sqlc.arg(grab_id),
    quality = sqlc.arg(quality), release_name = sqlc.arg(release_name), title_read = sqlc.arg(title_read), modified_at = sqlc.arg(modified_at), node_file_id = sqlc.arg(node_file_id),
    node_volume_id = sqlc.arg(node_volume_id), gone_at = sqlc.arg(gone_at), probed_at = sqlc.arg(probed_at),
    probe_error = sqlc.arg(probe_error)
WHERE id = sqlc.arg(id);

-- name: RenameFile :exec
UPDATE files_file SET path = sqlc.arg(path), label = sqlc.arg(label), extra = sqlc.arg(extra) WHERE id = sqlc.arg(id);

-- name: DeleteFile :exec
DELETE FROM files_file WHERE id = sqlc.arg(id);

-- name: ListFilesOfWorks :many
SELECT * FROM files_file WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY id;

-- name: ListFileReadsOfWorks :many
SELECT id, work_id, release_name, CAST(title_read AS TEXT) AS title_json FROM files_file WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListFileNamesOfWorks :many
SELECT id, work_id, release_name FROM files_file WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListEveryFile :many
SELECT * FROM files_file ORDER BY rowid;

-- name: ListBriefFilesOfWorks :many
SELECT id, work_id, item_id, cut, root_id, size, quality, extra, gone_at FROM files_file WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListEveryBriefFile :many
SELECT id, work_id, item_id, cut, root_id, size, quality, extra, gone_at FROM files_file ORDER BY rowid;

-- name: ListUnreadFiles :many
SELECT id, work_id, grab_id, path FROM files_file WHERE title_read IS NULL ORDER BY id;

-- name: SetFileRead :exec
UPDATE files_file SET release_name = sqlc.arg(release_name), title_read = sqlc.arg(title_read),
    source_name = CASE WHEN grab_id IS NULL THEN sqlc.arg(release_name) END
WHERE id = sqlc.arg(id) AND title_read IS NULL;

-- name: ListFileIDs :many
SELECT id FROM files_file ORDER BY id;

-- name: InsertFill :exec
INSERT INTO files_fill (file_id, target_id, item_id, edition, score)
VALUES (sqlc.arg(file_id), sqlc.arg(target_id), sqlc.arg(item_id), sqlc.arg(edition), sqlc.arg(score));

-- name: DeleteFill :exec
DELETE FROM files_fill WHERE item_id = sqlc.arg(item_id) AND edition = sqlc.arg(edition) AND target_id = sqlc.arg(target_id);

-- name: ListFillsOfWorks :many
SELECT files_fill.* FROM files_fill
JOIN files_file ON files_file.id = files_fill.file_id
WHERE files_file.work_id IN (sqlc.slice(work_ids))
ORDER BY files_fill.item_id, files_fill.edition, files_fill.target_id;

-- name: ListEveryFill :many
SELECT * FROM files_fill ORDER BY rowid;

-- name: ListFillsOfTargets :many
SELECT files_fill.target_id, files_file.work_id, files_file.item_id, files_file.size FROM files_fill
JOIN files_file ON files_file.id = files_fill.file_id
WHERE files_fill.target_id IN (sqlc.slice(target_ids))
ORDER BY files_fill.target_id, files_fill.file_id;

-- name: GetMove :one
SELECT * FROM files_move WHERE file_id = sqlc.arg(file_id);

-- name: InsertMove :exec
INSERT INTO files_move (file_id, sits, origin) VALUES (sqlc.arg(file_id), sqlc.arg(sits), sqlc.arg(sits));

-- name: SetRing :execrows
UPDATE files_move SET ring = sqlc.arg(ring) WHERE file_id = sqlc.arg(file_id) AND sits = sqlc.arg(sits) AND ring IS NULL;

-- name: SetPath :exec
UPDATE files_file SET path = sqlc.arg(path) WHERE id = sqlc.arg(id);

-- name: PathSettled :one
SELECT EXISTS (SELECT 1 FROM files_file f WHERE f.root_id = sqlc.arg(root_id) AND f.path = sqlc.arg(path)
    AND NOT EXISTS (SELECT 1 FROM files_move m WHERE m.file_id = f.id));

-- name: MoveReached :execrows
UPDATE files_move SET sits = sqlc.arg(reached) WHERE file_id = sqlc.arg(file_id) AND sits = sqlc.arg(sits);

-- name: DeleteFileMove :exec
DELETE FROM files_move WHERE file_id = sqlc.arg(file_id);

-- name: DeleteMove :execrows
DELETE FROM files_move WHERE file_id = sqlc.arg(file_id) AND sits = sqlc.arg(sits);

-- name: FinishMove :execrows
DELETE FROM files_move WHERE file_id = sqlc.arg(file_id) AND sits = sqlc.arg(sits)
    AND EXISTS (SELECT 1 FROM files_file WHERE files_file.id = files_move.file_id AND files_file.path = sqlc.arg(path));

-- name: ListMoves :many
SELECT files_move.file_id, files_move.sits, files_move.origin, files_move.ring, files_file.work_id, files_file.root_id, files_file.path
FROM files_move JOIN files_file ON files_file.id = files_move.file_id
ORDER BY files_move.file_id;

-- name: InsertRenamed :exec
INSERT INTO files_renamed (file_id, root_id, from_path, to_path, renamed_at)
VALUES (sqlc.arg(file_id), sqlc.arg(root_id), sqlc.arg(from_path), sqlc.arg(to_path), sqlc.arg(renamed_at));

-- name: InsertPrune :exec
INSERT INTO files_prune (root_id, folder) VALUES (sqlc.arg(root_id), sqlc.arg(folder)) ON CONFLICT DO NOTHING;

-- name: ListPrune :many
SELECT * FROM files_prune ORDER BY root_id, folder;

-- name: DeletePrune :exec
DELETE FROM files_prune WHERE root_id = sqlc.arg(root_id) AND folder = sqlc.arg(folder);

-- name: ListMovingFiles :many
SELECT file_id FROM files_move ORDER BY file_id;

-- name: ListMovingFilesOfWorks :many
SELECT m.file_id FROM files_move m JOIN files_file f ON f.id = m.file_id WHERE f.work_id IN (sqlc.slice(work_ids)) ORDER BY m.file_id;

-- name: CountMovingOfWork :one
SELECT count(*) FROM files_move m JOIN files_file f ON f.id = m.file_id WHERE f.work_id = sqlc.arg(work_id);

-- name: CountMovingByWork :many
SELECT f.work_id, count(*) AS files FROM files_move m JOIN files_file f ON f.id = m.file_id GROUP BY f.work_id ORDER BY f.work_id;

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
SELECT EXISTS (SELECT 1 FROM files_file WHERE work_id = sqlc.arg(work_id) AND gone_at IS NULL);

-- name: UseOfRoots :many
SELECT root_id, count(*) AS files, CAST(total(size) AS INTEGER) AS bytes FROM files_file
WHERE root_id IN (sqlc.slice(root_ids)) AND gone_at IS NULL GROUP BY root_id ORDER BY root_id;

-- name: DoomedOfRoots :many
SELECT root_id, count(*) AS files FROM files_doomed
WHERE root_id IN (sqlc.slice(root_ids)) GROUP BY root_id ORDER BY root_id;

-- name: ProofOfRoot :many
SELECT CAST(coalesce(m.sits, f.path) AS TEXT) AS path
FROM files_file f LEFT JOIN files_move m ON m.file_id = f.id
WHERE f.root_id = sqlc.arg(root_id) AND f.gone_at IS NULL ORDER BY f.id LIMIT 3;

-- name: RecordedOfRoot :many
SELECT f.id, f.path, m.sits, m.ring, f.size
FROM files_file f LEFT JOIN files_move m ON m.file_id = f.id
WHERE f.root_id = sqlc.arg(root_id) AND f.gone_at IS NULL ORDER BY f.id;

-- name: ScannedOfRoot :many
SELECT f.id, f.work_id, f.path, m.sits, m.ring, f.size, f.modified_at, f.node_file_id, f.node_volume_id, f.gone_at, f.probe_version
FROM files_file f LEFT JOIN files_move m ON m.file_id = f.id
WHERE f.root_id = sqlc.arg(root_id) ORDER BY f.id;

-- name: HeldOfRoot :many
SELECT CAST(m.sits AS TEXT) AS path FROM files_move m JOIN files_file f ON f.id = m.file_id WHERE f.root_id = sqlc.arg(root_id)
UNION SELECT d.path FROM files_doomed d WHERE d.root_id = sqlc.arg(root_id)
UNION SELECT CAST(e.sits AS TEXT) FROM files_doomed e WHERE e.root_id = sqlc.arg(root_id) AND e.sits IS NOT NULL
ORDER BY 1;

-- name: PathTaken :one
SELECT EXISTS (
    SELECT 1 FROM files_file a WHERE a.root_id = sqlc.arg(root_id) AND a.path = sqlc.arg(path)
    UNION ALL SELECT 1 FROM files_move m WHERE m.sits = sqlc.arg(path) AND EXISTS (SELECT 1 FROM files_file f WHERE f.id = m.file_id AND f.root_id = sqlc.arg(root_id))
    UNION ALL SELECT 1 FROM files_doomed d WHERE d.root_id = sqlc.arg(root_id) AND (d.path = sqlc.arg(path) OR d.sits = sqlc.arg(path)));

-- name: ListGone :many
SELECT root_id, path FROM files_file WHERE gone_at IS NOT NULL ORDER BY root_id, gone_at, id;

-- name: ListUnreadable :many
SELECT root_id, path, CAST(probe_error AS TEXT) AS line FROM files_file
WHERE probe_error IS NOT NULL AND gone_at IS NULL ORDER BY root_id, id;

-- name: CountGoneOfWorks :many
SELECT work_id, count(*) AS files FROM files_file WHERE gone_at IS NOT NULL AND work_id IN (sqlc.slice(work_ids)) GROUP BY work_id ORDER BY work_id;

-- name: CountEveryGone :many
SELECT work_id, count(*) AS files FROM files_file WHERE gone_at IS NOT NULL GROUP BY work_id ORDER BY work_id;

-- name: ListLeftIn :many
SELECT f.* FROM files_file f
WHERE f.work_id IN (sqlc.slice(work_ids)) AND f.gone_at IS NULL AND NOT EXISTS (SELECT 1 FROM files_move m WHERE m.file_id = f.id)
ORDER BY f.id;

-- name: ListFilesByID :many
SELECT * FROM files_file WHERE id IN (sqlc.slice(ids)) ORDER BY id;

-- name: ForgetGoneOfRoot :exec
DELETE FROM files_file WHERE root_id = sqlc.arg(root_id) AND gone_at IS NOT NULL;

-- name: ForgetGoneOfWork :exec
DELETE FROM files_file WHERE work_id = sqlc.arg(work_id) AND gone_at IS NOT NULL;
