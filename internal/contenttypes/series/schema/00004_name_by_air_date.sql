-- +goose Up
ALTER TABLE series_series ADD COLUMN name_by_air_date INTEGER NOT NULL DEFAULT 0 CHECK (name_by_air_date IN (0, 1));
-- NULL is the default, off.
ALTER TABLE series_settings ADD COLUMN add_name_by_air_date INTEGER CHECK (add_name_by_air_date IN (0, 1));
