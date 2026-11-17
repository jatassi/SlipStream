-- name: InsertWork :exec
INSERT INTO history_work (event_id, work_id) VALUES (sqlc.arg(event_id), sqlc.arg(work_id));
