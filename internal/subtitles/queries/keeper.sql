-- name: ListParkedFrom :many
SELECT * FROM subtitles_parked WHERE from_file = sqlc.arg(from_file) ORDER BY id;

-- name: ListParkedTo :many
SELECT * FROM subtitles_parked WHERE to_file = sqlc.arg(to_file) ORDER BY id;

-- name: InsertParked :exec
INSERT INTO subtitles_parked (id, from_file, to_file, what, sidecar, root_id, path, sha256, parked_at)
VALUES (sqlc.arg(id), sqlc.arg(from_file), sqlc.arg(to_file), sqlc.arg(what), sqlc.arg(sidecar), sqlc.arg(root_id), sqlc.arg(path), sqlc.arg(sha256), sqlc.arg(parked_at));

-- name: Repark :exec
UPDATE subtitles_parked SET to_file = sqlc.arg(by) WHERE to_file = sqlc.arg(d);

-- name: DeleteParked :exec
DELETE FROM subtitles_parked WHERE id = sqlc.arg(id);

-- name: ListOrphanParked :many
-- Parked rows whose replacer no longer exists: one deleted without a doom.
SELECT p.* FROM subtitles_parked p WHERE NOT EXISTS (SELECT 1 FROM files_file f WHERE f.id = p.to_file) ORDER BY p.id;

-- name: InsertRemoval :exec
INSERT INTO subtitles_removal (id, file_id, root_id, path, sha256, why, next_at)
VALUES (sqlc.arg(id), sqlc.arg(file_id), sqlc.arg(root_id), sqlc.arg(path), sqlc.arg(sha256), sqlc.arg(why), sqlc.arg(next_at))
ON CONFLICT (root_id, path, sha256) DO NOTHING;

-- name: ListRemovalsDue :many
SELECT * FROM subtitles_removal WHERE file_id = sqlc.arg(file_id) AND next_at <= sqlc.arg(now);

-- name: DeleteRemoval :exec
DELETE FROM subtitles_removal WHERE id = sqlc.arg(id);

-- name: DeferRemoval :exec
UPDATE subtitles_removal SET tries = tries + 1, next_at = sqlc.arg(next_at) WHERE id = sqlc.arg(id);

-- name: PlaceSidecar :execrows
UPDATE subtitles_sidecar SET stray = sqlc.narg(stray) WHERE id = sqlc.arg(id) AND sha256 = sqlc.arg(sha256);

-- name: DeleteIntent :execrows
DELETE FROM subtitles_write WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced)
    AND attempt_id = sqlc.arg(attempt_id);

-- name: FinishSurvey :execrows
-- A survey's verdicts hold only while the file is where it was listed and
-- no newer rename asked for another survey.
UPDATE subtitles_file SET survey_due = 0, survey_from = NULL, surveyed_at = sqlc.arg(surveyed_at), companions = sqlc.narg(companions)
WHERE subtitles_file.file_id = sqlc.arg(file_id) AND survey_from IS sqlc.narg(survey_from)
    AND EXISTS (SELECT 1 FROM files_file f WHERE f.id = sqlc.arg(file_id) AND f.path = sqlc.arg(video));

-- name: MarkSurveyDue :exec
UPDATE subtitles_file SET survey_due = 1, survey_from = coalesce(sqlc.narg(survey_from), survey_from) WHERE file_id = sqlc.arg(file_id);

-- name: MarkTimingDue :exec
UPDATE subtitles_sidecar SET timing_due = 1 WHERE file_id = sqlc.arg(file_id);

-- name: MarkFresh :exec
UPDATE subtitles_file SET fresh = 1, map_rank = min(map_rank, 2) WHERE file_id = sqlc.arg(file_id);

-- name: GetDoomedSits :one
SELECT sits FROM files_doomed WHERE file_id = sqlc.arg(file_id);

-- name: ListSurveyDue :many
SELECT file_id FROM subtitles_file WHERE survey_due = 1 ORDER BY file_id LIMIT sqlc.arg(n);

-- name: ListIntentFiles :many
SELECT file_id FROM subtitles_write ORDER BY file_id, language, forced LIMIT sqlc.arg(n);

-- name: ListPendingFiles :many
SELECT file_id FROM subtitles_sidecar WHERE timing_due = 1 OR stray IS NOT NULL ORDER BY file_id LIMIT sqlc.arg(n);

-- name: ListParkedFiles :many
SELECT from_file FROM subtitles_parked ORDER BY from_file LIMIT sqlc.arg(n);

-- name: ListRemovalFiles :many
SELECT file_id FROM subtitles_removal WHERE next_at <= sqlc.arg(now) ORDER BY next_at LIMIT sqlc.arg(n);
