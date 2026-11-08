-- name: ListTorrents :many
SELECT info_hash, save_folder, private, resume, announced_seconds, failed FROM downloading_torrent ORDER BY info_hash;

-- name: PutTorrent :one
-- A torrent fetched again keeps the larger count of announced seed time,
-- its own or the Controller's baseline.
INSERT INTO downloading_torrent (info_hash, save_folder, private, announced_seconds) VALUES (?, ?, ?, ?)
ON CONFLICT (info_hash) DO UPDATE SET
    save_folder = excluded.save_folder, private = excluded.private, resume = NULL,
    announced_seconds = MAX(announced_seconds, excluded.announced_seconds)
RETURNING announced_seconds;

-- name: SetResume :exec
UPDATE downloading_torrent SET resume = ? WHERE info_hash = ?;

-- name: SetFailed :exec
UPDATE downloading_torrent SET failed = sqlc.narg(failed) WHERE info_hash = sqlc.arg(info_hash);

-- name: StoreAnnounced :exec
UPDATE downloading_torrent SET announced_seconds = MAX(announced_seconds, sqlc.arg(announced_seconds))
WHERE info_hash = sqlc.arg(info_hash);

-- name: DeleteTorrent :exec
DELETE FROM downloading_torrent WHERE info_hash = ?;

-- name: DeleteTorrents :exec
DELETE FROM downloading_torrent;
