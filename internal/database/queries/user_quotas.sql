-- name: GetUserQuota :one
SELECT * FROM user_quotas WHERE user_id = ? LIMIT 1;

-- name: ListUserQuotas :many
SELECT * FROM user_quotas ORDER BY user_id;

-- name: CreateUserQuota :one
INSERT INTO user_quotas (
    user_id, movies_limit, seasons_limit, episodes_limit,
    movies_used, seasons_used, episodes_used, period_start
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpsertUserQuota :one
INSERT INTO user_quotas (
    user_id, movies_limit, seasons_limit, episodes_limit,
    movies_used, seasons_used, episodes_used, period_start
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    movies_limit = excluded.movies_limit,
    seasons_limit = excluded.seasons_limit,
    episodes_limit = excluded.episodes_limit,
    movies_used = excluded.movies_used,
    seasons_used = excluded.seasons_used,
    episodes_used = excluded.episodes_used,
    period_start = excluded.period_start,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateUserQuotaLimits :one
UPDATE user_quotas SET
    movies_limit = ?,
    seasons_limit = ?,
    episodes_limit = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ?
RETURNING *;

-- name: IncrementMovieQuota :one
UPDATE user_quotas SET
    movies_used = movies_used + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ?
RETURNING *;

-- name: IncrementSeasonQuota :one
UPDATE user_quotas SET
    seasons_used = seasons_used + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ?
RETURNING *;

-- name: IncrementEpisodeQuota :one
UPDATE user_quotas SET
    episodes_used = episodes_used + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ?
RETURNING *;

-- name: ResetUserQuota :one
UPDATE user_quotas SET
    movies_used = 0,
    seasons_used = 0,
    episodes_used = 0,
    period_start = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ?
RETURNING *;

-- name: ResetAllQuotas :exec
UPDATE user_quotas SET
    movies_used = 0,
    seasons_used = 0,
    episodes_used = 0,
    period_start = ?,
    updated_at = CURRENT_TIMESTAMP;

-- name: DeleteUserQuota :exec
DELETE FROM user_quotas WHERE user_id = ?;
