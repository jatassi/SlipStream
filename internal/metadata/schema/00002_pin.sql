-- +goose Up
-- A subscriber PIN rides beside the Admin's own key, sealed like it.
ALTER TABLE metadata_key ADD COLUMN pin_sealed BLOB;
