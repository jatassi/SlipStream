-- +goose Up
-- +goose StatementBegin

-- Add 'searching' to request status constraint
CREATE TABLE requests_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'series', 'season', 'episode')),
    tmdb_id INTEGER,
    tvdb_id INTEGER,
    title TEXT NOT NULL,
    year INTEGER,
    season_number INTEGER,
    episode_number INTEGER,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'denied', 'searching', 'downloading', 'failed', 'available')),
    monitor_type TEXT,
    denied_reason TEXT,
    approved_at DATETIME,
    approved_by INTEGER REFERENCES portal_users(id) ON DELETE SET NULL,
    media_id INTEGER,
    target_slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL,
    poster_url TEXT,
    requested_seasons TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO requests_new (id, user_id, media_type, tmdb_id, tvdb_id, title, year, season_number,
    episode_number, status, monitor_type, denied_reason, approved_at, approved_by, media_id,
    target_slot_id, poster_url, requested_seasons, created_at, updated_at)
SELECT id, user_id, media_type, tmdb_id, tvdb_id, title, year, season_number,
    episode_number, status, monitor_type, denied_reason, approved_at, approved_by, media_id,
    target_slot_id, poster_url, requested_seasons, created_at, updated_at
FROM requests;

DROP TABLE requests;
ALTER TABLE requests_new RENAME TO requests;

CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_tmdb_id ON requests(tmdb_id);
CREATE INDEX idx_requests_tvdb_id ON requests(tvdb_id);
CREATE INDEX idx_requests_media_type ON requests(media_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revert 'searching' rows to 'approved' and remove 'searching' from constraint
UPDATE requests SET status = 'approved' WHERE status = 'searching';

CREATE TABLE requests_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'series', 'season', 'episode')),
    tmdb_id INTEGER,
    tvdb_id INTEGER,
    title TEXT NOT NULL,
    year INTEGER,
    season_number INTEGER,
    episode_number INTEGER,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'denied', 'downloading', 'failed', 'available')),
    monitor_type TEXT,
    denied_reason TEXT,
    approved_at DATETIME,
    approved_by INTEGER REFERENCES portal_users(id) ON DELETE SET NULL,
    media_id INTEGER,
    target_slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL,
    poster_url TEXT,
    requested_seasons TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO requests_old (id, user_id, media_type, tmdb_id, tvdb_id, title, year, season_number,
    episode_number, status, monitor_type, denied_reason, approved_at, approved_by, media_id,
    target_slot_id, poster_url, requested_seasons, created_at, updated_at)
SELECT id, user_id, media_type, tmdb_id, tvdb_id, title, year, season_number,
    episode_number, status, monitor_type, denied_reason, approved_at, approved_by, media_id,
    target_slot_id, poster_url, requested_seasons, created_at, updated_at
FROM requests;

DROP TABLE requests;
ALTER TABLE requests_old RENAME TO requests;

CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_tmdb_id ON requests(tmdb_id);
CREATE INDEX idx_requests_tvdb_id ON requests(tvdb_id);
CREATE INDEX idx_requests_media_type ON requests(media_type);

-- +goose StatementEnd
