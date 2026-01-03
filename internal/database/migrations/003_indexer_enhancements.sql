-- +goose Up
-- +goose StatementBegin

-- Enhance indexers table with additional fields for protocol support
ALTER TABLE indexers ADD COLUMN settings TEXT DEFAULT '{}';  -- JSON for provider-specific settings
ALTER TABLE indexers ADD COLUMN protocol TEXT DEFAULT 'torrent' CHECK (protocol IN ('torrent', 'usenet'));
ALTER TABLE indexers ADD COLUMN privacy TEXT DEFAULT 'public' CHECK (privacy IN ('public', 'semi-private', 'private'));
ALTER TABLE indexers ADD COLUMN supports_search INTEGER DEFAULT 1;
ALTER TABLE indexers ADD COLUMN supports_rss INTEGER DEFAULT 1;
ALTER TABLE indexers ADD COLUMN api_path TEXT DEFAULT '/api';

-- Indexer status tracking (for health/failure management)
CREATE TABLE IF NOT EXISTS indexer_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    indexer_id INTEGER NOT NULL UNIQUE REFERENCES indexers(id) ON DELETE CASCADE,
    initial_failure DATETIME,
    most_recent_failure DATETIME,
    escalation_level INTEGER DEFAULT 0,
    disabled_till DATETIME,
    last_rss_sync DATETIME,
    cookies TEXT,  -- JSON for session cookies
    cookies_expiration DATETIME
);

CREATE INDEX idx_indexer_status_indexer_id ON indexer_status(indexer_id);

-- Search/grab history for indexers
CREATE TABLE IF NOT EXISTS indexer_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    indexer_id INTEGER NOT NULL REFERENCES indexers(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('query', 'rss', 'grab', 'auth')),
    successful INTEGER NOT NULL DEFAULT 1,
    query TEXT,
    categories TEXT,  -- JSON array
    results_count INTEGER,
    elapsed_ms INTEGER,
    data TEXT,  -- JSON for additional data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_indexer_history_indexer_id ON indexer_history(indexer_id);
CREATE INDEX idx_indexer_history_created_at ON indexer_history(created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop the new tables
DROP TABLE IF EXISTS indexer_history;
DROP TABLE IF EXISTS indexer_status;

-- SQLite doesn't support DROP COLUMN, so we recreate the indexers table
CREATE TABLE indexers_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('torznab', 'newznab')),
    url TEXT NOT NULL,
    api_key TEXT,
    categories TEXT DEFAULT '[]',
    supports_movies INTEGER NOT NULL DEFAULT 1,
    supports_tv INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 50,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO indexers_new (id, name, type, url, api_key, categories, supports_movies, supports_tv, priority, enabled, created_at, updated_at)
SELECT id, name, type, url, api_key, categories, supports_movies, supports_tv, priority, enabled, created_at, updated_at FROM indexers;

DROP TABLE indexers;
ALTER TABLE indexers_new RENAME TO indexers;

-- +goose StatementEnd
