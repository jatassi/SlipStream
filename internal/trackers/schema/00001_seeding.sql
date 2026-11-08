-- +goose Up
-- A Tracker has at most one rule per source. Releases never replace the
-- Admin's rule, so "Use what the Tracker states" deletes the admin row and
-- the rule from Releases is in force again. stated_at is the created_at of
-- the Grab a rule from Releases came from.
CREATE TABLE trackers_rule (
    tracker_id BLOB NOT NULL REFERENCES indexers_tracker (id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('releases', 'admin')),
    clears TEXT NOT NULL CHECK (clears IN ('time', 'ratio', 'either', 'both')),
    ratio_centi INTEGER CHECK (ratio_centi > 0),
    seed_seconds INTEGER CHECK (seed_seconds > 0),
    stated_at INTEGER,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (tracker_id, source),
    CHECK ((source = 'releases') = (stated_at IS NOT NULL)),
    CHECK ((clears IN ('ratio', 'either', 'both')) = (ratio_centi IS NOT NULL)),
    CHECK ((clears IN ('time', 'either', 'both')) = (seed_seconds IS NOT NULL))
) STRICT, WITHOUT ROWID;

-- A row only while the Tracker's goal differs from the default goal.
CREATE TABLE trackers_goal (
    tracker_id BLOB PRIMARY KEY NOT NULL REFERENCES indexers_tracker (id) ON DELETE CASCADE,
    combine TEXT NOT NULL CHECK (combine IN ('first', 'both', 'forever')),
    ratio_centi INTEGER CHECK (ratio_centi > 0),
    seed_seconds INTEGER CHECK (seed_seconds > 0),
    action TEXT CHECK (action IN ('pause', 'remove', 'remove-delete')),
    CHECK ((combine = 'forever') = (ratio_centi IS NULL)),
    CHECK ((combine = 'forever') = (seed_seconds IS NULL)),
    CHECK ((combine = 'forever') = (action IS NULL))
) STRICT, WITHOUT ROWID;

-- A private info hash's hit and run window. tracker_id is NULL while its
-- Tracker isn't known. The rule columns are the snapshot taken when it
-- opened, NULL while no rule was known. The owed columns are what the
-- latest miss still owed, and a regrab keeps them.
CREATE TABLE trackers_window (
    info_hash BLOB PRIMARY KEY NOT NULL CHECK (length(info_hash) = 20),
    tracker_id BLOB REFERENCES indexers_tracker (id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK (state IN ('owing', 'met', 'missed', 'released')),
    rule_clears TEXT CHECK (rule_clears IN ('time', 'ratio', 'either', 'both')),
    rule_ratio_centi INTEGER CHECK (rule_ratio_centi > 0),
    rule_seed_seconds INTEGER CHECK (rule_seed_seconds > 0),
    opened_at INTEGER NOT NULL,
    met_at INTEGER,
    closed_at INTEGER,
    regrabbed_at INTEGER,
    owed_clears TEXT CHECK (owed_clears IN ('time', 'ratio', 'either', 'both')),
    owed_seconds INTEGER CHECK (owed_seconds >= 0),
    owed_bytes INTEGER CHECK (owed_bytes >= 0),
    quiet_since INTEGER,
    CHECK ((state = 'met') = (met_at IS NOT NULL)),
    CHECK (state NOT IN ('missed', 'released') OR closed_at IS NOT NULL),
    CHECK (state = 'owing' OR quiet_since IS NULL),
    CHECK (rule_clears IS NOT NULL OR (rule_ratio_centi IS NULL AND rule_seed_seconds IS NULL)),
    CHECK (rule_clears IS NULL OR (rule_clears IN ('ratio', 'either', 'both')) = (rule_ratio_centi IS NOT NULL)),
    CHECK (rule_clears IS NULL OR (rule_clears IN ('time', 'either', 'both')) = (rule_seed_seconds IS NOT NULL)),
    CHECK ((owed_clears IS NULL) = (owed_seconds IS NULL)),
    CHECK ((owed_clears IS NULL) = (owed_bytes IS NULL))
) STRICT;

CREATE INDEX trackers_window_tracker ON trackers_window (tracker_id, state, opened_at);

-- A goal's action on one Torrent, written in the transaction that acts, so
-- a Torrent is acted on once. kept is why a remove kept the download copy.
-- A drain rewrites a pause to a kept remove.
CREATE TABLE trackers_goal_run (
    torrent_id BLOB PRIMARY KEY NOT NULL REFERENCES downloadclients_torrent (id),
    acted_at INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('pause', 'remove', 'remove-delete')),
    kept TEXT CHECK (kept IN ('not-imported', 'draining')),
    CHECK (kept IS NULL OR action = 'remove')
) STRICT, WITHOUT ROWID;

-- Each column is NULL for its setting's default. goal_combine NULL is the
-- default goal trackers.InitialGoal.
CREATE TABLE trackers_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    margin_seconds INTEGER,
    warn_after_seconds INTEGER,
    goal_combine TEXT CHECK (goal_combine IN ('first', 'both', 'forever')),
    goal_ratio_centi INTEGER CHECK (goal_ratio_centi > 0),
    goal_seed_seconds INTEGER CHECK (goal_seed_seconds > 0),
    goal_action TEXT CHECK (goal_action IN ('pause', 'remove', 'remove-delete')),
    CHECK (goal_combine IS NOT NULL OR (goal_ratio_centi IS NULL AND goal_seed_seconds IS NULL AND goal_action IS NULL)),
    CHECK (goal_combine IS NULL OR (goal_combine = 'forever') = (goal_ratio_centi IS NULL)),
    CHECK (goal_combine IS NULL OR (goal_combine = 'forever') = (goal_seed_seconds IS NULL)),
    CHECK (goal_combine IS NULL OR (goal_combine = 'forever') = (goal_action IS NULL))
) STRICT;

INSERT INTO trackers_settings (id) VALUES (1);
