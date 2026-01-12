-- +goose Up
-- Add import-related fields to download_clients
ALTER TABLE download_clients ADD COLUMN import_delay_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE download_clients ADD COLUMN cleanup_mode TEXT NOT NULL DEFAULT 'leave'
    CHECK(cleanup_mode IN ('leave', 'delete_after_import', 'delete_after_seed_ratio'));
ALTER TABLE download_clients ADD COLUMN seed_ratio_target REAL DEFAULT NULL;

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we note these would need manual removal
