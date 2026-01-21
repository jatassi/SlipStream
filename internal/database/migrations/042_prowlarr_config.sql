-- +goose Up
-- Prowlarr integration configuration table
-- Stores connection settings for using Prowlarr as the indexer backend instead of internal Cardigann indexers
CREATE TABLE prowlarr_config (
    id INTEGER PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    url TEXT NOT NULL DEFAULT '',
    api_key TEXT NOT NULL DEFAULT '',
    movie_categories TEXT NOT NULL DEFAULT '[2000,2010,2020,2030,2040,2045,2050,2060]',
    tv_categories TEXT NOT NULL DEFAULT '[5000,5010,5020,5030,5040,5045,5050,5060,5070,5080]',
    timeout INTEGER NOT NULL DEFAULT 90,
    skip_ssl_verify INTEGER NOT NULL DEFAULT 1,
    capabilities TEXT DEFAULT NULL,
    capabilities_updated_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert default row (singleton pattern - only one config row)
INSERT INTO prowlarr_config (id) VALUES (1);

-- +goose Down
DROP TABLE IF EXISTS prowlarr_config;
