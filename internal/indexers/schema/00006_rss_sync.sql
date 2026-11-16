-- +goose Up
-- S34 (Automatic search, Backlog search and RSS sync). Searches at once
-- bounds how many automatic searches go to an Indexer at a time.
ALTER TABLE indexers_indexer ADD COLUMN searches_at_once INTEGER NOT NULL DEFAULT 2
    CHECK (searches_at_once BETWEEN 1 AND 10);

-- Every search query names whose line it reserved: a press (Manual search
-- and the Admin's own searches) or a background line. The room rule counts
-- the in-flight ones per claim. SQLite can't add a CHECK across columns in
-- place, so the table is rebuilt by copy. Every search reserved before this
-- was a Manual search's.
CREATE TABLE indexers_query_next (
    id BLOB PRIMARY KEY NOT NULL,
    indexer_id BLOB NOT NULL REFERENCES indexers_indexer (id) ON DELETE CASCADE,
    use TEXT NOT NULL CHECK (use IN ('search', 'recent', 'fetch', 'test')),
    claim TEXT CHECK (claim IN ('press', 'background')),
    reserved_at INTEGER NOT NULL,
    started_at INTEGER,
    completed_at INTEGER,
    node_id BLOB,
    outcome TEXT CHECK (outcome IN ('ok', 'limited', 'failing', 'stopped', 'lost')),
    releases INTEGER CHECK (releases >= 0),
    revision INTEGER NOT NULL,
    sha TEXT,
    CHECK ((completed_at IS NULL) = (outcome IS NULL)),
    CHECK ((use = 'search') = (claim IS NOT NULL))
) STRICT;

INSERT INTO indexers_query_next (id, indexer_id, use, claim, reserved_at, started_at, completed_at, node_id, outcome, releases,
    revision, sha)
SELECT id, indexer_id, use, CASE use WHEN 'search' THEN 'press' END, reserved_at, started_at, completed_at, node_id, outcome,
    releases, revision, sha
FROM indexers_query;

DROP TABLE indexers_query;
ALTER TABLE indexers_query_next RENAME TO indexers_query;

CREATE INDEX indexers_query_window ON indexers_query (indexer_id, use, reserved_at);
CREATE INDEX indexers_query_reserved_at ON indexers_query (reserved_at);
CREATE INDEX indexers_query_inflight ON indexers_query (indexer_id, claim, reserved_at)
    WHERE completed_at IS NULL AND use = 'search';

-- indexers_rss is each feed's position for RSS sync. last_guids: up to 100
-- newest GUIDs, newest first. last_pages: how many pages the last fetch
-- used, which RSS sync's reserve in the search limit counts. gap_at and
-- gap_kind: the last fetch didn't reach the stored position, cleared when
-- one does.
CREATE TABLE indexers_rss (
    indexer_id BLOB PRIMARY KEY NOT NULL REFERENCES indexers_indexer (id) ON DELETE CASCADE,
    last_guids TEXT NOT NULL CHECK (json_valid(last_guids) AND json_type(last_guids) = 'array'
        AND json_array_length(last_guids) <= 100),
    last_published_at INTEGER,
    last_fetch_at INTEGER NOT NULL,
    last_pages INTEGER NOT NULL DEFAULT 1 CHECK (last_pages >= 1),
    gap_at INTEGER,
    gap_kind TEXT CHECK (gap_kind IN ('no_paging', 'page_cap', 'late')),
    CHECK ((gap_at IS NULL) = (gap_kind IS NULL))
) STRICT;
