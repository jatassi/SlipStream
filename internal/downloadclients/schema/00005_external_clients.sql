-- +goose Up
-- An external Download Client is a record on one downloading Node, and a
-- Torrent names its client, NULL for the Node's built-in one. SQLite
-- can't change the removed_reason CHECK in place, so
-- downloadclients_torrent is rebuilt with the same ids, which grabs_grab
-- references. Every column grabs 00005 inserts into stays, and every new
-- one has a default, so a fresh library still runs that insert.
CREATE TABLE downloadclients_client (
    id BLOB PRIMARY KEY NOT NULL,
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    kind TEXT NOT NULL CHECK (kind IN ('qbittorrent', 'transmission', 'deluge', 'rtorrent')),
    name TEXT NOT NULL CHECK (name <> ''),
    address TEXT NOT NULL,
    address_key TEXT NOT NULL,
    username TEXT CHECK (username <> ''),
    secrets_sealed BLOB,
    label TEXT NOT NULL CHECK (label <> ''),
    folder TEXT CHECK (folder <> ''),
    client_folders TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(client_folders)),
    daemon_host TEXT CHECK (daemon_host <> ''),
    daemon_port INTEGER CHECK (daemon_port BETWEEN 1 AND 65535),
    check_every_seconds INTEGER CHECK (check_every_seconds >= 2),
    import_delay_seconds INTEGER CHECK (import_delay_seconds BETWEEN 0 AND 86400),
    is_on INTEGER NOT NULL CHECK (is_on IN (0, 1)),
    settings_version INTEGER NOT NULL CHECK (settings_version >= 1),
    -- The last poll its Node reported, under settings
    -- poll_settings_version. NULL until the first poll, as for a client
    -- S54 (Import engine and import from v1) carries over with no Test.
    poll_state TEXT CHECK (poll_state IN ('responding', 'not_responding', 'sign_in_refused', 'needs_attention')),
    poll_since INTEGER,
    poll_problem TEXT CHECK (poll_problem IS NULL OR json_valid(poll_problem)),
    poll_settings_version INTEGER,
    poll_check BLOB,
    client_version TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER,
    CHECK ((daemon_host IS NULL) = (daemon_port IS NULL)),
    CHECK ((poll_state IS NULL) = (poll_since IS NULL)),
    CHECK ((poll_state IS NULL) = (poll_settings_version IS NULL)),
    CHECK ((poll_problem IS NULL) = (poll_state IS NULL OR poll_state = 'responding')),
    CHECK (deleted_at IS NULL OR secrets_sealed IS NULL)
) STRICT;

CREATE UNIQUE INDEX downloadclients_client_name ON downloadclients_client (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX downloadclients_client_address ON downloadclients_client (address_key) WHERE deleted_at IS NULL;
CREATE INDEX downloadclients_client_node ON downloadclients_client (node_id) WHERE deleted_at IS NULL;

CREATE TABLE downloadclients_torrent_next (
    id BLOB PRIMARY KEY NOT NULL,
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    client_id BLOB REFERENCES downloadclients_client (id),
    -- attached is whether the torrent was in its client before SlipStream
    -- attached to it, as the Node first reported. NULL until then, and
    -- always for the built-in client.
    attached INTEGER CHECK (attached IN (0, 1)),
    info_hash BLOB NOT NULL CHECK (length(info_hash) = 20),
    metainfo BLOB NOT NULL,
    name TEXT NOT NULL,
    size INTEGER NOT NULL CHECK (size >= 0),
    private INTEGER NOT NULL CHECK (private IN (0, 1)),
    trackers TEXT NOT NULL CHECK (json_valid(trackers)),
    content_type TEXT NOT NULL,
    tracker_id BLOB REFERENCES indexers_tracker (id),
    save_folder TEXT CHECK (save_folder <> ''),
    wanted TEXT NOT NULL CHECK (wanted IN ('started', 'paused')),
    added_order INTEGER NOT NULL UNIQUE,
    added_at INTEGER NOT NULL,
    removed_at INTEGER,
    removed_reason TEXT CHECK (removed_reason IN ('admin', 'mark_failed', 'node_written_off', 'client', 'client_deleted')),
    delete_files INTEGER NOT NULL DEFAULT 0 CHECK (delete_files IN (0, 1)),
    parted_at INTEGER,
    freed INTEGER CHECK (freed >= 0),
    uploaded INTEGER NOT NULL DEFAULT 0 CHECK (uploaded >= 0),
    downloaded INTEGER NOT NULL DEFAULT 0 CHECK (downloaded >= 0),
    active_seconds INTEGER NOT NULL DEFAULT 0 CHECK (active_seconds >= 0),
    seeding_seconds INTEGER NOT NULL DEFAULT 0 CHECK (seeding_seconds >= 0),
    totals_at INTEGER,
    -- raw_* are an external client's last raw counters, the baseline the
    -- next report's increase counts from. NULL until the first.
    raw_uploaded INTEGER,
    raw_downloaded INTEGER,
    raw_active_seconds INTEGER,
    raw_seeding_seconds INTEGER,
    report TEXT CHECK (report IS NULL OR json_valid(report)),
    reported_at INTEGER,
    completion TEXT CHECK (completion IS NULL OR json_valid(completion)),
    completed_at INTEGER,
    fault TEXT CHECK (fault IN ('engine_error', 'unregistered', 'no_payload')),
    fault_at INTEGER,
    CHECK ((removed_at IS NULL) = (removed_reason IS NULL)),
    CHECK (parted_at IS NULL OR removed_at IS NOT NULL),
    CHECK ((parted_at IS NULL) = (freed IS NULL) OR removed_reason IN ('node_written_off', 'client', 'client_deleted')),
    CHECK ((report IS NULL) = (reported_at IS NULL)),
    CHECK ((completion IS NULL) = (completed_at IS NULL)),
    CHECK ((fault IS NULL) = (fault_at IS NULL)),
    CHECK (client_id IS NOT NULL OR attached IS NULL AND raw_uploaded IS NULL),
    CHECK ((raw_uploaded IS NULL) = (raw_downloaded IS NULL)
       AND (raw_uploaded IS NULL) = (raw_active_seconds IS NULL)
       AND (raw_uploaded IS NULL) = (raw_seeding_seconds IS NULL))
) STRICT;

INSERT INTO downloadclients_torrent_next (
    id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id, save_folder, wanted,
    added_order, added_at, removed_at, removed_reason, delete_files, parted_at, freed, uploaded, downloaded,
    active_seconds, seeding_seconds, totals_at, report, reported_at, completion, completed_at, fault, fault_at)
SELECT
    id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id, save_folder, wanted,
    added_order, added_at, removed_at, removed_reason, delete_files, parted_at, freed, uploaded, downloaded,
    active_seconds, seeding_seconds, totals_at, report, reported_at, completion, completed_at, fault, fault_at
FROM downloadclients_torrent;

DROP TABLE downloadclients_torrent;
ALTER TABLE downloadclients_torrent_next RENAME TO downloadclients_torrent;

-- One live Torrent per Node and info hash, in any of its clients, until
-- its removal is confirmed: a new Grab for a hash being removed on that
-- Node waits for the confirmation.
CREATE UNIQUE INDEX downloadclients_torrent_held ON downloadclients_torrent (node_id, info_hash) WHERE parted_at IS NULL;
CREATE INDEX downloadclients_torrent_hash ON downloadclients_torrent (info_hash, removed_at);
CREATE INDEX downloadclients_torrent_node ON downloadclients_torrent (node_id, added_order);
CREATE INDEX downloadclients_torrent_client ON downloadclients_torrent (client_id) WHERE parted_at IS NULL;

-- grabs_to is where the Node's new Grabs go, NULL for its built-in client.
ALTER TABLE downloadclients_node ADD COLUMN grabs_to BLOB REFERENCES downloadclients_client (id);
