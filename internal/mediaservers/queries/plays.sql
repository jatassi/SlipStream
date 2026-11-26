-- name: StartPlay :exec
INSERT INTO mediaservers_play (
    server_id, id, session_key, file_id, server_path, path_key, item_keys, player_key, player, platform, product,
    user_name, decision, video, audio, subtitles, hardware, reasons, received, audio_track, subtitle_track,
    started_at, last_seen_at
) VALUES (
    sqlc.arg(server_id), sqlc.arg(id), sqlc.arg(session_key), sqlc.narg(file_id), sqlc.arg(server_path),
    sqlc.arg(path_key), sqlc.arg(item_keys), sqlc.arg(player_key), sqlc.arg(player), sqlc.arg(platform),
    sqlc.arg(product), sqlc.arg(user_name), sqlc.arg(decision), sqlc.arg(video), sqlc.arg(audio),
    sqlc.arg(subtitles), sqlc.arg(hardware), sqlc.arg(reasons), sqlc.arg(received), sqlc.narg(audio_track),
    sqlc.narg(subtitle_track), sqlc.arg(started_at), sqlc.arg(last_seen_at)
)
ON CONFLICT (server_id, id) DO NOTHING;

-- name: GetOpenPlay :one
SELECT last_seen_at, file_id, server_path, path_key, audio_track, subtitle_track,
    CAST(snapshot IS NOT NULL AS BOOLEAN) AS snapped
FROM mediaservers_play
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

-- name: FilePlay :exec
UPDATE mediaservers_play
SET file_id = sqlc.arg(file_id)
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: SetSnapshot :exec
UPDATE mediaservers_play
SET file_id = sqlc.arg(file_id),
    snapshot = sqlc.arg(snapshot)
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id) AND ended_at IS NULL;

-- name: ListUnfiledAt :many
SELECT id FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND file_id IS NULL AND ended_at IS NULL
    AND path_key IN (SELECT value FROM json_each(CAST(sqlc.arg(path_keys) AS TEXT)))
ORDER BY id;

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

-- name: EndServerPlays :many
UPDATE mediaservers_play
SET ended_at = last_seen_at
WHERE server_id = sqlc.arg(server_id) AND ended_at IS NULL
RETURNING server_id;

-- name: EndPlaysOn :many
UPDATE mediaservers_play
SET ended_at = last_seen_at
WHERE ended_at IS NULL
  AND server_id IN (SELECT id FROM mediaservers_server WHERE reaching_node_id = sqlc.arg(node_id))
RETURNING server_id;

-- name: PrunePlays :execrows
DELETE FROM mediaservers_play
WHERE rowid IN (
    SELECT p.rowid FROM mediaservers_play AS p
    WHERE p.ended_at IS NOT NULL AND p.ended_at < sqlc.arg(before)
    ORDER BY p.ended_at
    LIMIT sqlc.arg(n)
);

-- name: MovePlaysSince :exec
UPDATE mediaservers_server SET plays_since = sqlc.arg(cutoff) WHERE plays_since < sqlc.arg(cutoff);

-- name: ListPlaysSince :many
SELECT id, file_id, item_keys, player_key, player, platform, product, decision, video, audio, subtitles,
    reasons, CAST(received AS TEXT) AS received_json, CAST(coalesce(snapshot, '') AS TEXT) AS snapshot_json,
    started_at, last_seen_at
FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND started_at >= sqlc.arg(since)
ORDER BY started_at, id;

-- name: ListPlaysPage :many
SELECT * FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id)
    AND (started_at < sqlc.arg(before_at) OR (started_at = sqlc.arg(before_at) AND id < sqlc.arg(before_id)))
ORDER BY started_at DESC, id DESC
LIMIT sqlc.arg(n);

-- name: ListPlayerPlaysPage :many
SELECT * FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND player_key = sqlc.arg(player)
    AND (started_at < sqlc.arg(before_at) OR (started_at = sqlc.arg(before_at) AND id < sqlc.arg(before_id)))
ORDER BY started_at DESC, id DESC
LIMIT sqlc.arg(n);

-- name: GetPlayStart :one
SELECT started_at FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND id = sqlc.arg(id);

-- name: ListIndexedFiles :many
SELECT file_id, item_key FROM mediaservers_item
WHERE server_id = sqlc.arg(server_id)
    AND file_id IN (SELECT unhex(replace(value, '-', '')) FROM json_each(CAST(sqlc.arg(files) AS TEXT)))
ORDER BY file_id;

-- name: ListSnapshotWorks :many
SELECT id, CAST(snapshot ->> '$.work' AS TEXT) AS work FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id) AND snapshot IS NOT NULL
    AND id IN (SELECT unhex(replace(value, '-', '')) FROM json_each(CAST(sqlc.arg(plays) AS TEXT)))
ORDER BY id;

-- name: ListPlays :many
SELECT * FROM mediaservers_play
WHERE server_id = sqlc.arg(server_id)
ORDER BY started_at, id;
