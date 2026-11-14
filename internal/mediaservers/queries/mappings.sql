-- name: ListMappings :many
SELECT * FROM mediaservers_mapping WHERE server_id = sqlc.arg(server_id) ORDER BY root_id;

-- name: ListAllMappings :many
SELECT * FROM mediaservers_mapping ORDER BY server_id, root_id;

-- name: UpsertMapping :exec
INSERT INTO mediaservers_mapping (
    server_id, root_id, shown, server_folder, library_key, library_name, library_folder, guess, confirmed_at, confirmed_place
) VALUES (
    sqlc.arg(server_id), sqlc.arg(root_id), sqlc.arg(shown), sqlc.narg(server_folder), sqlc.narg(library_key),
    sqlc.narg(library_name), sqlc.narg(library_folder), 0, sqlc.arg(confirmed_at), sqlc.arg(confirmed_place)
)
ON CONFLICT (server_id, root_id) DO UPDATE
SET shown = excluded.shown,
    server_folder = excluded.server_folder,
    library_key = excluded.library_key,
    library_name = excluded.library_name,
    library_folder = excluded.library_folder,
    guess = 0,
    confirmed_at = excluded.confirmed_at,
    confirmed_place = excluded.confirmed_place;

-- name: DeleteMapping :exec
DELETE FROM mediaservers_mapping WHERE server_id = sqlc.arg(server_id) AND root_id = sqlc.arg(root_id);

-- Unconfirm is a re-home: the root's place moved past the one the Admin
-- confirmed.
-- name: Unconfirm :exec
UPDATE mediaservers_mapping SET confirmed_at = NULL, confirmed_place = NULL
WHERE server_id = sqlc.arg(server_id) AND root_id = sqlc.arg(root_id);

-- QueueLibrary queues a whole library refresh of a root's library, unless
-- one is open for it already.
-- name: QueueLibrary :exec
INSERT INTO mediaservers_change (id, server_id, kind, root_id, path, state, next_attempt_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(server_id), 'library', sqlc.arg(root_id), '', 'due', sqlc.arg(at), sqlc.arg(at))
ON CONFLICT (server_id, root_id, path)
    WHERE from_path IS NULL AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted')
    DO NOTHING;
