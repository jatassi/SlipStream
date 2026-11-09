-- +goose Up
-- The longest absolute path each Storage Node writes, in bytes, as it last
-- reported it. Kept while the Node is offline, so a plan made then measures
-- paths against the Node's own limit.
CREATE TABLE roots_limit (
    node_id BLOB PRIMARY KEY NOT NULL,
    path_limit INTEGER NOT NULL CHECK (path_limit > 0)
) STRICT;
