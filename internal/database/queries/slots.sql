-- Multi-Version Settings
-- name: GetMultiVersionSettings :one
SELECT * FROM multi_version_settings WHERE id = 1 LIMIT 1;

-- name: UpdateMultiVersionSettings :one
UPDATE multi_version_settings SET
    enabled = ?,
    dry_run_completed = ?,
    last_migration_at = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: SetMultiVersionEnabled :one
UPDATE multi_version_settings SET
    enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: SetDryRunCompleted :one
UPDATE multi_version_settings SET
    dry_run_completed = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- Version Slots
-- name: GetVersionSlot :one
SELECT * FROM version_slots WHERE id = ? LIMIT 1;

-- name: GetVersionSlotByNumber :one
SELECT * FROM version_slots WHERE slot_number = ? LIMIT 1;

-- name: ListVersionSlots :many
SELECT * FROM version_slots ORDER BY display_order;

-- name: ListEnabledVersionSlots :many
SELECT * FROM version_slots WHERE enabled = 1 ORDER BY display_order;

-- name: UpdateVersionSlot :one
UPDATE version_slots SET
    name = ?,
    enabled = ?,
    quality_profile_id = ?,
    display_order = ?,
    movie_root_folder_id = ?,
    tv_root_folder_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateVersionSlotName :one
UPDATE version_slots SET
    name = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateVersionSlotEnabled :one
UPDATE version_slots SET
    enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateVersionSlotProfile :one
UPDATE version_slots SET
    quality_profile_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateVersionSlotRootFolders :one
UPDATE version_slots SET
    movie_root_folder_id = ?,
    tv_root_folder_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: CountSlotsUsingProfile :one
SELECT COUNT(*) FROM version_slots WHERE quality_profile_id = ?;

-- Movie File Slot Assignments
-- name: GetMovieFileSlotAssignments :many
SELECT mf.*, vs.name as slot_name, vs.slot_number
FROM movie_files mf
LEFT JOIN version_slots vs ON mf.slot_id = vs.id
WHERE mf.movie_id = ?
ORDER BY vs.slot_number;

-- name: UpdateMovieFileSlot :one
UPDATE movie_files SET slot_id = ? WHERE id = ? RETURNING *;

-- name: ClearMovieFileSlot :exec
UPDATE movie_files SET slot_id = NULL WHERE id = ?;

-- name: ListMovieFilesInSlot :many
SELECT mf.*, m.title as movie_title
FROM movie_files mf
JOIN movies m ON mf.movie_id = m.id
WHERE mf.slot_id = ?
ORDER BY m.title;

-- name: CountMovieFilesInSlot :one
SELECT COUNT(*) FROM movie_files WHERE slot_id = ?;

-- Episode File Slot Assignments
-- name: GetEpisodeFileSlotAssignments :many
SELECT ef.*, vs.name as slot_name, vs.slot_number
FROM episode_files ef
LEFT JOIN version_slots vs ON ef.slot_id = vs.id
WHERE ef.episode_id = ?
ORDER BY vs.slot_number;

-- name: UpdateEpisodeFileSlot :one
UPDATE episode_files SET slot_id = ? WHERE id = ? RETURNING *;

-- name: ClearEpisodeFileSlot :exec
UPDATE episode_files SET slot_id = NULL WHERE id = ?;

-- name: ListEpisodeFilesInSlot :many
SELECT ef.*, e.title as episode_title, e.season_number, e.episode_number, s.title as series_title
FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
JOIN series s ON e.series_id = s.id
WHERE ef.slot_id = ?
ORDER BY s.title, e.season_number, e.episode_number;

-- name: CountEpisodeFilesInSlot :one
SELECT COUNT(*) FROM episode_files WHERE slot_id = ?;

-- Combined File Counts
-- name: CountAllFilesInSlot :one
SELECT
    (SELECT COUNT(*) FROM movie_files mf WHERE mf.slot_id = ?) +
    (SELECT COUNT(*) FROM episode_files ef WHERE ef.slot_id = ?) as total_count;

-- Slot with Profile (for joined queries)
-- name: ListVersionSlotsWithProfiles :many
SELECT
    vs.*,
    qp.id as profile_id,
    qp.name as profile_name,
    qp.cutoff as profile_cutoff,
    qp.items as profile_items,
    qp.hdr_settings as profile_hdr_settings,
    qp.video_codec_settings as profile_video_codec_settings,
    qp.audio_codec_settings as profile_audio_codec_settings,
    qp.audio_channel_settings as profile_audio_channel_settings
FROM version_slots vs
LEFT JOIN quality_profiles qp ON vs.quality_profile_id = qp.id
ORDER BY vs.display_order;

-- name: ListEnabledVersionSlotsWithProfiles :many
SELECT
    vs.*,
    qp.id as profile_id,
    qp.name as profile_name,
    qp.cutoff as profile_cutoff,
    qp.items as profile_items,
    qp.hdr_settings as profile_hdr_settings,
    qp.video_codec_settings as profile_video_codec_settings,
    qp.audio_codec_settings as profile_audio_codec_settings,
    qp.audio_channel_settings as profile_audio_channel_settings
FROM version_slots vs
LEFT JOIN quality_profiles qp ON vs.quality_profile_id = qp.id
WHERE vs.enabled = 1
ORDER BY vs.display_order;

-- =====================
-- Movie Slot Assignments
-- =====================

-- name: GetMovieSlotAssignment :one
SELECT * FROM movie_slot_assignments WHERE movie_id = ? AND slot_id = ? LIMIT 1;

-- name: ListMovieSlotAssignments :many
SELECT msa.*, vs.name as slot_name, vs.slot_number, vs.quality_profile_id
FROM movie_slot_assignments msa
JOIN version_slots vs ON msa.slot_id = vs.id
WHERE msa.movie_id = ?
ORDER BY vs.slot_number;

-- name: CreateMovieSlotAssignment :one
INSERT INTO movie_slot_assignments (movie_id, slot_id, file_id, monitored, status)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateMovieSlotAssignment :one
UPDATE movie_slot_assignments SET
    file_id = ?,
    monitored = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE movie_id = ? AND slot_id = ?
RETURNING *;

-- name: UpdateMovieSlotMonitored :exec
UPDATE movie_slot_assignments SET
    monitored = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE movie_id = ? AND slot_id = ?;

-- name: UpdateMovieSlotFile :exec
UPDATE movie_slot_assignments SET
    file_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE movie_id = ? AND slot_id = ?;

-- name: UpdateMovieSlotStatus :exec
UPDATE movie_slot_assignments SET
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE movie_id = ? AND slot_id = ?;

-- name: UpdateMovieSlotStatusWithDetails :exec
UPDATE movie_slot_assignments SET
    status = ?,
    active_download_id = ?,
    status_message = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE movie_id = ? AND slot_id = ?;

-- name: ClearMovieSlotFile :exec
UPDATE movie_slot_assignments SET
    file_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE movie_id = ? AND slot_id = ?;

-- name: DeleteMovieSlotAssignment :exec
DELETE FROM movie_slot_assignments WHERE movie_id = ? AND slot_id = ?;

-- name: DeleteMovieSlotAssignments :exec
DELETE FROM movie_slot_assignments WHERE movie_id = ?;

-- name: UpsertMovieSlotAssignment :one
INSERT INTO movie_slot_assignments (movie_id, slot_id, file_id, monitored, status)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (movie_id, slot_id) DO UPDATE SET
    file_id = excluded.file_id,
    monitored = excluded.monitored,
    status = excluded.status,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- =====================
-- Episode Slot Assignments
-- =====================

-- name: GetEpisodeSlotAssignment :one
SELECT * FROM episode_slot_assignments WHERE episode_id = ? AND slot_id = ? LIMIT 1;

-- name: ListEpisodeSlotAssignments :many
SELECT esa.*, vs.name as slot_name, vs.slot_number, vs.quality_profile_id
FROM episode_slot_assignments esa
JOIN version_slots vs ON esa.slot_id = vs.id
WHERE esa.episode_id = ?
ORDER BY vs.slot_number;

-- name: CreateEpisodeSlotAssignment :one
INSERT INTO episode_slot_assignments (episode_id, slot_id, file_id, monitored, status)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateEpisodeSlotAssignment :one
UPDATE episode_slot_assignments SET
    file_id = ?,
    monitored = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE episode_id = ? AND slot_id = ?
RETURNING *;

-- name: UpdateEpisodeSlotMonitored :exec
UPDATE episode_slot_assignments SET
    monitored = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE episode_id = ? AND slot_id = ?;

-- name: UpdateEpisodeSlotFile :exec
UPDATE episode_slot_assignments SET
    file_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE episode_id = ? AND slot_id = ?;

-- name: UpdateEpisodeSlotStatus :exec
UPDATE episode_slot_assignments SET
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE episode_id = ? AND slot_id = ?;

-- name: UpdateEpisodeSlotStatusWithDetails :exec
UPDATE episode_slot_assignments SET
    status = ?,
    active_download_id = ?,
    status_message = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE episode_id = ? AND slot_id = ?;

-- name: ClearEpisodeSlotFile :exec
UPDATE episode_slot_assignments SET
    file_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE episode_id = ? AND slot_id = ?;

-- name: DeleteEpisodeSlotAssignment :exec
DELETE FROM episode_slot_assignments WHERE episode_id = ? AND slot_id = ?;

-- name: DeleteEpisodeSlotAssignments :exec
DELETE FROM episode_slot_assignments WHERE episode_id = ?;

-- name: UpsertEpisodeSlotAssignment :one
INSERT INTO episode_slot_assignments (episode_id, slot_id, file_id, monitored, status)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (episode_id, slot_id) DO UPDATE SET
    file_id = excluded.file_id,
    monitored = excluded.monitored,
    status = excluded.status,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- =====================
-- Status Queries
-- =====================

-- name: ListMoviesMissingInMonitoredSlots :many
SELECT DISTINCT m.*
FROM movies m
CROSS JOIN version_slots vs
LEFT JOIN movie_slot_assignments msa ON m.id = msa.movie_id AND vs.id = msa.slot_id
WHERE vs.enabled = 1
  AND m.monitored = 1
  AND (msa.monitored = 1 OR msa.monitored IS NULL)
  AND (msa.file_id IS NULL OR msa.id IS NULL);

-- name: CountMovieSlotAssignmentsForSlot :one
SELECT COUNT(*) FROM movie_slot_assignments WHERE slot_id = ?;

-- name: CountEpisodeSlotAssignmentsForSlot :one
SELECT COUNT(*) FROM episode_slot_assignments WHERE slot_id = ?;

-- name: ListMovieSlotAssignmentsForSlot :many
SELECT msa.*, m.title as movie_title
FROM movie_slot_assignments msa
JOIN movies m ON msa.movie_id = m.id
WHERE msa.slot_id = ?
ORDER BY m.title;

-- name: ListEpisodeSlotAssignmentsForSlot :many
SELECT esa.*, e.title as episode_title, e.season_number, e.episode_number, s.title as series_title
FROM episode_slot_assignments esa
JOIN episodes e ON esa.episode_id = e.id
JOIN series s ON e.series_id = s.id
WHERE esa.slot_id = ?
ORDER BY s.title, e.season_number, e.episode_number;

-- Slot status queries for cached aggregate computation
-- name: ListMonitoredMovieSlotStatuses :many
SELECT msa.status FROM movie_slot_assignments msa
WHERE msa.movie_id = ? AND msa.monitored = 1;

-- name: ListMonitoredEpisodeSlotStatuses :many
SELECT esa.status FROM episode_slot_assignments esa
WHERE esa.episode_id = ? AND esa.monitored = 1;

-- Bulk status updates for unreleased/missing transitions (affects all slots for an item)
-- name: UpdateAllMovieSlotStatuses :exec
UPDATE movie_slot_assignments SET
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE movie_id = ?;

-- name: UpdateAllEpisodeSlotStatuses :exec
UPDATE episode_slot_assignments SET
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE episode_id = ?;

-- Slot-level search: find slots needing search
-- name: ListMovieSlotsNeedingSearch :many
SELECT msa.*, vs.name as slot_name, vs.slot_number, vs.quality_profile_id
FROM movie_slot_assignments msa
JOIN version_slots vs ON msa.slot_id = vs.id
WHERE msa.movie_id = ?
  AND msa.monitored = 1
  AND msa.status IN ('missing', 'upgradable')
ORDER BY vs.slot_number;

-- name: ListEpisodeSlotsNeedingSearch :many
SELECT esa.*, vs.name as slot_name, vs.slot_number, vs.quality_profile_id
FROM episode_slot_assignments esa
JOIN version_slots vs ON esa.slot_id = vs.id
WHERE esa.episode_id = ?
  AND esa.monitored = 1
  AND esa.status IN ('missing', 'upgradable')
ORDER BY vs.slot_number;

-- Slot-level quality recalculation
-- name: ListMovieSlotAssignmentsWithFilesForProfile :many
SELECT msa.movie_id, msa.slot_id, msa.status, msa.file_id, mf.quality_id as current_quality_id
FROM movie_slot_assignments msa
JOIN movie_files mf ON msa.file_id = mf.id
JOIN version_slots vs ON msa.slot_id = vs.id
WHERE vs.quality_profile_id = ?
  AND msa.status IN ('available', 'upgradable')
  AND mf.quality_id IS NOT NULL;

-- name: ListEpisodeSlotAssignmentsWithFilesForProfile :many
SELECT esa.episode_id, esa.slot_id, esa.status, esa.file_id, ef.quality_id as current_quality_id
FROM episode_slot_assignments esa
JOIN episode_files ef ON esa.file_id = ef.id
JOIN version_slots vs ON esa.slot_id = vs.id
WHERE vs.quality_profile_id = ?
  AND esa.status IN ('available', 'upgradable')
  AND ef.quality_id IS NOT NULL;

-- =====================
-- File Deletion Queries
-- =====================

-- name: GetMovieSlotAssignmentByFileID :one
SELECT * FROM movie_slot_assignments WHERE file_id = ? LIMIT 1;

-- name: GetEpisodeSlotAssignmentByFileID :one
SELECT * FROM episode_slot_assignments WHERE file_id = ? LIMIT 1;

-- name: ClearMovieSlotFileByFileID :exec
UPDATE movie_slot_assignments SET
    file_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE file_id = ?;

-- name: ClearEpisodeSlotFileByFileID :exec
UPDATE episode_slot_assignments SET
    file_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE file_id = ?;

-- =====================
-- Slot Disable Queries
-- =====================

-- name: ListFilesAssignedToSlot :many
SELECT
    'movie' as media_type,
    mf.id as file_id,
    mf.path as file_path,
    mf.movie_id as media_id,
    m.title as media_title
FROM movie_files mf
JOIN movies m ON mf.movie_id = m.id
WHERE mf.slot_id = ?
UNION ALL
SELECT
    'episode' as media_type,
    ef.id as file_id,
    ef.path as file_path,
    ef.episode_id as media_id,
    s.title || ' S' || printf('%02d', e.season_number) || 'E' || printf('%02d', e.episode_number) as media_title
FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
JOIN series s ON e.series_id = s.id
WHERE ef.slot_id = ?
ORDER BY media_title;

-- name: ClearAllMovieFileSlotsForSlot :exec
UPDATE movie_files SET slot_id = NULL WHERE slot_id = ?;

-- name: ClearAllEpisodeFileSlotsForSlot :exec
UPDATE episode_files SET slot_id = NULL WHERE slot_id = ?;

-- name: ClearAllMovieSlotAssignmentsForSlot :exec
UPDATE movie_slot_assignments SET
    file_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE slot_id = ?;

-- name: ClearAllEpisodeSlotAssignmentsForSlot :exec
UPDATE episode_slot_assignments SET
    file_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE slot_id = ?;

-- =====================
-- Review Queue Queries
-- =====================

-- name: CountMovieFilesWithoutSlot :one
SELECT COUNT(*) FROM movie_files WHERE slot_id IS NULL;

-- name: CountEpisodeFilesWithoutSlot :one
SELECT COUNT(*) FROM episode_files WHERE slot_id IS NULL;

-- name: ListMovieFilesWithoutSlot :many
SELECT mf.*, m.title as movie_title
FROM movie_files mf
JOIN movies m ON mf.movie_id = m.id
WHERE mf.slot_id IS NULL
ORDER BY m.title;

-- name: ListEpisodeFilesWithoutSlot :many
SELECT ef.*, e.title as episode_title, e.season_number, e.episode_number, s.title as series_title
FROM episode_files ef
JOIN episodes e ON ef.episode_id = e.id
JOIN series s ON e.series_id = s.id
WHERE ef.slot_id IS NULL
ORDER BY s.title, e.season_number, e.episode_number;
