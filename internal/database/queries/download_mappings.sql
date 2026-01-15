-- name: CreateDownloadMapping :one
INSERT INTO download_mappings (
    client_id, download_id, movie_id, series_id, season_number,
    episode_id, is_season_pack, is_complete_series, target_slot_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT (client_id, download_id) DO UPDATE SET
    movie_id = excluded.movie_id,
    series_id = excluded.series_id,
    season_number = excluded.season_number,
    episode_id = excluded.episode_id,
    is_season_pack = excluded.is_season_pack,
    is_complete_series = excluded.is_complete_series,
    target_slot_id = excluded.target_slot_id
RETURNING *;

-- name: GetDownloadMapping :one
SELECT * FROM download_mappings
WHERE client_id = ? AND download_id = ?;

-- name: GetDownloadMappingsByClientDownloadIDs :many
SELECT * FROM download_mappings
WHERE (client_id, download_id) IN (/*SLICE:client_download_ids*/sqlc.slice('client_download_ids'));

-- name: ListActiveDownloadMappings :many
SELECT * FROM download_mappings
ORDER BY created_at DESC;

-- name: DeleteDownloadMapping :exec
DELETE FROM download_mappings
WHERE client_id = ? AND download_id = ?;

-- name: DeleteOldDownloadMappings :exec
DELETE FROM download_mappings
WHERE created_at < datetime('now', '-7 days');

-- name: GetDownloadingMovieIDs :many
SELECT DISTINCT movie_id FROM download_mappings
WHERE movie_id IS NOT NULL;

-- name: GetDownloadingSeriesData :many
SELECT DISTINCT series_id, season_number, episode_id, is_season_pack, is_complete_series
FROM download_mappings
WHERE series_id IS NOT NULL;

-- name: IsMovieDownloading :one
SELECT EXISTS(
    SELECT 1 FROM download_mappings WHERE movie_id = ?
) AS downloading;

-- name: IsSeriesDownloading :one
SELECT EXISTS(
    SELECT 1 FROM download_mappings WHERE series_id = ? AND is_complete_series = 1
) AS downloading;

-- name: IsSeasonDownloading :one
SELECT EXISTS(
    SELECT 1 FROM download_mappings
    WHERE series_id = ? AND season_number = ? AND (is_season_pack = 1 OR is_complete_series = 1)
) AS downloading;

-- name: IsEpisodeDownloading :one
SELECT EXISTS(
    SELECT 1 FROM download_mappings
    WHERE episode_id = ?
       OR (series_id = ? AND season_number = ? AND (is_season_pack = 1 OR is_complete_series = 1))
       OR (series_id = ? AND is_complete_series = 1)
) AS downloading;

-- name: ClearDownloadMappingSlot :exec
-- Req 10.2.1: Clear slot when download fails or is rejected
UPDATE download_mappings SET target_slot_id = NULL
WHERE client_id = ? AND download_id = ?;

-- name: GetDownloadMappingsBySlot :many
-- Get all mappings targeting a specific slot
SELECT * FROM download_mappings
WHERE target_slot_id = ?
ORDER BY created_at DESC;
