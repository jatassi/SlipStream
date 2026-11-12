-- +goose Up
-- S33 (Import pipeline): a Grab ends at its download, and an Import takes
-- the files from there. grabs_carry keeps each in-flight Grab past its
-- download as this release found it, for imports' first start to carry
-- into an Import with the Grab's id. The table stays, empty once carried.
CREATE TABLE grabs_carry (
    grab_id BLOB PRIMARY KEY NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    was TEXT NOT NULL CHECK (was IN ('probing', 'transferring', 'importing')),
    file_index INTEGER,
    probe TEXT,
    root_id BLOB,
    path TEXT,
    -- The Targets it fills, in position order, as a JSON list of hex ids.
    fills TEXT NOT NULL CHECK (json_valid(fills)),
    CHECK (was = 'probing' OR (file_index IS NOT NULL AND probe IS NOT NULL AND root_id IS NOT NULL AND path IS NOT NULL))
) STRICT;

INSERT INTO grabs_carry (grab_id, was, file_index, probe, root_id, path, fills)
SELECT g.id, g.stage,
    CASE WHEN g.stage <> 'probing' THEN g.file_index END,
    CASE WHEN g.stage <> 'probing' THEN g.probe END,
    CASE WHEN g.stage <> 'probing' THEN g.root_id END,
    CASE WHEN g.stage <> 'probing' THEN g.path END,
    COALESCE((SELECT json_group_array(hex(f.target_id)) FROM (
        SELECT target_id FROM grabs_fill WHERE grab_id = g.id ORDER BY position) f), '[]')
FROM grabs_grab g WHERE g.stage IN ('probing', 'transferring', 'importing');

-- One importing row per Grab that passed probing, transferring or
-- importing, entered when the first of them was.
INSERT INTO grabs_stage (grab_id, stage, entered_at)
SELECT grab_id, 'importing', MIN(entered_at) FROM grabs_stage WHERE stage IN ('probing', 'transferring', 'importing') GROUP BY grab_id
ON CONFLICT (grab_id, stage) DO UPDATE SET entered_at = excluded.entered_at;
DELETE FROM grabs_stage WHERE stage IN ('probing', 'transferring');

-- grabs_grab keeps five stages. file_index, root_id, path and unwanted go.
-- probe stays, unread, because files 00002 selects it.
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
    stage TEXT NOT NULL CHECK (stage IN ('fetching', 'downloading', 'importing', 'imported', 'failed')),
    torrent_id BLOB REFERENCES downloadclients_torrent (id),
    probe TEXT,
    failed_stage TEXT,
    failure_code TEXT,
    failure_message TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    fetch_node_id BLOB REFERENCES nodes_node (id),
    metadata_started_at INTEGER,
    sent_built_in_on BLOB REFERENCES nodes_node (id),
    sent_client_id BLOB REFERENCES downloadclients_client (id),
    CHECK (stage <> 'fetching' OR release_link_sealed IS NOT NULL OR release_magnet IS NOT NULL),
    CHECK (stage IN ('fetching', 'failed') OR torrent_id IS NOT NULL),
    CHECK ((stage = 'failed') = (failure_code IS NOT NULL)),
    CHECK (sent_client_id IS NULL OR sent_built_in_on IS NULL),
    CHECK (source = 'manual' OR (manual_id IS NULL AND grabbed_by IS NULL AND grabbed_by_name IS NULL AND overrides = '{}'))
) STRICT;

INSERT INTO grabs_grab_next (
    id, work_id, item_id, edition, source, manual_id, grabbed_by, grabbed_by_name, overrides, indexer_id, tracker_id, node_id,
    release_title, release_size, release_link_sealed, release_magnet, release_info_hash, fetch_query_id, min_ratio,
    min_seed_seconds, download_factor, upload_factor, stage, torrent_id, probe, failed_stage, failure_code, failure_message,
    created_at, updated_at, fetch_node_id, metadata_started_at, sent_built_in_on, sent_client_id)
SELECT
    id, work_id, item_id, edition, source, manual_id, grabbed_by, grabbed_by_name, overrides, indexer_id, tracker_id, node_id,
    release_title, release_size, release_link_sealed, release_magnet, release_info_hash, fetch_query_id, min_ratio,
    min_seed_seconds, download_factor, upload_factor,
    CASE WHEN stage IN ('probing', 'transferring') THEN 'importing' ELSE stage END,
    torrent_id, probe,
    CASE WHEN failed_stage IN ('probing', 'transferring') THEN 'importing' ELSE failed_stage END,
    failure_code, failure_message, created_at, updated_at, fetch_node_id, metadata_started_at, sent_built_in_on, sent_client_id
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

DROP TABLE grabs_fill;
