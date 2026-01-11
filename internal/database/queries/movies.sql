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
    path, root_folder_id, quality_profile_id, monitored, status,
    release_date, digital_release_date, physical_release_date, released
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
    release_date = ?,
    digital_release_date = ?,
    physical_release_date = ?,
    released = ?,
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
INSERT INTO movie_files (movie_id, path, size, quality, quality_id, video_codec, audio_codec, resolution)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
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

-- Calendar queries
-- name: GetMoviesInDateRange :many
SELECT * FROM movies
WHERE (release_date BETWEEN ? AND ?)
   OR (digital_release_date BETWEEN ? AND ?)
   OR (physical_release_date BETWEEN ? AND ?)
ORDER BY COALESCE(release_date, digital_release_date, physical_release_date);

-- name: UpdateMovieReleaseDates :exec
UPDATE movies SET
    release_date = ?,
    digital_release_date = ?,
    physical_release_date = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- Availability queries
-- name: UpdateMoviesReleasedByDate :execresult
UPDATE movies SET released = 1, updated_at = CURRENT_TIMESTAMP
WHERE released = 0 AND release_date IS NOT NULL AND release_date <= date('now');

-- name: UpdateMovieReleased :exec
UPDATE movies SET released = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: GetUnreleasedMoviesWithPastDate :many
SELECT * FROM movies
WHERE released = 0 AND release_date IS NOT NULL AND release_date <= date('now');

-- name: UpdateMovieAvailabilityStatus :exec
UPDATE movies SET availability_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateAllMoviesAvailabilityStatus :execresult
UPDATE movies SET
    availability_status = CASE WHEN released = 1 THEN 'Available' ELSE 'Unreleased' END,
    updated_at = CURRENT_TIMESTAMP;

-- Missing movies queries
-- name: ListMissingMovies :many
SELECT m.* FROM movies m
LEFT JOIN movie_files mf ON m.id = mf.movie_id
WHERE m.released = 1 AND m.monitored = 1 AND mf.id IS NULL
ORDER BY m.release_date DESC;

-- name: CountMissingMovies :one
SELECT COUNT(*) FROM movies m
LEFT JOIN movie_files mf ON m.id = mf.movie_id
WHERE m.released = 1 AND m.monitored = 1 AND mf.id IS NULL;

-- Upgrade candidate queries (movies with files below quality cutoff)
-- name: ListMovieUpgradeCandidates :many
SELECT m.*, mf.id as file_id, mf.quality_id as current_quality_id, qp.cutoff
FROM movies m
JOIN movie_files mf ON m.id = mf.movie_id
JOIN quality_profiles qp ON m.quality_profile_id = qp.id
WHERE m.monitored = 1
  AND m.released = 1
  AND mf.quality_id IS NOT NULL
  AND mf.quality_id < qp.cutoff
ORDER BY m.release_date DESC;

-- name: CountMovieUpgradeCandidates :one
SELECT COUNT(*) FROM movies m
JOIN movie_files mf ON m.id = mf.movie_id
JOIN quality_profiles qp ON m.quality_profile_id = qp.id
WHERE m.monitored = 1
  AND m.released = 1
  AND mf.quality_id IS NOT NULL
  AND mf.quality_id < qp.cutoff;

-- name: GetMovieWithFileQuality :one
SELECT m.*, mf.id as file_id, mf.quality_id as current_quality_id
FROM movies m
LEFT JOIN movie_files mf ON m.id = mf.movie_id
WHERE m.id = ?
LIMIT 1;

-- name: UpdateMovieFileQualityID :exec
UPDATE movie_files SET quality_id = ? WHERE id = ?;
