-- name: GetMovie :one
SELECT * FROM movies WHERE id = ? LIMIT 1;

-- name: GetMovieWithAddedBy :one
SELECT m.*, pu.username AS added_by_username FROM movies m
LEFT JOIN portal_users pu ON m.added_by = pu.id
WHERE m.id = ? LIMIT 1;

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
    release_date, physical_release_date, theatrical_release_date,
    studio, tvdb_id, content_rating, added_by
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
    physical_release_date = ?,
    theatrical_release_date = ?,
    studio = ?,
    tvdb_id = ?,
    content_rating = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: GetMovieByTvdbID :one
SELECT * FROM movies WHERE tvdb_id = ? LIMIT 1;

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
WHERE title LIKE sqlc.arg(search_term) OR sort_title LIKE sqlc.arg(search_term)
   OR REPLACE(title, '''', '') LIKE sqlc.arg(search_term)
   OR REPLACE(sort_title, '''', '') LIKE sqlc.arg(search_term)
ORDER BY sort_title
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: ListMoviesPaginated :many
SELECT * FROM movies
ORDER BY sort_title
LIMIT ? OFFSET ?;

-- name: GetMovieByPath :one
SELECT * FROM movies WHERE path = ? LIMIT 1;

-- name: ListMoviesByRootFolder :many
SELECT * FROM movies WHERE root_folder_id = ? ORDER BY sort_title;

-- name: UpdateMovieMonitored :exec
UPDATE movies SET monitored = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateMovieStatus :exec
UPDATE movies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateMovieStatusWithDetails :exec
UPDATE movies SET
    status = ?,
    active_download_id = ?,
    status_message = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- Movie Files
-- name: GetMovieFile :one
SELECT * FROM movie_files WHERE id = ? LIMIT 1;

-- name: ListMovieFiles :many
SELECT * FROM movie_files WHERE movie_id = ? ORDER BY path;

-- name: CreateMovieFile :one
INSERT INTO movie_files (movie_id, path, size, quality, quality_id, video_codec, audio_codec, resolution, audio_channels, dynamic_range)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
   OR (physical_release_date BETWEEN ? AND ?)
   OR (theatrical_release_date BETWEEN ? AND ?)
ORDER BY COALESCE(release_date, physical_release_date, theatrical_release_date);

-- name: UpdateMovieReleaseDates :exec
UPDATE movies SET
    release_date = ?,
    physical_release_date = ?,
    theatrical_release_date = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- Status refresh queries
-- Note: substr(col, 1, 10) is required because the Go SQLite driver (modernc.org/sqlite)
-- stores time.Time in RFC3339 format with timezone offset (e.g. '2026-02-10T00:00:00-07:00').
-- SQLite's date() function cannot parse this format (returns NULL), and raw string comparison
-- fails on same-day dates since 'T' extends past the date-only string from date('now').

-- name: UpdateUnreleasedMoviesToMissing :execresult
UPDATE movies SET status = 'missing', updated_at = CURRENT_TIMESTAMP
WHERE status = 'unreleased' AND (
    (release_date IS NOT NULL AND substr(release_date, 1, 10) <= date('now'))
    OR (physical_release_date IS NOT NULL AND substr(physical_release_date, 1, 10) <= date('now'))
    OR (theatrical_release_date IS NOT NULL AND date(substr(theatrical_release_date, 1, 10), '+90 days') <= date('now'))
);

-- name: GetUnreleasedMoviesWithPastDate :many
SELECT * FROM movies
WHERE status = 'unreleased' AND (
    (release_date IS NOT NULL AND substr(release_date, 1, 10) <= date('now'))
    OR (physical_release_date IS NOT NULL AND substr(physical_release_date, 1, 10) <= date('now'))
    OR (theatrical_release_date IS NOT NULL AND date(substr(theatrical_release_date, 1, 10), '+90 days') <= date('now'))
);

-- name: UpdateMoviesToUnreleased :execresult
UPDATE movies SET status = 'unreleased', updated_at = CURRENT_TIMESTAMP
WHERE status = 'missing'
    AND (release_date IS NULL OR substr(release_date, 1, 10) > date('now'))
    AND (physical_release_date IS NULL OR substr(physical_release_date, 1, 10) > date('now'))
    AND (theatrical_release_date IS NULL OR date(substr(theatrical_release_date, 1, 10), '+90 days') > date('now'));

-- Missing movies queries (status-based)
-- name: ListMissingMovies :many
SELECT m.* FROM movies m
WHERE m.status IN ('missing', 'failed')
  AND m.monitored = 1
ORDER BY m.release_date DESC;

-- name: CountMissingMovies :one
SELECT COUNT(*) FROM movies m
WHERE m.status IN ('missing', 'failed')
  AND m.monitored = 1;

-- Upgrade candidate queries (status-based)
-- name: ListMovieUpgradeCandidates :many
SELECT m.*, mf.quality_id as current_quality_id FROM movies m
JOIN movie_files mf ON mf.id = (
    SELECT id FROM movie_files WHERE movie_id = m.id ORDER BY id DESC LIMIT 1
)
WHERE m.status = 'upgradable'
  AND m.monitored = 1
ORDER BY m.release_date DESC;

-- name: CountMovieUpgradeCandidates :one
SELECT COUNT(*) FROM movies m
WHERE m.status = 'upgradable'
  AND m.monitored = 1;

-- name: GetMovieWithFileQuality :one
SELECT m.*, mf.id as file_id, mf.quality_id as current_quality_id
FROM movies m
LEFT JOIN movie_files mf ON m.id = mf.movie_id
WHERE m.id = ?
LIMIT 1;

-- name: UpdateMovieFileQualityID :exec
UPDATE movie_files SET quality_id = ? WHERE id = ?;

-- Import-related movie file operations
-- name: CreateMovieFileWithImportInfo :one
INSERT INTO movie_files (
    movie_id, path, size, quality, quality_id, video_codec, audio_codec, resolution,
    audio_channels, dynamic_range, original_path, original_filename, imported_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateMovieFileImportInfo :one
UPDATE movie_files SET
    original_path = ?,
    original_filename = ?,
    imported_at = ?
WHERE id = ?
RETURNING *;

-- name: GetMovieFilesWithImportInfo :many
SELECT * FROM movie_files WHERE movie_id = ? ORDER BY imported_at DESC;

-- name: UpdateMovieFilePath :exec
UPDATE movie_files SET path = ? WHERE id = ?;

-- name: UpdateMovieFileMediaInfo :exec
UPDATE movie_files SET
    video_codec = ?,
    audio_codec = ?,
    resolution = ?
WHERE movie_id = ?;

-- name: GetMovieFileByOriginalPath :one
SELECT * FROM movie_files WHERE original_path = ? LIMIT 1;

-- name: IsOriginalPathImportedMovie :one
SELECT EXISTS(SELECT 1 FROM movie_files WHERE original_path = ?) AS imported;

-- name: ListAllMovieFilePaths :many
SELECT path FROM movie_files;

-- name: ListDownloadingMovies :many
SELECT id, active_download_id FROM movies
WHERE status = 'downloading' AND active_download_id IS NOT NULL;

-- name: ListMovieFilesForRootFolder :many
SELECT mf.id as file_id, mf.path, mf.movie_id, m.status as movie_status
FROM movie_files mf
JOIN movies m ON mf.movie_id = m.id
WHERE m.root_folder_id = ?
  AND m.status IN ('available', 'upgradable');

-- Upgradable movies with current file quality
-- name: ListUpgradableMoviesWithQuality :many
SELECT m.*, mf.quality_id as current_quality_id
FROM movies m
JOIN movie_files mf ON mf.id = (
    SELECT id FROM movie_files WHERE movie_id = m.id ORDER BY id DESC LIMIT 1
)
WHERE m.status = 'upgradable' AND m.monitored = 1
ORDER BY m.release_date DESC;

-- name: UpdateMoviesMonitoredByIDs :exec
UPDATE movies SET monitored = ?, updated_at = CURRENT_TIMESTAMP WHERE id IN (sqlc.slice('ids'));

-- Quality profile recalculation: find movies with files to evaluate against new cutoff
-- name: ListMoviesWithFilesForProfile :many
SELECT m.id, m.status, mf.id as file_id, mf.quality_id as current_quality_id
FROM movies m
JOIN movie_files mf ON m.id = mf.movie_id
WHERE m.quality_profile_id = ?
  AND m.status IN ('available', 'upgradable')
  AND mf.quality_id IS NOT NULL;
