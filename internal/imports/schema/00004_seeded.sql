-- +goose Up
-- seeded is a folder file a torrent held when the folder was opened,
-- which Move files can't take.
ALTER TABLE imports_file ADD COLUMN seeded INTEGER NOT NULL DEFAULT 0 CHECK (seeded IN (0, 1));
