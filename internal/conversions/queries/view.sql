-- name: ListFailed :many
SELECT c.* FROM conversions_conversion AS c
WHERE c.stage = 'failed' AND c.finished_at IS NULL
ORDER BY coalesce((SELECT max(a.ended_at) FROM conversions_attempt AS a WHERE a.conversion_id = c.id), c.queued_at) DESC, c.id;

-- name: ListOpenOfWork :many
SELECT * FROM conversions_conversion WHERE work_id = ? AND finished_at IS NULL ORDER BY id;

-- name: CountForShell :one
SELECT CAST(coalesce(sum(stage IN ('running', 'paused')), 0) AS INTEGER) AS started,
    CAST(coalesce(sum(stage = 'failed'), 0) AS INTEGER) AS failed,
    CAST((SELECT count(*) FROM conversions_cause) AS INTEGER) AS causes
FROM conversions_conversion
WHERE finished_at IS NULL;
