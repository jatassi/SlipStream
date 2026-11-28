-- name: InsertNudge :exec
INSERT INTO conversions_nudge (work_id, why, since, backfill_id, created_at) VALUES (?, ?, ?, ?, ?);

-- name: TakeNudges :many
SELECT * FROM conversions_nudge ORDER BY id LIMIT ?;

-- name: DeleteNudges :exec
DELETE FROM conversions_nudge WHERE id IN (sqlc.slice(ids));

-- name: InsertBackfill :exec
INSERT INTO conversions_backfill (id, profile_id, target_id, kind, created_at, count) VALUES (?, ?, ?, ?, ?, 0);

-- name: AddToBackfill :exec
UPDATE conversions_backfill SET count = count + @n WHERE id = @id;

-- name: ListOpenOfWorks :many
SELECT * FROM conversions_conversion WHERE work_id IN (sqlc.slice(work_ids)) AND finished_at IS NULL ORDER BY id;

-- name: ListDeclinedOf :many
SELECT * FROM conversions_declined WHERE source_file_id IN (sqlc.slice(source_file_ids)) ORDER BY source_file_id, target_id;

-- name: DeleteDecline :exec
DELETE FROM conversions_declined WHERE source_file_id = ? AND target_id = ?;

-- name: DeleteDeclinesOfSource :execrows
DELETE FROM conversions_declined WHERE source_file_id = ?;

-- name: TailOfLane :one
SELECT CAST(coalesce(max(position), '') AS TEXT) FROM conversions_conversion WHERE lane = ? AND finished_at IS NULL;

-- name: NextInLane :one
SELECT CAST(coalesce(min(position), '') AS TEXT) FROM conversions_conversion
WHERE lane = ? AND position > ? AND finished_at IS NULL;

-- name: InsertConversion :exec
INSERT INTO conversions_conversion (id, work_id, type, items, edition, source_file_id, target_id, version, lane, position,
    backfill_id, stage, not_before, snapshot, queued_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'queued', ?, ?, ?);

-- name: MoveConversion :exec
UPDATE conversions_conversion SET lane = ?, position = ? WHERE id = ?;

-- name: ListOpenOfSource :many
SELECT id, work_id FROM conversions_conversion WHERE source_file_id = ? AND finished_at IS NULL ORDER BY id;

-- name: ListOpenOfBackfill :many
SELECT id FROM conversions_conversion WHERE backfill_id = ? AND finished_at IS NULL ORDER BY id;

-- name: ListCellsOf :many
SELECT c.id, c.items, c.edition, c.target_id, c.stage, c.node_id, c.not_before, c.failure_code,
    CAST(coalesce((SELECT a.node_name FROM conversions_attempt AS a WHERE a.conversion_id = c.id ORDER BY a.n DESC LIMIT 1), '') AS TEXT) AS node_name
FROM conversions_conversion AS c
WHERE c.work_id IN (sqlc.slice(work_ids)) AND c.finished_at IS NULL AND c.target_id IS NOT NULL
ORDER BY c.id;

-- name: ListEveryCell :many
SELECT c.id, c.items, c.edition, c.target_id, c.stage, c.node_id, c.not_before, c.failure_code,
    CAST(coalesce((SELECT a.node_name FROM conversions_attempt AS a WHERE a.conversion_id = c.id ORDER BY a.n DESC LIMIT 1), '') AS TEXT) AS node_name
FROM conversions_conversion AS c
WHERE c.finished_at IS NULL AND c.target_id IS NOT NULL
ORDER BY c.id;

-- name: ListHolding :many
SELECT id, root_id, target_id, attempt FROM conversions_conversion
WHERE root_id IS NOT NULL AND target_id IS NOT NULL AND attempt >= 1
    AND stage IN ('ready', 'running', 'paused', 'returning', 'checking', 'landing')
ORDER BY id;

-- name: ListReading :many
SELECT DISTINCT source_file_id FROM conversions_conversion
WHERE source_file_id IN (sqlc.slice(source_file_ids)) AND stage IN ('copying', 'ready', 'running', 'paused')
ORDER BY source_file_id;

-- name: GetNodeSettings :one
SELECT * FROM conversions_node WHERE node_id = ?;

-- name: UpsertNodeSettings :exec
INSERT INTO conversions_node (node_id, at_once, hours_on, hours_start, hours_end, processor)
VALUES (@node_id, @at_once, @hours_on, @hours_start, @hours_end, @processor)
ON CONFLICT (node_id) DO UPDATE SET
    at_once = excluded.at_once, hours_on = excluded.hours_on, hours_start = excluded.hours_start,
    hours_end = excluded.hours_end, processor = excluded.processor;

-- name: ListProofsOf :many
SELECT * FROM conversions_proof WHERE node_id = ? ORDER BY shape, class;

-- name: ListMadeBefore :many
SELECT work_id, source_file_id, output_file_id FROM conversions_conversion
WHERE target_id = @target_id AND version < @version AND stage = 'done' AND output_file_id IS NOT NULL
ORDER BY work_id, output_file_id;
