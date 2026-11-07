-- name: ListWorksOf :many
SELECT * FROM works_work WHERE id IN (sqlc.slice(ids)) ORDER BY id;

-- name: ListWorksToMonitor :many
SELECT DISTINCT work_id FROM works_item
WHERE monitored <> sqlc.arg(monitored) AND work_id IN (sqlc.slice(work_ids))
ORDER BY work_id;

-- name: MonitorWorks :exec
UPDATE works_item SET monitored = sqlc.arg(monitored) WHERE work_id IN (sqlc.slice(work_ids));
