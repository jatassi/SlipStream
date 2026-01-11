-- +goose Up
-- +goose StatementBegin

-- Add auto_search_enabled column to indexers table
-- When enabled, the indexer will be used for automatic/scheduled searches
-- Manual searches always use all enabled indexers regardless of this setting
ALTER TABLE indexers ADD COLUMN auto_search_enabled INTEGER NOT NULL DEFAULT 1;

-- Create table for tracking automatic search status per item
-- Used for backoff mechanism when items repeatedly fail to find releases
CREATE TABLE IF NOT EXISTS autosearch_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_type TEXT NOT NULL CHECK (item_type IN ('movie', 'episode')),
    item_id INTEGER NOT NULL,
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_searched_at DATETIME,
    last_meta_change_at DATETIME,
    UNIQUE(item_type, item_id)
);

CREATE INDEX idx_autosearch_status_item ON autosearch_status(item_type, item_id);
CREATE INDEX idx_autosearch_status_failure_count ON autosearch_status(failure_count);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop the autosearch_status table
DROP TABLE IF EXISTS autosearch_status;

-- SQLite doesn't support DROP COLUMN directly, but for the down migration
-- we can use a workaround. For simplicity, we'll leave the column in place
-- as it won't affect functionality when auto-search feature is removed.
-- In production, a full table recreation would be needed to truly remove the column.

-- +goose StatementEnd
