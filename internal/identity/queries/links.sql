-- name: InsertLink :exec
INSERT INTO identity_link (id, kind, token_hash, token_sealed, email, account_id, remove_passkeys, issued_by, issued_at, expires_at)
VALUES (
    sqlc.arg(id), sqlc.arg(kind), sqlc.arg(token_hash), sqlc.arg(token_sealed), sqlc.arg(email),
    sqlc.narg(account_id), sqlc.arg(remove_passkeys), sqlc.narg(issued_by), sqlc.arg(issued_at), sqlc.arg(expires_at)
);

-- name: WithdrawLiveSignupLink :one
UPDATE identity_link
SET ended_at = sqlc.arg(now), end_reason = 'withdrawn', token_sealed = NULL
WHERE kind = 'signup' AND email = sqlc.arg(email) AND ended_at IS NULL AND expires_at > sqlc.arg(now)
RETURNING issued_at;

-- name: WithdrawLiveAccountLink :one
UPDATE identity_link
SET ended_at = sqlc.arg(now), end_reason = 'withdrawn', token_sealed = NULL
WHERE kind = sqlc.arg(kind) AND account_id = sqlc.arg(account_id) AND ended_at IS NULL AND expires_at > sqlc.arg(now)
RETURNING issued_at;

-- name: EraseExpiredSignupTokens :exec
UPDATE identity_link SET token_sealed = NULL
WHERE kind = 'signup' AND token_sealed IS NOT NULL AND expires_at <= sqlc.arg(now);

-- name: GetLinkByHash :one
SELECT
    l.*,
    COALESCE(issuer.display_name, '') AS issuer_name
FROM identity_link AS l
LEFT JOIN identity_account AS issuer ON issuer.id = l.issued_by
WHERE l.token_hash = sqlc.arg(token_hash);

-- name: RedeemLink :one
UPDATE identity_link
SET ended_at = sqlc.arg(now), end_reason = 'used', token_sealed = NULL
WHERE token_hash = sqlc.arg(token_hash) AND kind = sqlc.arg(kind) AND ended_at IS NULL AND expires_at > sqlc.arg(now)
RETURNING *;

-- name: SetLinkAccount :exec
UPDATE identity_link SET account_id = sqlc.arg(account_id) WHERE id = sqlc.arg(id);

-- name: ListLiveSignupLinks :many
SELECT
    l.id,
    l.email,
    l.issued_at,
    l.expires_at,
    COALESCE(issuer.display_name, '') AS issuer_name
FROM identity_link AS l
LEFT JOIN identity_account AS issuer ON issuer.id = l.issued_by
WHERE l.kind = 'signup' AND l.ended_at IS NULL AND l.expires_at > sqlc.arg(now)
ORDER BY l.issued_at, l.id;

-- name: GetLiveSignupLink :one
SELECT
    l.id,
    l.email,
    l.token_sealed,
    l.issued_at,
    l.expires_at,
    COALESCE(issuer.display_name, '') AS issuer_name
FROM identity_link AS l
LEFT JOIN identity_account AS issuer ON issuer.id = l.issued_by
WHERE l.id = sqlc.arg(id) AND l.kind = 'signup' AND l.ended_at IS NULL AND l.expires_at > sqlc.arg(now);

-- name: WithdrawSignupLink :one
UPDATE identity_link
SET ended_at = sqlc.arg(now), end_reason = 'withdrawn', token_sealed = NULL
WHERE id = sqlc.arg(id) AND kind = 'signup' AND ended_at IS NULL AND expires_at > sqlc.arg(now)
RETURNING email;
