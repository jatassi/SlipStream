-- +goose Up
-- First setup's one stored decision: finished_at is NULL while first setup
-- is open and is set once by Finish. library.db is born at the claim, so the
-- row is open from the claim on. A library.db that already has an enabled
-- Admin was claimed before first setup existed, so its row starts finished.
CREATE TABLE firstrun_state (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    finished_at INTEGER
) STRICT;
INSERT INTO firstrun_state (id, finished_at)
SELECT 1, CASE
    WHEN EXISTS (SELECT 1 FROM identity_account WHERE admin = 1 AND disabled_at IS NULL)
    THEN CAST(strftime('%s', 'now') AS INTEGER) * 1000000
END;

-- An Admin's Continue or Skip on a step, by step id.
CREATE TABLE firstrun_step (
    id TEXT PRIMARY KEY NOT NULL CHECK (id <> ''),
    outcome TEXT NOT NULL CHECK (outcome IN ('done', 'skipped')),
    at INTEGER NOT NULL
) STRICT, WITHOUT ROWID;
