-- The path index: what listings, lookups and steps learned of each server
-- path, and the listings that keep it.

-- name: UpsertItem :exec
INSERT INTO mediaservers_item (server_id, path_key, server_path, library_key, file_id, item_key, top_key, seen_at, stale)
VALUES (
    sqlc.arg(server_id), sqlc.arg(path_key), sqlc.arg(server_path), sqlc.arg(library_key), sqlc.narg(file_id),
    sqlc.arg(item_key), sqlc.arg(top_key), sqlc.arg(seen_at), 0
)
ON CONFLICT (server_id, path_key) DO UPDATE
SET server_path = excluded.server_path,
    library_key = excluded.library_key,
    file_id = excluded.file_id,
    item_key = excluded.item_key,
    top_key = excluded.top_key,
    seen_at = excluded.seen_at,
    stale = 0;

-- name: DeleteItem :exec
DELETE FROM mediaservers_item WHERE server_id = sqlc.arg(server_id) AND path_key = sqlc.arg(path_key);

-- PruneLibrary forgets each entry of a library its running listing didn't
-- see, once the listing read its last page.
-- name: PruneLibrary :exec
DELETE FROM mediaservers_item AS i
WHERE i.server_id = sqlc.arg(server_id) AND i.library_key = sqlc.arg(library_key)
    AND i.seen_at < (
        SELECT l.started_at FROM mediaservers_listing AS l
        WHERE l.server_id = sqlc.arg(server_id) AND l.library_key = sqlc.arg(library_key)
    );

-- name: ListStaleItems :many
SELECT server_path FROM mediaservers_item
WHERE server_id = sqlc.arg(server_id) AND stale = 1
ORDER BY path_key
LIMIT sqlc.arg(n);

-- name: GetItem :one
SELECT * FROM mediaservers_item WHERE server_id = sqlc.arg(server_id) AND path_key = sqlc.arg(path_key);

-- name: ListListings :many
SELECT * FROM mediaservers_listing WHERE server_id = sqlc.arg(server_id) ORDER BY library_key;

-- name: StartListing :exec
INSERT INTO mediaservers_listing (server_id, library_key, started_at, cursor)
VALUES (sqlc.arg(server_id), sqlc.arg(library_key), sqlc.arg(started_at), '')
ON CONFLICT (server_id, library_key) DO UPDATE SET started_at = excluded.started_at, cursor = '';

-- name: SetListingCursor :exec
UPDATE mediaservers_listing SET cursor = sqlc.arg(cursor)
WHERE server_id = sqlc.arg(server_id) AND library_key = sqlc.arg(library_key);

-- name: EndListing :exec
UPDATE mediaservers_listing SET started_at = NULL, cursor = '', listed_at = sqlc.arg(listed_at)
WHERE server_id = sqlc.arg(server_id) AND library_key = sqlc.arg(library_key);
