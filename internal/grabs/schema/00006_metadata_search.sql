-- +goose Up
-- metadata_started_at is when a Grab of a magnet link first asked its
-- Downloading Node for the link's metadata. The search's deadline counts
-- from it, so a restart of either Node keeps it.
ALTER TABLE grabs_grab ADD COLUMN metadata_started_at INTEGER;
