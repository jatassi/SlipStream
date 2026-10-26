-- +goose Up
-- A Transfer is a purpose's request to copy files from one Node to
-- another. Its live state is never stored: the Controller judges it from
-- these rows, both Nodes' facts and standing, and the destination volume.
-- The destination's Node isn't stored either, so a re-home moves it.
CREATE TABLE transfers_record (
    id BLOB PRIMARY KEY NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose <> ''),
    job TEXT NOT NULL,
    source_node BLOB NOT NULL REFERENCES nodes_node (id),
    destination_kind TEXT NOT NULL CHECK (destination_kind IN ('root')),
    destination_root BLOB REFERENCES roots_root (id),
    -- First come, first served: (created_at, id).
    created_at INTEGER NOT NULL,
    failure_code TEXT CHECK (failure_code IN ('source_gone', 'source_failed', 'damaged', 'source_written_off', 'rerouted')),
    -- The file the failure names, NULL when it names none.
    failure_position INTEGER CHECK (failure_position >= 0),
    -- The engine's text, or 'read_back' for damage found only on reading
    -- the staged file back.
    failure_detail TEXT NOT NULL DEFAULT '',
    UNIQUE (purpose, job),
    CHECK ((destination_kind = 'root') = (destination_root IS NOT NULL)),
    CHECK (failure_code IS NOT NULL OR (failure_position IS NULL AND failure_detail = ''))
) STRICT;
CREATE INDEX transfers_record_line ON transfers_record (created_at, id);

CREATE TABLE transfers_file (
    transfer_id BLOB NOT NULL REFERENCES transfers_record (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    -- The Item key the purpose groups files by, '' for none.
    item TEXT NOT NULL,
    name TEXT NOT NULL CHECK (name <> ''),
    source_kind TEXT NOT NULL CHECK (source_kind IN ('torrent')),
    info_hash BLOB CHECK (length(info_hash) = 20),
    file_index INTEGER CHECK (file_index >= 0),
    size INTEGER NOT NULL CHECK (size >= 0),
    -- The destination's place the file was verified at, NULL until it was.
    -- The file counts verified only while this equals the destination's
    -- place, so a re-home unverifies every unplaced file.
    verified_place INTEGER CHECK (verified_place > 0),
    placed INTEGER NOT NULL DEFAULT 0 CHECK (placed IN (0, 1)),
    PRIMARY KEY (transfer_id, position),
    CHECK ((source_kind = 'torrent') = (info_hash IS NOT NULL AND file_index IS NOT NULL))
) STRICT;

-- One recent speed and the last route that moved bytes per direction
-- between two Nodes.
CREATE TABLE transfers_pair (
    source_node BLOB NOT NULL REFERENCES nodes_node (id),
    destination_node BLOB NOT NULL REFERENCES nodes_node (id),
    -- Bytes a second, exponentially weighted.
    rate INTEGER NOT NULL CHECK (rate >= 0),
    rate_at INTEGER NOT NULL,
    route INTEGER CHECK (route IN (1, 2, 3)),
    route_at INTEGER,
    PRIMARY KEY (source_node, destination_node),
    CHECK ((route IS NULL) = (route_at IS NULL))
) STRICT;

-- No row is every default.
CREATE TABLE transfers_node (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id),
    -- Bytes a second.
    send_limit INTEGER CHECK (send_limit >= 0)
) STRICT;
