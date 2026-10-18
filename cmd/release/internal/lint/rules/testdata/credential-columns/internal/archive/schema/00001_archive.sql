-- +goose Up
CREATE TABLE archive_feed (
    id BLOB PRIMARY KEY NOT NULL,
    token TEXT
) STRICT;
INSERT INTO archive_feed (id) SELECT id FROM history_feed;
