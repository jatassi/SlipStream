-- name: GetEntry :one
SELECT * FROM imports_pending WHERE id = ?;

-- name: GetEntryAt :one
SELECT * FROM imports_pending WHERE root_id = ? AND folder = ?;

-- name: ListEntries :many
SELECT * FROM imports_pending ORDER BY root_id, folder;

-- name: ListFiles :many
SELECT * FROM imports_pending_file ORDER BY pending_id, path;

-- name: ListFilesOf :many
SELECT * FROM imports_pending_file WHERE pending_id = ? ORDER BY path;

-- name: InsertEntry :exec
INSERT INTO imports_pending (id, root_id, folder, content_type, hint, suggestion, found_at, ignored_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateEntry :exec
UPDATE imports_pending SET hint = ?, suggestion = ?, ignored_at = ? WHERE id = ?;

-- name: DeleteEntry :exec
DELETE FROM imports_pending WHERE id = ?;

-- name: InsertEntryFile :exec
INSERT INTO imports_pending_file (pending_id, path, size, modified_at, node_file_id, node_volume_id, probe, quality, cut, items, reason)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteEntryFiles :exec
DELETE FROM imports_pending_file WHERE pending_id = ?;

-- name: ListRootsOf :many
SELECT DISTINCT root_id FROM imports_pending WHERE id IN (sqlc.slice(ids)) ORDER BY root_id;

-- name: Ignore :exec
UPDATE imports_pending SET ignored_at = coalesce(ignored_at, sqlc.arg(ignored_at)) WHERE id IN (sqlc.slice(ids));

-- name: Unignore :exec
UPDATE imports_pending SET ignored_at = NULL WHERE id IN (sqlc.slice(ids));

-- name: CountNeedsYou :one
SELECT count(*) FROM imports_pending WHERE ignored_at IS NULL;

-- name: CountNeedsYouByRoot :many
SELECT root_id, count(*) AS entries FROM imports_pending WHERE ignored_at IS NULL GROUP BY root_id ORDER BY root_id;

-- name: ListCompared :many
SELECT * FROM imports_root ORDER BY root_id;

-- name: GetCompared :one
SELECT * FROM imports_root WHERE root_id = ?;

-- name: RecordComparison :exec
INSERT INTO imports_root (root_id, place, compared_at, found, gone, returned, added, stopped_gone, stopped_of)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (root_id) DO UPDATE SET
    place = excluded.place, compared_at = excluded.compared_at, found = excluded.found, gone = excluded.gone,
    returned = excluded.returned, added = excluded.added, stopped_gone = excluded.stopped_gone, stopped_of = excluded.stopped_of;

-- name: ConsumeConfirmed :exec
UPDATE imports_root SET confirmed = NULL WHERE root_id = ? AND confirmed = ?;

-- name: ConfirmGone :execrows
UPDATE imports_root SET confirmed = stopped_gone WHERE root_id = ? AND stopped_gone IS NOT NULL;

-- name: GetSettings :one
SELECT * FROM imports_settings;

-- name: SetSettings :exec
UPDATE imports_settings SET stop_share = ?, probes = ?;
