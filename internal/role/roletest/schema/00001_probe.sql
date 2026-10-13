-- +goose Up
CREATE TABLE probe_intent (
    node_id BLOB PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;

CREATE TABLE probe_applied (
    id INTEGER PRIMARY KEY,
    node_id BLOB NOT NULL,
    n INTEGER NOT NULL
) STRICT;
