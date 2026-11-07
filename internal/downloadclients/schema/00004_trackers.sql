-- +goose Up
-- tracker_id is the Tracker a Grab's Release reached, set when the Grab
-- attaches the Torrent. Older Torrents keep NULL, which the foreign key
-- check passes before indexers_tracker exists.
ALTER TABLE downloadclients_torrent ADD COLUMN tracker_id BLOB REFERENCES indexers_tracker (id);
