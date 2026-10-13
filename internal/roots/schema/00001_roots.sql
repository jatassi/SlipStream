-- +goose Up
CREATE TABLE roots_root (
    id BLOB PRIMARY KEY NOT NULL,
    content_type TEXT NOT NULL UNIQUE,
    node_id BLOB NOT NULL,
    path TEXT NOT NULL CHECK (path <> '')
) STRICT;
