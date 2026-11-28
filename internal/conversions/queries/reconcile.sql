-- name: ListUnfinished :many
SELECT * FROM conversions_conversion
WHERE finished_at IS NULL AND stage <> 'failed'
ORDER BY CASE lane WHEN 'moved_up' THEN 0 WHEN 'new' THEN 1 WHEN 'upgrade' THEN 2 ELSE 3 END, position;

-- name: ListOnNode :many
SELECT * FROM conversions_conversion
WHERE node_id = ? AND stage IN ('copying', 'ready', 'running', 'paused', 'returning')
ORDER BY CASE lane WHEN 'moved_up' THEN 0 WHEN 'new' THEN 1 WHEN 'upgrade' THEN 2 ELSE 3 END, position;

-- name: ListChecking :many
SELECT * FROM conversions_conversion WHERE stage = 'checking' ORDER BY id;

-- name: ListCopying :many
SELECT id FROM conversions_conversion WHERE node_id = ? AND source_file_id = ? AND stage = 'copying' ORDER BY id;

-- name: ListPlacedNodes :many
SELECT DISTINCT node_id FROM conversions_conversion
WHERE node_id IS NOT NULL AND stage IN ('copying', 'ready', 'running', 'paused', 'returning')
ORDER BY node_id;

-- name: ListKnownNodes :many
SELECT node_id FROM conversions_node
UNION SELECT node_id FROM conversions_proof
UNION SELECT node_id FROM conversions_cause WHERE node_id IS NOT NULL
ORDER BY 1;

-- name: ListNodeSettings :many
SELECT * FROM conversions_node ORDER BY node_id;

-- name: ListProofs :many
SELECT * FROM conversions_proof ORDER BY node_id, shape, class;

-- name: DeleteProofsOf :exec
DELETE FROM conversions_proof WHERE node_id = ?;

-- name: DeleteNodeSettings :exec
DELETE FROM conversions_node WHERE node_id = ?;

-- name: DeleteCausesOf :exec
DELETE FROM conversions_cause WHERE node_id = ?;

-- name: ListCauses :many
SELECT * FROM conversions_cause ORDER BY key;

-- name: InsertCause :exec
INSERT INTO conversions_cause (key, why, class, shape, target_id, node_id, count, opened_at, notified_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetCauseCount :exec
UPDATE conversions_cause SET count = ? WHERE key = ?;

-- name: DeleteCause :exec
DELETE FROM conversions_cause WHERE key = ?;
