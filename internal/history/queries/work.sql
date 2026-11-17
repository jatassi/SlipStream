-- name: InsertWork :exec
INSERT INTO history_work (event_id, work_id) VALUES (sqlc.arg(event_id), sqlc.arg(work_id));

-- name: ListWorksOf :many
SELECT * FROM history_work WHERE event_id IN (sqlc.slice(ids)) ORDER BY event_id, work_id;
