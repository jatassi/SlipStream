-- name: GetPortalUser :one
SELECT * FROM portal_users WHERE id = ? LIMIT 1;

-- name: GetPortalUserByUsername :one
SELECT * FROM portal_users WHERE username = ? LIMIT 1;

-- name: ListPortalUsers :many
SELECT * FROM portal_users ORDER BY created_at DESC;

-- name: ListEnabledPortalUsers :many
SELECT * FROM portal_users WHERE enabled = 1 ORDER BY display_name;

-- name: CreatePortalUser :one
INSERT INTO portal_users (
    username, password_hash, display_name, quality_profile_id, auto_approve, enabled
) VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdatePortalUser :one
UPDATE portal_users SET
    username = ?,
    display_name = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdatePortalUserPassword :exec
UPDATE portal_users SET
    password_hash = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdatePortalUserQualityProfile :one
UPDATE portal_users SET
    quality_profile_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdatePortalUserAutoApprove :one
UPDATE portal_users SET
    auto_approve = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdatePortalUserEnabled :one
UPDATE portal_users SET
    enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeletePortalUser :exec
DELETE FROM portal_users WHERE id = ?;

-- name: CountPortalUsers :one
SELECT COUNT(*) FROM portal_users;

-- name: CountEnabledPortalUsers :one
SELECT COUNT(*) FROM portal_users WHERE enabled = 1;

-- name: GetAdminUser :one
SELECT * FROM portal_users WHERE is_admin = 1 LIMIT 1;

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM portal_users WHERE is_admin = 1;

-- name: CreateAdminUser :one
INSERT INTO portal_users (
    username, password_hash, display_name, is_admin, auto_approve, enabled
) VALUES (?, ?, ?, 1, 1, 1)
RETURNING *;

-- name: CreatePortalUserWithID :one
-- Used for copying users to dev database while preserving IDs
INSERT INTO portal_users (
    id, username, password_hash, display_name, quality_profile_id, auto_approve, enabled, is_admin
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;
