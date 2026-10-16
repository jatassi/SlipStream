-- name: CountEnabledAdmins :one
SELECT COUNT(*) FROM identity_account WHERE admin = 1 AND disabled_at IS NULL;

-- name: ListAccounts :many
SELECT
    a.*,
    (SELECT COUNT(*) FROM identity_passkey AS p WHERE p.account_id = a.id) AS passkey_count
FROM identity_account AS a
ORDER BY a.created_at, a.id;

-- name: SetDisplayName :exec
UPDATE identity_account SET display_name = sqlc.arg(display_name) WHERE id = sqlc.arg(id);

-- name: SetEmail :exec
UPDATE identity_account SET email = sqlc.arg(email) WHERE id = sqlc.arg(id);

-- name: SetAdmin :exec
UPDATE identity_account SET admin = sqlc.arg(admin) WHERE id = sqlc.arg(id);

-- name: SetDisabledAt :exec
UPDATE identity_account SET disabled_at = sqlc.narg(disabled_at) WHERE id = sqlc.arg(id);

-- name: SetKeyPrompt :exec
UPDATE identity_account SET key_prompt = sqlc.arg(key_prompt) WHERE id = sqlc.arg(id);

-- name: DeleteAccount :exec
DELETE FROM identity_account WHERE id = sqlc.arg(id);

-- name: ListSessions :many
SELECT id, device, address, created_at, last_seen_at
FROM identity_session
WHERE account_id = sqlc.arg(account_id)
ORDER BY last_seen_at DESC, id;

-- name: ListSessionIDs :many
SELECT id FROM identity_session WHERE account_id = sqlc.arg(account_id) ORDER BY id;

-- name: DeleteAccountSessions :exec
DELETE FROM identity_session WHERE account_id = sqlc.arg(account_id);

-- name: DeleteAccountSession :execrows
DELETE FROM identity_session WHERE account_id = sqlc.arg(account_id) AND id = sqlc.arg(id);

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
