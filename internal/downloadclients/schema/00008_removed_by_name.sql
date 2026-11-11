-- +goose Up
-- S32 (Manual search and grab) names the Admin who removed a Torrent or marked
-- it failed. removed_by_name is NULL for the API key and every other reason.
ALTER TABLE downloadclients_torrent ADD COLUMN removed_by_name TEXT CHECK (removed_by_name <> '');
