-- +goose Up
-- dht is a Node's DHT switch, and metadata_wait_seconds how long a
-- magnet link's metadata search runs before its Grab fails.
ALTER TABLE downloadclients_node ADD COLUMN dht INTEGER CHECK (dht IN (0, 1));
ALTER TABLE downloadclients_settings ADD COLUMN metadata_wait_seconds INTEGER CHECK (metadata_wait_seconds > 0);
