-- name: ListAddedByScanSince :many
SELECT id, label FROM works_work
WHERE added_by = 'scan' AND added_at > coalesce(sqlc.arg(since), 0)
ORDER BY added_at, id;

-- name: ListTallyRows :many
SELECT id, content_type, label, sort_label FROM works_work
WHERE id IN (sqlc.slice(ids))
ORDER BY id;
