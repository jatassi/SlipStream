-- +goose Up
-- How long a torrent unit waits for its destination client's check
-- before it rolls back, in minutes. NULL is its default, kept in Go.
ALTER TABLE moves_settings ADD COLUMN check_bound_minutes INTEGER CHECK (check_bound_minutes BETWEEN 30 AND 10080);
