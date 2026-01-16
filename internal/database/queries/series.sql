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
    path, root_folder_id, quality_profile_id, monitored, season_folder, status, network, released, format_type
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
    network = ?,
    released = ?,
    format_type = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteSeries :exec
DELETE FROM series WHERE id = ?;

-- name: DeleteSeriesByRootFolder :exec
DELETE FROM series WHERE root_folder_id = ?;

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
    series_id, season_number, episode_number, title, overview, air_date, monitored, released
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateEpisode :one
UPDATE episodes SET
    title = ?,
    overview = ?,
    air_date = ?,
    monitored = ?,
    released = ?
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
INSERT INTO seasons (series_id, season_number, monitored, released)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateSeason :one
UPDATE seasons SET monitored = ?, released = ? WHERE id = ? RETURNING *;

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
INSERT INTO episode_files (episode_id, path, size, quality, quality_id, video_codec, audio_codec, resolution)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteEpisodeFile :exec
DELETE FROM episode_files WHERE id = ?;

-- name: DeleteEpisodeFilesByEpisode :exec
DELETE FROM episode_files WHERE episode_id = ?;

-- name: GetEpisodeFileByPath :one
SELECT * FROM episode_files WHERE path = ? LIMIT 1;

-- name: CountEpisodeFiles :one
SELECT COUNT(*) FROM episode_files WHERE episode_id = ?;

-- name: CountEpisodeFilesBySeries :one
SELECT COUNT(*) FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
WHERE e.series_id = ?;

-- name: CountEpisodeFilesBySeason :one
SELECT COUNT(*) FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
WHERE e.series_id = ? AND e.season_number = ?;

-- name: ListUnmatchedSeriesByRootFolder :many
SELECT * FROM series
WHERE root_folder_id = ?
  AND (tvdb_id IS NULL OR tvdb_id = 0)
  AND (tmdb_id IS NULL OR tmdb_id = 0)
ORDER BY sort_title;

-- name: UpsertSeason :one
INSERT INTO seasons (series_id, season_number, monitored, overview, poster_url, released)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(series_id, season_number) DO UPDATE SET
    overview = COALESCE(excluded.overview, seasons.overview),
    poster_url = COALESCE(excluded.poster_url, seasons.poster_url)
RETURNING *;

-- name: UpsertEpisode :one
INSERT INTO episodes (series_id, season_number, episode_number, title, overview, air_date, monitored, released)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(series_id, season_number, episode_number) DO UPDATE SET
    title = COALESCE(excluded.title, episodes.title),
    overview = COALESCE(excluded.overview, episodes.overview),
    air_date = COALESCE(excluded.air_date, episodes.air_date)
RETURNING *;

-- name: UpdateSeasonMetadata :one
UPDATE seasons SET
    overview = ?,
    poster_url = ?
WHERE id = ?
RETURNING *;

-- Calendar queries
-- name: GetEpisodesInDateRange :many
SELECT
    e.id,
    e.series_id,
    e.season_number,
    e.episode_number,
    e.title,
    e.overview,
    e.air_date,
    e.monitored,
    s.title as series_title,
    s.network,
    s.tmdb_id as series_tmdb_id
FROM episodes e
JOIN series s ON e.series_id = s.id
WHERE e.air_date BETWEEN ? AND ?
ORDER BY e.air_date, s.title, e.season_number, e.episode_number;

-- name: UpdateSeriesNetwork :exec
UPDATE series SET network = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- Availability queries
-- name: UpdateEpisodesReleasedByDate :execresult
UPDATE episodes SET released = 1
WHERE released = 0 AND air_date IS NOT NULL AND air_date <= date('now');

-- name: UpdateEpisodeReleased :exec
UPDATE episodes SET released = ? WHERE id = ?;

-- name: UpdateSeasonReleased :exec
UPDATE seasons SET released = ? WHERE id = ?;

-- name: UpdateSeriesReleased :exec
UPDATE series SET released = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateSeasonReleasedFromEpisodes :exec
UPDATE seasons SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM episodes WHERE episodes.series_id = seasons.series_id
    AND episodes.season_number = seasons.season_number
) WHERE seasons.id = ?;

-- name: UpdateAllSeasonsReleased :execresult
UPDATE seasons SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM episodes WHERE episodes.series_id = seasons.series_id
    AND episodes.season_number = seasons.season_number
);

-- name: UpdateSeriesReleasedFromSeasons :exec
UPDATE series SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM seasons WHERE seasons.series_id = series.id
), updated_at = CURRENT_TIMESTAMP WHERE series.id = ?;

-- name: UpdateAllSeriesReleased :execresult
UPDATE series SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM seasons WHERE seasons.series_id = series.id
), updated_at = CURRENT_TIMESTAMP;

-- name: GetSeasonsBySeriesID :many
SELECT * FROM seasons WHERE series_id = ? ORDER BY season_number;

-- name: UpdateSeriesAvailabilityStatus :exec
UPDATE series SET availability_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: GetSeriesAvailabilityData :one
SELECT
    s.id,
    s.status,
    s.released,
    (SELECT COUNT(*) FROM seasons WHERE series_id = s.id AND season_number > 0) as total_seasons,
    (SELECT COUNT(*) FROM seasons WHERE series_id = s.id AND season_number > 0 AND released = 1) as released_seasons,
    (SELECT MIN(season_number) FROM seasons WHERE series_id = s.id AND season_number > 0 AND released = 0) as first_unreleased_season,
    (SELECT COUNT(*) FROM episodes e
     JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
     WHERE sea.series_id = s.id AND sea.released = 0 AND e.released = 1) as aired_eps_in_unreleased_seasons
FROM series s
WHERE s.id = ?;

-- name: ListAllSeriesForAvailability :many
SELECT
    s.id,
    s.status,
    s.released,
    (SELECT COUNT(*) FROM seasons WHERE series_id = s.id AND season_number > 0) as total_seasons,
    (SELECT COUNT(*) FROM seasons WHERE series_id = s.id AND season_number > 0 AND released = 1) as released_seasons,
    (SELECT MIN(season_number) FROM seasons WHERE series_id = s.id AND season_number > 0 AND released = 0) as first_unreleased_season,
    (SELECT COUNT(*) FROM episodes e
     JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
     WHERE sea.series_id = s.id AND sea.released = 0 AND e.released = 1) as aired_eps_in_unreleased_seasons
FROM series s;

-- Missing episodes queries (respects cascading monitoring: series -> season -> episode)
-- name: ListMissingEpisodes :many
SELECT
    e.*,
    s.title as series_title,
    s.tvdb_id as series_tvdb_id,
    s.tmdb_id as series_tmdb_id,
    s.imdb_id as series_imdb_id,
    s.year as series_year,
    s.quality_profile_id as series_quality_profile_id
FROM episodes e
JOIN series s ON e.series_id = s.id
JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
LEFT JOIN episode_files ef ON e.id = ef.episode_id
WHERE e.released = 1
  AND s.monitored = 1
  AND sea.monitored = 1
  AND e.monitored = 1
  AND ef.id IS NULL
ORDER BY e.air_date DESC;

-- name: CountMissingEpisodes :one
SELECT COUNT(*) FROM episodes e
JOIN series s ON e.series_id = s.id
JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
LEFT JOIN episode_files ef ON e.id = ef.episode_id
WHERE e.released = 1
  AND s.monitored = 1
  AND sea.monitored = 1
  AND e.monitored = 1
  AND ef.id IS NULL;

-- name: GetMissingEpisodesBySeries :many
SELECT e.* FROM episodes e
JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
LEFT JOIN episode_files ef ON e.id = ef.episode_id
WHERE e.series_id = ?
  AND e.released = 1
  AND sea.monitored = 1
  AND e.monitored = 1
  AND ef.id IS NULL
ORDER BY e.season_number, e.episode_number;

-- name: CountMissingEpisodesBySeries :one
SELECT COUNT(*) FROM episodes e
JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
LEFT JOIN episode_files ef ON e.id = ef.episode_id
WHERE e.series_id = ?
  AND e.released = 1
  AND sea.monitored = 1
  AND e.monitored = 1
  AND ef.id IS NULL;

-- name: ListSeriesWithMissingEpisodes :many
SELECT DISTINCT s.* FROM series s
JOIN episodes e ON s.id = e.series_id
JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
LEFT JOIN episode_files ef ON e.id = ef.episode_id
WHERE e.released = 1
  AND s.monitored = 1
  AND sea.monitored = 1
  AND e.monitored = 1
  AND ef.id IS NULL
ORDER BY s.sort_title;

-- Upgrade candidate queries (episodes with files below quality cutoff)
-- name: ListEpisodeUpgradeCandidates :many
SELECT
    e.*,
    s.title as series_title,
    s.tvdb_id as series_tvdb_id,
    s.tmdb_id as series_tmdb_id,
    s.imdb_id as series_imdb_id,
    s.year as series_year,
    s.quality_profile_id as series_quality_profile_id,
    ef.id as file_id,
    ef.quality_id as current_quality_id,
    qp.cutoff
FROM episodes e
JOIN series s ON e.series_id = s.id
JOIN episode_files ef ON e.id = ef.episode_id
JOIN quality_profiles qp ON s.quality_profile_id = qp.id
WHERE e.monitored = 1
  AND e.released = 1
  AND s.monitored = 1
  AND ef.quality_id IS NOT NULL
  AND ef.quality_id < qp.cutoff
  AND qp.upgrades_enabled = 1
ORDER BY e.air_date DESC;

-- name: CountEpisodeUpgradeCandidates :one
SELECT COUNT(*) FROM episodes e
JOIN series s ON e.series_id = s.id
JOIN episode_files ef ON e.id = ef.episode_id
JOIN quality_profiles qp ON s.quality_profile_id = qp.id
WHERE e.monitored = 1
  AND e.released = 1
  AND s.monitored = 1
  AND ef.quality_id IS NOT NULL
  AND ef.quality_id < qp.cutoff
  AND qp.upgrades_enabled = 1;

-- name: ListEpisodeUpgradeCandidatesBySeries :many
SELECT
    e.*,
    ef.id as file_id,
    ef.quality_id as current_quality_id,
    qp.cutoff
FROM episodes e
JOIN series s ON e.series_id = s.id
JOIN episode_files ef ON e.id = ef.episode_id
JOIN quality_profiles qp ON s.quality_profile_id = qp.id
WHERE e.series_id = ?
  AND e.monitored = 1
  AND e.released = 1
  AND ef.quality_id IS NOT NULL
  AND ef.quality_id < qp.cutoff
  AND qp.upgrades_enabled = 1
ORDER BY e.season_number, e.episode_number;

-- name: GetEpisodeWithFileQuality :one
SELECT
    e.*,
    s.quality_profile_id as series_quality_profile_id,
    ef.id as file_id,
    ef.quality_id as current_quality_id
FROM episodes e
JOIN series s ON e.series_id = s.id
LEFT JOIN episode_files ef ON e.id = ef.episode_id
WHERE e.id = ?
LIMIT 1;

-- name: UpdateEpisodeFileQualityID :exec
UPDATE episode_files SET quality_id = ? WHERE id = ?;

-- Bulk monitoring updates for add flow
-- name: UpdateAllEpisodesMonitoredBySeries :exec
UPDATE episodes SET monitored = ? WHERE series_id = ?;

-- name: UpdateEpisodesMonitoredBySeason :exec
UPDATE episodes SET monitored = ? WHERE series_id = ? AND season_number = ?;

-- name: UpdateEpisodesMonitoredExcludingSeason :exec
UPDATE episodes SET monitored = ? WHERE series_id = ? AND season_number != ?;

-- name: UpdateSeasonMonitoredBySeries :exec
UPDATE seasons SET monitored = ? WHERE series_id = ?;

-- name: UpdateSeasonMonitoredByNumber :exec
UPDATE seasons SET monitored = ? WHERE series_id = ? AND season_number = ?;

-- name: GetLatestSeasonNumber :one
SELECT COALESCE(MAX(season_number), 0) as latest FROM seasons WHERE series_id = ? AND season_number > 0;

-- name: UpdateFutureEpisodesMonitored :exec
UPDATE episodes SET monitored = ? WHERE series_id = ? AND released = 0;

-- name: UpdateFutureSeasonsMonitored :exec
UPDATE seasons SET monitored = ?
WHERE seasons.series_id = ? AND seasons.season_number IN (
    SELECT DISTINCT e.season_number FROM episodes e
    WHERE e.series_id = ? AND e.released = 0
);

-- name: UpdateEpisodesMonitoredByIDs :exec
UPDATE episodes SET monitored = ? WHERE id IN (sqlc.slice('ids'));

-- name: UpdateSeasonsMonitoredExcluding :exec
UPDATE seasons SET monitored = ? WHERE series_id = ? AND season_number NOT IN (sqlc.slice('excludeSeasons'));

-- name: UpdateEpisodesMonitoredExcludingSpecials :exec
UPDATE episodes SET monitored = ? WHERE series_id = ? AND season_number > 0;

-- name: UpdateSeasonsMonitoredExcludingSpecials :exec
UPDATE seasons SET monitored = ? WHERE series_id = ? AND season_number > 0;

-- Monitoring status queries
-- name: GetSeriesMonitoringStats :one
SELECT
    (SELECT COUNT(*) FROM seasons s WHERE s.series_id = ?) as total_seasons,
    (SELECT COUNT(*) FROM seasons s WHERE s.series_id = ? AND s.monitored = 1) as monitored_seasons,
    (SELECT COUNT(*) FROM episodes e WHERE e.series_id = ?) as total_episodes,
    (SELECT COUNT(*) FROM episodes e WHERE e.series_id = ? AND e.monitored = 1) as monitored_episodes;

-- Import-related episode file operations
-- name: CreateEpisodeFileWithImportInfo :one
INSERT INTO episode_files (
    episode_id, path, size, quality, quality_id, video_codec, audio_codec, resolution,
    original_path, original_filename, imported_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateEpisodeFileImportInfo :one
UPDATE episode_files SET
    original_path = ?,
    original_filename = ?,
    imported_at = ?
WHERE id = ?
RETURNING *;

-- name: GetEpisodeFilesWithImportInfo :many
SELECT ef.*, e.series_id, e.season_number, e.episode_number
FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
WHERE e.series_id = ?
ORDER BY ef.imported_at DESC;

-- Format type override queries
-- name: UpdateSeriesFormatType :one
UPDATE series SET format_type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? RETURNING *;

-- name: GetSeriesFormatType :one
SELECT format_type FROM series WHERE id = ?;

-- name: UpdateEpisodeFilePath :exec
UPDATE episode_files SET path = ? WHERE id = ?;
