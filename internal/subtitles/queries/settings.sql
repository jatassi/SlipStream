-- name: GetSettings :one
SELECT * FROM subtitles_settings WHERE id = 1;

-- name: SetSettings :exec
UPDATE subtitles_settings SET search_on = sqlc.narg(search_on), search_upgrades = sqlc.narg(search_upgrades),
    longest_gap_seconds = sqlc.narg(longest_gap_seconds), full_pace_seconds = sqlc.narg(full_pace_seconds),
    report_gap_seconds = sqlc.narg(report_gap_seconds), tries_per_search = sqlc.narg(tries_per_search),
    sync_match = sqlc.narg(sync_match), sync_max_offset_seconds = sqlc.narg(sync_max_offset_seconds),
    sync_framerates = sqlc.narg(sync_framerates), provider_warn_after_seconds = sqlc.narg(provider_warn_after_seconds)
WHERE id = 1;

-- name: GetNodeSlots :one
SELECT map_slots FROM subtitles_node WHERE node_id = sqlc.arg(node_id);

-- name: SetNodeSlots :exec
-- A row with neither an override nor a fault is deleted.
INSERT INTO subtitles_node (node_id, map_slots) VALUES (sqlc.arg(node_id), sqlc.narg(map_slots))
ON CONFLICT (node_id) DO UPDATE SET map_slots = excluded.map_slots;

-- name: DropIdleNode :exec
DELETE FROM subtitles_node WHERE node_id = sqlc.arg(node_id) AND map_slots IS NULL AND fault_at IS NULL;

-- name: MapsDueByRoot :many
SELECT root_id, count(*) AS due FROM subtitles_file WHERE map_state = 'due' GROUP BY root_id ORDER BY root_id;

-- name: ListNodeFaults :many
SELECT node_id, fault_at, fault_words FROM subtitles_node WHERE fault_at IS NOT NULL ORDER BY node_id;

-- name: ListWantedTypes :many
-- The Content Types whose Profiles list a Wanted Language.
SELECT DISTINCT p.content_type FROM targets_profile p
WHERE EXISTS (SELECT 1 FROM targets_profile_language l WHERE l.profile_id = p.id)
ORDER BY p.content_type;

-- name: PutIgnored :exec
INSERT INTO subtitles_track_ignored (file_id, stream_index, codec, language, forced)
VALUES (sqlc.arg(file_id), sqlc.arg(stream_index), sqlc.arg(codec), sqlc.arg(language), sqlc.arg(forced))
ON CONFLICT (file_id, stream_index) DO UPDATE SET codec = excluded.codec, language = excluded.language, forced = excluded.forced;
