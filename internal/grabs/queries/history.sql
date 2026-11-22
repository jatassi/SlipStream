-- name: ListExisting :many
SELECT id FROM grabs_grab WHERE id IN (sqlc.slice(ids)) ORDER BY id;

-- name: ListFailed :many
SELECT id, work_id, failure_code, source, created_at FROM grabs_grab
WHERE stage = 'failed' ORDER BY created_at DESC;
