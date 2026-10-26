-- name: GetFinishedAt :one
SELECT finished_at FROM firstrun_state WHERE id = 1;

-- name: SetFinishedAt :execrows
UPDATE firstrun_state SET finished_at = sqlc.arg(finished_at) WHERE id = 1 AND finished_at IS NULL;

-- name: ListSteps :many
SELECT id, outcome FROM firstrun_step ORDER BY id;

-- name: PutStep :execrows
INSERT INTO firstrun_step (id, outcome, at) VALUES (sqlc.arg(id), sqlc.arg(outcome), sqlc.arg(at))
ON CONFLICT (id) DO UPDATE SET outcome = excluded.outcome, at = excluded.at
WHERE firstrun_step.outcome <> excluded.outcome;
