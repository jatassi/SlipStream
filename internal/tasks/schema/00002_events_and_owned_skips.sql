-- +goose Up
CREATE TABLE tasks_run_next (
    id INTEGER PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks_task (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('schedule', 'catch-up', 'run-now', 'returned', 'event')),
    requested_by BLOB,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    result TEXT CHECK (result IN ('succeeded', 'partial', 'failed', 'skipped', 'canceled')),
    summary TEXT,
    cause TEXT,
    units TEXT,
    CHECK ((ended_at IS NULL) = (result IS NULL)),
    CHECK (kind = 'run-now' OR requested_by IS NULL)
) STRICT;

INSERT INTO tasks_run_next (id, task_id, kind, requested_by, started_at, ended_at, result, summary, cause, units)
SELECT id, task_id, kind, requested_by, started_at, ended_at, result, summary, cause, units FROM tasks_run;

DROP TABLE tasks_run;
ALTER TABLE tasks_run_next RENAME TO tasks_run;

-- An open run now lists the units it queued, which a stop mid-run closes as
-- canceled. A run left open before this migration queued every unit.
UPDATE tasks_run SET units = (
    SELECT json_group_array(json(queued.line)) FROM (
        SELECT json_object('key', unit_key, 'label', label) AS line
        FROM tasks_unit WHERE tasks_unit.task_id = tasks_run.task_id
        ORDER BY position
    ) AS queued
) WHERE ended_at IS NULL;

CREATE UNIQUE INDEX tasks_run_open ON tasks_run (task_id) WHERE ended_at IS NULL;

CREATE INDEX tasks_run_task_id ON tasks_run (task_id, id);

-- last_owned: the unit's last outcome was a skip its feature's own Health
-- category raises, so the Scheduled tasks item leaves it out.
ALTER TABLE tasks_unit ADD COLUMN last_owned INTEGER NOT NULL DEFAULT 0 CHECK (last_owned IN (0, 1));
