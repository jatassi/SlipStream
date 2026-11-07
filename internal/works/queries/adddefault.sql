-- name: GetAddMonitor :one
SELECT monitor FROM works_add_default WHERE content_type = sqlc.arg(content_type);

-- name: SetAddMonitor :exec
INSERT INTO works_add_default (content_type, monitor)
VALUES (sqlc.arg(content_type), sqlc.arg(monitor))
ON CONFLICT (content_type) DO UPDATE SET monitor = excluded.monitor;

-- name: ClearAddMonitor :exec
DELETE FROM works_add_default WHERE content_type = sqlc.arg(content_type);
