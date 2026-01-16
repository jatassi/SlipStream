-- +goose Up
ALTER TABLE quality_profiles ADD COLUMN upgrades_enabled INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE quality_profiles DROP COLUMN upgrades_enabled;
