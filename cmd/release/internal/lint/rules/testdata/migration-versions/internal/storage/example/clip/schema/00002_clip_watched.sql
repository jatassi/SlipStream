-- +goose Up
-- A comment that mentions +goose Down is not a directive.
ALTER TABLE clip ADD COLUMN watched_at INTEGER;
