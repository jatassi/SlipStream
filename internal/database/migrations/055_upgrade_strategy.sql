-- +goose Up
ALTER TABLE quality_profiles ADD COLUMN upgrade_strategy TEXT NOT NULL DEFAULT 'balanced';

-- +goose Down
ALTER TABLE quality_profiles DROP COLUMN upgrade_strategy;
