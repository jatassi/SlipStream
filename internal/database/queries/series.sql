-- name: GetSeries :one
SELECT * FROM series WHERE id = ? LIMIT 1;

-- name: GetSeriesByTvdbID :one
SELECT * FROM series WHERE tvdb_id = ? LIMIT 1;

-- name: ListSeries :many
SELECT * FROM series ORDER BY sort_title;

-- name: ListMonitoredSeries :many
SELECT * FROM series WHERE monitored = 1 ORDER BY sort_title;

-- name: CreateSeries :one
INSERT INTO series (
    title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime,
    path, root_folder_id, quality_profile_id, monitored, season_folder, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateSeries :one
UPDATE series SET
    title = ?,
    sort_title = ?,
    year = ?,
    tvdb_id = ?,
    tmdb_id = ?,
    imdb_id = ?,
    overview = ?,
    runtime = ?,
    path = ?,
    root_folder_id = ?,
    quality_profile_id = ?,
    monitored = ?,
    season_folder = ?,
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteSeries :exec
DELETE FROM series WHERE id = ?;

-- name: CountSeries :one
SELECT COUNT(*) FROM series;

-- Episodes
-- name: GetEpisode :one
SELECT * FROM episodes WHERE id = ? LIMIT 1;

-- name: ListEpisodesBySeries :many
SELECT * FROM episodes WHERE series_id = ? ORDER BY season_number, episode_number;

-- name: ListEpisodesBySeason :many
SELECT * FROM episodes WHERE series_id = ? AND season_number = ? ORDER BY episode_number;

-- name: CreateEpisode :one
INSERT INTO episodes (
    series_id, season_number, episode_number, title, overview, air_date, monitored
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateEpisode :one
UPDATE episodes SET
    title = ?,
    overview = ?,
    air_date = ?,
    monitored = ?
WHERE id = ?
RETURNING *;

-- name: DeleteEpisode :exec
DELETE FROM episodes WHERE id = ?;

-- Seasons
-- name: GetSeason :one
SELECT * FROM seasons WHERE id = ? LIMIT 1;

-- name: ListSeasonsBySeries :many
SELECT * FROM seasons WHERE series_id = ? ORDER BY season_number;

-- name: CreateSeason :one
INSERT INTO seasons (series_id, season_number, monitored)
VALUES (?, ?, ?)
RETURNING *;

-- name: UpdateSeason :one
UPDATE seasons SET monitored = ? WHERE id = ? RETURNING *;

-- name: SearchSeries :many
SELECT * FROM series
WHERE title LIKE ? OR sort_title LIKE ?
ORDER BY sort_title
LIMIT ? OFFSET ?;

-- name: ListSeriesPaginated :many
SELECT * FROM series
ORDER BY sort_title
LIMIT ? OFFSET ?;

-- name: GetSeriesByPath :one
SELECT * FROM series WHERE path = ? LIMIT 1;

-- name: ListSeriesByRootFolder :many
SELECT * FROM series WHERE root_folder_id = ? ORDER BY sort_title;

-- name: UpdateSeriesStatus :exec
UPDATE series SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteSeasonsBySeries :exec
DELETE FROM seasons WHERE series_id = ?;

-- name: DeleteEpisodesBySeries :exec
DELETE FROM episodes WHERE series_id = ?;

-- name: GetSeasonByNumber :one
SELECT * FROM seasons WHERE series_id = ? AND season_number = ? LIMIT 1;

-- name: GetEpisodeByNumber :one
SELECT * FROM episodes
WHERE series_id = ? AND season_number = ? AND episode_number = ?
LIMIT 1;

-- name: CountEpisodesBySeries :one
SELECT COUNT(*) FROM episodes WHERE series_id = ?;

-- name: CountEpisodesBySeason :one
SELECT COUNT(*) FROM episodes WHERE series_id = ? AND season_number = ?;

-- Episode Files
-- name: GetEpisodeFile :one
SELECT * FROM episode_files WHERE id = ? LIMIT 1;

-- name: ListEpisodeFilesBySeries :many
SELECT ef.* FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
WHERE e.series_id = ?
ORDER BY e.season_number, e.episode_number;

-- name: ListEpisodeFilesBySeason :many
SELECT ef.* FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
WHERE e.series_id = ? AND e.season_number = ?
ORDER BY e.episode_number;

-- name: ListEpisodeFilesByEpisode :many
SELECT * FROM episode_files WHERE episode_id = ? ORDER BY path;

-- name: CreateEpisodeFile :one
INSERT INTO episode_files (episode_id, path, size, quality, video_codec, audio_codec, resolution)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteEpisodeFile :exec
DELETE FROM episode_files WHERE id = ?;

-- name: DeleteEpisodeFilesByEpisode :exec
DELETE FROM episode_files WHERE episode_id = ?;

-- name: GetEpisodeFileByPath :one
SELECT * FROM episode_files WHERE path = ? LIMIT 1;

-- name: CountEpisodeFilesBySeries :one
SELECT COUNT(*) FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
WHERE e.series_id = ?;

-- name: CountEpisodeFilesBySeason :one
SELECT COUNT(*) FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
WHERE e.series_id = ? AND e.season_number = ?;
