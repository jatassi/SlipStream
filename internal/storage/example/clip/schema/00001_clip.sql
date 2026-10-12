-- +goose Up
CREATE TABLE clip (
    id INTEGER PRIMARY KEY,
    shelf_id INTEGER NOT NULL REFERENCES shelf (id),
    public_id BLOB NOT NULL UNIQUE,
    title TEXT NOT NULL,
    created_at INTEGER NOT NULL
) STRICT;

CREATE INDEX clip_shelf_id ON clip (shelf_id);
