-- +goose Up
CREATE TABLE secret_check (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    value BLOB NOT NULL
) STRICT;
