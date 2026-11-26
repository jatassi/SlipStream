-- name: InsertAccess :exec
INSERT INTO mediaservers_member_access (
    id, server_id, way, account_id, member_name, state, changed_at, offered_by, offered_at, plex_id, plex_uuid, plex_home
) VALUES (
    sqlc.arg(id), sqlc.arg(server_id), sqlc.arg(way), sqlc.arg(account_id), sqlc.arg(member_name), sqlc.arg(state),
    sqlc.arg(at), sqlc.narg(offered_by), sqlc.arg(at), sqlc.narg(plex_id), sqlc.narg(plex_uuid), sqlc.arg(plex_home)
);

-- name: GetAccess :one
SELECT * FROM mediaservers_member_access WHERE id = sqlc.arg(id);

-- name: ListAccessOf :many
SELECT * FROM mediaservers_member_access WHERE account_id = sqlc.arg(account_id) ORDER BY offered_at, id;

-- name: ListAccessIn :many
SELECT * FROM mediaservers_member_access WHERE state = sqlc.arg(state) ORDER BY changed_at, id;

-- name: ListAccess :many
SELECT * FROM mediaservers_member_access ORDER BY offered_at, id;

-- name: GetAccessOn :one
SELECT * FROM mediaservers_member_access WHERE server_id = sqlc.arg(server_id) AND account_id = sqlc.arg(account_id);

-- name: MoveAccess :execrows
UPDATE mediaservers_member_access
SET state = sqlc.arg(state), changed_at = sqlc.arg(changed_at), account_id = sqlc.narg(account_id),
    plex_id = sqlc.narg(plex_id), plex_uuid = sqlc.narg(plex_uuid), shared_server_id = sqlc.narg(shared_server_id),
    by_hand = sqlc.arg(by_hand), plex_home = sqlc.arg(plex_home), server_user_id = sqlc.narg(server_user_id),
    username = sqlc.narg(username), problem_code = sqlc.narg(problem_code), problem_detail = sqlc.narg(problem_detail)
WHERE id = sqlc.arg(id) AND state = sqlc.arg(from_state);

-- name: DeleteAccessIn :execrows
DELETE FROM mediaservers_member_access WHERE id = sqlc.arg(id) AND state = sqlc.arg(state);

-- name: ListMemberPlex :many
SELECT * FROM mediaservers_member_plex ORDER BY account_id;

-- name: GetMemberPlex :one
SELECT * FROM mediaservers_member_plex WHERE account_id = sqlc.arg(account_id);

-- name: IdleMemberToken :one
SELECT client_identifier, token_sealed FROM mediaservers_member_plex AS p
WHERE p.account_id = sqlc.arg(account_id) AND p.token_sealed IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM mediaservers_member_access AS a
        WHERE a.account_id = p.account_id AND a.state IN ('waiting', 'shared')
    );

-- name: ClearMemberToken :exec
UPDATE mediaservers_member_plex SET token_sealed = NULL WHERE account_id = sqlc.arg(account_id);
