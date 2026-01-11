-- +goose Up
-- +goose StatementBegin

-- Add quality_id column to movie_files for efficient upgrade candidate queries
-- This stores the numeric Quality ID (1-17) parsed from the quality text field
ALTER TABLE movie_files ADD COLUMN quality_id INTEGER;

-- Add quality_id column to episode_files for efficient upgrade candidate queries
ALTER TABLE episode_files ADD COLUMN quality_id INTEGER;

-- Create indexes for upgrade candidate queries
CREATE INDEX IF NOT EXISTS idx_movie_files_quality_id ON movie_files(quality_id);
CREATE INDEX IF NOT EXISTS idx_episode_files_quality_id ON episode_files(quality_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_episode_files_quality_id;
DROP INDEX IF EXISTS idx_movie_files_quality_id;

-- SQLite doesn't support DROP COLUMN directly
-- The columns will remain but won't affect functionality

-- +goose StatementEnd
