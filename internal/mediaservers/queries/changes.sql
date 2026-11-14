-- Changed and Prepare fold each file change into the rows below, and the
-- API lists them.

-- name: ListShowing :many
SELECT server_id FROM mediaservers_mapping
WHERE root_id = sqlc.arg(root_id) AND shown = 1
ORDER BY server_id;

-- name: OpenCarry :many
SELECT * FROM mediaservers_change
WHERE server_id = sqlc.arg(server_id) AND file_id = sqlc.arg(file_id) AND from_path IS NOT NULL
    AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted')
ORDER BY created_at, id;

-- name: OpenAtPaths :many
SELECT * FROM mediaservers_change
WHERE server_id = sqlc.arg(server_id) AND root_id = sqlc.arg(root_id) AND from_path IS NULL
    AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted')
    AND path IN (sqlc.slice(paths))
ORDER BY created_at, id;

-- OpenPartner is the other half of an upgrade: for a removed naming its
-- replacer, that file's added; for an added, the removed naming its file.
-- name: OpenPartner :many
SELECT * FROM mediaservers_change
WHERE server_id = sqlc.arg(server_id) AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted')
    AND ((kind = 'added' AND upgrade = 1 AND file_id = sqlc.narg(replacer))
        OR (kind = 'removed' AND replaced_by = sqlc.narg(added)))
ORDER BY created_at, id;

-- name: InsertChange :exec
INSERT INTO mediaservers_change (
    id, server_id, kind, root_id, path, from_path, file_id, replaced_by, upgrade,
    state, step, step_at, snapshot, attempts, next_attempt_at, created_at, sent_at, result, lost, problem
) VALUES (
    sqlc.arg(id), sqlc.arg(server_id), sqlc.arg(kind), sqlc.arg(root_id), sqlc.arg(path), sqlc.narg(from_path),
    sqlc.arg(file_id), sqlc.arg(replaced_by), sqlc.arg(upgrade), sqlc.arg(state), sqlc.arg(step), sqlc.arg(step_at),
    sqlc.narg(snapshot), sqlc.arg(attempts), sqlc.arg(next_attempt_at), sqlc.arg(created_at), sqlc.arg(sent_at),
    sqlc.narg(result), sqlc.arg(lost), sqlc.narg(problem)
);

-- UpdateChange writes every column a fold or a step moves. A row keeps
-- its id, server, root and age.
-- name: UpdateChange :exec
UPDATE mediaservers_change
SET kind = sqlc.arg(kind),
    path = sqlc.arg(path),
    from_path = sqlc.narg(from_path),
    file_id = sqlc.arg(file_id),
    replaced_by = sqlc.arg(replaced_by),
    upgrade = sqlc.arg(upgrade),
    state = sqlc.arg(state),
    step = sqlc.arg(step),
    step_at = sqlc.arg(step_at),
    snapshot = sqlc.narg(snapshot),
    attempts = sqlc.arg(attempts),
    next_attempt_at = sqlc.arg(next_attempt_at),
    sent_at = sqlc.arg(sent_at),
    result = sqlc.narg(result),
    lost = sqlc.arg(lost),
    problem = sqlc.narg(problem)
WHERE id = sqlc.arg(id);

-- name: DeleteChange :exec
DELETE FROM mediaservers_change WHERE id = sqlc.arg(id);

-- ListUnsent is every unsent row of root at or from one of paths, or of
-- one of files. Prepare leaves the snapshot of a rename such a row meets
-- to that row's own send.
-- name: ListUnsent :many
SELECT * FROM mediaservers_change
WHERE root_id = sqlc.arg(root_id) AND state <> 'sent'
    AND (path IN (sqlc.slice(paths)) OR from_path IN (sqlc.slice(origins)) OR file_id IN (sqlc.slice(files)))
ORDER BY created_at, id;

-- name: DeletePrepared :exec
DELETE FROM mediaservers_change WHERE file_id = sqlc.arg(file_id) AND state = 'prepared';

-- DeleteStalePrepared deletes the prepared rows of files no pending
-- rename names any more: a plan cancelled or replaced.
-- name: DeleteStalePrepared :exec
DELETE FROM mediaservers_change WHERE state = 'prepared' AND file_id NOT IN (sqlc.slice(pending));

-- name: DeleteAllPrepared :exec
DELETE FROM mediaservers_change WHERE state = 'prepared';

-- name: ListChanges :many
SELECT * FROM mediaservers_change WHERE server_id = sqlc.arg(server_id) ORDER BY id DESC LIMIT sqlc.arg(n);

-- name: ListChangesBefore :many
SELECT * FROM mediaservers_change
WHERE server_id = sqlc.arg(server_id) AND id < sqlc.arg(before)
ORDER BY id DESC
LIMIT sqlc.arg(n);

-- name: PruneSent :execrows
DELETE FROM mediaservers_change WHERE state = 'sent' AND sent_at < sqlc.arg(before);
