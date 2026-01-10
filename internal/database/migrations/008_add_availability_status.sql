-- +goose Up
-- +goose StatementBegin

-- Add availability_status column to series for storing calculated badge text
-- Values: "Available", "Unreleased", "Airing", "Partial", "Seasons 1-2 Available", etc.
ALTER TABLE series ADD COLUMN availability_status TEXT NOT NULL DEFAULT 'Unreleased';

-- Add availability_status column to movies for consistency
-- Values: "Available", "Unreleased"
ALTER TABLE movies ADD COLUMN availability_status TEXT NOT NULL DEFAULT 'Unreleased';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN directly, need to recreate tables
-- For simplicity, we'll leave the columns in place during rollback
-- In production, you'd recreate the tables without these columns

-- +goose StatementEnd
