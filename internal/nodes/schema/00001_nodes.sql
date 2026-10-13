-- +goose Up
CREATE TABLE nodes_node (
    id BLOB PRIMARY KEY NOT NULL,
    public_key BLOB NOT NULL UNIQUE CHECK (length(public_key) = 32),
    name TEXT NOT NULL UNIQUE,
    paired_at INTEGER NOT NULL
) STRICT;

CREATE TABLE nodes_address (
    node_id BLOB NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    address TEXT NOT NULL,
    PRIMARY KEY (node_id, position)
) STRICT;

CREATE TABLE nodes_role (
    node_id BLOB NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    assigned_at INTEGER NOT NULL,
    PRIMARY KEY (node_id, role)
) STRICT;

CREATE TABLE nodes_join_code (
    secret_hash BLOB PRIMARY KEY NOT NULL CHECK (length(secret_hash) = 32),
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used_by BLOB REFERENCES nodes_node (id) ON DELETE CASCADE,
    used_at INTEGER,
    CHECK ((used_by IS NULL) = (used_at IS NULL))
) STRICT;

CREATE INDEX nodes_join_code_used_by ON nodes_join_code (used_by);

CREATE TABLE nodes_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    join_code_lifetime_seconds INTEGER NOT NULL CHECK (join_code_lifetime_seconds BETWEEN 600 AND 604800)
) STRICT;

INSERT INTO nodes_settings (id, join_code_lifetime_seconds) VALUES (1, 3600);
