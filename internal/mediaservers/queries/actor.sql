-- The sender of each server: it takes due rows, records each step, and
-- writes the server's sends columns.

-- name: ListServerIDs :many
SELECT id FROM mediaservers_server ORDER BY id;

-- name: ListDue :many
SELECT * FROM mediaservers_change
WHERE server_id = sqlc.arg(server_id) AND state = 'due'
ORDER BY created_at, id;

-- name: TakeChanges :exec
UPDATE mediaservers_change SET state = 'sending'
WHERE server_id = sqlc.arg(server_id) AND state = 'due' AND id IN (sqlc.slice(ids));

-- name: GetChanges :many
SELECT * FROM mediaservers_change WHERE id IN (sqlc.slice(ids)) ORDER BY created_at, id;

-- name: ListSending :many
SELECT * FROM mediaservers_change WHERE state = 'sending' ORDER BY server_id, created_at, id;

-- CoverLibrary marks sent every due refresh in roots, the roots of one
-- library, that existed before a whole library refresh of it was sent.
-- name: CoverLibrary :exec
UPDATE mediaservers_change
SET state = 'sent', sent_at = sqlc.arg(at), result = 'covered', problem = NULL
WHERE server_id = sqlc.arg(server_id) AND state = 'due' AND from_path IS NULL AND kind <> 'library'
    AND created_at <= sqlc.arg(before) AND root_id IN (sqlc.slice(roots));

-- name: SetSendsOK :exec
UPDATE mediaservers_server SET sends = 'ok', sends_since = NULL, last_send_at = sqlc.arg(at)
WHERE id = sqlc.arg(id);

-- name: SetSendsFailing :exec
UPDATE mediaservers_server SET sends = 'failing', sends_since = coalesce(sends_since, sqlc.arg(at))
WHERE id = sqlc.arg(id);
