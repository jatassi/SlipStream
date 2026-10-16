-- name: ListTasks :many
SELECT * FROM tasks_task ORDER BY id;

-- name: InsertTask :exec
INSERT INTO tasks_task (id, first_seen_at) VALUES (sqlc.arg(id), sqlc.arg(first_seen_at))
ON CONFLICT (id) DO NOTHING;

-- name: DeleteTask :exec
DELETE FROM tasks_task WHERE id = sqlc.arg(id);

-- name: SetTaskEvery :exec
UPDATE tasks_task SET every_seconds = sqlc.narg(every_seconds) WHERE id = sqlc.arg(id);

-- name: SetTaskOverdue :exec
UPDATE tasks_task SET overdue = sqlc.narg(overdue) WHERE id = sqlc.arg(id);

-- name: AdvanceSlot :exec
UPDATE tasks_task SET slot_started_at = sqlc.arg(slot_started_at), jitter_frac = sqlc.arg(jitter_frac)
WHERE id = sqlc.arg(id);

-- name: ListUnits :many
SELECT * FROM tasks_unit WHERE task_id = sqlc.arg(task_id) ORDER BY position;

-- name: ListDueUnits :many
SELECT * FROM tasks_unit WHERE due_on_return = 1 ORDER BY task_id, position;

-- name: UpsertUnit :exec
INSERT INTO tasks_unit (task_id, unit_key, position, label, tied_to, first_seen_at)
VALUES (sqlc.arg(task_id), sqlc.arg(unit_key), sqlc.arg(position), sqlc.arg(label), sqlc.arg(tied_to), sqlc.arg(first_seen_at))
ON CONFLICT (task_id, unit_key) DO UPDATE SET
    position = excluded.position,
    label = excluded.label,
    tied_to = excluded.tied_to,
    due_on_return = CASE WHEN excluded.tied_to IS NULL THEN 0 ELSE tasks_unit.due_on_return END;

-- name: DeleteUnit :exec
DELETE FROM tasks_unit WHERE task_id = sqlc.arg(task_id) AND unit_key = sqlc.arg(unit_key);

-- name: RecordUnit :exec
UPDATE tasks_unit SET
    last_run_id = sqlc.arg(last_run_id),
    last_outcome = sqlc.arg(last_outcome),
    last_cause = sqlc.narg(last_cause),
    last_node = sqlc.arg(last_node),
    last_ended_at = sqlc.arg(last_ended_at),
    last_success_at = coalesce(sqlc.arg(last_success_at), last_success_at),
    due_on_return = sqlc.arg(due_on_return)
WHERE task_id = sqlc.arg(task_id) AND unit_key = sqlc.arg(unit_key);

-- name: InsertRun :one
INSERT INTO tasks_run (task_id, kind, requested_by, started_at)
VALUES (sqlc.arg(task_id), sqlc.arg(kind), sqlc.arg(requested_by), sqlc.arg(started_at))
RETURNING id;

-- name: CloseRun :exec
UPDATE tasks_run SET
    ended_at = sqlc.arg(ended_at),
    result = sqlc.arg(result),
    summary = sqlc.arg(summary),
    cause = sqlc.narg(cause),
    units = sqlc.arg(units)
WHERE id = sqlc.arg(id) AND ended_at IS NULL;

-- name: ListOpenRuns :many
SELECT * FROM tasks_run WHERE ended_at IS NULL ORDER BY id;

-- name: ListLatestRuns :many
SELECT * FROM tasks_run r
WHERE r.id = (SELECT max(l.id) FROM tasks_run l WHERE l.task_id = r.task_id AND l.ended_at IS NOT NULL)
ORDER BY r.task_id;

-- name: PruneRuns :exec
DELETE FROM tasks_run AS old
WHERE old.ended_at IS NOT NULL AND (
    SELECT count(*) FROM tasks_run AS newer
    WHERE newer.task_id = old.task_id AND newer.ended_at IS NOT NULL AND newer.id > old.id
) >= CAST(sqlc.arg(kept) AS INTEGER);

-- name: GetSettings :one
SELECT * FROM tasks_settings WHERE id = 1;

-- name: SetSettings :exec
UPDATE tasks_settings SET
    window_start = sqlc.narg(window_start),
    window_end = sqlc.narg(window_end),
    jitter_percent = sqlc.narg(jitter_percent),
    jitter_cap_seconds = sqlc.narg(jitter_cap_seconds),
    runs_kept = sqlc.narg(runs_kept),
    header_delay_seconds = sqlc.narg(header_delay_seconds),
    partial_warn_intervals = sqlc.narg(partial_warn_intervals),
    overdue = sqlc.narg(overdue)
WHERE id = 1;
