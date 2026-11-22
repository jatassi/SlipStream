-- name: HoldsLease :one
SELECT count(*) FROM subtitles_want WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced)
    AND attempt_id = sqlc.arg(attempt_id);

-- name: InsertIntent :exec
INSERT INTO subtitles_write (file_id, language, forced, attempt_id, tail, sha256, sidecar, replaces, started_at)
VALUES (sqlc.arg(file_id), sqlc.arg(language), sqlc.arg(forced), sqlc.arg(attempt_id), sqlc.arg(tail), sqlc.arg(sha256), sqlc.arg(sidecar),
    sqlc.narg(replaces), sqlc.arg(started_at));

-- name: GetSidecarHeld :one
-- The Sidecar while it still holds the bytes a settle read.
SELECT count(*) FROM subtitles_sidecar WHERE id = sqlc.arg(id) AND sha256 = sqlc.arg(sha256);

-- name: PutTried :exec
INSERT INTO subtitles_tried (file_id, language, forced, candidate_key, provider, release, match, outcome, tried_at)
VALUES (sqlc.arg(file_id), sqlc.arg(language), sqlc.arg(forced), sqlc.arg(candidate_key), sqlc.narg(provider), sqlc.narg(release),
    sqlc.narg(match), sqlc.arg(outcome), sqlc.arg(tried_at))
ON CONFLICT (file_id, language, forced, candidate_key) DO UPDATE SET provider = excluded.provider, release = excluded.release,
    match = excluded.match, outcome = excluded.outcome, tried_at = excluded.tried_at;

-- name: DueNow :exec
UPDATE subtitles_want SET due_at = sqlc.arg(now) WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced);

-- name: DeleteParkedOnce :execrows
DELETE FROM subtitles_parked WHERE id = sqlc.arg(id);

-- name: GetGate :one
SELECT sync_match, sync_max_offset_seconds, sync_framerates FROM subtitles_settings WHERE id = 1;

-- name: ListWriting :many
SELECT f.file_id FROM subtitles_file f
WHERE f.file_id IN (sqlc.slice(ids)) AND (f.map_state = 'building' OR EXISTS (SELECT 1 FROM subtitles_write w WHERE w.file_id = f.file_id))
ORDER BY f.file_id;
