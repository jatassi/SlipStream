-- name: ListLanguagesAdded :many
SELECT profile_id, language, forced, added_at FROM targets_profile_language ORDER BY profile_id, position;

-- name: ListProviders :many
SELECT * FROM subtitles_provider ORDER BY position;

-- name: SeedProvider :exec
INSERT INTO subtitles_provider (id, position, is_on)
VALUES (sqlc.arg(id), (SELECT coalesce(max(position), 0) + 1 FROM subtitles_provider), 1)
ON CONFLICT (id) DO NOTHING;

-- name: StampGained :exec
-- A missing want last searched before a covering provider gained the power
-- to search is due now.
UPDATE subtitles_want SET due_at = sqlc.arg(now)
WHERE kind = sqlc.arg(kind) AND unmet = 1 AND last_searched_at < sqlc.arg(gained)
    AND (due_at IS NULL OR due_at > sqlc.arg(now)) AND file_id IN (sqlc.slice(ids));

-- name: ClearLeases :execrows
UPDATE subtitles_want SET attempt_id = NULL WHERE attempt_id IS NOT NULL;

-- name: MarkAllSurveyDue :exec
UPDATE subtitles_file SET survey_due = 1 WHERE kind IS NOT NULL AND survey_due = 0;
