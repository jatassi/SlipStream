-- name: InsertChange :exec
INSERT INTO mediaservers_change (
    id, server_id, kind, root_id, path, from_path, file_id, replaced_by, upgrade,
    state, step, step_at, snapshot, next_attempt_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
