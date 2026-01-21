-- +goose Up
-- Per-indexer settings for Prowlarr indexers
-- Allows customizing priority, content type filtering, categories, and tracks failure stats
CREATE TABLE prowlarr_indexer_settings (
    prowlarr_indexer_id INTEGER PRIMARY KEY,
    priority INTEGER NOT NULL DEFAULT 25,
    content_type TEXT NOT NULL DEFAULT 'both',
    movie_categories TEXT DEFAULT NULL,
    tv_categories TEXT DEFAULT NULL,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_failure_at DATETIME,
    last_failure_reason TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS prowlarr_indexer_settings;
