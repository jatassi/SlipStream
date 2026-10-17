-- +goose Up
CREATE TABLE history_feed_sealed (
    id BLOB PRIMARY KEY NOT NULL,
    api_key_sealed BLOB
) STRICT;
INSERT INTO history_feed_sealed (id, api_key_sealed) SELECT id, NULL FROM history_feed;
DROP TABLE history_feed;
ALTER TABLE history_feed_sealed RENAME TO history_feed;
