-- +goose Up
CREATE TABLE role_rejected (
    node_id BLOB NOT NULL,
    epoch BLOB NOT NULL,
    seq INTEGER NOT NULL,
    scope TEXT NOT NULL,
    reason TEXT NOT NULL,
    rejected_at INTEGER NOT NULL,
    PRIMARY KEY (node_id, epoch, seq)
) STRICT;
