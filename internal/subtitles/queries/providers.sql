-- name: SetProviderOn :exec
UPDATE subtitles_provider SET is_on = sqlc.arg(is_on), gained_at = coalesce(sqlc.narg(gained_at), gained_at) WHERE id = sqlc.arg(id);

-- name: ShiftPositions :exec
UPDATE subtitles_provider SET position = position + sqlc.arg(by);

-- name: SetPosition :exec
UPDATE subtitles_provider SET position = sqlc.arg(position) WHERE id = sqlc.arg(id);

-- name: PutProviderKey :execrows
-- An accepted key ends a pause for a refused key, and one for a spent
-- allowance when its check reported downloads left, and stamps gained_at.
UPDATE subtitles_provider SET api_key_sealed = sqlc.arg(api_key_sealed), account = sqlc.narg(account), plan = sqlc.narg(plan),
    allowance = sqlc.narg(allowance), allowance_at = sqlc.narg(allowance_at), key_checked_at = sqlc.arg(now), gained_at = sqlc.arg(now),
    paused_until = CASE WHEN pause_reason = 'key' OR (pause_reason = 'quota' AND CAST(sqlc.arg(downloads_left) AS INTEGER) = 1) THEN NULL ELSE paused_until END,
    pause_words = CASE WHEN pause_reason = 'key' OR (pause_reason = 'quota' AND CAST(sqlc.arg(downloads_left) AS INTEGER) = 1) THEN NULL ELSE pause_words END,
    pause_reason = CASE WHEN pause_reason = 'key' OR (pause_reason = 'quota' AND CAST(sqlc.arg(downloads_left) AS INTEGER) = 1) THEN NULL ELSE pause_reason END
WHERE id = sqlc.arg(id);

-- name: DeleteProviderKey :execrows
UPDATE subtitles_provider SET api_key_sealed = NULL, account = NULL, plan = NULL, allowance = NULL, allowance_at = NULL,
    paused_until = CASE WHEN pause_reason = 'key' THEN NULL ELSE paused_until END,
    pause_words = CASE WHEN pause_reason = 'key' THEN NULL ELSE pause_words END,
    pause_reason = CASE WHEN pause_reason = 'key' THEN NULL ELSE pause_reason END
WHERE id = sqlc.arg(id);
