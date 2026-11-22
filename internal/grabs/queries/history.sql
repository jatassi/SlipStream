-- name: ListExisting :many
SELECT id FROM grabs_grab WHERE id IN (sqlc.slice(ids)) ORDER BY id;

-- name: ListFailed :many
SELECT id, work_id, failure_code, source, created_at FROM grabs_grab INDEXED BY grabs_grab_failing
WHERE stage = 'failed' AND replaced = 0 ORDER BY created_at DESC;
