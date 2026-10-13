-- +goose Up
CREATE TABLE role_watermark (
    node_id BLOB NOT NULL,
    epoch BLOB NOT NULL,
    seq INTEGER NOT NULL,
    PRIMARY KEY (node_id, epoch)
) STRICT;
