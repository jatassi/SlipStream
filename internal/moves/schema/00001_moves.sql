-- +goose Up
-- S48 (Moves). A batch is one Start of a previewed plan. Each unit moves one
-- Work's files to another root, or one torrent to another Download Client.
-- A unit's state is its phase and the columns that phase has; the CHECKs
-- refuse a row that no Stage parses from. Root, file, Work and torrent ids
-- carry no foreign key: their owners delete rows in their own Writes, and a
-- finished unit outlives a deleted root.
CREATE TABLE moves_batch (
    id BLOB PRIMARY KEY NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('work', 'bulk', 'root', 'drain', 'client', 'torrent')),
    created_at INTEGER NOT NULL,
    -- The Admin who started it, NULL for the API key.
    created_by BLOB,
    finished_at INTEGER,
    canceled_at INTEGER,
    homed_works INTEGER NOT NULL DEFAULT 0 CHECK (homed_works >= 0),
    include_torrents INTEGER NOT NULL CHECK (include_torrents IN (0, 1)),
    repoint_routing INTEGER NOT NULL CHECK (repoint_routing IN (0, 1)),
    delete_emptied_roots INTEGER NOT NULL CHECK (delete_emptied_roots IN (0, 1)),
    plan_token_hash BLOB NOT NULL UNIQUE,
    notes TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(notes))
) STRICT;
CREATE INDEX moves_batch_listed ON moves_batch (finished_at IS NOT NULL, created_at DESC, id);

CREATE TABLE moves_unit (
    id BLOB PRIMARY KEY NOT NULL,
    batch_id BLOB NOT NULL REFERENCES moves_batch (id) ON DELETE CASCADE,
    seq INTEGER NOT NULL UNIQUE,
    position INTEGER NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('work', 'torrent')),
    work_id BLOB,
    content_type TEXT,
    torrent_id BLOB,
    from_root_id BLOB,
    from_node_id BLOB NOT NULL,
    from_client_id BLOB,
    to_root_id BLOB,
    to_node_id BLOB NOT NULL,
    to_client_id BLOB,
    to_folder TEXT,
    lane TEXT NOT NULL CHECK (lane <> ''),
    phase TEXT NOT NULL CHECK (phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup', 'done', 'failed', 'canceled')),
    waiting TEXT CHECK (waiting IN ('pair_busy', 'low_priority', 'node_offline', 'no_route', 'space', 'renames',
        'downloading', 'importing', 'open_files', 'playing', 'media_servers', 'guard', 'linked_torrent')),
    waiting_at INTEGER,
    due_at INTEGER,
    handoff TEXT CHECK (handoff IN ('guard', 'stopping', 'starting')),
    handoff_at INTEGER,
    pass INTEGER NOT NULL DEFAULT 0 CHECK (pass >= 0),
    switch_now_at INTEGER,
    failure_code TEXT CHECK (failure_code IN ('source_gone', 'source_failed', 'damaged', 'source_written_off',
        'destination_gone', 'destination_check_failed', 'switch_refused', 'destination_unreachable')),
    failure_detail TEXT,
    failed_phase TEXT CHECK (failed_phase IN ('queued', 'transfer', 'place', 'switch')),
    keeps_copy INTEGER NOT NULL DEFAULT 0 CHECK (keeps_copy IN (0, 1)),
    undone_at INTEGER,
    reconcile INTEGER NOT NULL DEFAULT 0 CHECK (reconcile IN (0, 1)),
    transfer_id BLOB,
    dest_torrent_id BLOB,
    pairs_with_unit_id BLOB REFERENCES moves_unit (id) ON DELETE SET NULL,
    planned_bytes INTEGER NOT NULL CHECK (planned_bytes >= 0),
    planned_at INTEGER NOT NULL,
    switched_at INTEGER,
    cleaned_at INTEGER,
    notes TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(notes)),
    CHECK ((kind = 'work') = (work_id IS NOT NULL)),
    CHECK ((kind = 'work') = (content_type IS NOT NULL)),
    CHECK ((kind = 'torrent') = (torrent_id IS NOT NULL)),
    CHECK ((kind = 'work') = (from_root_id IS NOT NULL AND to_root_id IS NOT NULL)),
    CHECK (kind = 'work' OR (from_root_id IS NULL AND to_root_id IS NULL)),
    CHECK ((waiting IS NULL) = (waiting_at IS NULL)),
    CHECK (waiting IS NULL OR phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup')),
    CHECK ((handoff IS NULL) = (handoff_at IS NULL)),
    CHECK (handoff IS NULL OR (kind = 'torrent' AND phase = 'switch')),
    CHECK ((phase = 'failed') = (failure_code IS NOT NULL)),
    CHECK ((phase IN ('failed', 'canceled')) = (failed_phase IS NOT NULL)),
    CHECK (keeps_copy = 0 OR (phase = 'failed' AND kind = 'torrent')),
    CHECK (undone_at IS NULL OR phase IN ('failed', 'canceled')),
    CHECK (phase <> 'done' OR due_at IS NULL),
    CHECK ((phase IN ('cleanup', 'done')) = (switched_at IS NOT NULL)),
    CHECK ((phase = 'done') = (cleaned_at IS NOT NULL)),
    CHECK (phase <> 'transfer' OR transfer_id IS NOT NULL)
) STRICT;
-- One unit per Node pair (or same-Node volume pair) from its Transfer until its Place ends.
CREATE UNIQUE INDEX moves_unit_lane_held ON moves_unit (lane) WHERE phase IN ('transfer', 'place');
-- The head of each lane, without reading the queued units that step aside.
CREATE INDEX moves_unit_head ON moves_unit (lane, seq)
    WHERE phase = 'queued' AND (waiting IS NULL OR waiting IN ('pair_busy', 'node_offline'));
-- What the runner looks at: a unit is due once due_at passes.
CREATE INDEX moves_unit_due ON moves_unit (due_at) WHERE due_at IS NOT NULL;
-- One open unit per Work and per torrent ("already moving"); a kept copy counts.
CREATE UNIQUE INDEX moves_unit_open_work ON moves_unit (work_id)
    WHERE work_id IS NOT NULL AND phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup');
CREATE UNIQUE INDEX moves_unit_open_torrent ON moves_unit (torrent_id)
    WHERE torrent_id IS NOT NULL AND (phase IN ('queued', 'transfer', 'place', 'switch', 'cleanup') OR keeps_copy = 1);
CREATE INDEX moves_unit_waiting ON moves_unit (waiting, seq) WHERE waiting IS NOT NULL;
CREATE INDEX moves_unit_batch ON moves_unit (batch_id, position);
CREATE INDEX moves_unit_from_root ON moves_unit (from_root_id) WHERE from_root_id IS NOT NULL;
CREATE INDEX moves_unit_to_root ON moves_unit (to_root_id) WHERE to_root_id IS NOT NULL;
CREATE INDEX moves_unit_nodes ON moves_unit (from_node_id, to_node_id) WHERE phase IN ('queued', 'transfer', 'place', 'switch');
CREATE INDEX moves_unit_clients ON moves_unit (from_client_id, to_client_id) WHERE kind = 'torrent';

-- A unit's files. sits_rel is where the bytes sit at the source and the Rel
-- they take at the destination. modified_at and the node_* columns are the
-- source's Seen at the plan or the last final pass, the placed_* ones the
-- destination's once placed.
CREATE TABLE moves_unit_file (
    unit_id BLOB NOT NULL REFERENCES moves_unit (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('library', 'companion', 'sidecar', 'payload')),
    via TEXT NOT NULL DEFAULT 'library' CHECK (via IN ('library', 'arriving')),
    file_id BLOB,
    sits_rel TEXT NOT NULL CHECK (sits_rel <> ''),
    payload_index INTEGER,
    size INTEGER NOT NULL CHECK (size >= 0),
    modified_at INTEGER NOT NULL,
    node_file_id TEXT,
    node_volume_id TEXT,
    link_group INTEGER,
    ledger INTEGER NOT NULL DEFAULT 0 CHECK (ledger IN (0, 1)),
    placed INTEGER NOT NULL DEFAULT 0 CHECK (placed IN (0, 1)),
    placed_modified_at INTEGER,
    placed_node_file_id TEXT,
    placed_node_volume_id TEXT,
    copy_bytes INTEGER NOT NULL DEFAULT 0 CHECK (copy_bytes >= 0),
    dropped_at INTEGER,
    cleanup TEXT CHECK (cleanup IN ('pending', 'done', 'skipped')),
    PRIMARY KEY (unit_id, position),
    CHECK (kind <> 'library' OR file_id IS NOT NULL),
    CHECK (kind NOT IN ('companion', 'payload') OR file_id IS NULL),
    CHECK ((kind = 'payload' OR via = 'arriving') = (payload_index IS NOT NULL)),
    CHECK (via = 'library' OR kind <> 'payload'),
    CHECK (ledger = 0 OR kind = 'sidecar'),
    CHECK (placed = (placed_modified_at IS NOT NULL)),
    CHECK (dropped_at IS NULL OR (placed = 1 AND cleanup IS NULL))
) STRICT, WITHOUT ROWID;
CREATE INDEX moves_unit_file_file ON moves_unit_file (file_id) WHERE file_id IS NOT NULL;

-- One row of nullable overrides. A NULL is its default, kept in Go.
CREATE TABLE moves_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    needs_you_after_minutes INTEGER CHECK (needs_you_after_minutes BETWEEN 5 AND 10080),
    plex_bound_minutes INTEGER CHECK (plex_bound_minutes BETWEEN 1 AND 1440)
) STRICT;
INSERT INTO moves_settings (id) VALUES (1);
