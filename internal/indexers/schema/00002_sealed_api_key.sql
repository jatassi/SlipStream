-- +goose Up
-- Migrations run before library.key is open, so they can't seal. v2 has no
-- installs whose Indexers need keeping: every key is dropped and an Admin
-- enters it again.
CREATE TABLE indexers_indexer_sealed (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE,
    protocol TEXT NOT NULL CHECK (protocol = 'torznab'),
    url TEXT NOT NULL,
    api_key_sealed BLOB,
    home_node_id BLOB NOT NULL,
    caps TEXT NOT NULL,
    created_at INTEGER NOT NULL
) STRICT;

INSERT INTO indexers_indexer_sealed (id, name, protocol, url, api_key_sealed, home_node_id, caps, created_at)
SELECT id, name, protocol, url, NULL, home_node_id, caps, created_at FROM indexers_indexer;

DROP TABLE indexers_indexer;

ALTER TABLE indexers_indexer_sealed RENAME TO indexers_indexer;
