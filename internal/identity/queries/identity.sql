-- name: HasAdmin :one
SELECT EXISTS (SELECT 1 FROM identity_account WHERE admin = 1);

-- name: InsertAccount :exec
INSERT INTO identity_account (id, email, display_name, admin, pin_hash, created_at)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(display_name), sqlc.arg(admin), sqlc.arg(pin_hash), sqlc.arg(created_at));

-- name: GetAccountByEmail :one
SELECT * FROM identity_account WHERE email = sqlc.arg(email);

-- name: GetAccount :one
SELECT * FROM identity_account WHERE id = sqlc.arg(id);

-- name: StampSignIn :exec
UPDATE identity_account SET last_sign_in_at = sqlc.arg(last_sign_in_at) WHERE id = sqlc.arg(id);

-- name: SetPIN :exec
UPDATE identity_account
SET pin_hash = sqlc.arg(pin_hash), pin_failures = 0, pin_wait_until = NULL, pin_disabled_at = NULL
WHERE id = sqlc.arg(id);

-- name: SavePINLedger :exec
UPDATE identity_account
SET pin_failures = sqlc.arg(pin_failures), pin_wait_until = sqlc.arg(pin_wait_until), pin_disabled_at = sqlc.arg(pin_disabled_at)
WHERE id = sqlc.arg(id);

-- name: InsertSession :exec
INSERT INTO identity_session (id, token_hash, account_id, address, created_at, last_seen_at, confirmed_at, device, pin_required, prompted)
VALUES (
    sqlc.arg(id), sqlc.arg(token_hash), sqlc.arg(account_id), sqlc.arg(address),
    sqlc.arg(created_at), sqlc.arg(last_seen_at), sqlc.arg(confirmed_at), sqlc.arg(device),
    sqlc.arg(pin_required), sqlc.arg(prompted)
);

-- name: GetSession :one
SELECT
    s.id AS session_id,
    s.last_seen_at,
    s.confirmed_at,
    s.pin_required,
    a.id AS account_id,
    a.email,
    a.display_name,
    a.admin,
    a.key_prompt,
    st.session_lifetime_seconds,
    st.reconfirm_window_seconds
FROM identity_session AS s
JOIN identity_account AS a ON a.id = s.account_id
JOIN identity_settings AS st ON st.id = 1
WHERE s.token_hash = sqlc.arg(token_hash);

-- name: GetSessionUse :one
SELECT s.last_seen_at, st.session_lifetime_seconds
FROM identity_session AS s
JOIN identity_settings AS st ON st.id = 1
WHERE s.id = sqlc.arg(id);

-- name: TouchSession :exec
UPDATE identity_session SET last_seen_at = sqlc.arg(last_seen_at) WHERE id = sqlc.arg(id);

-- name: ConfirmSession :exec
UPDATE identity_session SET confirmed_at = sqlc.arg(confirmed_at) WHERE id = sqlc.arg(id);

-- name: ClearPINRequired :exec
UPDATE identity_session SET pin_required = 0 WHERE account_id = sqlc.arg(account_id);

-- name: DeleteSession :exec
DELETE FROM identity_session WHERE id = sqlc.arg(id);

-- name: ListIdleSessions :many
SELECT id, account_id FROM identity_session WHERE last_seen_at < sqlc.arg(seen_before)
ORDER BY account_id, id;

-- name: DeleteIdleSessions :exec
DELETE FROM identity_session WHERE last_seen_at < sqlc.arg(seen_before);

-- name: InsertAPIKey :exec
INSERT INTO identity_api_key (id, token_hash, created_at) VALUES (1, sqlc.arg(token_hash), sqlc.arg(created_at));

-- name: ReplaceAPIKey :exec
UPDATE identity_api_key SET token_hash = sqlc.arg(token_hash), created_at = sqlc.arg(created_at) WHERE id = 1;

-- name: GetAPIKeyHash :one
SELECT token_hash FROM identity_api_key WHERE id = 1;

-- name: GetAPIKeyCreatedAt :one
SELECT created_at FROM identity_api_key WHERE id = 1;

-- name: GetSettings :one
SELECT * FROM identity_settings WHERE id = 1;

-- name: SetSignInSettings :exec
UPDATE identity_settings
SET
    pin_wait_after = sqlc.narg(pin_wait_after),
    pin_wait_seconds = sqlc.narg(pin_wait_seconds),
    pin_off_after = sqlc.narg(pin_off_after),
    address_failures = sqlc.narg(address_failures),
    address_window_seconds = sqlc.narg(address_window_seconds),
    reconfirm_window_seconds = sqlc.narg(reconfirm_window_seconds),
    session_lifetime_seconds = sqlc.narg(session_lifetime_seconds),
    signup_link_lifetime_seconds = sqlc.narg(signup_link_lifetime_seconds),
    reset_link_lifetime_seconds = sqlc.narg(reset_link_lifetime_seconds),
    email_link_lifetime_seconds = sqlc.narg(email_link_lifetime_seconds)
WHERE id = 1;

-- name: SetAddress :exec
UPDATE identity_settings SET address = sqlc.arg(address) WHERE id = 1;
