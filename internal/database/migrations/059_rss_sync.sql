-- +goose Up
-- RSS Sync support: per-indexer RSS toggle and cache boundary tracking.

ALTER TABLE indexers ADD COLUMN rss_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE prowlarr_indexer_settings ADD COLUMN rss_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE indexer_status ADD COLUMN last_rss_release_url TEXT;
ALTER TABLE indexer_status ADD COLUMN last_rss_release_date DATETIME;

-- +goose Down
-- SQLite doesn't support DROP COLUMN before 3.35.0, so we recreate tables.
-- For indexers, we drop the rss_enabled column.
-- For prowlarr_indexer_settings, we drop the rss_enabled column.
-- For indexer_status, we drop the last_rss_release_url and last_rss_release_date columns.

-- Note: These ALTER TABLE DROP COLUMN statements require SQLite 3.35.0+
-- which is available in Go's modernc.org/sqlite driver.
ALTER TABLE indexers DROP COLUMN rss_enabled;
ALTER TABLE prowlarr_indexer_settings DROP COLUMN rss_enabled;
ALTER TABLE indexer_status DROP COLUMN last_rss_release_url;
ALTER TABLE indexer_status DROP COLUMN last_rss_release_date;
