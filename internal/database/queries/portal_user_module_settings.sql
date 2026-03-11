-- name: GetUserModuleSettings :one
SELECT * FROM portal_user_module_settings
WHERE user_id = ? AND module_type = ? LIMIT 1;

-- name: ListUserModuleSettings :many
SELECT * FROM portal_user_module_settings
WHERE user_id = ?
ORDER BY module_type;

-- name: UpsertUserModuleSettings :one
INSERT INTO portal_user_module_settings (
    user_id, module_type, quota_limit, quota_used, quality_profile_id, period_start
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, module_type) DO UPDATE SET
    quota_limit = excluded.quota_limit,
    quota_used = excluded.quota_used,
    quality_profile_id = excluded.quality_profile_id,
    period_start = excluded.period_start,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateUserModuleQuotaLimit :one
UPDATE portal_user_module_settings SET
    quota_limit = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: UpdateUserModuleQualityProfile :one
UPDATE portal_user_module_settings SET
    quality_profile_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: IncrementUserModuleQuota :one
UPDATE portal_user_module_settings SET
    quota_used = quota_used + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: ResetUserModuleQuota :one
UPDATE portal_user_module_settings SET
    quota_used = 0,
    period_start = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: ResetAllModuleQuotas :exec
UPDATE portal_user_module_settings SET
    quota_used = 0,
    period_start = ?,
    updated_at = CURRENT_TIMESTAMP;

-- name: DeleteUserModuleSettings :exec
DELETE FROM portal_user_module_settings WHERE user_id = ?;

-- name: DeleteUserModuleSettingsForModule :exec
DELETE FROM portal_user_module_settings WHERE user_id = ? AND module_type = ?;
