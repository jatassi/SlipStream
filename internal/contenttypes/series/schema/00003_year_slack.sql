-- +goose Up
-- NULL is the default.
ALTER TABLE series_settings ADD COLUMN year_slack INTEGER CHECK (year_slack >= 0);
