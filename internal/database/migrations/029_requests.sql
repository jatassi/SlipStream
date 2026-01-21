-- +goose Up
-- Media requests from portal users
CREATE TABLE requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'series', 'season', 'episode')),
    tmdb_id INTEGER,
    tvdb_id INTEGER,
    title TEXT NOT NULL,
    year INTEGER,
    season_number INTEGER,
    episode_number INTEGER,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'downloading', 'available')),
    monitor_type TEXT,
    denied_reason TEXT,
    approved_at DATETIME,
    approved_by INTEGER REFERENCES portal_users(id) ON DELETE SET NULL,
    media_id INTEGER,
    target_slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_tmdb_id ON requests(tmdb_id);
CREATE INDEX idx_requests_tvdb_id ON requests(tvdb_id);
CREATE INDEX idx_requests_media_type ON requests(media_type);

-- +goose Down
DROP INDEX IF EXISTS idx_requests_media_type;
DROP INDEX IF EXISTS idx_requests_tvdb_id;
DROP INDEX IF EXISTS idx_requests_tmdb_id;
DROP INDEX IF EXISTS idx_requests_status;
DROP INDEX IF EXISTS idx_requests_user_id;
DROP TABLE IF EXISTS requests;
