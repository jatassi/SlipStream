-- name: WriteConversion :exec
UPDATE conversions_conversion SET
    stage = @stage, node_id = @node_id, root_id = @root_id, pin_node_id = @pin_node_id, exclude = @exclude,
    attempt = @attempt, budget_from = @budget_from, lane = @lane, position = @position, version = @version,
    paused_by = @paused_by, failure_code = @failure_code, failure_class = @failure_class,
    failure_detail = @failure_detail, cancel_why = @cancel_why, started_at = @started_at,
    finished_at = @finished_at, output_file_id = @output_file_id
WHERE id = @id;

-- name: HeadOfLane :one
SELECT CAST(coalesce(min(position), '') AS TEXT) FROM conversions_conversion WHERE lane = ? AND finished_at IS NULL;

-- name: OpenAttempt :exec
INSERT INTO conversions_attempt (conversion_id, n, node_id, node_name, path, api, started_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: CloseAttempt :exec
UPDATE conversions_attempt SET ended_at = @ended_at, outcome = @outcome, code = @code, class = @class, tail = @tail
WHERE conversion_id = @conversion_id AND n = @n AND ended_at IS NULL;

-- name: GetAttempt :one
SELECT * FROM conversions_attempt WHERE conversion_id = ? AND n = ?;

-- name: ListAttempts :many
SELECT * FROM conversions_attempt WHERE conversion_id = ? ORDER BY n;

-- name: Decline :exec
INSERT INTO conversions_declined (source_file_id, target_id) VALUES (?, ?) ON CONFLICT DO NOTHING;

-- name: ListPlacedOn :many
SELECT id, stage, attempt FROM conversions_conversion
WHERE node_id = ? AND stage IN ('copying', 'ready', 'running', 'paused', 'returning')
ORDER BY id;

-- name: ListPinnedTo :many
SELECT id FROM conversions_conversion WHERE pin_node_id = ? AND stage = 'queued' ORDER BY id;

-- name: CountReaders :one
SELECT count(*) FROM conversions_conversion
WHERE node_id = @node_id AND source_file_id = @source_file_id AND id <> @id
    AND stage IN ('copying', 'ready', 'running', 'paused');

-- name: CountHeld :one
SELECT CAST(count(*) AS INTEGER) AS units, CAST(coalesce(sum(stage IN ('running', 'paused')), 0) AS INTEGER) AS started
FROM conversions_conversion
WHERE node_id = ? AND stage IN ('copying', 'ready', 'running', 'paused', 'returning');

-- name: ListWritingOn :many
SELECT id, source_file_id, target_id FROM conversions_conversion
WHERE node_id = ? AND root_id IS NOT NULL AND stage IN ('copying', 'ready', 'running', 'paused')
ORDER BY id;

-- name: UpsertProof :exec
INSERT INTO conversions_proof (node_id, signature, shape, class, path, api, device, speed, pauses, proven_at, failed_at, error)
VALUES (@node_id, @signature, @shape, @class, @path, @api, @device, @speed, @pauses, @proven_at, @failed_at, @error)
ON CONFLICT (node_id, signature, shape, class) DO UPDATE SET
    path = excluded.path, api = excluded.api, device = excluded.device, speed = excluded.speed,
    pauses = excluded.pauses, proven_at = excluded.proven_at, failed_at = excluded.failed_at,
    lost_at = NULL, error = excluded.error;

-- name: GetProof :one
SELECT * FROM conversions_proof WHERE node_id = ? AND shape = ? AND class = ?;

-- name: LoseProof :exec
UPDATE conversions_proof SET lost_at = @lost_at
WHERE node_id = @node_id AND shape = @shape AND class = @class AND proven_at IS NOT NULL AND lost_at IS NULL;

-- name: SetProofSpeed :exec
UPDATE conversions_proof SET speed = @speed
WHERE node_id = @node_id AND shape = @shape AND class = @class AND proven_at IS NOT NULL;

-- name: DeleteProofsBesides :exec
DELETE FROM conversions_proof WHERE node_id = @node_id AND signature <> @signature;

-- name: DeleteFinishedBefore :execrows
DELETE FROM conversions_conversion WHERE id IN (
    SELECT f.id FROM conversions_conversion AS f WHERE f.finished_at < sqlc.arg(before) ORDER BY f.finished_at LIMIT sqlc.arg(n)
);

-- name: DeleteEmptyBackfills :execrows
DELETE FROM conversions_backfill WHERE id IN (
    SELECT conversions_backfill.id FROM conversions_backfill
    WHERE conversions_backfill.created_at < @before
        AND NOT EXISTS (SELECT 1 FROM conversions_conversion WHERE conversions_conversion.backfill_id = conversions_backfill.id)
    LIMIT @n
);
