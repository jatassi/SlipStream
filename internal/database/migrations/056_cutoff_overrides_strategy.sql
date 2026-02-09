-- +goose Up
ALTER TABLE quality_profiles ADD COLUMN cutoff_overrides_strategy INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE quality_profiles DROP COLUMN cutoff_overrides_strategy;
