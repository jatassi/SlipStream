-- name: ListGenresOf :many
SELECT work_id, name FROM movie_genre WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id, position;
