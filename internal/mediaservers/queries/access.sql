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
