-- +goose Up
ALTER TABLE prowlarr_indexer_settings DROP COLUMN rss_enabled;

-- +goose Down
ALTER TABLE prowlarr_indexer_settings ADD COLUMN rss_enabled INTEGER NOT NULL DEFAULT 1;
