-- +goose Up
-- S26 (Indexers and Trackers) rebuilds the Indexers, and indexers 00003
-- runs after this set and drops every existing one. v2 has no installs to
-- keep: search runs and their refusals go, Grabs still fetching fail as
-- indexer_removed, and every reference to an Indexer is nulled. A
-- migration can't seal, so every stored release link is dropped too.
DELETE FROM grabs_refusal;
DELETE FROM grabs_search;

INSERT INTO grabs_stage (grab_id, stage, entered_at)
SELECT id, 'failed', CAST(strftime('%s', 'now') AS INTEGER) * 1000000 FROM grabs_grab WHERE stage = 'fetching';

UPDATE grabs_grab SET
    stage = 'failed',
    failed_stage = 'fetching',
    failure_code = 'indexer_removed',
    failure_message = 'The Indexer was removed before the .torrent was fetched'
WHERE stage = 'fetching';

CREATE TABLE grabs_grab_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source = 'search'),
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
    CHECK (stage <> 'fetching' OR release_link_sealed IS NOT NULL OR release_magnet IS NOT NULL),
    CHECK (stage IN ('fetching', 'failed') OR torrent_id IS NOT NULL),
    CHECK (stage NOT IN ('transferring', 'importing', 'imported') OR path IS NOT NULL),
    CHECK ((stage = 'failed') = (failure_code IS NOT NULL))
) STRICT;

INSERT INTO grabs_grab_next (
    id, work_id, item_id, edition, source, node_id, release_title, release_size, release_info_hash, stage, torrent_id,
    file_index, probe, root_id, path, failed_stage, failure_code, failure_message, created_at, updated_at, unwanted,
    fetch_node_id, metadata_started_at)
SELECT
    id, work_id, item_id, edition, source, node_id, release_title, release_size, release_info_hash, stage, torrent_id,
    file_index, probe, root_id, path, failed_stage, failure_code, failure_message, created_at, updated_at, unwanted,
    fetch_node_id, metadata_started_at
FROM grabs_grab;

DROP TABLE grabs_grab;
ALTER TABLE grabs_grab_next RENAME TO grabs_grab;

CREATE INDEX grabs_grab_stage ON grabs_grab (stage, created_at);
CREATE INDEX grabs_grab_work_id ON grabs_grab (work_id);
CREATE INDEX grabs_grab_node_id ON grabs_grab (node_id);
CREATE INDEX grabs_grab_torrent_id ON grabs_grab (torrent_id);
CREATE INDEX grabs_grab_indexer_id ON grabs_grab (indexer_id);
CREATE INDEX grabs_grab_tracker_id ON grabs_grab (tracker_id);

-- A search run is one Item searched on every usable Indexer. grab_id is
-- its Grab: the decision's, or the rerun's once a dropped Grab reran it.
CREATE TABLE grabs_search_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    started_at INTEGER NOT NULL,
    finished_at INTEGER,
    outcome TEXT CHECK (outcome IN ('grabbed', 'nothing_acceptable')),
    candidates INTEGER NOT NULL DEFAULT 0,
    grab_id BLOB REFERENCES grabs_grab (id),
    notes TEXT CHECK (notes IS NULL OR json_valid(notes)),
    CHECK ((finished_at IS NULL) = (outcome IS NULL)),
    CHECK ((outcome = 'grabbed') = (grab_id IS NOT NULL))
) STRICT;

CREATE TABLE grabs_refusal_next (
    search_id BLOB NOT NULL REFERENCES grabs_search_next (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    title TEXT NOT NULL,
    reasons TEXT NOT NULL CHECK (json_valid(reasons)),
    PRIMARY KEY (search_id, position)
) STRICT;

DROP TABLE grabs_refusal;
DROP TABLE grabs_search;
ALTER TABLE grabs_search_next RENAME TO grabs_search;
ALTER TABLE grabs_refusal_next RENAME TO grabs_refusal;

CREATE INDEX grabs_search_work_id ON grabs_search (work_id, started_at);
CREATE INDEX grabs_search_open ON grabs_search (finished_at, started_at);

-- Each Indexer a search run asked or skipped, in the run's order, with its
-- name at search time. outcome is NULL while the Indexer's job runs.
CREATE TABLE grabs_search_indexer (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    query_id BLOB,
    outcome TEXT CHECK (outcome IN ('found', 'no_releases', 'limited', 'failing', 'stopped', 'waiting', 'late',
        'limit_reached', 'no_categories', 'removed')),
    releases INTEGER CHECK (releases >= 0),
    until_at INTEGER,
    since_at INTEGER,
    node_id BLOB,
    lapsed INTEGER CHECK (lapsed IN (0, 1)),
    cause TEXT,
    text TEXT,
    PRIMARY KEY (search_id, position)
) STRICT;

CREATE INDEX grabs_search_indexer_indexer_id ON grabs_search_indexer (indexer_id);

-- A run's Releases, kept while its decision may rerun after a dropped
-- Grab. release is the normalized Release as JSON, with its link sealed
-- apart.
CREATE TABLE grabs_release (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    release TEXT NOT NULL CHECK (json_valid(release)),
    link_sealed BLOB,
    PRIMARY KEY (search_id, position)
) STRICT;

CREATE INDEX grabs_release_indexer_id ON grabs_release (indexer_id);
