-- +goose Up
CREATE TABLE metadata_announced (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    day TEXT NOT NULL CHECK (day GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]')
) STRICT;
