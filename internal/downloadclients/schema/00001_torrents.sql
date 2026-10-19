-- +goose Up
CREATE TABLE downloadclients_torrent (
    id BLOB PRIMARY KEY NOT NULL,
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    info_hash BLOB NOT NULL CHECK (length(info_hash) = 20),
    metainfo BLOB NOT NULL,
    name TEXT NOT NULL,
    size INTEGER NOT NULL CHECK (size >= 0),
    private INTEGER NOT NULL CHECK (private IN (0, 1)),
    trackers TEXT NOT NULL CHECK (json_valid(trackers)),
    content_type TEXT NOT NULL,
    save_folder TEXT CHECK (save_folder <> ''),
    wanted TEXT NOT NULL CHECK (wanted IN ('started', 'paused')),
    added_order INTEGER NOT NULL UNIQUE,
    added_at INTEGER NOT NULL,
    removed_at INTEGER,
    removed_reason TEXT CHECK (removed_reason IN ('admin', 'mark_failed', 'node_written_off')),
    delete_files INTEGER NOT NULL DEFAULT 0 CHECK (delete_files IN (0, 1)),
    parted_at INTEGER,
    freed INTEGER CHECK (freed >= 0),
    uploaded INTEGER NOT NULL DEFAULT 0 CHECK (uploaded >= 0),
    downloaded INTEGER NOT NULL DEFAULT 0 CHECK (downloaded >= 0),
    active_seconds INTEGER NOT NULL DEFAULT 0 CHECK (active_seconds >= 0),
    seeding_seconds INTEGER NOT NULL DEFAULT 0 CHECK (seeding_seconds >= 0),
    totals_at INTEGER,
    report TEXT CHECK (report IS NULL OR json_valid(report)),
    reported_at INTEGER,
    completion TEXT CHECK (completion IS NULL OR json_valid(completion)),
    completed_at INTEGER,
    fault TEXT CHECK (fault IN ('engine_error', 'unregistered', 'no_payload')),
    fault_at INTEGER,
    CHECK ((removed_at IS NULL) = (removed_reason IS NULL)),
    CHECK (parted_at IS NULL OR removed_at IS NOT NULL),
    CHECK ((parted_at IS NULL) = (freed IS NULL) OR removed_reason = 'node_written_off'),
    CHECK ((report IS NULL) = (reported_at IS NULL)),
    CHECK ((completion IS NULL) = (completed_at IS NULL)),
    CHECK ((fault IS NULL) = (fault_at IS NULL))
) STRICT;

-- One Torrent per Node and info hash until its removal is confirmed: a new
-- Grab for a hash being removed on that Node waits for the confirmation.
CREATE UNIQUE INDEX downloadclients_torrent_held ON downloadclients_torrent (node_id, info_hash) WHERE parted_at IS NULL;
CREATE INDEX downloadclients_torrent_hash ON downloadclients_torrent (info_hash, removed_at);
CREATE INDEX downloadclients_torrent_node ON downloadclients_torrent (node_id, added_order);

-- No row is every default.
CREATE TABLE downloadclients_node (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id),
    download_limit INTEGER CHECK (download_limit >= 0),
    upload_limit INTEGER CHECK (upload_limit >= 0),
    active_downloads INTEGER CHECK (active_downloads >= 0),
    active_checks INTEGER CHECK (active_checks >= 1),
    connections INTEGER CHECK (connections >= 1),
    utp INTEGER CHECK (utp IN (0, 1)),
    stalled_after_seconds INTEGER CHECK (stalled_after_seconds > 0)
) STRICT;

CREATE TABLE downloadclients_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    fault_stall_seconds INTEGER CHECK (fault_stall_seconds > 0),
    unregistered_gap_seconds INTEGER CHECK (unregistered_gap_seconds > 0)
) STRICT;

INSERT INTO downloadclients_settings (id) VALUES (1);
