-- +goose Up
-- announced_seconds is the seed time the torrent's trackers credited it.
-- The Node stores it in the Write that queues each report carrying it, so
-- it never reports a count that a restart could fall below. failed is the
-- engine's error text while the torrent is failed, NULL otherwise.
ALTER TABLE downloading_torrent ADD COLUMN announced_seconds INTEGER NOT NULL DEFAULT 0 CHECK (announced_seconds >= 0);
ALTER TABLE downloading_torrent ADD COLUMN failed TEXT CHECK (failed <> '');
