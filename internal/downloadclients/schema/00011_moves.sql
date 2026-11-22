-- +goose Up
-- S48 (Moves) moves a seeding torrent to another Download Client: the
-- destination is a new Torrent of the same hash, arriving until the
-- Move's switch, which removes the source with reason moved. Neither an
-- arriving Torrent nor a moved one holds its hash on its Node, so a Move
-- between two clients of one Node holds the hash twice until the switch.
-- downloadclients_torrent is rebuilt as 00010 did, keeping its ids and
-- every column.
CREATE TABLE downloadclients_torrent_next (
    id BLOB PRIMARY KEY NOT NULL,
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    client_id BLOB REFERENCES downloadclients_client (id),
    -- attached is whether the torrent was in its client before SlipStream
    -- attached to it, as the Node first reported. NULL until then, and
    -- always for the built-in client.
    attached INTEGER CHECK (attached IN (0, 1)),
    info_hash BLOB NOT NULL CHECK (length(info_hash) = 20),
    metainfo BLOB,
    name TEXT NOT NULL,
    size INTEGER NOT NULL CHECK (size >= 0),
    private INTEGER NOT NULL CHECK (private IN (0, 1)),
    trackers TEXT NOT NULL CHECK (json_valid(trackers)),
    content_type TEXT,
    adopted INTEGER NOT NULL DEFAULT 0 CHECK (adopted IN (0, 1)),
    withheld_at INTEGER,
    tracker_id BLOB REFERENCES indexers_tracker (id),
    save_folder TEXT CHECK (save_folder <> ''),
    wanted TEXT NOT NULL CHECK (wanted IN ('started', 'paused')),
    added_order INTEGER NOT NULL UNIQUE,
    added_at INTEGER NOT NULL,
    removed_at INTEGER,
    removed_reason TEXT CHECK (removed_reason IN ('admin', 'mark_failed', 'node_written_off', 'client', 'client_deleted', 'goal_met', 'release_fault', 'canceled', 'moved')),
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
    client_error_ms INTEGER NOT NULL DEFAULT 0 CHECK (client_error_ms >= 0),
    client_error_at INTEGER,
    -- arriving_unit is the Move's unit while this Torrent is its
    -- destination, kept until it parts. arrival is what the destination
    -- Node reported of it: a complete add's check or a judged start, with
    -- the Node's reason for a refused or check_failed. arrived_at is the
    -- Controller's time it received the arrival.
    arriving_unit BLOB,
    arrival TEXT CHECK (arrival IN ('complete', 'check_failed', 'started', 'refused')),
    arrival_reason TEXT,
    arrived_at INTEGER,
    -- start_asked_at is when the Controller asked this destination to
    -- start and judge the start; start_until_at is the end of the window
    -- the Node may start it in.
    start_asked_at INTEGER,
    start_until_at INTEGER,
    -- paused_at is the Controller's time it saw a Move's source paused.
    paused_at INTEGER,
    -- announced_at is the latest successful announce over the trackers,
    -- never moved back, and announce_due_at the earliest next one, both
    -- on the Controller's clock, while a Move watches the announces.
    announced_at INTEGER,
    announce_due_at INTEGER,
    watch_announce INTEGER NOT NULL DEFAULT 0 CHECK (watch_announce IN (0, 1)),
    CHECK ((removed_at IS NULL) = (removed_reason IS NULL)),
    CHECK (parted_at IS NULL OR removed_at IS NOT NULL),
    CHECK ((parted_at IS NULL) = (freed IS NULL) OR removed_reason IN ('node_written_off', 'client', 'client_deleted')),
    CHECK ((report IS NULL) = (reported_at IS NULL)),
    CHECK ((completion IS NULL) = (completed_at IS NULL)),
    CHECK ((fault IS NULL) = (fault_at IS NULL)),
    CHECK (client_id IS NOT NULL OR attached IS NULL AND raw_uploaded IS NULL),
    CHECK ((raw_uploaded IS NULL) = (raw_downloaded IS NULL)
       AND (raw_uploaded IS NULL) = (raw_active_seconds IS NULL)
       AND (raw_uploaded IS NULL) = (raw_seeding_seconds IS NULL)),
    CHECK (adopted = 1 OR content_type IS NOT NULL),
    CHECK (adopted = 1 OR metainfo IS NOT NULL),
    CHECK (metainfo IS NOT NULL OR client_id IS NOT NULL),
    CHECK (arriving_unit IS NOT NULL OR (arrival IS NULL AND start_asked_at IS NULL)),
    CHECK ((start_asked_at IS NULL) = (start_until_at IS NULL)),
    CHECK ((arrival IS NULL) = (arrived_at IS NULL)),
    CHECK (arrival NOT IN ('started', 'refused') OR start_asked_at IS NOT NULL),
    CHECK ((arrival IN ('refused', 'check_failed')) = (arrival_reason IS NOT NULL))
) STRICT;

INSERT INTO downloadclients_torrent_next (
    id, node_id, client_id, attached, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id,
    save_folder, wanted, added_order, added_at, removed_at, removed_reason, removed_by_name, delete_files, parted_at,
    freed, uploaded, downloaded, active_seconds, seeding_seconds, announced_seconds, totals_at, raw_uploaded,
    raw_downloaded, raw_active_seconds, raw_seeding_seconds, report, reported_at, completion, completed_at, fault,
    fault_at, retry, client_error_ms, client_error_at, adopted, withheld_at)
SELECT
    id, node_id, client_id, attached, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id,
    save_folder, wanted, added_order, added_at, removed_at, removed_reason, removed_by_name, delete_files, parted_at,
    freed, uploaded, downloaded, active_seconds, seeding_seconds, announced_seconds, totals_at, raw_uploaded,
    raw_downloaded, raw_active_seconds, raw_seeding_seconds, report, reported_at, completion, completed_at, fault,
    fault_at, retry, client_error_ms, client_error_at, adopted, withheld_at
FROM downloadclients_torrent;

DROP TABLE downloadclients_torrent;
ALTER TABLE downloadclients_torrent_next RENAME TO downloadclients_torrent;

-- One live Torrent per Node and info hash, in any of its clients, until
-- its removal is confirmed: a new Grab for a hash being removed on that
-- Node waits for the confirmation. A Move's arriving Torrent and its
-- moved source don't hold it.
CREATE UNIQUE INDEX downloadclients_torrent_held ON downloadclients_torrent (node_id, info_hash)
    WHERE parted_at IS NULL AND arriving_unit IS NULL AND removed_reason IS NOT 'moved';
-- One Move's arriving Torrent per info hash.
CREATE UNIQUE INDEX downloadclients_torrent_arriving ON downloadclients_torrent (info_hash)
    WHERE arriving_unit IS NOT NULL AND removed_at IS NULL;
CREATE INDEX downloadclients_torrent_watched ON downloadclients_torrent (node_id) WHERE watch_announce = 1;
CREATE INDEX downloadclients_torrent_hash ON downloadclients_torrent (info_hash, removed_at);
CREATE INDEX downloadclients_torrent_node ON downloadclients_torrent (node_id, added_order);
CREATE INDEX downloadclients_torrent_client ON downloadclients_torrent (client_id) WHERE parted_at IS NULL;

-- The Torrents with a release fault, which grabs blocklists.
CREATE INDEX downloadclients_torrent_fault ON downloadclients_torrent (id) WHERE fault IS NOT NULL;
