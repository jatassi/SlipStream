-- name: GetSettings :one
SELECT * FROM mail_settings WHERE id = 1;

-- name: SaveSettings :exec
UPDATE mail_settings
SET
    host = sqlc.arg(host),
    port = sqlc.arg(port),
    security = sqlc.arg(security),
    username = sqlc.arg(username),
    password_sealed = sqlc.arg(password_sealed),
    from_address = sqlc.arg(from_address),
    from_name = sqlc.arg(from_name)
WHERE id = 1;
