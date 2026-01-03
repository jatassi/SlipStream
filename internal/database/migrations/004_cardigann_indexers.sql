-- +goose Up
-- +goose StatementBegin

-- Migrate indexers table to Cardigann-based system
-- SQLite requires recreating the table to modify constraints

-- Create new indexers table with Cardigann support
CREATE TABLE indexers_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    definition_id TEXT NOT NULL,              -- References Cardigann definition ID
    settings TEXT DEFAULT '{}',               -- User-provided settings (JSON)
    categories TEXT DEFAULT '[]',             -- Category filter (JSON array)
    supports_movies INTEGER NOT NULL DEFAULT 1,
    supports_tv INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 50,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Migrate existing data (map torznab/newznab to generic definition IDs)
-- For existing indexers, use the URL host as a pseudo-definition-id
-- Users will need to reconfigure these with proper Cardigann definitions
INSERT INTO indexers_new (id, name, definition_id, settings, categories, supports_movies, supports_tv, priority, enabled, created_at, updated_at)
SELECT
    id,
    name,
    CASE
        WHEN type = 'torznab' THEN 'torznab-generic'
        WHEN type = 'newznab' THEN 'newznab-generic'
        ELSE 'unknown'
    END as definition_id,
    json_object(
        'url', url,
        'apiKey', api_key,
        'apiPath', COALESCE(api_path, '/api')
    ) as settings,
    categories,
    supports_movies,
    supports_tv,
    priority,
    enabled,
    created_at,
    updated_at
FROM indexers;

-- Replace the old table
DROP TABLE indexers;
ALTER TABLE indexers_new RENAME TO indexers;

-- Create index for definition lookups
CREATE INDEX idx_indexers_definition_id ON indexers(definition_id);
CREATE INDEX idx_indexers_enabled ON indexers(enabled);

-- Track cached definition metadata
CREATE TABLE IF NOT EXISTS definition_metadata (
    id TEXT PRIMARY KEY,                      -- Definition ID (e.g., "1337x", "rarbg")
    name TEXT NOT NULL,
    description TEXT,
    privacy TEXT CHECK (privacy IN ('public', 'semi-private', 'private')),
    language TEXT,
    protocol TEXT CHECK (protocol IN ('torrent', 'usenet')),
    cached_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    file_path TEXT
);

CREATE INDEX idx_definition_metadata_protocol ON definition_metadata(protocol);
CREATE INDEX idx_definition_metadata_privacy ON definition_metadata(privacy);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore the original indexers table structure
CREATE TABLE indexers_old (
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
    settings TEXT DEFAULT '{}',
    protocol TEXT DEFAULT 'torrent',
    privacy TEXT DEFAULT 'public',
    supports_search INTEGER DEFAULT 1,
    supports_rss INTEGER DEFAULT 1,
    api_path TEXT DEFAULT '/api',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Try to restore data (this is best-effort, some data may be lost)
INSERT INTO indexers_old (id, name, type, url, api_key, categories, supports_movies, supports_tv, priority, enabled, settings, created_at, updated_at)
SELECT
    id,
    name,
    CASE
        WHEN definition_id = 'torznab-generic' THEN 'torznab'
        WHEN definition_id = 'newznab-generic' THEN 'newznab'
        ELSE 'torznab'
    END as type,
    COALESCE(json_extract(settings, '$.url'), '') as url,
    json_extract(settings, '$.apiKey') as api_key,
    categories,
    supports_movies,
    supports_tv,
    priority,
    enabled,
    settings,
    created_at,
    updated_at
FROM indexers;

DROP TABLE indexers;
ALTER TABLE indexers_old RENAME TO indexers;

-- Drop the definition metadata table
DROP TABLE IF EXISTS definition_metadata;

-- +goose StatementEnd
