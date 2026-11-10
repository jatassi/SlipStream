-- +goose Up
-- S32 (Manual search and grab). A Work keeps one Manual search, a group of
-- runs that list Releases and take nothing. The Blocklist holds Releases
-- known to be bad by info hash. Grabs record who chose them and what they
-- overrode.

CREATE TABLE grabs_manual (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL UNIQUE REFERENCES works_work (id) ON DELETE CASCADE,
    scope TEXT NOT NULL CHECK (json_valid(scope)),
    query TEXT CHECK (query IS NULL OR length(query) BETWEEN 1 AND 200),
    started_by BLOB REFERENCES identity_account (id) ON DELETE SET NULL,
    started_by_name TEXT,
    started_at INTEGER NOT NULL
) STRICT;

CREATE INDEX grabs_manual_started_by ON grabs_manual (started_by);

-- grabs_search: manual_id and the outcome 'listed'. A Manual run never
-- grabs.
CREATE TABLE grabs_search_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    manual_id BLOB REFERENCES grabs_manual (id) ON DELETE CASCADE,
    started_at INTEGER NOT NULL,
    finished_at INTEGER,
    outcome TEXT CHECK (outcome IN ('grabbed', 'nothing_acceptable', 'listed')),
    candidates INTEGER NOT NULL DEFAULT 0,
    grab_id BLOB REFERENCES grabs_grab (id),
    notes TEXT CHECK (notes IS NULL OR json_valid(notes)),
    CHECK ((finished_at IS NULL) = (outcome IS NULL)),
    CHECK ((outcome = 'grabbed') = (grab_id IS NOT NULL)),
    CHECK ((outcome = 'listed') = (manual_id IS NOT NULL AND outcome IS NOT NULL)),
    CHECK (manual_id IS NULL OR outcome IS NULL OR outcome = 'listed')
) STRICT;

INSERT INTO grabs_search_next (id, work_id, item_id, started_at, finished_at, outcome, candidates, grab_id, notes)
SELECT id, work_id, item_id, started_at, finished_at, outcome, candidates, grab_id, notes FROM grabs_search;

DROP TABLE grabs_search;
ALTER TABLE grabs_search_next RENAME TO grabs_search;

CREATE INDEX grabs_search_work_id ON grabs_search (work_id, started_at);
CREATE INDEX grabs_search_open ON grabs_search (finished_at, started_at);
CREATE INDEX grabs_search_item_id ON grabs_search (item_id, finished_at);
CREATE INDEX grabs_search_manual_id ON grabs_search (manual_id);

-- grabs_search_indexer: the skip 'rss_only'.
CREATE TABLE grabs_search_indexer_next (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    query_id BLOB,
    outcome TEXT CHECK (outcome IN ('found', 'no_releases', 'limited', 'failing', 'stopped', 'waiting', 'late',
        'limit_reached', 'no_categories', 'removed', 'rss_only')),
    releases INTEGER CHECK (releases >= 0),
    until_at INTEGER,
    since_at INTEGER,
    node_id BLOB,
    lapsed INTEGER CHECK (lapsed IN (0, 1)),
    cause TEXT,
    text TEXT,
    PRIMARY KEY (search_id, position)
) STRICT;

INSERT INTO grabs_search_indexer_next (search_id, position, indexer_id, indexer_name, query_id, outcome, releases, until_at,
    since_at, node_id, lapsed, cause, text)
SELECT search_id, position, indexer_id, indexer_name, query_id, outcome, releases, until_at, since_at, node_id, lapsed, cause,
    text
FROM grabs_search_indexer;

DROP TABLE grabs_search_indexer;
ALTER TABLE grabs_search_indexer_next RENAME TO grabs_search_indexer;

CREATE INDEX grabs_search_indexer_indexer_id ON grabs_search_indexer (indexer_id);

-- grabs_release: a Manual run's rows are 'listed', with the title reading
-- in read and nothing judged.
CREATE TABLE grabs_release_next (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    title TEXT NOT NULL,
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    size INTEGER NOT NULL,
    seeders INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('grabbed', 'not_taken', 'outranked', 'kept', 'refused', 'listed')),
    rank INTEGER CHECK (rank >= 1),
    reasons TEXT NOT NULL CHECK (json_valid(reasons)),
    lost_step TEXT CHECK (lost_step IN ('want', 'tier', 'revision', 'score', 'priority', 'freeleech', 'seeders', 'size', 'title')),
    lost_target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL,
    lost_own TEXT,
    lost_ref TEXT,
    scores TEXT NOT NULL CHECK (json_valid(scores)),
    formats TEXT NOT NULL CHECK (json_valid(formats)),
    covers TEXT NOT NULL CHECK (json_valid(covers)),
    copy_of INTEGER CHECK (copy_of >= 0),
    release TEXT CHECK (release IS NULL OR json_valid(release)),
    link_sealed BLOB,
    read TEXT CHECK (read IS NULL OR json_valid(read)),
    PRIMARY KEY (search_id, position),
    CHECK (outcome IN ('refused', 'listed') OR rank IS NOT NULL),
    CHECK (lost_step IS NULL OR outcome IN ('outranked', 'kept')),
    CHECK ((lost_step IS NULL) = (lost_own IS NULL) AND (lost_step IS NULL) = (lost_ref IS NULL)),
    CHECK (link_sealed IS NULL OR release IS NOT NULL),
    CHECK ((outcome = 'listed') = (read IS NOT NULL)),
    CHECK (outcome <> 'listed' OR (rank IS NULL AND copy_of IS NULL AND release IS NOT NULL AND link_sealed IS NOT NULL))
) STRICT;

INSERT INTO grabs_release_next (search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, rank, reasons,
    lost_step, lost_target_id, lost_own, lost_ref, scores, formats, covers, copy_of, release, link_sealed)
SELECT search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, rank, reasons, lost_step, lost_target_id,
    lost_own, lost_ref, scores, formats, covers, copy_of, release, link_sealed
FROM grabs_release;

DROP TABLE grabs_release;
ALTER TABLE grabs_release_next RENAME TO grabs_release;

CREATE INDEX grabs_release_indexer_id ON grabs_release (indexer_id);
CREATE INDEX grabs_release_lost_target_id ON grabs_release (lost_target_id);

-- grabs_grab: the record. source is now 'search' or 'manual', and only a
-- Manual Grab names its Manual search, its Admin and its overrides.
CREATE TABLE grabs_grab_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'search' CHECK (source IN ('search', 'manual')),
    manual_id BLOB REFERENCES grabs_manual (id) ON DELETE SET NULL,
    grabbed_by BLOB REFERENCES identity_account (id) ON DELETE SET NULL,
    grabbed_by_name TEXT,
    overrides TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(overrides)),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    tracker_id BLOB REFERENCES indexers_tracker (id),
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    release_title TEXT NOT NULL,
    release_size INTEGER NOT NULL,
    release_link_sealed BLOB,
    release_magnet TEXT,
    release_info_hash BLOB,
    fetch_query_id BLOB,
    min_ratio REAL,
    min_seed_seconds INTEGER,
    download_factor REAL,
    upload_factor REAL,
    stage TEXT NOT NULL CHECK (stage IN ('fetching', 'downloading', 'probing', 'transferring', 'importing', 'imported', 'failed')),
    torrent_id BLOB REFERENCES downloadclients_torrent (id),
    file_index INTEGER,
    probe TEXT,
    root_id BLOB REFERENCES roots_root (id),
    path TEXT,
    failed_stage TEXT,
    failure_code TEXT,
    failure_message TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    unwanted TEXT CHECK (unwanted IN ('delete', 'keep')),
    fetch_node_id BLOB REFERENCES nodes_node (id),
    metadata_started_at INTEGER,
    sent_built_in_on BLOB REFERENCES nodes_node (id),
    sent_client_id BLOB REFERENCES downloadclients_client (id),
    CHECK (stage <> 'fetching' OR release_link_sealed IS NOT NULL OR release_magnet IS NOT NULL),
    CHECK (stage IN ('fetching', 'failed') OR torrent_id IS NOT NULL),
    CHECK (stage NOT IN ('transferring', 'importing', 'imported') OR path IS NOT NULL),
    CHECK ((stage = 'failed') = (failure_code IS NOT NULL)),
    CHECK (sent_client_id IS NULL OR sent_built_in_on IS NULL),
    CHECK (source = 'manual' OR (manual_id IS NULL AND grabbed_by IS NULL AND grabbed_by_name IS NULL AND overrides = '{}'))
) STRICT;

INSERT INTO grabs_grab_next (
    id, work_id, item_id, edition, source, indexer_id, tracker_id, node_id, release_title, release_size, release_link_sealed,
    release_magnet, release_info_hash, fetch_query_id, min_ratio, min_seed_seconds, download_factor, upload_factor, stage,
    torrent_id, file_index, probe, root_id, path, failed_stage, failure_code, failure_message, created_at, updated_at, unwanted,
    fetch_node_id, metadata_started_at, sent_built_in_on, sent_client_id)
SELECT
    id, work_id, item_id, edition, source, indexer_id, tracker_id, node_id, release_title, release_size, release_link_sealed,
    release_magnet, release_info_hash, fetch_query_id, min_ratio, min_seed_seconds, download_factor, upload_factor, stage,
    torrent_id, file_index, probe, root_id, path, failed_stage, failure_code, failure_message, created_at, updated_at, unwanted,
    fetch_node_id, metadata_started_at, sent_built_in_on, sent_client_id
FROM grabs_grab;

DROP TABLE grabs_grab;
ALTER TABLE grabs_grab_next RENAME TO grabs_grab;

CREATE INDEX grabs_grab_stage ON grabs_grab (stage, created_at);
CREATE INDEX grabs_grab_work_id ON grabs_grab (work_id);
CREATE INDEX grabs_grab_node_id ON grabs_grab (node_id);
CREATE INDEX grabs_grab_torrent_id ON grabs_grab (torrent_id);
CREATE INDEX grabs_grab_indexer_id ON grabs_grab (indexer_id);
CREATE INDEX grabs_grab_tracker_id ON grabs_grab (tracker_id);
CREATE INDEX grabs_grab_manual_id ON grabs_grab (manual_id);
CREATE INDEX grabs_grab_grabbed_by ON grabs_grab (grabbed_by);

-- The Blocklist, global and keyed by the SHA-1 info hash. Names sit beside
-- ids, since the account, Indexer or Work can go. A release fault's entry
-- has no Admin.
CREATE TABLE grabs_blocklist (
    info_hash BLOB PRIMARY KEY NOT NULL CHECK (length(info_hash) = 20),
    release_title TEXT NOT NULL,
    release_size INTEGER NOT NULL CHECK (release_size >= 0),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT,
    tracker_id BLOB REFERENCES indexers_tracker (id) ON DELETE SET NULL,
    work_id BLOB REFERENCES works_work (id) ON DELETE SET NULL,
    grab_id BLOB REFERENCES grabs_grab (id) ON DELETE SET NULL,
    cause TEXT NOT NULL CHECK (cause IN ('unregistered', 'no_payload', 'engine_error', 'marked_failed', 'by_hand')),
    client_name TEXT,
    added_by BLOB REFERENCES identity_account (id) ON DELETE SET NULL,
    added_by_name TEXT,
    added_at INTEGER NOT NULL,
    CHECK (cause NOT IN ('unregistered', 'no_payload', 'engine_error') OR (added_by IS NULL AND added_by_name IS NULL)),
    CHECK (cause <> 'engine_error' OR client_name IS NOT NULL)
) STRICT;

CREATE INDEX grabs_blocklist_added_at ON grabs_blocklist (added_at, info_hash);
CREATE INDEX grabs_blocklist_indexer_id ON grabs_blocklist (indexer_id);
CREATE INDEX grabs_blocklist_tracker_id ON grabs_blocklist (tracker_id);
CREATE INDEX grabs_blocklist_work_id ON grabs_blocklist (work_id);
CREATE INDEX grabs_blocklist_grab_id ON grabs_blocklist (grab_id);
CREATE INDEX grabs_blocklist_added_by ON grabs_blocklist (added_by);

-- A Torrent's release fault grabs has handled, so removing its entry by
-- hand never brings it back.
CREATE TABLE grabs_fault_settled (
    torrent_id BLOB PRIMARY KEY NOT NULL REFERENCES downloadclients_torrent (id) ON DELETE CASCADE,
    settled_at INTEGER NOT NULL
) STRICT;
