-- name: ListPasskeys :many
SELECT id, name, relying_party, created_at, last_used_at
FROM identity_passkey
WHERE account_id = sqlc.arg(account_id)
ORDER BY created_at, id;

-- name: CountPasskeysAt :one
SELECT COUNT(*) FROM identity_passkey
WHERE account_id = sqlc.arg(account_id) AND relying_party = sqlc.arg(relying_party);

-- name: DeletePasskeys :exec
DELETE FROM identity_passkey WHERE account_id = sqlc.arg(account_id);

-- name: ListCredentialsAt :many
SELECT * FROM identity_passkey
WHERE account_id = sqlc.arg(account_id) AND relying_party = sqlc.arg(relying_party)
ORDER BY created_at, id;

-- name: InsertPasskey :exec
INSERT INTO identity_passkey (
    id, account_id, relying_party, credential_id, public_key, sign_count, aaguid, transports,
    backup_eligible, backup_state, name, created_at
) VALUES (
    sqlc.arg(id), sqlc.arg(account_id), sqlc.arg(relying_party), sqlc.arg(credential_id), sqlc.arg(public_key),
    sqlc.arg(sign_count), sqlc.arg(aaguid), sqlc.arg(transports), sqlc.arg(backup_eligible), sqlc.arg(backup_state),
    sqlc.arg(name), sqlc.arg(created_at)
);

-- name: RecordPasskeyUse :exec
UPDATE identity_passkey
SET sign_count = sqlc.arg(sign_count), backup_state = sqlc.arg(backup_state), last_used_at = sqlc.arg(last_used_at)
WHERE account_id = sqlc.arg(account_id) AND credential_id = sqlc.arg(credential_id);

-- name: RenamePasskey :one
UPDATE identity_passkey SET name = sqlc.arg(name)
WHERE account_id = sqlc.arg(account_id) AND id = sqlc.arg(id)
RETURNING id, name, relying_party, created_at, last_used_at;

-- name: DeletePasskey :execrows
DELETE FROM identity_passkey WHERE account_id = sqlc.arg(account_id) AND id = sqlc.arg(id);
