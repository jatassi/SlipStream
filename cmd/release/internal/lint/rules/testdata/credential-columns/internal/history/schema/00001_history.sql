-- +goose Up
CREATE TABLE history_feed (
    id BLOB PRIMARY KEY NOT NULL,
    api_key TEXT NOT NULL
) STRICT;
