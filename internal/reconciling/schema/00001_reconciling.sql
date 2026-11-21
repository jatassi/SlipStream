-- +goose Up
-- S49 (Backups and restore): Reconciling after a restore, and the keeping
-- of every Node's torrents until SlipStream adopts its Holdings.

-- Present only while a restore's Reconciling runs. restore_id is the
-- restore that began it, so the restore's Write runs once.
CREATE TABLE reconciling_state (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    restore_id TEXT NOT NULL CHECK (restore_id <> ''),
    started_at INTEGER NOT NULL,
    cause TEXT NOT NULL CHECK (cause IN ('restore', 'manual_rollback')),
    backup_name TEXT NOT NULL,
    backup_taken_at INTEGER NOT NULL,
    new_machine INTEGER NOT NULL CHECK (new_machine IN (0, 1)),
    -- NULL until the old Controller's folders are found on a new machine;
    -- set when Reconciling begins in place.
    rehome_saved_at INTEGER,
    -- When the History rows of the restore went out.
    announced_at INTEGER
) STRICT;

-- One row per Node that keeps every torrent until its Holdings are
-- adopted: every Node at a restore, and every pairing. It outlives
-- reconciling_state: a Node continued without keeps everything until it
-- reports, and granted_at names a Role adoption gave.
CREATE TABLE reconciling_node (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    cause TEXT NOT NULL CHECK (cause IN ('restore', 'pairing')),
    since INTEGER NOT NULL,
    -- The Holdings round the Node's intent asks for.
    round INTEGER NOT NULL CHECK (round >= 1),
    -- The round's page count once a page arrived, and the distinct pages
    -- of it stored.
    pages INTEGER CHECK (pages >= 1),
    pages_seen INTEGER NOT NULL DEFAULT 0,
    -- Every page of the round arrived.
    reported_at INTEGER,
    -- How many torrents the round's pages listed.
    held INTEGER NOT NULL DEFAULT 0 CHECK (held >= 0),
    adopted_at INTEGER,
    -- Adoption gave the Node the Downloading Role.
    granted_at INTEGER,
    -- A page SlipStream refused, and why: the Node keeps everything.
    refused_at INTEGER,
    refused TEXT,
    continued_without_at INTEGER,
    CHECK (pages_seen <= coalesce(pages, 0)),
    CHECK (reported_at IS NULL OR pages_seen = pages),
    CHECK (granted_at IS NULL OR adopted_at IS NOT NULL),
    CHECK ((refused_at IS NULL) = (refused IS NULL))
) STRICT;
CREATE INDEX reconciling_node_pending ON reconciling_node (node_id) WHERE adopted_at IS NULL;

-- A page of a round seen, so a repeated page counts once.
CREATE TABLE reconciling_page (
    node_id BLOB NOT NULL REFERENCES reconciling_node (node_id) ON DELETE CASCADE,
    round INTEGER NOT NULL,
    page INTEGER NOT NULL CHECK (page >= 0),
    PRIMARY KEY (node_id, round, page)
) STRICT;

-- Each Node's own Holdings: its facts, written only from its own pages,
-- with adoption's decision. Nothing another Node says changes them. name
-- and size are NULL for a torrent the Node never saved resume data for.
CREATE TABLE reconciling_held (
    node_id BLOB NOT NULL REFERENCES reconciling_node (node_id) ON DELETE CASCADE,
    info_hash BLOB NOT NULL CHECK (length(info_hash) = 20),
    -- NULL is the built-in client.
    client_id BLOB,
    name TEXT,
    size INTEGER CHECK (size >= 0),
    done_ppm INTEGER NOT NULL CHECK (done_ppm BETWEEN 0 AND 1000000),
    private INTEGER NOT NULL CHECK (private IN (0, 1)),
    seed_seconds INTEGER NOT NULL CHECK (seed_seconds >= 0),
    hosts TEXT NOT NULL CHECK (json_valid(hosts)),
    save_folder TEXT,
    decision TEXT CHECK (decision IN ('adopt', 'kept', 'move', 'wait', 'conflict', 'stranded')),
    -- The decision is final. A wait is decided again until it is.
    settled_at INTEGER,
    CHECK ((name IS NULL) = (size IS NULL)),
    CHECK (settled_at IS NULL OR decision IS NOT NULL),
    PRIMARY KEY (node_id, info_hash)
) STRICT;
CREATE INDEX reconciling_held_hash ON reconciling_held (info_hash);
CREATE INDEX reconciling_held_open ON reconciling_held (node_id, info_hash) WHERE settled_at IS NULL;
CREATE INDEX reconciling_held_keep ON reconciling_held (node_id, info_hash) WHERE decision IN ('wait', 'conflict', 'stranded');

-- One row per library root while Reconciling: finding the old
-- Controller's folders, and the scan.
CREATE TABLE reconciling_root (
    root_id BLOB PRIMARY KEY NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    -- NULL is Later: no scan reads the root.
    saved_at INTEGER,
    -- A scan finished at or after Reconciling started.
    scanned_at INTEGER,
    continued_without_at INTEGER
) STRICT;

-- The review list: recorded torrents no Node holds, and torrents two
-- Nodes hold. It outlives Reconciling.
CREATE TABLE reconciling_torrent (
    torrent_id BLOB PRIMARY KEY NOT NULL REFERENCES downloadclients_torrent (id) ON DELETE CASCADE,
    -- The Node it was recorded on.
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    reason TEXT NOT NULL CHECK (reason IN ('not_held', 'conflict')),
    -- NULL until its Node looked for its files.
    files_found INTEGER CHECK (files_found IN (0, 1)),
    looked_at INTEGER,
    since INTEGER NOT NULL,
    -- Re-add pressed while Reconciling, onto this Node.
    readd_node BLOB REFERENCES nodes_node (id),
    readd_at INTEGER,
    CHECK ((readd_node IS NULL) = (readd_at IS NULL))
) STRICT;
CREATE INDEX reconciling_torrent_list ON reconciling_torrent (reason, files_found, since, torrent_id);
