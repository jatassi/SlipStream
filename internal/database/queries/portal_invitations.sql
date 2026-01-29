-- name: GetPortalInvitation :one
SELECT * FROM portal_invitations WHERE id = ? LIMIT 1;

-- name: GetPortalInvitationByToken :one
SELECT * FROM portal_invitations WHERE token = ? LIMIT 1;

-- name: GetPortalInvitationByUsername :one
SELECT * FROM portal_invitations WHERE username = ? ORDER BY created_at DESC LIMIT 1;

-- name: ListPortalInvitations :many
SELECT * FROM portal_invitations ORDER BY created_at DESC;

-- name: ListPendingPortalInvitations :many
SELECT * FROM portal_invitations
WHERE used_at IS NULL
ORDER BY created_at DESC;

-- name: CreatePortalInvitation :one
INSERT INTO portal_invitations (username, token, expires_at, quality_profile_id, auto_approve)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: MarkPortalInvitationUsed :exec
UPDATE portal_invitations SET used_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdatePortalInvitationToken :one
UPDATE portal_invitations SET
    token = ?,
    expires_at = ?,
    used_at = NULL
WHERE id = ?
RETURNING *;

-- name: DeletePortalInvitation :exec
DELETE FROM portal_invitations WHERE id = ?;

-- name: DeleteExpiredPortalInvitations :exec
DELETE FROM portal_invitations
WHERE expires_at < CURRENT_TIMESTAMP AND used_at IS NULL;

-- name: CountPendingPortalInvitations :one
SELECT COUNT(*) FROM portal_invitations WHERE used_at IS NULL;
