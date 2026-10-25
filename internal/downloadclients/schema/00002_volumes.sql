-- +goose Up
-- Each downloading Node's downloads folder volume as its client last
-- sampled it, so an offline Node's figures still count.
CREATE TABLE downloadclients_volume (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id),
    mount TEXT NOT NULL,
    name TEXT NOT NULL,
    total INTEGER NOT NULL CHECK (total >= 0),
    free INTEGER NOT NULL CHECK (free >= 0),
    sampled_at INTEGER NOT NULL
) STRICT;
