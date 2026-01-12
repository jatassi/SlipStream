-- +goose Up
-- Add import tracking fields to movie_files
ALTER TABLE movie_files ADD COLUMN original_path TEXT;
ALTER TABLE movie_files ADD COLUMN original_filename TEXT;
ALTER TABLE movie_files ADD COLUMN imported_at DATETIME;

-- Add import tracking fields to episode_files
ALTER TABLE episode_files ADD COLUMN original_path TEXT;
ALTER TABLE episode_files ADD COLUMN original_filename TEXT;
ALTER TABLE episode_files ADD COLUMN imported_at DATETIME;

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we need to recreate the tables
-- For simplicity, we just note that these columns would need to be removed manually
-- In practice, the Down migration is rarely used for column additions
