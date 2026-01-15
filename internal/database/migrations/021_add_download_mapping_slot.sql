-- +goose Up
-- Add target_slot_id to download_mappings for multi-version slot tracking
ALTER TABLE download_mappings ADD COLUMN target_slot_id INTEGER REFERENCES version_slots(id);

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly, but this migration is reversible
-- by recreating the table without the column if needed
