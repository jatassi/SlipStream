-- name: PutStaging :exec
INSERT INTO transfers_staging (transfer_id, folder) VALUES (sqlc.arg(transfer_id), sqlc.arg(folder))
ON CONFLICT (transfer_id) DO UPDATE SET folder = excluded.folder;

-- name: Staging :one
SELECT folder FROM transfers_staging WHERE transfer_id = sqlc.arg(transfer_id);

-- name: ListStaging :many
SELECT transfer_id, folder FROM transfers_staging ORDER BY transfer_id;

-- name: DeleteStaging :exec
DELETE FROM transfers_staging WHERE transfer_id = sqlc.arg(transfer_id);

-- name: Resume :one
SELECT * FROM transfers_resume
WHERE transfer_id = sqlc.arg(transfer_id) AND position = sqlc.arg(position) AND place = sqlc.arg(place);

-- name: PutResume :exec
INSERT INTO transfers_resume (transfer_id, position, place, size, mtime_ns, file_key, verified)
VALUES (sqlc.arg(transfer_id), sqlc.arg(position), sqlc.arg(place), sqlc.arg(size), sqlc.arg(mtime_ns), sqlc.arg(file_key), sqlc.arg(verified))
ON CONFLICT (transfer_id, position, place) DO UPDATE
SET size = excluded.size, mtime_ns = excluded.mtime_ns, file_key = excluded.file_key, verified = excluded.verified, done = 0;

-- name: DeleteResumeElsewhere :exec
DELETE FROM transfers_resume
WHERE transfer_id = sqlc.arg(transfer_id) AND position = sqlc.arg(position) AND place <> sqlc.arg(place);

-- name: SetVerified :exec
UPDATE transfers_resume SET verified = sqlc.arg(verified), done = sqlc.arg(done)
WHERE transfer_id = sqlc.arg(transfer_id) AND position = sqlc.arg(position) AND place = sqlc.arg(place);
