-- +goose Up
-- S48 (Moves) carries a file from one root to another under the same Rel.
-- from_root_id is the root from_path is in when it isn't root_id: a Move's
-- carry. It has no foreign key, because a Move's carry outlives the source
-- root's delete only until it is sent, and moves deletes an emptied root
-- only after OpenMovesFrom is zero. source_gone_at is when the Move
-- removed the source files, which the planner waits for before a step that
-- needs the old item gone. mediaservers_change is rebuilt, keeping its
-- ids, because SQLite can't change the CHECK that refused a carry whose
-- path equals its from_path.
CREATE TABLE mediaservers_change_next (
    id BLOB PRIMARY KEY NOT NULL,
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('added', 'replaced', 'removed', 'moved',
        'sidecar_added', 'sidecar_replaced', 'sidecar_removed', 'library')),
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    from_root_id BLOB,
    from_path TEXT CHECK (from_path <> ''),
    file_id BLOB,
    -- A removed whose file an upgrade replaced: the replacer's id.
    replaced_by BLOB,
    -- An added whose file an upgrade's removed will follow.
    upgrade INTEGER NOT NULL DEFAULT 0 CHECK (upgrade IN (0, 1)),
    state TEXT NOT NULL CHECK (state IN ('prepared', 'due', 'sending', 'sent')),
    step TEXT NOT NULL DEFAULT 'none' CHECK (step <> ''),
    step_at INTEGER,
    snapshot TEXT CHECK (snapshot IS NULL OR json_valid(snapshot)),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    sent_at INTEGER,
    source_gone_at INTEGER,
    result TEXT CHECK (result IN ('accepted', 'covered', 'converted')),
    lost TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(lost) AND json_type(lost) = 'array'),
    -- The last failure the server itself responded with, as link.Problem.
    problem TEXT CHECK (problem IS NULL OR json_valid(problem)),
    CHECK ((kind = 'library') = (path = '')),
    CHECK ((kind = 'library') = (file_id IS NULL)),
    CHECK (from_path IS NULL OR kind IN ('moved', 'replaced')),
    CHECK (kind <> 'moved' OR from_path IS NOT NULL),
    CHECK (from_path IS NULL OR from_path <> path OR (from_root_id IS NOT NULL AND from_root_id <> root_id)),
    CHECK (from_root_id IS NULL OR from_path IS NOT NULL),
    CHECK (source_gone_at IS NULL OR from_root_id IS NOT NULL),
    CHECK (state <> 'prepared' OR from_path IS NOT NULL),
    CHECK (replaced_by IS NULL OR kind = 'removed' OR (state = 'prepared' AND kind = 'replaced')),
    CHECK (upgrade = 0 OR kind = 'added'),
    CHECK ((state = 'sent') = (sent_at IS NOT NULL)),
    CHECK ((state = 'sent') = (result IS NOT NULL)),
    CHECK ((step = 'none') = (step_at IS NULL)),
    CHECK (snapshot IS NULL OR step <> 'none')
) STRICT;

INSERT INTO mediaservers_change_next (
    id, server_id, kind, root_id, path, from_path, file_id, replaced_by, upgrade, state, step, step_at, snapshot,
    attempts, next_attempt_at, created_at, sent_at, result, lost, problem)
SELECT
    id, server_id, kind, root_id, path, from_path, file_id, replaced_by, upgrade, state, step, step_at, snapshot,
    attempts, next_attempt_at, created_at, sent_at, result, lost, problem
FROM mediaservers_change;

DROP TABLE mediaservers_change;
ALTER TABLE mediaservers_change_next RENAME TO mediaservers_change;

-- Compacted per path: one foldable row per server, root and path, and one
-- foldable carry per server and file. Their predicate is queue.Foldable
-- (TestFoldablePredicate pins the two together).
CREATE UNIQUE INDEX mediaservers_change_open_path ON mediaservers_change (server_id, root_id, path)
    WHERE from_path IS NULL AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted');
CREATE UNIQUE INDEX mediaservers_change_open_carry ON mediaservers_change (server_id, file_id)
    WHERE from_path IS NOT NULL AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted');
CREATE INDEX mediaservers_change_due ON mediaservers_change (server_id, created_at)
    WHERE state IN ('due', 'sending');
CREATE INDEX mediaservers_change_file ON mediaservers_change (server_id, file_id)
    WHERE file_id IS NOT NULL AND state IN ('prepared', 'due');
CREATE INDEX mediaservers_change_partner ON mediaservers_change (server_id, replaced_by)
    WHERE replaced_by IS NOT NULL AND state IN ('prepared', 'due');
CREATE INDEX mediaservers_change_server ON mediaservers_change (server_id, id);
CREATE INDEX mediaservers_change_sent ON mediaservers_change (sent_at) WHERE state = 'sent';
-- A Move's carries by file and source root: Safe, Snapshotted, SourcesGone
-- and OpenMovesFrom read through it.
CREATE INDEX mediaservers_change_move ON mediaservers_change (file_id, from_root_id) WHERE from_root_id IS NOT NULL;
