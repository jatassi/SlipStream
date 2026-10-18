-- +goose Up
CREATE TABLE nodes_join_code_new (
    id BLOB PRIMARY KEY NOT NULL,
    secret_hash BLOB NOT NULL UNIQUE CHECK (length(secret_hash) = 32),
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used_by BLOB REFERENCES nodes_node (id) ON DELETE CASCADE,
    used_at INTEGER,
    dial_address TEXT,
    dial_at INTEGER,
    CHECK ((used_by IS NULL) = (used_at IS NULL)),
    CHECK ((dial_address IS NULL) = (dial_at IS NULL))
) STRICT;

INSERT INTO nodes_join_code_new (id, secret_hash, created_at, expires_at, used_by, used_at)
SELECT randomblob(16), secret_hash, created_at, expires_at, used_by, used_at FROM nodes_join_code;

DROP TABLE nodes_join_code;
ALTER TABLE nodes_join_code_new RENAME TO nodes_join_code;
CREATE INDEX nodes_join_code_used_by ON nodes_join_code (used_by);

CREATE TABLE nodes_dial (
    node_id BLOB NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    address TEXT NOT NULL,
    PRIMARY KEY (node_id, position)
) STRICT;
