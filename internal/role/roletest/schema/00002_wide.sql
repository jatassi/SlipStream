-- +goose Up
CREATE TABLE probe_wide (
    node_id BLOB PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;
