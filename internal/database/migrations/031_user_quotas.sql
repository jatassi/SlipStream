-- +goose Up
-- User quota tracking and overrides
CREATE TABLE user_quotas (
    user_id INTEGER PRIMARY KEY REFERENCES portal_users(id) ON DELETE CASCADE,
    movies_limit INTEGER,
    seasons_limit INTEGER,
    episodes_limit INTEGER,
    movies_used INTEGER NOT NULL DEFAULT 0,
    seasons_used INTEGER NOT NULL DEFAULT 0,
    episodes_used INTEGER NOT NULL DEFAULT 0,
    period_start DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS user_quotas;
