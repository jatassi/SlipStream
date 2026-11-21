-- +goose Up
-- Old Controller keys this Controller replaced at a restore, each with the
-- proof the restore minted from the archive's fence secret.
CREATE TABLE nodes_superseded (
    public_key BLOB PRIMARY KEY NOT NULL CHECK (length(public_key) = 32),
    node_id BLOB NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    proof BLOB NOT NULL CHECK (length(proof) = 36),
    superseded_at INTEGER NOT NULL
) STRICT;

-- On an old Controller: what a pinned Worker said replaced it, and how far
-- this Controller is fenced.
CREATE TABLE nodes_fence (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    state TEXT NOT NULL CHECK (state IN ('unproven', 'proven', 'stopped', 'retired')),
    by_key BLOB CHECK (by_key IS NULL OR length(by_key) = 32),
    by_node BLOB REFERENCES nodes_node (id) ON DELETE SET NULL,
    addresses TEXT NOT NULL CHECK (json_valid(addresses)),
    fenced_at INTEGER NOT NULL,
    CHECK (state = 'retired' OR by_key IS NOT NULL)
) STRICT;
