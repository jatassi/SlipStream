-- +goose Up
-- A file's release name is its Grab's Release title, or its own file name
-- when no Grab brought it, and title_read is release.Read's JSON of it.
-- Both are NULL only until the catch-up at start fills them.
ALTER TABLE files_file ADD COLUMN release_name TEXT;
ALTER TABLE files_file ADD COLUMN title_read TEXT CHECK (json_valid(title_read));

-- A fill's score is its file's Custom Format score at the fill's Target.
-- No format exists before this migration, so every fill scores 0.
ALTER TABLE files_fill ADD COLUMN score INTEGER NOT NULL DEFAULT 0;
