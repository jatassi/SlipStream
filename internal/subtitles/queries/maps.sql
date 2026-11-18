-- name: MarkMapsDue :exec
-- A file's map is due while a want of it is unmet or a Sidecar of it waits
-- to be timed again, and it has none current. A file never surveyed waits
-- for its survey, so found subtitles count first.
UPDATE subtitles_file SET map_state = 'due'
WHERE subtitles_file.file_id IN (sqlc.slice(ids)) AND map_state = 'idle' AND kind IS NOT NULL AND wantless = 0 AND surveyed_at IS NOT NULL
    AND (EXISTS (SELECT 1 FROM subtitles_want w WHERE w.file_id = subtitles_file.file_id AND w.unmet = 1)
        OR EXISTS (SELECT 1 FROM subtitles_sidecar c WHERE c.file_id = subtitles_file.file_id AND c.timing_due = 1));

-- name: ForgetMap :exec
-- The file's bytes changed: its map, and any fault its old bytes had, no
-- longer hold.
UPDATE subtitles_file SET map_state = 'idle', map_fault = NULL, map_words = NULL, map_built_at = NULL, map_tries = 0, map_next_at = NULL
WHERE file_id = sqlc.arg(file_id);

-- name: MapStale :exec
UPDATE subtitles_file SET map_state = 'due', map_built_at = NULL WHERE file_id = sqlc.arg(file_id) AND map_state = 'built';

-- name: ListMapRoots :many
SELECT DISTINCT root_id FROM subtitles_file WHERE map_state = 'due' ORDER BY root_id;

-- name: NextMaps :many
SELECT * FROM subtitles_file WHERE map_state = 'due' AND root_id = sqlc.arg(root_id)
ORDER BY map_rank, map_since DESC LIMIT sqlc.arg(n);

-- name: StartMap :execrows
UPDATE subtitles_file SET map_state = 'building' WHERE file_id = sqlc.arg(file_id) AND map_state = 'due';

-- name: ResetBuilding :exec
-- A build the last process started is due again: its call ended with it.
UPDATE subtitles_file SET map_state = 'due' WHERE map_state = 'building';

-- name: MapBuilt :exec
UPDATE subtitles_file SET map_state = 'built', map_fault = NULL, map_words = NULL, map_node_id = sqlc.arg(node_id), map_stream = sqlc.arg(stream),
    map_identity = sqlc.arg(identity), map_version = sqlc.arg(version), map_frames = sqlc.arg(frames), map_density = sqlc.arg(density),
    map_tries = 0, map_next_at = NULL, map_built_at = sqlc.arg(built_at)
WHERE file_id = sqlc.arg(file_id) AND map_state = 'building';

-- name: MapFailed :exec
UPDATE subtitles_file SET map_state = 'failed', map_fault = sqlc.arg(fault), map_words = sqlc.narg(words), map_node_id = sqlc.arg(node_id),
    map_tries = map_tries + 1, map_next_at = sqlc.narg(next_at)
WHERE file_id = sqlc.arg(file_id) AND map_state = 'building';

-- name: MapDueAgain :exec
UPDATE subtitles_file SET map_state = 'due' WHERE file_id = sqlc.arg(file_id) AND map_state = 'building';

-- name: RetryMaps :exec
UPDATE subtitles_file SET map_state = 'due', map_fault = NULL, map_words = NULL
WHERE map_state = 'failed' AND map_next_at IS NOT NULL AND map_next_at <= sqlc.arg(now);

-- name: GetMapSlots :one
SELECT CAST(coalesce((SELECT map_slots FROM subtitles_node WHERE node_id = sqlc.arg(node_id)), 0) AS INTEGER) AS slots;

-- name: NodeFault :exec
INSERT INTO subtitles_node (node_id, fault_at, fault_words) VALUES (sqlc.arg(node_id), sqlc.arg(fault_at), sqlc.arg(fault_words))
ON CONFLICT (node_id) DO UPDATE SET fault_at = excluded.fault_at, fault_words = excluded.fault_words;

-- name: NodeBuilt :exec
UPDATE subtitles_node SET fault_at = NULL, fault_words = NULL WHERE node_id = sqlc.arg(node_id) AND fault_at IS NOT NULL;
