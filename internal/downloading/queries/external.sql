-- name: ListExternal :many
SELECT client_id, info_hash, origin, sent_at, resends, seen, applied, finished_at, completed
FROM downloading_external WHERE client_id = ? ORDER BY info_hash;

-- name: PutExternal :exec
INSERT INTO downloading_external (client_id, info_hash, origin, sent_at, resends, seen, applied, finished_at, completed)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (client_id, info_hash) DO UPDATE SET
    origin = excluded.origin, sent_at = excluded.sent_at, resends = excluded.resends, seen = excluded.seen,
    applied = excluded.applied, finished_at = excluded.finished_at, completed = excluded.completed;

-- name: DeleteExternal :exec
DELETE FROM downloading_external WHERE client_id = ? AND info_hash = ?;

-- name: DeleteClientExternals :exec
DELETE FROM downloading_external WHERE client_id = ?;

-- name: DeleteExternals :exec
DELETE FROM downloading_external;
