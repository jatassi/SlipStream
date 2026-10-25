-- name: ListTorrents :many
SELECT info_hash, save_folder, private, resume FROM downloading_torrent ORDER BY info_hash;

-- name: PutTorrent :exec
INSERT INTO downloading_torrent (info_hash, save_folder, private) VALUES (?, ?, ?)
ON CONFLICT (info_hash) DO UPDATE SET save_folder = excluded.save_folder, private = excluded.private, resume = NULL;

-- name: SetResume :exec
UPDATE downloading_torrent SET resume = ? WHERE info_hash = ?;

-- name: DeleteTorrent :exec
DELETE FROM downloading_torrent WHERE info_hash = ?;

-- name: DeleteTorrents :exec
DELETE FROM downloading_torrent;
