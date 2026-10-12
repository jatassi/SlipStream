-- name: HasAdmin :one
SELECT EXISTS (SELECT 1 FROM identity_account WHERE admin = 1);

-- name: InsertAccount :exec
INSERT INTO identity_account (id, email, display_name, admin, pin_hash, created_at)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(display_name), sqlc.arg(admin), sqlc.arg(pin_hash), sqlc.arg(created_at));

-- name: GetAccountByEmail :one
SELECT * FROM identity_account WHERE email = sqlc.arg(email);

-- name: InsertSession :exec
INSERT INTO identity_session (id, token_hash, account_id, address, created_at, last_seen_at)
VALUES (sqlc.arg(id), sqlc.arg(token_hash), sqlc.arg(account_id), sqlc.arg(address), sqlc.arg(created_at), sqlc.arg(last_seen_at));

-- name: GetSession :one
SELECT
    s.id AS session_id,
    s.last_seen_at,
    a.id AS account_id,
    a.email,
    a.display_name,
    a.admin
FROM identity_session AS s
JOIN identity_account AS a ON a.id = s.account_id
WHERE s.token_hash = sqlc.arg(token_hash);

-- name: TouchSession :exec
UPDATE identity_session SET last_seen_at = sqlc.arg(last_seen_at) WHERE id = sqlc.arg(id);

-- name: DeleteSession :exec
DELETE FROM identity_session WHERE id = sqlc.arg(id);

-- name: DeleteIdleSessions :exec
DELETE FROM identity_session
WHERE account_id = sqlc.arg(account_id) AND last_seen_at < sqlc.arg(seen_before);

-- name: InsertAPIKey :exec
INSERT INTO identity_api_key (id, token_hash, created_at) VALUES (1, sqlc.arg(token_hash), sqlc.arg(created_at));

-- name: GetAPIKeyHash :one
SELECT token_hash FROM identity_api_key WHERE id = 1;

-- name: GetSessionLifetimeDays :one
SELECT session_lifetime_days FROM identity_settings WHERE id = 1;
