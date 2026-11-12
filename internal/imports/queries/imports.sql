-- name: GetImport :one
SELECT * FROM imports_import WHERE id = ?;

-- name: GetRootImportAt :one
SELECT * FROM imports_import WHERE source = 'root' AND root_id = ? AND folder = ? AND finished_at IS NULL;

-- name: ListUnfinishedRootImports :many
SELECT * FROM imports_import WHERE source = 'root' AND finished_at IS NULL ORDER BY root_id, folder;

-- name: ListFiles :many
SELECT * FROM imports_file ORDER BY import_id, position;

-- name: ListFilesOf :many
SELECT * FROM imports_file WHERE import_id = ? ORDER BY position;

-- name: ListFileItems :many
SELECT * FROM imports_file_item ORDER BY import_id, position, "order";

-- name: ListFileItemsOf :many
SELECT * FROM imports_file_item WHERE import_id = ? ORDER BY position, "order";

-- name: InsertImport :exec
INSERT INTO imports_import (id, source, grab_id, node_id, info_hash, root_id, folder, content_type, hint, suggestion, work_id, edition,
                            move_files, failure_code, ended, created_at, ignored_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateImport :exec
UPDATE imports_import SET hint = ?, suggestion = ?, work_id = ?, edition = ?, move_files = ?, failure_code = ?, ended = ?,
                          ignored_at = ?, finished_at = ?
WHERE id = ?;

-- name: DeleteImport :exec
DELETE FROM imports_import WHERE id = ?;

-- name: InsertFile :exec
INSERT INTO imports_file (import_id, position, path, source_index, size, modified_at, node_file_id, node_volume_id, probe, quality, cut,
                          release_name, source_name, outcome, reason, hold, confirmed, root_id, dest, sent_transfer, sent_order,
                          placed_by, file_id, source_removed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetFile :execrows
UPDATE imports_file SET outcome = sqlc.arg(outcome), reason = sqlc.arg(reason), hold = sqlc.arg(hold), confirmed = sqlc.arg(confirmed),
                        probe = sqlc.arg(probe), quality = sqlc.arg(quality), cut = sqlc.arg(cut), release_name = sqlc.arg(release_name),
                        root_id = sqlc.arg(root_id), dest = sqlc.arg(dest), sent_transfer = sqlc.arg(sent_transfer),
                        sent_order = sqlc.arg(sent_order), placed_by = sqlc.arg(placed_by), file_id = sqlc.arg(file_id),
                        source_removed_at = sqlc.arg(source_removed_at)
WHERE import_id = sqlc.arg(import_id) AND position = sqlc.arg(position) AND outcome = sqlc.arg(was);

-- name: DeleteFile :exec
DELETE FROM imports_file WHERE import_id = ? AND position = ?;

-- name: InsertFileItem :exec
INSERT INTO imports_file_item (import_id, position, item_id, "order") VALUES (?, ?, ?, ?);

-- name: DeleteFileItems :exec
DELETE FROM imports_file_item WHERE import_id = ? AND position = ?;

-- name: ListRootsOf :many
SELECT DISTINCT root_id FROM imports_import WHERE id IN (sqlc.slice(ids)) AND root_id IS NOT NULL ORDER BY root_id;

-- name: Ignore :exec
UPDATE imports_import SET ignored_at = coalesce(ignored_at, sqlc.arg(ignored_at)) WHERE id IN (sqlc.slice(ids)) AND finished_at IS NULL;

-- name: Unignore :exec
UPDATE imports_import SET ignored_at = NULL WHERE id IN (sqlc.slice(ids));

-- name: CountNeedsYou :one
SELECT count(*) FROM imports_import WHERE source = 'root' AND finished_at IS NULL AND ignored_at IS NULL;

-- name: CountNeedsYouByRoot :many
SELECT root_id, count(*) AS entries FROM imports_import WHERE source = 'root' AND finished_at IS NULL AND ignored_at IS NULL
GROUP BY root_id ORDER BY root_id;

-- name: ListCompared :many
SELECT * FROM imports_root ORDER BY root_id;

-- name: GetCompared :one
SELECT * FROM imports_root WHERE root_id = ?;

-- name: RecordComparison :exec
INSERT INTO imports_root (root_id, place, compared_at, found, changed, gone, returned, added, stopped_gone, stopped_of)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (root_id) DO UPDATE SET
    place = excluded.place, compared_at = excluded.compared_at, found = excluded.found, changed = excluded.changed, gone = excluded.gone,
    returned = excluded.returned, added = excluded.added, stopped_gone = excluded.stopped_gone, stopped_of = excluded.stopped_of;

-- name: ConsumeConfirmed :exec
UPDATE imports_root SET confirmed = NULL WHERE root_id = ? AND confirmed = ?;

-- name: ConfirmGone :execrows
UPDATE imports_root SET confirmed = stopped_gone WHERE root_id = ? AND stopped_gone IS NOT NULL;

-- name: GetSettings :one
SELECT * FROM imports_settings;

-- name: SetSettings :exec
UPDATE imports_settings SET stop_share = ?, probes = ?;

-- name: SetImportingSettings :exec
UPDATE imports_settings SET video_extensions = ?, smallest_video = ?, sample_shorter = ?;

-- name: ListUnfinishedPaths :many
SELECT f.path FROM imports_file f JOIN imports_import i ON i.id = f.import_id
WHERE i.finished_at IS NULL AND f.outcome NOT IN ('left_out', 'not_better', 'placed', 'declined', 'failed') ORDER BY f.import_id, f.position;

-- name: ListFills :many
SELECT * FROM imports_file_fill ORDER BY import_id, position, item_id, edition, target_id;

-- name: ListFillsOf :many
SELECT * FROM imports_file_fill WHERE import_id = ? ORDER BY position, item_id, edition, target_id;

-- name: InsertFill :exec
INSERT INTO imports_file_fill (import_id, position, item_id, edition, target_id, forced) VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteFills :exec
DELETE FROM imports_file_fill WHERE import_id = ? AND position = ?;

-- name: ListBusy :many
SELECT DISTINCT fi.item_id, f.import_id FROM imports_file_item fi
JOIN imports_file f ON f.import_id = fi.import_id AND f.position = fi.position
JOIN imports_import i ON i.id = f.import_id
WHERE i.finished_at IS NULL AND i.work_id IS NOT NULL AND fi.item_id IN (sqlc.slice(item_ids)) AND f.import_id <> sqlc.arg(except)
  AND (f.outcome IN ('sending', 'placing') OR (f.outcome = 'pending' AND i.created_at < sqlc.arg(created)))
ORDER BY fi.item_id, f.import_id;

-- name: ListImporting :many
SELECT DISTINCT fi.item_id, f.import_id FROM imports_file_item fi
JOIN imports_file f ON f.import_id = fi.import_id AND f.position = fi.position
JOIN imports_import i ON i.id = f.import_id
WHERE i.finished_at IS NULL AND i.work_id IS NOT NULL AND fi.item_id IN (sqlc.slice(item_ids))
  AND f.outcome IN ('pending', 'waiting', 'sending', 'placing')
ORDER BY fi.item_id, f.import_id;

-- name: ListCopies :many
SELECT f.import_id, f.position, f.root_id, f.size FROM imports_file f JOIN imports_import i ON i.id = f.import_id
WHERE i.finished_at IS NULL AND f.outcome = 'placing' AND f.placed_by = 'copy' ORDER BY f.import_id, f.position;

-- name: ListUnfinished :many
SELECT * FROM imports_import WHERE finished_at IS NULL AND source <> 'root' OR finished_at IS NULL AND id IN (
    SELECT import_id FROM imports_file WHERE outcome IN ('pending', 'placing') OR outcome = 'waiting' AND json_extract(reason, '$.code') = 'busy'
) ORDER BY id;

-- name: ListHolding :many
SELECT f.root_id, f.dest FROM imports_file f JOIN imports_import i ON i.id = f.import_id
WHERE i.finished_at IS NULL AND f.dest IS NOT NULL AND i.source <> 'root'
  AND (f.outcome IN ('pending', 'waiting', 'sending', 'placing') OR f.outcome = 'failed' AND json_extract(f.reason, '$.code') = 'import_failed')
ORDER BY f.root_id, f.dest;

-- name: ListUnfinishedOf :many
SELECT id FROM imports_import WHERE finished_at IS NULL AND work_id IN (sqlc.slice(work_ids)) ORDER BY id;

-- name: CountUnfinishedInto :one
SELECT count(*) FROM imports_import i
WHERE i.finished_at IS NULL AND (
    i.source <> 'root' AND (i.work_id IN (sqlc.slice(work_ids)) OR EXISTS (SELECT 1 FROM imports_file f WHERE f.import_id = i.id AND f.root_id = sqlc.arg(root_id)))
    OR i.source = 'root' AND i.root_id = sqlc.arg(root_id) AND EXISTS (SELECT 1 FROM imports_file f WHERE f.import_id = i.id AND f.outcome IN ('pending', 'placing'))
);

-- name: ForgetRoot :exec
UPDATE imports_file SET root_id = NULL, dest = NULL WHERE imports_file.root_id = ? AND import_id IN (SELECT id FROM imports_import WHERE finished_at IS NOT NULL);
