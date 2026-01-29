-- +goose Up
-- Rename on_download to on_import for clarity (triggers on file import, not download completion)
ALTER TABLE notifications RENAME COLUMN on_download TO on_import;

-- +goose Down
ALTER TABLE notifications RENAME COLUMN on_import TO on_download;
