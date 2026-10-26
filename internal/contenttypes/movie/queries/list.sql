-- name: ListListedOf :many
SELECT work_id, year, runtime_minutes, certification FROM movie_movie WHERE work_id IN (sqlc.slice(work_ids));

-- name: ListEveryListed :many
SELECT work_id, year, runtime_minutes, certification FROM movie_movie;

-- name: ListGenresOf :many
SELECT work_id, name FROM movie_genre WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id, position;

-- name: ListEveryGenre :many
SELECT work_id, name FROM movie_genre ORDER BY work_id, position;
