-- +goose Up
-- S27 (Ported site Indexers) adds the native kind. SQLite can't change a
-- CHECK in place, so indexers_indexer and indexers_status are rebuilt by
-- copy, keeping every Indexer and what references it. The migration
-- connection runs with foreign keys off, so the drop cascades nowhere.
CREATE TABLE indexers_indexer_new (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL CHECK (kind IN ('catalog', 'torznab', 'rss', 'native')),
    home_node_id BLOB NOT NULL REFERENCES nodes_node (id),
    is_on INTEGER NOT NULL CHECK (is_on IN (0, 1)),
    auto_search INTEGER NOT NULL CHECK (auto_search IN (0, 1)),
    rss INTEGER NOT NULL CHECK (rss IN (0, 1)),
    allow_failover INTEGER NOT NULL CHECK (allow_failover IN (0, 1)),
    priority INTEGER NOT NULL CHECK (priority BETWEEN 1 AND 50),
    categories TEXT CHECK (categories IS NULL OR json_valid(categories)),
    media TEXT CHECK (media IS NULL OR json_valid(media)),
    search_limit INTEGER NOT NULL CHECK (search_limit >= 0),
    search_window TEXT NOT NULL CHECK (search_window IN ('hour', 'day')),
    grab_limit INTEGER NOT NULL CHECK (grab_limit >= 0),
    grab_window TEXT NOT NULL CHECK (grab_window IN ('hour', 'day')),
    definition_id TEXT,
    definition_sha TEXT,
    port_id TEXT,
    base_address TEXT,
    plain TEXT NOT NULL CHECK (json_valid(plain)),
    secrets_sealed BLOB NOT NULL,
    feed_host TEXT,
    tracker_name TEXT,
    allow_no_size INTEGER NOT NULL CHECK (allow_no_size IN (0, 1)),
    rss_reading TEXT CHECK (rss_reading IS NULL OR json_valid(rss_reading)),
    caps TEXT NOT NULL CHECK (json_valid(caps)),
    notes TEXT NOT NULL CHECK (json_valid(notes)),
    revision INTEGER NOT NULL CHECK (revision >= 1),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (definition_id, definition_sha) REFERENCES indexers_definition (id, sha),
    CHECK ((kind = 'catalog') = (definition_id IS NOT NULL)),
    CHECK ((definition_id IS NULL) = (definition_sha IS NULL)),
    CHECK ((kind = 'native') = (port_id IS NOT NULL)),
    CHECK ((kind IN ('catalog', 'native')) = (base_address IS NOT NULL)),
    CHECK ((kind IN ('catalog', 'native')) <> (feed_host IS NOT NULL)),
    CHECK ((kind = 'rss') = (media IS NOT NULL))
) STRICT;

INSERT INTO indexers_indexer_new (id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority, categories,
    media, search_limit, search_window, grab_limit, grab_window, definition_id, definition_sha, base_address, plain,
    secrets_sealed, feed_host, tracker_name, allow_no_size, rss_reading, caps, notes, revision, created_at, updated_at)
SELECT id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority, categories,
    media, search_limit, search_window, grab_limit, grab_window, definition_id, definition_sha, base_address, plain,
    secrets_sealed, feed_host, tracker_name, allow_no_size, rss_reading, caps, notes, revision, created_at, updated_at
FROM indexers_indexer;

DROP TABLE indexers_indexer;
ALTER TABLE indexers_indexer_new RENAME TO indexers_indexer;

CREATE INDEX indexers_indexer_home ON indexers_indexer (home_node_id);
CREATE INDEX indexers_indexer_definition ON indexers_indexer (definition_id, definition_sha);

CREATE TABLE indexers_status_new (
    indexer_id BLOB PRIMARY KEY NOT NULL REFERENCES indexers_indexer (id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK (state IN ('ok', 'limited', 'failing', 'stopped', 'home_only')),
    rung INTEGER NOT NULL CHECK (rung >= 0),
    until_at INTEGER,
    since_at INTEGER,
    cause TEXT CHECK (cause IN ('credentials_rejected', 'captcha_needed', 'challenge_unsolved', 'cookie_expired',
        'definition_unloadable', 'account_refused', 'request_rejected', 'code_needed', 'port_retired')),
    text TEXT,
    node_id BLOB,
    solver_tried INTEGER NOT NULL CHECK (solver_tried IN (0, 1)),
    last_failure TEXT,
    last_failure_at INTEGER,
    success_at INTEGER,
    revision INTEGER NOT NULL,
    CHECK ((state IN ('stopped', 'home_only')) = (cause IS NOT NULL)),
    CHECK ((state IN ('limited', 'failing')) = (until_at IS NOT NULL)),
    CHECK ((last_failure IS NULL) = (last_failure_at IS NULL))
) STRICT;

INSERT INTO indexers_status_new SELECT * FROM indexers_status;

DROP TABLE indexers_status;
ALTER TABLE indexers_status_new RENAME TO indexers_status;

-- A token-family site's bearer token, sealed beside its cookies.
ALTER TABLE indexers_session ADD COLUMN token_sealed BLOB;
