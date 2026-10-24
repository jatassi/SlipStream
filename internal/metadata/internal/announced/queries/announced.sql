-- name: GetDay :one
SELECT day FROM metadata_announced WHERE id = 1;

-- name: PutDay :exec
INSERT INTO metadata_announced (id, day) VALUES (1, sqlc.arg(day))
ON CONFLICT (id) DO UPDATE SET day = excluded.day;
