-- name: SetMembersFields :exec
UPDATE mediaservers_server
SET members_see = sqlc.arg(members_see), member_address = sqlc.narg(member_address)
WHERE id = sqlc.arg(id);

-- name: ListMemberLibraries :many
SELECT * FROM mediaservers_member_library WHERE server_id = sqlc.arg(server_id) ORDER BY name, library_key;

-- name: DeleteMemberLibraries :exec
DELETE FROM mediaservers_member_library WHERE server_id = sqlc.arg(server_id);

-- name: InsertMemberLibrary :exec
INSERT INTO mediaservers_member_library (server_id, library_key, name)
VALUES (sqlc.arg(server_id), sqlc.arg(library_key), sqlc.arg(name));

-- name: ListAllMemberLibraries :many
SELECT * FROM mediaservers_member_library ORDER BY server_id, name, library_key;

-- name: ListShownLibraries :many
SELECT DISTINCT library_key, library_name FROM mediaservers_mapping
WHERE server_id = sqlc.arg(server_id) AND shown = 1 AND library_key IS NOT NULL AND library_name IS NOT NULL
ORDER BY library_name, library_key;

-- name: CountMembers :many
SELECT server_id, count(*) AS members FROM mediaservers_member_access
WHERE state = 'active' GROUP BY server_id ORDER BY server_id;
