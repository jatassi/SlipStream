-- name: CountUnfinishedOfWorks :many
SELECT work_id, count(*) AS grabs FROM grabs_grab
WHERE stage NOT IN ('imported', 'failed') AND work_id IN (sqlc.slice(work_ids))
GROUP BY work_id ORDER BY work_id;

-- name: ListGrabsOfWorks :many
SELECT id FROM grabs_grab WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY id;
