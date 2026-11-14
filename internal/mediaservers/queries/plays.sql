-- name: StartPlay :exec
INSERT INTO mediaservers_play (
    server_id, id, session_key, file_id, server_path, user_name, player, product,
    decision, hardware, reasons, started_at, last_seen_at
) VALUES (
    sqlc.arg(server_id), sqlc.arg(id), sqlc.arg(session_key), sqlc.narg(file_id), sqlc.arg(server_path),
    sqlc.arg(user_name), sqlc.arg(player), sqlc.arg(product), sqlc.arg(decision), sqlc.arg(hardware),
    sqlc.arg(reasons), sqlc.arg(started_at), sqlc.arg(last_seen_at)
)
ON CONFLICT (server_id, id) DO NOTHING;

-- name: GetOpenPlay :one
SELECT last_seen_at FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: ChangePlay :exec
UPDATE mediaservers_play
SET decision = sqlc.arg(decision),
    hardware = sqlc.arg(hardware),
    reasons = sqlc.arg(reasons),
    last_seen_at = sqlc.arg(last_seen_at)
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: SeePlay :exec
UPDATE mediaservers_play
SET last_seen_at = sqlc.arg(last_seen_at)
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: EndPlay :exec
UPDATE mediaservers_play
SET last_seen_at = sqlc.arg(ended_at),
    ended_at = sqlc.arg(ended_at)
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: EndPlayAsSeen :exec
UPDATE mediaservers_play
SET ended_at = last_seen_at
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: ListOpenPlays :many
SELECT id FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND ended_at IS NULL
ORDER BY id;

-- name: EndServerPlays :execrows
UPDATE mediaservers_play
SET ended_at = last_seen_at
WHERE server_id = sqlc.arg(server_id) AND ended_at IS NULL;

-- name: EndPlaysOn :execrows
UPDATE mediaservers_play
SET ended_at = last_seen_at
WHERE ended_at IS NULL
  AND server_id IN (SELECT id FROM mediaservers_server WHERE reaching_node_id = sqlc.arg(node_id));

-- name: PrunePlays :execrows
DELETE FROM mediaservers_play
WHERE ended_at IS NOT NULL AND ended_at < sqlc.arg(before);

-- name: ListPlays :many
SELECT * FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id)
ORDER BY started_at, id;
