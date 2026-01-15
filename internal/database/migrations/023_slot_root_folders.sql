-- +goose Up
-- +goose StatementBegin

-- Add root folder assignments to version slots
-- Each slot can have a dedicated root folder per media type (movie and TV)
-- This enables storing different quality versions on different storage devices
ALTER TABLE version_slots ADD COLUMN movie_root_folder_id INTEGER REFERENCES root_folders(id) ON DELETE SET NULL;
ALTER TABLE version_slots ADD COLUMN tv_root_folder_id INTEGER REFERENCES root_folders(id) ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN directly
-- The columns will remain but won't affect functionality

-- +goose StatementEnd
