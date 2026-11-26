-- name: InsertMemberDevice :exec
INSERT INTO mediaservers_member_plex (account_id, client_identifier)
VALUES (sqlc.arg(account_id), sqlc.arg(client_identifier));

-- name: HolderOfPlex :one
SELECT account_id FROM mediaservers_member_plex WHERE plex_id = sqlc.arg(plex_id);

-- name: SignMemberIn :exec
UPDATE mediaservers_member_plex
SET plex_id = sqlc.arg(plex_id),
    plex_uuid = sqlc.arg(plex_uuid),
    username = sqlc.arg(username),
    email = sqlc.arg(email),
    signed_in_at = sqlc.arg(signed_in_at),
    token_sealed = sqlc.arg(token_sealed)
WHERE account_id = sqlc.arg(account_id);

-- name: ForgetMemberPlex :exec
UPDATE mediaservers_member_plex
SET plex_id = NULL, plex_uuid = NULL, username = NULL, email = NULL, signed_in_at = NULL, token_sealed = NULL
WHERE account_id = sqlc.arg(account_id);

-- name: ListMemberTokens :many
SELECT account_id, client_identifier, token_sealed FROM mediaservers_member_plex
WHERE token_sealed IS NOT NULL
ORDER BY account_id;

-- name: ClearMemberTokens :exec
UPDATE mediaservers_member_plex SET token_sealed = NULL WHERE token_sealed IS NOT NULL;
