-- name: ListMappingsIn :many
SELECT * FROM mediaservers_mapping WHERE root_id IN (sqlc.slice(roots)) ORDER BY server_id, root_id;

-- name: ListPlayingFiles :many
SELECT DISTINCT server_id, file_id FROM mediaservers_play
WHERE ended_at IS NULL AND file_id IN (sqlc.slice(files))
ORDER BY server_id, file_id;

-- name: ListUnfiledPlays :many
SELECT server_id, server_path FROM mediaservers_play
WHERE ended_at IS NULL AND file_id IS NULL
ORDER BY server_id, id;

-- name: ListPlayedSinceSnapshot :many
-- Each server and file of files whose Move snapshot from from_root_id a
-- play still open, or seen since it was taken, makes stale.
SELECT DISTINCT c.server_id, c.file_id FROM mediaservers_change c
JOIN mediaservers_play p ON p.server_id = c.server_id AND p.file_id = c.file_id
WHERE c.file_id IN (sqlc.slice(files)) AND c.from_root_id = sqlc.arg(from_root_id) AND c.state = 'prepared'
    AND (p.ended_at IS NULL OR p.last_seen_at >= c.step_at)
ORDER BY c.server_id, c.file_id;

-- name: ListMoveRows :many
SELECT * FROM mediaservers_change
WHERE file_id IN (sqlc.slice(files)) AND from_root_id = sqlc.arg(from_root_id)
ORDER BY server_id, file_id;

-- name: DeleteMovePreparedOf :exec
DELETE FROM mediaservers_change
WHERE server_id = sqlc.arg(server_id) AND file_id = sqlc.arg(file_id) AND from_root_id IS NOT NULL AND state = 'prepared';

-- name: DeleteMovePrepared :exec
DELETE FROM mediaservers_change
WHERE file_id IN (sqlc.slice(files)) AND from_root_id IS NOT NULL AND state = 'prepared';

-- name: StampSourcesGone :exec
UPDATE mediaservers_change SET source_gone_at = sqlc.arg(at)
WHERE file_id IN (sqlc.slice(files)) AND from_root_id = sqlc.arg(from_root_id)
    AND source_gone_at IS NULL AND state <> 'sent';

-- name: ListServersOfMoveFiles :many
SELECT DISTINCT server_id FROM mediaservers_change
WHERE file_id IN (sqlc.slice(files)) AND from_root_id = sqlc.arg(from_root_id) AND state <> 'sent'
ORDER BY server_id;

-- name: CountOpenMovesFrom :one
SELECT count(*) FROM mediaservers_change WHERE from_root_id = sqlc.arg(root_id) AND state <> 'sent';

-- name: ListShowingRoots :many
SELECT DISTINCT root_id FROM mediaservers_mapping
WHERE shown = 1 AND server_id IN (sqlc.slice(servers))
ORDER BY root_id;
