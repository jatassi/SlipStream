-- name: ListFinishedBefore :many
SELECT id, work_id, torrent_id, stage, failure_code, source, updated_at FROM grabs_grab INDEXED BY grabs_grab_finished
WHERE stage IN ('imported', 'failed') AND updated_at < sqlc.arg(before)
    AND updated_at >= coalesce(sqlc.narg(after_at), 0)
    AND (updated_at > coalesce(sqlc.narg(after_at), 0) OR id > sqlc.narg(after_id))
ORDER BY updated_at, id LIMIT sqlc.arg(limit);

-- name: DeleteGrabs :exec
DELETE FROM grabs_grab WHERE id IN (sqlc.slice(ids));
