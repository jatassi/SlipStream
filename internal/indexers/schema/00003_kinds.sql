-- +goose Up
-- S26 (Indexers and Trackers) rebuilds the Indexers with their kinds, and
-- SQLite can't change the protocol CHECK in place. v2 has no installs to
-- keep: grabs 00007 already nulled every reference, and every Indexer is
-- dropped. The Fake world creates its Indexers again on start.
DROP TABLE indexers_indexer;

-- Every catalog definition version an Indexer pins or the catalog lists,
-- and every custom definition, with its YAML. summary is what Load read,
-- refusal why SlipStream can't run it.
CREATE TABLE indexers_definition (
    id TEXT NOT NULL,
    sha TEXT NOT NULL,
    file TEXT NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('catalog', 'custom')),
    yaml BLOB NOT NULL,
    summary TEXT CHECK (summary IS NULL OR json_valid(summary)),
    refusal TEXT,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (id, sha),
    CHECK ((summary IS NULL) <> (refusal IS NULL))
) STRICT;

-- The Indexer catalog as its last good refresh listed it.
CREATE TABLE indexers_catalog (
    id TEXT PRIMARY KEY NOT NULL,
    file TEXT NOT NULL UNIQUE,
    sha TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    privacy TEXT NOT NULL CHECK (privacy IN ('private', 'semi-private', 'public')),
    language TEXT NOT NULL,
    categories TEXT NOT NULL CHECK (json_valid(categories))
) STRICT;

-- address is NULL for the endpoints.indexers.catalog default.
CREATE TABLE indexers_catalog_state (
    one INTEGER PRIMARY KEY CHECK (one = 1),
    refreshed_at INTEGER,
    last_modified TEXT,
    address TEXT
) STRICT;

INSERT INTO indexers_catalog_state (one) VALUES (1);

-- long_backoff_seconds is NULL for indexers.longBackoff's default.
CREATE TABLE indexers_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    long_backoff_seconds INTEGER
) STRICT;

INSERT INTO indexers_settings (id) VALUES (1);

-- secrets_sealed is one sealed JSON object: every secret field value, a
-- feed's whole address and its API key. revision bumps on every saved
-- edit, and a job's answer from an older revision writes no status.
CREATE TABLE indexers_indexer (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL CHECK (kind IN ('catalog', 'torznab', 'rss')),
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
    CHECK ((kind = 'catalog') = (base_address IS NOT NULL)),
    CHECK ((kind = 'catalog') <> (feed_host IS NOT NULL)),
    CHECK ((kind = 'rss') = (media IS NOT NULL))
) STRICT;

CREATE INDEX indexers_indexer_home ON indexers_indexer (home_node_id);
CREATE INDEX indexers_indexer_definition ON indexers_indexer (definition_id, definition_sha);

-- One status per Indexer, written only from a job's answer. A stop or a
-- home_only wait has its cause, and node_id is where a node-bound cause
-- was met.
CREATE TABLE indexers_status (
    indexer_id BLOB PRIMARY KEY NOT NULL REFERENCES indexers_indexer (id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK (state IN ('ok', 'limited', 'failing', 'stopped', 'home_only')),
    rung INTEGER NOT NULL CHECK (rung >= 0),
    until_at INTEGER,
    since_at INTEGER,
    cause TEXT CHECK (cause IN ('credentials_rejected', 'captcha_needed', 'challenge_unsolved', 'cookie_expired',
        'definition_unloadable', 'account_refused', 'request_rejected')),
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

-- An Indexer's session on one Node: a sign-in a job earned, a challenge's
-- clearance, or a cookie an Admin pasted. A newer earned_at replaces it.
CREATE TABLE indexers_session (
    indexer_id BLOB NOT NULL REFERENCES indexers_indexer (id) ON DELETE CASCADE,
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    cookies_sealed BLOB NOT NULL,
    user_agent TEXT NOT NULL,
    pasted INTEGER NOT NULL CHECK (pasted IN (0, 1)),
    earned_at INTEGER NOT NULL,
    PRIMARY KEY (indexer_id, node_id)
) STRICT;

CREATE INDEX indexers_session_node_id ON indexers_session (node_id);

-- One row per job: a search, a recent-items call, a fetch or a Test. A
-- search's and a fetch's row is reserved in the Write that decides it, so
-- the limits count reserved rows. The row's id is the job's id.
CREATE TABLE indexers_query (
    id BLOB PRIMARY KEY NOT NULL,
    indexer_id BLOB NOT NULL REFERENCES indexers_indexer (id) ON DELETE CASCADE,
    use TEXT NOT NULL CHECK (use IN ('search', 'recent', 'fetch', 'test')),
    reserved_at INTEGER NOT NULL,
    started_at INTEGER,
    completed_at INTEGER,
    node_id BLOB,
    outcome TEXT CHECK (outcome IN ('ok', 'limited', 'failing', 'stopped', 'lost')),
    releases INTEGER CHECK (releases >= 0),
    revision INTEGER NOT NULL,
    sha TEXT,
    CHECK ((completed_at IS NULL) = (outcome IS NULL))
) STRICT;

CREATE INDEX indexers_query_window ON indexers_query (indexer_id, use, reserved_at);
CREATE INDEX indexers_query_reserved_at ON indexers_query (reserved_at);

CREATE TABLE indexers_tracker (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE,
    privacy TEXT NOT NULL CHECK (privacy IN ('private', 'semi-private', 'public', 'unknown')),
    created_at INTEGER NOT NULL
) STRICT;

-- A key belongs to exactly one Tracker: "definition:<id>",
-- "prowlarr:<feed Indexer>:<source id>" or "named:<id>".
CREATE TABLE indexers_tracker_key (
    key TEXT PRIMARY KEY NOT NULL,
    tracker_id BLOB NOT NULL REFERENCES indexers_tracker (id) ON DELETE CASCADE
) STRICT;

CREATE INDEX indexers_tracker_key_tracker_id ON indexers_tracker_key (tracker_id);

-- The announce hosts of a Tracker's private torrents.
CREATE TABLE indexers_tracker_host (
    tracker_id BLOB NOT NULL REFERENCES indexers_tracker (id) ON DELETE CASCADE,
    host TEXT NOT NULL,
    PRIMARY KEY (tracker_id, host)
) STRICT;

-- Which Indexers reach which Tracker.
CREATE TABLE indexers_reach (
    indexer_id BLOB NOT NULL REFERENCES indexers_indexer (id) ON DELETE CASCADE,
    tracker_id BLOB NOT NULL REFERENCES indexers_tracker (id) ON DELETE CASCADE,
    PRIMARY KEY (indexer_id, tracker_id)
) STRICT;

CREATE INDEX indexers_reach_tracker_id ON indexers_reach (tracker_id);

-- Each Searching Node's Challenge solver.
CREATE TABLE indexers_node (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id),
    solver_address TEXT NOT NULL,
    solver_timeout_seconds INTEGER CHECK (solver_timeout_seconds BETWEEN 10 AND 180)
) STRICT;
