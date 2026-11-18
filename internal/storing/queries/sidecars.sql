-- name: GetSpeechMap :one
SELECT * FROM subtitles_speechmap WHERE file_id = ?;

-- name: PutSpeechMap :exec
INSERT INTO subtitles_speechmap (file_id, root_id, identity, version, stream, frames, density, size, built_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (file_id) DO UPDATE SET
    root_id = excluded.root_id, identity = excluded.identity, version = excluded.version, stream = excluded.stream,
    frames = excluded.frames, density = excluded.density, size = excluded.size, built_at = excluded.built_at;

-- name: DeleteSpeechMap :exec
DELETE FROM subtitles_speechmap WHERE file_id = ?;

-- name: SpeechMapIDs :many
SELECT file_id FROM subtitles_speechmap ORDER BY file_id;

-- name: SpeechMapsFrom :many
SELECT file_id FROM subtitles_speechmap WHERE file_id >= sqlc.arg(after) ORDER BY file_id;

-- name: SpeechMapsBefore :many
SELECT file_id FROM subtitles_speechmap WHERE file_id < sqlc.arg(before) ORDER BY file_id;

-- name: SpeechMapsBetween :many
SELECT file_id FROM subtitles_speechmap WHERE file_id >= sqlc.arg(after) AND file_id < sqlc.arg(before) ORDER BY file_id;

-- name: PutLedger :exec
INSERT INTO subtitles_ledger (root_id, sha256, path, written_at) VALUES (?, ?, ?, ?)
ON CONFLICT (root_id, sha256) DO UPDATE SET path = excluded.path, written_at = excluded.written_at;

-- name: LedgerHolds :many
SELECT sha256 FROM subtitles_ledger WHERE root_id = ? AND sha256 IN (sqlc.slice(hashes)) ORDER BY sha256;
