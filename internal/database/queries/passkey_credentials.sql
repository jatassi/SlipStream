-- name: CreatePasskeyCredential :exec
INSERT INTO passkey_credentials (
    id, user_id, credential_id, public_key, attestation_type,
    transport, flags_user_present, flags_user_verified,
    flags_backup_eligible, flags_backup_state, sign_count, name
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPasskeyCredentialsByUserID :many
SELECT * FROM passkey_credentials WHERE user_id = ? ORDER BY created_at DESC;

-- name: GetPasskeyCredentialByCredentialID :one
SELECT * FROM passkey_credentials WHERE credential_id = ?;

-- name: GetPasskeyCredentialByID :one
SELECT * FROM passkey_credentials WHERE id = ?;

-- name: UpdatePasskeyCredentialSignCount :exec
UPDATE passkey_credentials
SET sign_count = ?, last_used_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdatePasskeyCredentialName :exec
UPDATE passkey_credentials SET name = ? WHERE id = ? AND user_id = ?;

-- name: DeletePasskeyCredential :exec
DELETE FROM passkey_credentials WHERE id = ? AND user_id = ?;

-- name: GetAllPasskeyCredentialsForLogin :many
SELECT pc.*, pu.username, pu.is_admin
FROM passkey_credentials pc
JOIN portal_users pu ON pc.user_id = pu.id
WHERE pu.enabled = 1;
