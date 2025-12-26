-- name: GetMovie :one
SELECT * FROM movies WHERE id = ? LIMIT 1;

-- name: GetMovieByTmdbID :one
SELECT * FROM movies WHERE tmdb_id = ? LIMIT 1;

-- name: ListMovies :many
SELECT * FROM movies ORDER BY sort_title;

-- name: ListMonitoredMovies :many
SELECT * FROM movies WHERE monitored = 1 ORDER BY sort_title;

-- name: CreateMovie :one
INSERT INTO movies (
    title, sort_title, year, tmdb_id, imdb_id, overview, runtime,
    path, root_folder_id, quality_profile_id, monitored, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateMovie :one
UPDATE movies SET
    title = ?,
    sort_title = ?,
    year = ?,
    tmdb_id = ?,
    imdb_id = ?,
    overview = ?,
    runtime = ?,
    path = ?,
    root_folder_id = ?,
    quality_profile_id = ?,
    monitored = ?,
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteMovie :exec
DELETE FROM movies WHERE id = ?;

-- name: DeleteMoviesByRootFolder :exec
DELETE FROM movies WHERE root_folder_id = ?;

-- name: CountMovies :one
SELECT COUNT(*) FROM movies;

-- name: CountMonitoredMovies :one
SELECT COUNT(*) FROM movies WHERE monitored = 1;

-- name: SearchMovies :many
SELECT * FROM movies
WHERE title LIKE ? OR sort_title LIKE ?
ORDER BY sort_title
LIMIT ? OFFSET ?;

-- name: ListMoviesPaginated :many
SELECT * FROM movies
ORDER BY sort_title
LIMIT ? OFFSET ?;

-- name: GetMovieByPath :one
SELECT * FROM movies WHERE path = ? LIMIT 1;

-- name: ListMoviesByRootFolder :many
SELECT * FROM movies WHERE root_folder_id = ? ORDER BY sort_title;

-- name: UpdateMovieStatus :exec
UPDATE movies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- Movie Files
-- name: GetMovieFile :one
SELECT * FROM movie_files WHERE id = ? LIMIT 1;

-- name: ListMovieFiles :many
SELECT * FROM movie_files WHERE movie_id = ? ORDER BY path;

-- name: CreateMovieFile :one
INSERT INTO movie_files (movie_id, path, size, quality, video_codec, audio_codec, resolution)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteMovieFile :exec
DELETE FROM movie_files WHERE id = ?;

-- name: DeleteMovieFilesByMovie :exec
DELETE FROM movie_files WHERE movie_id = ?;

-- name: GetMovieFileByPath :one
SELECT * FROM movie_files WHERE path = ? LIMIT 1;

-- name: CountMovieFiles :one
SELECT COUNT(*) FROM movie_files WHERE movie_id = ?;

-- name: ListUnmatchedMoviesByRootFolder :many
SELECT * FROM movies
WHERE root_folder_id = ?
  AND (tmdb_id IS NULL OR tmdb_id = 0)
ORDER BY sort_title;
