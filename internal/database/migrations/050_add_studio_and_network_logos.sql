-- +goose Up
ALTER TABLE movies ADD COLUMN studio TEXT;
ALTER TABLE movies ADD COLUMN tvdb_id INTEGER;
ALTER TABLE series ADD COLUMN network_logo_url TEXT;

CREATE TABLE network_logos (
    name TEXT PRIMARY KEY,
    logo_url TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS network_logos;
