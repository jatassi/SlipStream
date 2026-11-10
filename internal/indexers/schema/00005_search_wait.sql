-- +goose Up
-- S32 (Manual search and grab): "Wait for each Indexer up to", in seconds.
-- NULL is the default, 2 minutes.
ALTER TABLE indexers_settings ADD COLUMN search_wait_seconds INTEGER
    CHECK (search_wait_seconds IS NULL OR search_wait_seconds BETWEEN 10 AND 600);
