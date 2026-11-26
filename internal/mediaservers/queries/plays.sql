-- name: StartPlay :exec
INSERT INTO mediaservers_play (
    server_id, id, session_key, file_id, server_path, path_key, item_key, player_key, player, platform, product,
    user_name, decision, video, audio, subtitles, hardware, reasons, received, audio_track, subtitle_track,
    started_at, last_seen_at
) VALUES (
    sqlc.arg(server_id), sqlc.arg(id), sqlc.arg(session_key), sqlc.narg(file_id), sqlc.arg(server_path),
    sqlc.arg(path_key), sqlc.arg(item_key), sqlc.arg(player_key), sqlc.arg(player), sqlc.arg(platform),
    sqlc.arg(product), sqlc.arg(user_name), sqlc.arg(decision), sqlc.arg(video), sqlc.arg(audio),
    sqlc.arg(subtitles), sqlc.arg(hardware), sqlc.arg(reasons), sqlc.arg(received), sqlc.narg(audio_track),
    sqlc.narg(subtitle_track), sqlc.arg(started_at), sqlc.arg(last_seen_at)
)
ON CONFLICT (server_id, id) DO NOTHING;

-- name: GetOpenPlay :one
SELECT last_seen_at FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: ChangePlay :exec
UPDATE mediaservers_play
SET decision = sqlc.arg(decision),
    video = sqlc.arg(video),
    audio = sqlc.arg(audio),
    subtitles = sqlc.arg(subtitles),
    hardware = sqlc.arg(hardware),
    reasons = sqlc.arg(reasons),
    received = sqlc.arg(received),
    audio_track = sqlc.narg(audio_track),
    subtitle_track = sqlc.narg(subtitle_track),
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
