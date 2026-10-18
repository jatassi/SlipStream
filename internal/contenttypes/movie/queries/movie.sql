-- name: UpsertMovie :exec
INSERT INTO movie_movie (work_id, tmdb_id, imdb_id, title, year, released, runtime_minutes)
VALUES (sqlc.arg(work_id), sqlc.arg(tmdb_id), sqlc.arg(imdb_id), sqlc.arg(title), sqlc.arg(year), sqlc.arg(released), sqlc.arg(runtime_minutes))
ON CONFLICT (work_id) DO UPDATE SET imdb_id = excluded.imdb_id, title = excluded.title, year = excluded.year,
    released = excluded.released, runtime_minutes = excluded.runtime_minutes;

-- name: GetMovie :one
SELECT * FROM movie_movie WHERE work_id = sqlc.arg(work_id);

-- name: ListMovies :many
SELECT * FROM movie_movie WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;
