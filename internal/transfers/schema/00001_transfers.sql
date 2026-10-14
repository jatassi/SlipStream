-- +goose Up
CREATE TABLE transfers_transfer (
    id BLOB PRIMARY KEY NOT NULL,
    source_node BLOB NOT NULL,
    info_hash BLOB NOT NULL CHECK (length(info_hash) = 20),
    file_index INTEGER NOT NULL CHECK (file_index >= 0),
    size INTEGER CHECK (size >= 0),
    chunk_size INTEGER CHECK (chunk_size > 0),
    chunk_hashes BLOB,
    staging_path TEXT NOT NULL CHECK (staging_path <> ''),
    created_at INTEGER NOT NULL,
    CHECK ((size IS NULL) = (chunk_size IS NULL) AND (size IS NULL) = (chunk_hashes IS NULL))
) STRICT;

CREATE TABLE transfers_chunk (
    transfer_id BLOB NOT NULL REFERENCES transfers_transfer (id) ON DELETE CASCADE,
    idx INTEGER NOT NULL CHECK (idx >= 0),
    PRIMARY KEY (transfer_id, idx)
) STRICT;
