-- +goose Up
-- Prowlarr kit@7ee91a1 (per-Indexer query timeout). Query timeout bounds
-- each request to the Indexer, in seconds. NULL is the Node's own request
-- timeout.
ALTER TABLE indexers_indexer ADD COLUMN query_timeout_seconds INTEGER
    CHECK (query_timeout_seconds IS NULL OR query_timeout_seconds BETWEEN 10 AND 300);
