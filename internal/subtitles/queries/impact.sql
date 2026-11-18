-- name: ImpactGained :one
-- The searches one gained want makes under a Profile, and how many of them
-- need the audio scanned first. A file the bulk pass hasn't reached counts
-- as searchable and unscanned. A file whose last survey listed an extra
-- naming the want is met: the bulk pass marks it for a survey that adopts
-- it. no_audio asks for audio not in the language, spoken for audio in it.
SELECT count(*) AS searches, CAST(coalesce(sum(s.map_state IS NOT 'built'), 0) AS INTEGER) AS need_scan
FROM targets_wants tw
JOIN files_file f ON f.work_id = tw.work_id
LEFT JOIN subtitles_file s ON s.file_id = f.id
WHERE tw.profile_id = sqlc.arg(profile_id)
    AND (s.file_id IS NULL OR s.kind IS NOT NULL AND s.wantless = 0)
    AND instr(coalesce(s.embedded, ','), sqlc.arg(key)) = 0
    AND (CAST(sqlc.arg(no_audio) AS INTEGER) = 0 OR instr(coalesce(s.audio, ','), sqlc.arg(base)) = 0)
    AND (CAST(sqlc.arg(spoken) AS INTEGER) = 0 OR instr(coalesce(s.audio, ','), sqlc.arg(base)) > 0)
    AND NOT EXISTS (SELECT 1 FROM subtitles_sidecar c WHERE c.file_id = f.id AND c.language = sqlc.arg(language) AND c.forced = sqlc.arg(forced))
    AND NOT EXISTS (SELECT 1 FROM json_each(s.companions) WHERE json_extract(value, '$.want') = CAST(sqlc.arg(want) AS TEXT));

-- name: ImpactDropped :one
-- The subtitles SlipStream downloaded for one want under a Profile.
SELECT count(*) FROM subtitles_sidecar c
JOIN subtitles_file s ON s.file_id = c.file_id
JOIN targets_wants tw ON tw.work_id = s.work_id
WHERE c.language = sqlc.arg(language) AND c.forced = sqlc.arg(forced) AND c.source IN ('provider', 'sibling')
    AND tw.profile_id = sqlc.arg(profile_id);

-- name: ListDropped :many
SELECT c.id, c.file_id, c.tail, c.stray, c.sha256, s.root_id, f.path FROM subtitles_sidecar c
JOIN subtitles_file s ON s.file_id = c.file_id
JOIN targets_wants tw ON tw.work_id = s.work_id
JOIN files_file f ON f.id = c.file_id
WHERE c.language = sqlc.arg(language) AND c.forced = sqlc.arg(forced) AND c.source IN ('provider', 'sibling')
    AND tw.profile_id = sqlc.arg(profile_id)
ORDER BY c.id;

-- name: ListDroppedIntents :many
SELECT w.file_id, w.attempt_id, w.tail, w.sha256, s.root_id, f.path FROM subtitles_write w
JOIN subtitles_file s ON s.file_id = w.file_id
JOIN targets_wants tw ON tw.work_id = s.work_id
JOIN files_file f ON f.id = w.file_id
WHERE w.language = sqlc.arg(language) AND w.forced = sqlc.arg(forced) AND tw.profile_id = sqlc.arg(profile_id)
ORDER BY w.file_id, w.language, w.forced;

-- name: DeleteWantsOfProfile :exec
DELETE FROM subtitles_want WHERE language = sqlc.arg(language) AND forced = sqlc.arg(forced)
    AND work_id IN (SELECT work_id FROM targets_wants WHERE profile_id = sqlc.arg(profile_id));

-- name: ListLibraryMarks :many
SELECT work_id, max(last_result = 'none_passed') AS none_passed FROM subtitles_want WHERE unmet = 1 GROUP BY work_id ORDER BY work_id;

-- name: NextQueued :many
SELECT * FROM subtitles_want WHERE kind = sqlc.arg(kind) AND queued_at IS NOT NULL AND attempt_id IS NULL
ORDER BY queue_rank, queue_order, file_id LIMIT sqlc.arg(n);

-- name: PastDue :many
SELECT * FROM subtitles_want WHERE queued_at IS NULL AND due_at <= sqlc.arg(now) ORDER BY due_at LIMIT sqlc.arg(n);

-- name: MissingCount :one
SELECT count(*) FROM subtitles_want WHERE unmet = 1;

-- name: WaitingCount :one
SELECT count(*) FROM subtitles_want WHERE kind = sqlc.arg(kind) AND queued_at IS NOT NULL AND attempt_id IS NULL;
