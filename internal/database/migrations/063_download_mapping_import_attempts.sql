-- +goose Up
ALTER TABLE download_mappings ADD COLUMN import_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE download_mappings ADD COLUMN last_import_error TEXT;

-- +goose Down
ALTER TABLE download_mappings DROP COLUMN import_attempts;
ALTER TABLE download_mappings DROP COLUMN last_import_error;
