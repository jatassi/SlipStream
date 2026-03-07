-- name: GetInvitationModuleSettings :many
SELECT * FROM portal_invitation_module_settings
WHERE invitation_id = ?
ORDER BY module_type;

-- name: UpsertInvitationModuleSetting :one
INSERT INTO portal_invitation_module_settings (invitation_id, module_type, quality_profile_id)
VALUES (?, ?, ?)
ON CONFLICT(invitation_id, module_type) DO UPDATE SET
    quality_profile_id = excluded.quality_profile_id
RETURNING *;

-- name: DeleteInvitationModuleSettings :exec
DELETE FROM portal_invitation_module_settings WHERE invitation_id = ?;
