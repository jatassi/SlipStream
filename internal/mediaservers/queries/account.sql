-- name: GetPlexAccount :one
SELECT * FROM mediaservers_plex_account WHERE id = 1;

-- name: InsertPlexDevice :exec
INSERT INTO mediaservers_plex_account (id, client_identifier, key_sealed)
VALUES (1, sqlc.arg(client_identifier), sqlc.arg(key_sealed));

-- name: SignInToPlex :exec
UPDATE mediaservers_plex_account
SET jwt_sealed = sqlc.arg(jwt_sealed),
    jwt_expires_at = sqlc.arg(jwt_expires_at),
    account_name = sqlc.arg(account_name),
    plex_pass = sqlc.arg(plex_pass),
    signed_in_at = sqlc.arg(signed_in_at),
    renewed_at = NULL,
    refused_at = NULL
WHERE id = 1;

-- name: RenewPlex :exec
UPDATE mediaservers_plex_account
SET jwt_sealed = sqlc.arg(jwt_sealed),
    jwt_expires_at = sqlc.arg(jwt_expires_at),
    renewed_at = sqlc.arg(renewed_at),
    refused_at = NULL
WHERE id = 1;

-- name: RefusePlex :execrows
UPDATE mediaservers_plex_account
SET refused_at = sqlc.arg(refused_at)
WHERE id = 1 AND jwt_sealed IS NOT NULL AND refused_at IS NULL;

-- name: SignOutOfPlex :exec
UPDATE mediaservers_plex_account
SET jwt_sealed = NULL, jwt_expires_at = NULL, account_name = NULL, plex_pass = NULL,
    signed_in_at = NULL, renewed_at = NULL, refused_at = NULL
WHERE id = 1;

-- name: ListPlexServers :many
SELECT * FROM mediaservers_server WHERE kind = 'plex' ORDER BY id;

-- name: DeletePlexServers :many
DELETE FROM mediaservers_server WHERE kind = 'plex' RETURNING id, reaching_node_id;
