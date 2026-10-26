-- name: DeleteWorks :exec
DELETE FROM works_work WHERE id IN (sqlc.slice(ids));
