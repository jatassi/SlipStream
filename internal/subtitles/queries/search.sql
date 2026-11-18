-- name: TakeLease :execrows
-- The want's lease, refused while another attempt holds it, an intent of
-- the key waits for the Keeper, or a Sidecar of the file is carried.
UPDATE subtitles_want SET attempt_id = sqlc.arg(attempt_id)
WHERE subtitles_want.file_id = sqlc.arg(file_id) AND subtitles_want.language = sqlc.arg(language) AND subtitles_want.forced = sqlc.arg(forced)
    AND subtitles_want.attempt_id IS NULL
    AND NOT EXISTS (SELECT 1 FROM subtitles_write w WHERE w.file_id = sqlc.arg(file_id) AND w.language = sqlc.arg(language) AND w.forced = sqlc.arg(forced))
    AND NOT EXISTS (SELECT 1 FROM subtitles_sidecar c WHERE c.file_id = sqlc.arg(file_id) AND (c.timing_due = 1 OR c.stray IS NOT NULL));

-- name: ReleaseLease :exec
UPDATE subtitles_want SET attempt_id = NULL
WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced) AND attempt_id = sqlc.arg(attempt_id);

-- name: GetWant :one
SELECT * FROM subtitles_want WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced);

-- name: Searched :execrows
-- An attempt's end, off the queue, under the attempt's own lease.
UPDATE subtitles_want SET last_searched_at = sqlc.arg(now), last_found = sqlc.arg(found), best_match = sqlc.narg(best_match),
    last_result = sqlc.arg(result), gap_seconds = sqlc.arg(gap_seconds), due_at = sqlc.narg(due_at),
    queued_at = NULL, queue_rank = NULL, queue_order = NULL, priority = NULL
WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced) AND attempt_id = sqlc.arg(attempt_id);

-- name: Unqueue :exec
-- A queued want that can't search now leaves the queue and keeps its due
-- time and priority, so it queues again once it can.
UPDATE subtitles_want SET queued_at = NULL, queue_rank = NULL, queue_order = NULL
WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced);

-- name: QueueDue :execrows
-- Up to n wants past due whose file's audio is scanned join the queue,
-- all of them or only fresh imports' and prioritized ones: reports, the
-- Admin's searches, missing wants of fresh imports newest first, older
-- missing wants, then upgrades.
UPDATE subtitles_want SET queued_at = sqlc.arg(now),
    queue_rank = CASE priority WHEN 'report' THEN 0 WHEN 'admin' THEN 1 ELSE
        CASE WHEN unmet = 0 THEN 4 WHEN (SELECT f.fresh FROM subtitles_file f WHERE f.file_id = subtitles_want.file_id) = 1 THEN 2 ELSE 3 END END,
    queue_order = CASE WHEN priority IS NULL AND unmet = 1 AND (SELECT f.fresh FROM subtitles_file f WHERE f.file_id = subtitles_want.file_id) = 1
        THEN -(SELECT f.map_since FROM subtitles_file f WHERE f.file_id = subtitles_want.file_id) ELSE due_at END
WHERE rowid IN (
    SELECT w.rowid FROM subtitles_want w JOIN subtitles_file f ON f.file_id = w.file_id
    WHERE w.queued_at IS NULL AND w.due_at <= sqlc.arg(now) AND f.map_state = 'built' AND f.wantless = 0
        AND (w.unmet = 1 OR CAST(sqlc.arg(upgrades) AS INTEGER) = 1)
        AND (CAST(sqlc.arg(all) AS INTEGER) = 1 OR f.fresh = 1 OR w.priority IS NOT NULL)
    ORDER BY w.due_at LIMIT sqlc.arg(n));

-- name: SearchNow :execrows
-- The want searches ahead of the schedule, at rank, with its gap reset.
UPDATE subtitles_want SET priority = sqlc.arg(priority), due_at = sqlc.arg(now), gap_seconds = sqlc.arg(gap_seconds),
    queued_at = CASE WHEN (SELECT f.map_state FROM subtitles_file f WHERE f.file_id = subtitles_want.file_id) = 'built' THEN sqlc.arg(now) END,
    queue_rank = CASE WHEN (SELECT f.map_state FROM subtitles_file f WHERE f.file_id = subtitles_want.file_id) = 'built' THEN sqlc.arg(rank) END,
    queue_order = sqlc.arg(now)
WHERE subtitles_want.file_id = sqlc.arg(file_id) AND subtitles_want.language = sqlc.arg(language) AND subtitles_want.forced = sqlc.arg(forced);

-- name: ClearFresh :exec
UPDATE subtitles_file SET fresh = 0 WHERE file_id = sqlc.arg(file_id);

-- name: ListTried :many
SELECT * FROM subtitles_tried WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced)
ORDER BY candidate_key;

-- name: PruneTried :execrows
-- Tried before rows older than before go, a page at a time; a Mark bad
-- stays.
DELETE FROM subtitles_tried WHERE rowid IN (
    SELECT t.rowid FROM subtitles_tried t WHERE t.outcome <> 'marked_bad' AND t.tried_at < sqlc.arg(before) ORDER BY t.tried_at LIMIT sqlc.arg(n));

-- name: PutProviderState :exec
UPDATE subtitles_provider SET paused_until = sqlc.narg(paused_until), pause_reason = sqlc.narg(pause_reason), pause_words = sqlc.narg(pause_words),
    failing_since = sqlc.narg(failing_since), last_error = sqlc.narg(last_error), allowance = sqlc.narg(allowance),
    allowance_at = sqlc.narg(allowance_at), key_checked_at = sqlc.narg(key_checked_at)
WHERE id = sqlc.arg(id);

-- name: ListSiblings :many
-- The Sidecars of one want on the other files of a Work, the newest
-- first. The Work's files lead the join, so a common language never
-- reads every Sidecar of it.
SELECT c.* FROM subtitles_file s CROSS JOIN subtitles_sidecar c ON c.file_id = s.file_id
WHERE s.work_id = sqlc.arg(work_id) AND c.file_id <> sqlc.arg(file_id) AND c.language = sqlc.arg(language) AND c.forced = sqlc.arg(forced)
    AND c.stray IS NULL AND c.timing_due = 0
ORDER BY c.added_at DESC, c.id;

-- name: RankMap :exec
-- A file whose map an Admin's search or a report waits for builds ahead
-- of the others.
UPDATE subtitles_file SET map_rank = min(map_rank, CAST(sqlc.arg(rank) AS INTEGER)) WHERE file_id = sqlc.arg(file_id) AND map_state <> 'built';

-- name: DueNowOf :exec
-- A fresh import's missing wants search once its audio is scanned.
UPDATE subtitles_want SET due_at = sqlc.arg(now) WHERE file_id = sqlc.arg(file_id) AND unmet = 1;
