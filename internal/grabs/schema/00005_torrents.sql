-- +goose Up
-- Each Node and info hash a Grab downloaded becomes one Torrent, with the
-- id of its first Grab. It is live while any of its Grabs still seeds,
-- and parted by a write-off when every one was dropped. v2 has no
-- installs to keep: the name is the release's, the Torrent counts as
-- private with no known trackers, and no save folder is recorded, so
-- payloads under the old flat layout download again.
WITH ranked AS (
    SELECT g.*,
           ROW_NUMBER() OVER held AS n,
           MAX(g.seed_dropped_at IS NULL) OVER held AS live,
           MAX(g.seed_dropped_at) OVER held AS dropped,
           MAX(g.download_files) OVER held AS files
    FROM grabs_grab g
    WHERE g.info_hash IS NOT NULL
    WINDOW held AS (PARTITION BY g.node_id, g.info_hash ORDER BY g.created_at, g.id
                    ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
)
INSERT INTO downloadclients_torrent (
    id, node_id, info_hash, metainfo, name, size, private, trackers, content_type,
    wanted, added_order, added_at, removed_at, removed_reason, parted_at, completion, completed_at)
SELECT
    r.id, r.node_id, r.info_hash, r.metainfo, r.release_title, r.release_size, 1, '[]', w.content_type,
    'started', ROW_NUMBER() OVER (ORDER BY r.created_at, r.id), r.created_at,
    CASE WHEN r.live = 0 THEN r.dropped END,
    CASE WHEN r.live = 0 THEN 'node_written_off' END,
    CASE WHEN r.live = 0 THEN r.dropped END,
    CASE WHEN r.files IS NOT NULL THEN json_object('files', json(r.files)) END,
    CASE WHEN r.files IS NOT NULL THEN r.updated_at END
FROM ranked r JOIN works_work w ON w.id = r.work_id
WHERE r.n = 1;

CREATE TABLE grabs_grab_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source = 'search'),
    indexer_id BLOB NOT NULL REFERENCES indexers_indexer (id),
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    release_title TEXT NOT NULL,
    release_size INTEGER NOT NULL,
    release_link TEXT NOT NULL,
    release_info_hash BLOB,
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
    CHECK (stage IN ('fetching', 'failed') OR torrent_id IS NOT NULL),
    CHECK (stage NOT IN ('transferring', 'importing', 'imported') OR path IS NOT NULL),
    CHECK ((stage = 'failed') = (failure_code IS NOT NULL))
) STRICT;

INSERT INTO grabs_grab_next (
    id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size, release_link,
    release_info_hash, stage, torrent_id, file_index, probe, root_id, path, failed_stage, failure_code,
    failure_message, created_at, updated_at, unwanted, fetch_node_id)
SELECT
    g.id, g.work_id, g.item_id, g.edition, g.source, g.indexer_id, g.node_id, g.release_title, g.release_size,
    g.release_link, g.release_info_hash, g.stage,
    (SELECT t.id FROM downloadclients_torrent t WHERE t.node_id = g.node_id AND t.info_hash = g.info_hash),
    g.file_index, g.probe, g.root_id, g.path, g.failed_stage, g.failure_code, g.failure_message, g.created_at,
    g.updated_at, g.unwanted, g.fetch_node_id
FROM grabs_grab g;

DROP TABLE grabs_grab;
ALTER TABLE grabs_grab_next RENAME TO grabs_grab;

CREATE INDEX grabs_grab_stage ON grabs_grab (stage, created_at);
CREATE INDEX grabs_grab_work_id ON grabs_grab (work_id);
CREATE INDEX grabs_grab_node_id ON grabs_grab (node_id);
CREATE INDEX grabs_grab_torrent_id ON grabs_grab (torrent_id);
