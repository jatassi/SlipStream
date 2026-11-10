-- name: IsBlocked :one
SELECT EXISTS (SELECT 1 FROM grabs_blocklist WHERE info_hash = sqlc.arg(info_hash));

-- name: InsertBlocked :exec
-- A hash already listed keeps its first entry.
INSERT INTO grabs_blocklist (info_hash, release_title, release_size, indexer_id, indexer_name, tracker_id, work_id, grab_id, cause,
    client_name, added_by, added_by_name, added_at)
VALUES (sqlc.arg(info_hash), sqlc.arg(release_title), sqlc.arg(release_size), sqlc.narg(indexer_id), sqlc.narg(indexer_name),
    sqlc.narg(tracker_id), sqlc.narg(work_id), sqlc.narg(grab_id), sqlc.arg(cause), sqlc.narg(client_name), sqlc.narg(added_by),
    sqlc.narg(added_by_name), sqlc.arg(added_at))
ON CONFLICT (info_hash) DO NOTHING;

-- name: ListBlocksOf :many
SELECT * FROM grabs_blocklist WHERE info_hash IN (sqlc.slice(info_hashes)) ORDER BY info_hash;

-- name: ListBlocked :many
SELECT * FROM grabs_blocklist
WHERE instr(lower(release_title), lower(sqlc.arg(title))) > 0
ORDER BY added_at DESC, info_hash DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(skip);

-- name: CountBlocked :one
SELECT count(*) FROM grabs_blocklist WHERE instr(lower(release_title), lower(sqlc.arg(title))) > 0;

-- name: DeleteBlocked :exec
DELETE FROM grabs_blocklist WHERE info_hash IN (sqlc.slice(info_hashes));

-- name: ListDownloadingGrabsOfTorrents :many
SELECT id, torrent_id, work_id, indexer_id, tracker_id, release_title, release_size FROM grabs_grab
WHERE stage = 'downloading' AND torrent_id IN (sqlc.slice(torrent_ids))
ORDER BY created_at, id;
