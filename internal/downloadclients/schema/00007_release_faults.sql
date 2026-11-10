-- +goose Up
-- S32 (Manual search and grab) removes a Torrent with reason release_fault
-- when its Release is blocklisted for a release fault, and with reason
-- canceled when an Admin cancels its downloading Grab, and names the Admin
-- who removed or marked it failed. downloadclients_torrent is rebuilt as
-- 00006 did, keeping its ids and every column.
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
    removed_reason TEXT CHECK (removed_reason IN ('admin', 'mark_failed', 'node_written_off', 'client', 'client_deleted', 'goal_met', 'release_fault', 'canceled')),
    -- removed_by_name is the name of the Admin who removed or marked it
    -- failed, NULL for the API key and every other reason.
    removed_by_name TEXT CHECK (removed_by_name <> ''),
    delete_files INTEGER NOT NULL DEFAULT 0 CHECK (delete_files IN (0, 1)),
    parted_at INTEGER,
    freed INTEGER CHECK (freed >= 0),
    uploaded INTEGER NOT NULL DEFAULT 0 CHECK (uploaded >= 0),
    downloaded INTEGER NOT NULL DEFAULT 0 CHECK (downloaded >= 0),
    active_seconds INTEGER NOT NULL DEFAULT 0 CHECK (active_seconds >= 0),
    seeding_seconds INTEGER NOT NULL DEFAULT 0 CHECK (seeding_seconds >= 0),
    -- announced_seconds is the seed time the Torrent's trackers credited
    -- it, as its Node counts it.
    announced_seconds INTEGER NOT NULL DEFAULT 0 CHECK (announced_seconds >= 0),
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
    -- retry is the last Retry of the failed Torrent and what its check
    -- found, NULL for none.
    retry TEXT CHECK (retry IS NULL OR json_valid(retry)),
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
    id, node_id, client_id, attached, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id,
    save_folder, wanted, added_order, added_at, removed_at, removed_reason, delete_files, parted_at, freed, uploaded,
    downloaded, active_seconds, seeding_seconds, totals_at, raw_uploaded, raw_downloaded, raw_active_seconds,
    raw_seeding_seconds, report, reported_at, completion, completed_at, fault, fault_at, announced_seconds, retry)
SELECT
    id, node_id, client_id, attached, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id,
    save_folder, wanted, added_order, added_at, removed_at, removed_reason, delete_files, parted_at, freed, uploaded,
    downloaded, active_seconds, seeding_seconds, totals_at, raw_uploaded, raw_downloaded, raw_active_seconds,
    raw_seeding_seconds, report, reported_at, completion, completed_at, fault, fault_at, announced_seconds, retry
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

-- The Torrents with a release fault, which grabs blocklists.
CREATE INDEX downloadclients_torrent_fault ON downloadclients_torrent (id) WHERE fault IS NOT NULL;
