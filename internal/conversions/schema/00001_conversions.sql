-- +goose Up
-- S44 (Conversion): making a Target's file from another Target's file on a
-- Node that holds the Converting Role.

-- An Admin's backfill or remake, or an adoption, that queued Conversions
-- for many Works at once. count is how many it queued.
CREATE TABLE conversions_backfill (
    id BLOB PRIMARY KEY NOT NULL,
    profile_id BLOB REFERENCES targets_profile (id) ON DELETE SET NULL,
    target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL,
    kind TEXT NOT NULL CHECK (kind IN ('backfill', 'remake', 'adopted')),
    created_at INTEGER NOT NULL,
    count INTEGER NOT NULL CHECK (count >= 0)
) STRICT;

-- One Conversion of a source file for a Target. A failed one is
-- unfinished, so it holds its pair until Retry, a new source or a cancel.
-- position orders a lane's unfinished rows. snapshot keeps the Work title,
-- Target name and source file name for a row whose links are gone.
CREATE TABLE conversions_conversion (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    items TEXT NOT NULL CHECK (json_valid(items) AND json_type(items) = 'array'),
    edition TEXT NOT NULL,
    source_file_id BLOB REFERENCES files_file (id) ON DELETE SET NULL,
    target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL,
    version INTEGER NOT NULL CHECK (version >= 1),
    lane TEXT NOT NULL CHECK (lane IN ('moved_up', 'new', 'upgrade', 'backfill')),
    position TEXT NOT NULL CHECK (position <> ''),
    backfill_id BLOB REFERENCES conversions_backfill (id) ON DELETE SET NULL,
    stage TEXT NOT NULL CHECK (stage IN ('queued', 'copying', 'ready', 'running', 'paused', 'checking', 'returning',
        'landing', 'done', 'failed', 'canceled')),
    node_id BLOB REFERENCES nodes_node (id) ON DELETE SET NULL,
    root_id BLOB REFERENCES roots_root (id) ON DELETE SET NULL,
    pin_node_id BLOB REFERENCES nodes_node (id) ON DELETE SET NULL,
    exclude TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(exclude) AND json_type(exclude) = 'array'),
    attempt INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    -- The first attempt the retry budget counts, past an Admin's Retry.
    budget_from INTEGER NOT NULL DEFAULT 1 CHECK (budget_from >= 1),
    not_before INTEGER NOT NULL,
    -- The Media Servers whose video transcodes paused the run.
    paused_by TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(paused_by) AND json_type(paused_by) = 'array'),
    failure_code TEXT CHECK (failure_code <> ''),
    failure_class TEXT CHECK (failure_class IN ('run', 'source')),
    failure_detail TEXT,
    cancel_why TEXT CHECK (cancel_why IN ('admin', 'not_wanted', 'source_gone', 'import_ended', 'import_declined')),
    snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
    queued_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER,
    output_file_id BLOB REFERENCES files_file (id) ON DELETE SET NULL,
    CHECK ((finished_at IS NULL) = (stage NOT IN ('done', 'canceled'))),
    CHECK (stage NOT IN ('copying', 'ready', 'running', 'paused', 'returning') OR node_id IS NOT NULL),
    CHECK (stage <> 'queued' OR node_id IS NULL),
    CHECK ((stage = 'failed') = (failure_code IS NOT NULL)),
    CHECK ((failure_code IS NULL) = (failure_class IS NULL) AND (failure_code IS NULL) = (failure_detail IS NULL)),
    CHECK ((stage = 'canceled') = (cancel_why IS NOT NULL)),
    CHECK (pin_node_id IS NULL OR stage = 'queued')
) STRICT;
CREATE UNIQUE INDEX conversions_conversion_pair ON conversions_conversion (source_file_id, target_id) WHERE finished_at IS NULL;
CREATE UNIQUE INDEX conversions_conversion_order ON conversions_conversion (lane, position) WHERE finished_at IS NULL;
CREATE INDEX conversions_conversion_due ON conversions_conversion (finished_at, not_before);
CREATE INDEX conversions_conversion_node ON conversions_conversion (node_id) WHERE node_id IS NOT NULL;
CREATE INDEX conversions_conversion_work ON conversions_conversion (work_id, finished_at);
-- So a deleted file or backfill SET NULLs its rows without a scan.
CREATE INDEX conversions_conversion_source ON conversions_conversion (source_file_id);
CREATE INDEX conversions_conversion_output ON conversions_conversion (output_file_id) WHERE output_file_id IS NOT NULL;
CREATE INDEX conversions_conversion_backfill ON conversions_conversion (backfill_id) WHERE backfill_id IS NOT NULL;

-- Each placement of a Conversion on a Node, and how it ended. node_name
-- names the Node after it is gone. path and api are the pipeline's.
CREATE TABLE conversions_attempt (
    conversion_id BLOB NOT NULL REFERENCES conversions_conversion (id) ON DELETE CASCADE,
    n INTEGER NOT NULL CHECK (n >= 1),
    node_id BLOB REFERENCES nodes_node (id) ON DELETE SET NULL,
    node_name TEXT NOT NULL,
    path TEXT NOT NULL CHECK (path IN ('hardware', 'hardware_opencl', 'processor')),
    api TEXT NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    outcome TEXT CHECK (outcome IN ('failed', 'stalled', 'disk_full', 'put_back')),
    code TEXT CHECK (code <> ''),
    class TEXT CHECK (class IN ('run', 'source')),
    -- The end of ffmpeg's error output, at most 2 KiB.
    tail TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (conversion_id, n),
    CHECK ((ended_at IS NULL) = (outcome IS NULL)),
    CHECK ((code IS NULL) = (class IS NULL)),
    CHECK (code IS NULL OR outcome IS NOT NULL)
) STRICT;

-- A test run's answer for one shape and source class on a Node, under the
-- Node's tool and driver signature. A proven pipeline that later fails in
-- a way that costs is lost until a new test run proves it again.
CREATE TABLE conversions_proof (
    node_id BLOB NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    signature TEXT NOT NULL,
    shape TEXT NOT NULL,
    class TEXT NOT NULL,
    path TEXT NOT NULL CHECK (path IN ('hardware', 'hardware_opencl', 'processor')),
    api TEXT NOT NULL,
    device TEXT NOT NULL DEFAULT '',
    -- Encoded seconds per second of wall time.
    speed REAL CHECK (speed > 0),
    pauses INTEGER NOT NULL DEFAULT 0 CHECK (pauses IN (0, 1)),
    proven_at INTEGER,
    failed_at INTEGER,
    lost_at INTEGER,
    error TEXT,
    PRIMARY KEY (node_id, signature, shape, class),
    CHECK (proven_at IS NOT NULL OR failed_at IS NOT NULL),
    CHECK (lost_at IS NULL OR proven_at IS NOT NULL),
    CHECK (proven_at IS NULL OR speed IS NOT NULL)
) STRICT;

-- A Node's Converting settings. A Node with no row has the defaults:
-- at_once NULL is DefaultAtOnce. The hours are minutes after midnight in
-- the Maintenance window's zone.
CREATE TABLE conversions_node (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    at_once INTEGER CHECK (at_once BETWEEN 1 AND 16),
    hours_on INTEGER NOT NULL DEFAULT 0 CHECK (hours_on IN (0, 1)),
    hours_start INTEGER NOT NULL DEFAULT 60 CHECK (hours_start BETWEEN 0 AND 1439),
    hours_end INTEGER NOT NULL DEFAULT 420 CHECK (hours_end BETWEEN 0 AND 1439),
    processor INTEGER NOT NULL DEFAULT 1 CHECK (processor IN (0, 1))
) STRICT;

-- An open No Node can convert cause, keyed by what it names. count is the
-- Conversions it holds back; notified_at is when it was recorded.
CREATE TABLE conversions_cause (
    key TEXT PRIMARY KEY NOT NULL CHECK (key <> ''),
    why TEXT NOT NULL CHECK (why <> ''),
    class TEXT,
    shape TEXT,
    target_id BLOB,
    node_id BLOB,
    count INTEGER NOT NULL CHECK (count >= 1),
    opened_at INTEGER NOT NULL,
    notified_at INTEGER
) STRICT;

-- A pair an Admin canceled or declined, which the drain never queues again
-- until its source file changes or the Admin asks for that Work by name.
CREATE TABLE conversions_declined (
    source_file_id BLOB NOT NULL REFERENCES files_file (id) ON DELETE CASCADE,
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    PRIMARY KEY (source_file_id, target_id)
) STRICT, WITHOUT ROWID;
CREATE INDEX conversions_declined_target ON conversions_declined (target_id);

-- A Work whose Conversions the drain converges next, oldest first. since
-- is when the grace of a download-then-derive row counts from.
CREATE TABLE conversions_nudge (
    id INTEGER PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    why TEXT NOT NULL CHECK (why IN ('new', 'adopted', 'profile', 'remake', 'changed')),
    since INTEGER NOT NULL,
    backfill_id BLOB,
    created_at INTEGER NOT NULL
) STRICT;
CREATE INDEX conversions_nudge_work ON conversions_nudge (work_id);
