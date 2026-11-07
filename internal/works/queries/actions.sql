-- name: ListWorksOf :many
SELECT * FROM works_work WHERE id IN (sqlc.slice(ids)) ORDER BY id;

-- name: CountMonitoredOfWorks :many
SELECT work_id, aside, monitored, count(*) AS items FROM works_item
WHERE work_id IN (sqlc.slice(work_ids)) AND withdrawn_at IS NULL
GROUP BY work_id, aside, monitored
ORDER BY work_id;

-- name: CountEveryMonitored :many
SELECT work_id, aside, monitored, count(*) AS items FROM works_item
WHERE withdrawn_at IS NULL
GROUP BY work_id, aside, monitored
ORDER BY work_id;

-- name: SetMonitoredOfWorks :exec
UPDATE works_item SET monitored = sqlc.arg(monitored)
WHERE aside = sqlc.arg(aside) AND withdrawn_at IS NULL AND work_id IN (sqlc.slice(work_ids));

-- name: SetItemsMonitored :exec
UPDATE works_item SET monitored = sqlc.arg(monitored) WHERE id IN (sqlc.slice(item_ids));

-- name: SetWorkRules :exec
UPDATE works_work SET monitor_new = sqlc.arg(monitor_new), monitor_asides = sqlc.arg(monitor_asides) WHERE id IN (sqlc.slice(ids));

-- name: SetMonitorNew :exec
UPDATE works_work SET monitor_new = sqlc.arg(monitor_new) WHERE id IN (sqlc.slice(ids));
