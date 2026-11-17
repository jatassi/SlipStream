-- name: ListLive :many
SELECT id FROM downloadclients_torrent WHERE id IN (sqlc.slice(ids)) AND removed_at IS NULL ORDER BY id;
