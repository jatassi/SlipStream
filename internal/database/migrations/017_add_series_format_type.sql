-- +goose Up
-- Add format type override for series (standard/daily/anime)
-- NULL means auto-detect from metadata
ALTER TABLE series ADD COLUMN format_type TEXT DEFAULT NULL
    CHECK(format_type IS NULL OR format_type IN ('standard', 'daily', 'anime'));

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we note this would need manual removal
