-- +goose Up
CREATE TABLE grabs_grab (
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
    info_hash BLOB,
    metainfo BLOB,
    download_files TEXT,
    download_error TEXT,
    file_index INTEGER,
    probe TEXT,
    root_id BLOB REFERENCES roots_root (id),
    path TEXT,
    failed_stage TEXT,
    failure_code TEXT,
    failure_message TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK ((info_hash IS NULL) = (metainfo IS NULL)),
    CHECK (stage IN ('fetching', 'failed') OR info_hash IS NOT NULL),
    CHECK (download_files IS NULL OR download_error IS NULL),
    CHECK (stage NOT IN ('transferring', 'importing', 'imported') OR path IS NOT NULL),
    CHECK ((stage = 'failed') = (failure_code IS NOT NULL))
) STRICT;

CREATE INDEX grabs_grab_stage ON grabs_grab (stage, created_at);
CREATE INDEX grabs_grab_work_id ON grabs_grab (work_id);
CREATE INDEX grabs_grab_node_id ON grabs_grab (node_id, info_hash);

-- grabs_target is the Targets a Grab was grabbed for, in Profile order.
CREATE TABLE grabs_target (
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    target_id BLOB NOT NULL REFERENCES targets_target (id),
    PRIMARY KEY (grab_id, position)
) STRICT;

-- grabs_fill is the Targets the probed file fills, in Profile order: the
-- plan probing decides before any Transfer.
CREATE TABLE grabs_fill (
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    target_id BLOB NOT NULL REFERENCES targets_target (id),
    PRIMARY KEY (grab_id, position)
) STRICT;

CREATE TABLE grabs_stage (
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    stage TEXT NOT NULL,
    entered_at INTEGER NOT NULL,
    PRIMARY KEY (grab_id, stage)
) STRICT;

CREATE TABLE grabs_search (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    indexer_id BLOB NOT NULL REFERENCES indexers_indexer (id),
    started_at INTEGER NOT NULL,
    finished_at INTEGER,
    outcome TEXT CHECK (outcome IN ('grabbed', 'nothing_acceptable', 'indexer_unavailable', 'indexer_refused')),
    candidates INTEGER NOT NULL DEFAULT 0,
    grab_id BLOB REFERENCES grabs_grab (id),
    CHECK ((finished_at IS NULL) = (outcome IS NULL)),
    CHECK ((outcome = 'grabbed') = (grab_id IS NOT NULL))
) STRICT;

CREATE INDEX grabs_search_work_id ON grabs_search (work_id, started_at);
CREATE INDEX grabs_search_open ON grabs_search (finished_at, started_at);
