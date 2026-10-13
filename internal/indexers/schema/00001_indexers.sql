-- +goose Up
CREATE TABLE indexers_indexer (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE,
    protocol TEXT NOT NULL CHECK (protocol = 'torznab'),
    url TEXT NOT NULL,
    api_key TEXT NOT NULL,
    home_node_id BLOB NOT NULL,
    caps TEXT NOT NULL,
    created_at INTEGER NOT NULL
) STRICT;
