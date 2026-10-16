-- +goose Up
CREATE TABLE tasks_task (
    id TEXT PRIMARY KEY NOT NULL,
    every_seconds INTEGER CHECK (every_seconds > 0),
    overdue TEXT CHECK (overdue IN ('catch-up', 'wait')),
    first_seen_at INTEGER NOT NULL,
    slot_started_at INTEGER,
    jitter_frac REAL CHECK (jitter_frac >= 0 AND jitter_frac < 1)
) STRICT;

CREATE TABLE tasks_run (
    id INTEGER PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks_task (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('schedule', 'catch-up', 'run-now', 'returned')),
    requested_by BLOB,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    result TEXT CHECK (result IN ('succeeded', 'partial', 'failed', 'skipped', 'cancelled')),
    summary TEXT,
    cause TEXT,
    units TEXT,
    CHECK ((ended_at IS NULL) = (result IS NULL)),
    CHECK ((kind = 'run-now') = (requested_by IS NOT NULL))
) STRICT;

CREATE UNIQUE INDEX tasks_run_open ON tasks_run (task_id) WHERE ended_at IS NULL;

CREATE INDEX tasks_run_task_id ON tasks_run (task_id, id);

CREATE TABLE tasks_unit (
    task_id TEXT NOT NULL REFERENCES tasks_task (id) ON DELETE CASCADE,
    unit_key TEXT NOT NULL,
    position INTEGER NOT NULL,
    label TEXT NOT NULL,
    tied_to BLOB,
    last_run_id INTEGER REFERENCES tasks_run (id) ON DELETE SET NULL,
    last_outcome TEXT CHECK (last_outcome IN ('succeeded', 'failed', 'skipped', 'cancelled')),
    last_cause TEXT,
    last_node BLOB,
    last_ended_at INTEGER,
    last_success_at INTEGER,
    first_seen_at INTEGER NOT NULL,
    due_on_return INTEGER NOT NULL DEFAULT 0 CHECK (due_on_return IN (0, 1)),
    CHECK (due_on_return = 0 OR tied_to IS NOT NULL),
    PRIMARY KEY (task_id, unit_key)
) STRICT, WITHOUT ROWID;

CREATE INDEX tasks_unit_last_run_id ON tasks_unit (last_run_id);

CREATE TABLE tasks_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    window_start INTEGER,
    window_end INTEGER,
    jitter_percent INTEGER,
    jitter_cap_seconds INTEGER,
    runs_kept INTEGER,
    header_delay_seconds INTEGER,
    partial_warn_intervals INTEGER,
    overdue TEXT CHECK (overdue IN ('catch-up', 'wait')),
    CHECK ((window_start IS NULL) = (window_end IS NULL))
) STRICT;

INSERT INTO tasks_settings (id) VALUES (1);
