-- name: ListClientsOfTorrents :many
-- The external client of each of ids an external client holds.
SELECT t.id AS torrent_id, sqlc.embed(c) FROM downloadclients_torrent t JOIN downloadclients_client c ON c.id = t.client_id
WHERE t.id IN (sqlc.slice(ids));

-- name: ListLiveOfClient :many
SELECT id FROM downloadclients_torrent WHERE client_id = sqlc.arg(client_id) AND removed_at IS NULL ORDER BY id;
