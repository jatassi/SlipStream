-- +goose Up
ALTER TABLE download_mappings ADD COLUMN next_import_retry_at DATETIME;

-- +goose Down
ALTER TABLE download_mappings DROP COLUMN next_import_retry_at;
