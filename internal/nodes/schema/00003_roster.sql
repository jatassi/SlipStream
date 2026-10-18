-- +goose Up
CREATE TABLE nodes_node_new (
    id BLOB PRIMARY KEY NOT NULL,
    public_key BLOB CHECK (public_key IS NULL OR length(public_key) = 32),
    name TEXT NOT NULL,
    paired_at INTEGER NOT NULL,
    seen_at INTEGER,
    removed_at INTEGER,
    removed_key BLOB CHECK (removed_key IS NULL OR length(removed_key) = 32),
    CHECK ((removed_at IS NULL) = (public_key IS NOT NULL)),
    CHECK ((removed_at IS NULL) = (removed_key IS NULL))
) STRICT;

INSERT INTO nodes_node_new (id, public_key, name, paired_at)
SELECT id, public_key, name, paired_at FROM nodes_node;

DROP TABLE nodes_node;

ALTER TABLE nodes_node_new RENAME TO nodes_node;

CREATE UNIQUE INDEX nodes_node_live_key ON nodes_node (public_key) WHERE removed_at IS NULL;

CREATE UNIQUE INDEX nodes_node_live_name ON nodes_node (name) WHERE removed_at IS NULL;

CREATE INDEX nodes_node_removed_key ON nodes_node (removed_key) WHERE removed_key IS NOT NULL;

CREATE TABLE nodes_forfeit (
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    role TEXT NOT NULL,
    declared_at INTEGER NOT NULL,
    PRIMARY KEY (node_id, role)
) STRICT;

ALTER TABLE nodes_settings ADD COLUMN offline_grace_seconds INTEGER;
