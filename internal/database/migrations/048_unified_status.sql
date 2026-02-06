-- +goose Up
-- +goose StatementBegin

-- =============================================================================
-- Unified Media Status Consolidation
-- Replaces fragmented status/released/availabilityStatus fields with a single
-- unified status enum at the leaf level (movies, episodes, slot assignments).
-- =============================================================================

-- ---- MOVIES: Recreate with new status constraint, add new columns, drop old ones ----

CREATE TABLE movies_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    sort_title TEXT NOT NULL,
    year INTEGER,
    tmdb_id INTEGER UNIQUE,
    imdb_id TEXT,
    overview TEXT,
    runtime INTEGER,
    path TEXT,
    root_folder_id INTEGER REFERENCES root_folders(id),
    quality_profile_id INTEGER REFERENCES quality_profiles(id),
    monitored INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'missing'
        CHECK (status IN ('unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available')),
    active_download_id TEXT,
    status_message TEXT,
    release_date DATE,
    physical_release_date DATE,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO movies_new (id, title, sort_title, year, tmdb_id, imdb_id, overview, runtime, path,
    root_folder_id, quality_profile_id, monitored, status, release_date, physical_release_date,
    added_at, updated_at)
SELECT id, title, sort_title, year, tmdb_id, imdb_id, overview, runtime, path,
    root_folder_id, quality_profile_id, monitored,
    CASE
        WHEN status = 'available' THEN 'available'
        WHEN status = 'downloading' THEN 'downloading'
        WHEN status = 'missing' AND released = 1 THEN 'missing'
        WHEN status = 'missing' AND released = 0 THEN 'unreleased'
        ELSE 'missing'
    END,
    release_date, physical_release_date,
    added_at, updated_at
FROM movies;

DROP TABLE movies;
ALTER TABLE movies_new RENAME TO movies;

CREATE INDEX idx_movies_tmdb_id ON movies(tmdb_id);
CREATE INDEX idx_movies_monitored ON movies(monitored);
CREATE INDEX idx_movies_status ON movies(status);
CREATE INDEX idx_movies_release_date ON movies(release_date);
CREATE INDEX idx_movies_physical_release ON movies(physical_release_date);

-- ---- EPISODES: Add status columns, drop released ----

CREATE TABLE episodes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    episode_number INTEGER NOT NULL,
    title TEXT,
    overview TEXT,
    air_date DATETIME,
    monitored INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'unreleased'
        CHECK (status IN ('unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available')),
    active_download_id TEXT,
    status_message TEXT,
    UNIQUE(series_id, season_number, episode_number)
);

INSERT INTO episodes_new (id, series_id, season_number, episode_number, title, overview, air_date,
    monitored, status)
SELECT id, series_id, season_number, episode_number, title, overview, air_date,
    monitored,
    CASE
        WHEN released = 0 THEN 'unreleased'
        WHEN released = 1 AND id IN (SELECT episode_id FROM episode_files) THEN 'available'
        WHEN released = 1 THEN 'missing'
        ELSE 'unreleased'
    END
FROM episodes;

DROP TABLE episodes;
ALTER TABLE episodes_new RENAME TO episodes;

CREATE INDEX idx_episodes_series_id ON episodes(series_id);
CREATE INDEX idx_episodes_air_date ON episodes(air_date);
CREATE INDEX idx_episodes_status ON episodes(status);

-- ---- SERIES: Rename status to production_status, drop released and availability_status ----

CREATE TABLE series_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    sort_title TEXT NOT NULL,
    year INTEGER,
    tvdb_id INTEGER UNIQUE,
    tmdb_id INTEGER,
    imdb_id TEXT,
    overview TEXT,
    runtime INTEGER,
    path TEXT,
    root_folder_id INTEGER REFERENCES root_folders(id),
    quality_profile_id INTEGER REFERENCES quality_profiles(id),
    monitored INTEGER NOT NULL DEFAULT 1,
    season_folder INTEGER NOT NULL DEFAULT 1,
    production_status TEXT NOT NULL DEFAULT 'continuing'
        CHECK (production_status IN ('continuing', 'ended', 'upcoming')),
    network TEXT,
    format_type TEXT DEFAULT NULL
        CHECK(format_type IS NULL OR format_type IN ('standard', 'daily', 'anime')),
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO series_new (id, title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime,
    path, root_folder_id, quality_profile_id, monitored, season_folder, production_status, network,
    format_type, added_at, updated_at)
SELECT id, title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime,
    path, root_folder_id, quality_profile_id, monitored, season_folder, status, network,
    format_type, added_at, updated_at
FROM series;

DROP TABLE series;
ALTER TABLE series_new RENAME TO series;

CREATE INDEX idx_series_tvdb_id ON series(tvdb_id);
CREATE INDEX idx_series_tmdb_id ON series(tmdb_id);
CREATE INDEX idx_series_monitored ON series(monitored);
CREATE INDEX idx_series_network ON series(network);

-- ---- SEASONS: Drop released column ----

CREATE TABLE seasons_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    overview TEXT,
    poster_url TEXT,
    UNIQUE(series_id, season_number)
);

INSERT INTO seasons_new (id, series_id, season_number, monitored, overview, poster_url)
SELECT id, series_id, season_number, monitored, overview, poster_url
FROM seasons;

DROP TABLE seasons;
ALTER TABLE seasons_new RENAME TO seasons;

CREATE INDEX idx_seasons_series_id ON seasons(series_id);

-- ---- SLOT ASSIGNMENTS: Add status fields ----

ALTER TABLE movie_slot_assignments ADD COLUMN status TEXT NOT NULL DEFAULT 'missing'
    CHECK (status IN ('unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available'));
ALTER TABLE movie_slot_assignments ADD COLUMN active_download_id TEXT;
ALTER TABLE movie_slot_assignments ADD COLUMN status_message TEXT;

ALTER TABLE episode_slot_assignments ADD COLUMN status TEXT NOT NULL DEFAULT 'missing'
    CHECK (status IN ('unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available'));
ALTER TABLE episode_slot_assignments ADD COLUMN active_download_id TEXT;
ALTER TABLE episode_slot_assignments ADD COLUMN status_message TEXT;

-- Initialize slot assignment statuses from media status + file presence
UPDATE movie_slot_assignments SET status = CASE
    WHEN (SELECT m.status FROM movies m WHERE m.id = movie_slot_assignments.movie_id) = 'unreleased'
        THEN 'unreleased'
    WHEN file_id IS NOT NULL THEN 'available'
    ELSE 'missing'
END;

UPDATE episode_slot_assignments SET status = CASE
    WHEN (SELECT e.status FROM episodes e WHERE e.id = episode_slot_assignments.episode_id) = 'unreleased'
        THEN 'unreleased'
    WHEN file_id IS NOT NULL THEN 'available'
    ELSE 'missing'
END;

CREATE INDEX idx_movie_slot_assignments_status ON movie_slot_assignments(status);
CREATE INDEX idx_episode_slot_assignments_status ON episode_slot_assignments(status);

-- ---- AUTOSEARCH_STATUS: Add search_type for separate missing/upgrade backoff ----

-- SQLite requires table recreation to change UNIQUE constraint
CREATE TABLE autosearch_status_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_type TEXT NOT NULL CHECK (item_type IN ('movie', 'episode')),
    item_id INTEGER NOT NULL,
    search_type TEXT NOT NULL DEFAULT 'missing' CHECK (search_type IN ('missing', 'upgrade')),
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_searched_at DATETIME,
    last_meta_change_at DATETIME,
    UNIQUE(item_type, item_id, search_type)
);

INSERT INTO autosearch_status_new (id, item_type, item_id, search_type, failure_count,
    last_searched_at, last_meta_change_at)
SELECT id, item_type, item_id, 'missing', failure_count, last_searched_at, last_meta_change_at
FROM autosearch_status;

DROP TABLE autosearch_status;
ALTER TABLE autosearch_status_new RENAME TO autosearch_status;

CREATE INDEX idx_autosearch_status_item ON autosearch_status(item_type, item_id, search_type);
CREATE INDEX idx_autosearch_status_failure_count ON autosearch_status(failure_count);

-- ---- REQUESTS: Add 'failed' to status constraint ----

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

-- Reverse migration: restore old schema
-- Movies: restore old status constraint and columns
CREATE TABLE movies_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    sort_title TEXT NOT NULL,
    year INTEGER,
    tmdb_id INTEGER UNIQUE,
    imdb_id TEXT,
    overview TEXT,
    runtime INTEGER,
    path TEXT,
    root_folder_id INTEGER REFERENCES root_folders(id),
    quality_profile_id INTEGER REFERENCES quality_profiles(id),
    monitored INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'missing' CHECK (status IN ('missing', 'downloading', 'available')),
    release_date DATE,
    digital_release_date DATE,
    physical_release_date DATE,
    released INTEGER NOT NULL DEFAULT 0,
    availability_status TEXT NOT NULL DEFAULT 'Unreleased',
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO movies_old (id, title, sort_title, year, tmdb_id, imdb_id, overview, runtime, path,
    root_folder_id, quality_profile_id, monitored, status, release_date, physical_release_date,
    released, availability_status, added_at, updated_at)
SELECT id, title, sort_title, year, tmdb_id, imdb_id, overview, runtime, path,
    root_folder_id, quality_profile_id, monitored,
    CASE
        WHEN status IN ('available', 'upgradable') THEN 'available'
        WHEN status = 'downloading' THEN 'downloading'
        ELSE 'missing'
    END,
    release_date, physical_release_date,
    CASE WHEN status != 'unreleased' THEN 1 ELSE 0 END,
    CASE WHEN status IN ('available', 'upgradable') THEN 'Available' ELSE 'Unreleased' END,
    added_at, updated_at
FROM movies;

DROP TABLE movies;
ALTER TABLE movies_old RENAME TO movies;

CREATE INDEX idx_movies_tmdb_id ON movies(tmdb_id);
CREATE INDEX idx_movies_monitored ON movies(monitored);
CREATE INDEX idx_movies_released ON movies(released);
CREATE INDEX idx_movies_release_date ON movies(release_date);
CREATE INDEX idx_movies_digital_release ON movies(digital_release_date);
CREATE INDEX idx_movies_physical_release ON movies(physical_release_date);

-- Episodes: restore old schema
CREATE TABLE episodes_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    episode_number INTEGER NOT NULL,
    title TEXT,
    overview TEXT,
    air_date DATE,
    monitored INTEGER NOT NULL DEFAULT 1,
    released INTEGER NOT NULL DEFAULT 0,
    UNIQUE(series_id, season_number, episode_number)
);

INSERT INTO episodes_old (id, series_id, season_number, episode_number, title, overview, air_date,
    monitored, released)
SELECT id, series_id, season_number, episode_number, title, overview, air_date,
    monitored,
    CASE WHEN status != 'unreleased' THEN 1 ELSE 0 END
FROM episodes;

DROP TABLE episodes;
ALTER TABLE episodes_old RENAME TO episodes;

CREATE INDEX idx_episodes_series_id ON episodes(series_id);
CREATE INDEX idx_episodes_air_date ON episodes(air_date);
CREATE INDEX idx_episodes_released ON episodes(released);

-- Series: restore old schema
CREATE TABLE series_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    sort_title TEXT NOT NULL,
    year INTEGER,
    tvdb_id INTEGER UNIQUE,
    tmdb_id INTEGER,
    imdb_id TEXT,
    overview TEXT,
    runtime INTEGER,
    path TEXT,
    root_folder_id INTEGER REFERENCES root_folders(id),
    quality_profile_id INTEGER REFERENCES quality_profiles(id),
    monitored INTEGER NOT NULL DEFAULT 1,
    season_folder INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'continuing' CHECK (status IN ('continuing', 'ended', 'upcoming')),
    network TEXT,
    released INTEGER NOT NULL DEFAULT 0,
    availability_status TEXT NOT NULL DEFAULT 'Unreleased',
    format_type TEXT DEFAULT NULL CHECK(format_type IS NULL OR format_type IN ('standard', 'daily', 'anime')),
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO series_old (id, title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime,
    path, root_folder_id, quality_profile_id, monitored, season_folder, status, network,
    format_type, added_at, updated_at)
SELECT id, title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime,
    path, root_folder_id, quality_profile_id, monitored, season_folder, production_status, network,
    format_type, added_at, updated_at
FROM series;

DROP TABLE series;
ALTER TABLE series_old RENAME TO series;

CREATE INDEX idx_series_tvdb_id ON series(tvdb_id);
CREATE INDEX idx_series_monitored ON series(monitored);
CREATE INDEX idx_series_network ON series(network);
CREATE INDEX idx_series_released ON series(released);

-- Seasons: restore released column
CREATE TABLE seasons_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    overview TEXT,
    poster_url TEXT,
    released INTEGER NOT NULL DEFAULT 0,
    UNIQUE(series_id, season_number)
);

INSERT INTO seasons_old (id, series_id, season_number, monitored, overview, poster_url)
SELECT id, series_id, season_number, monitored, overview, poster_url
FROM seasons;

DROP TABLE seasons;
ALTER TABLE seasons_old RENAME TO seasons;

CREATE INDEX idx_seasons_series_id ON seasons(series_id);
CREATE INDEX idx_seasons_released ON seasons(released);

-- Autosearch: restore old schema without search_type
CREATE TABLE autosearch_status_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_type TEXT NOT NULL CHECK (item_type IN ('movie', 'episode')),
    item_id INTEGER NOT NULL,
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_searched_at DATETIME,
    last_meta_change_at DATETIME,
    UNIQUE(item_type, item_id)
);

INSERT INTO autosearch_status_old (item_type, item_id, failure_count, last_searched_at, last_meta_change_at)
SELECT item_type, item_id, MAX(failure_count), MAX(last_searched_at), MAX(last_meta_change_at)
FROM autosearch_status
GROUP BY item_type, item_id;

DROP TABLE autosearch_status;
ALTER TABLE autosearch_status_old RENAME TO autosearch_status;

CREATE INDEX idx_autosearch_status_item ON autosearch_status(item_type, item_id);
CREATE INDEX idx_autosearch_status_failure_count ON autosearch_status(failure_count);

-- Requests: restore old constraint without 'failed'
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
        CHECK (status IN ('pending', 'approved', 'denied', 'downloading', 'available')),
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
    episode_number,
    CASE WHEN status = 'failed' THEN 'downloading' ELSE status END,
    monitor_type, denied_reason, approved_at, approved_by, media_id,
    target_slot_id, poster_url, requested_seasons, created_at, updated_at
FROM requests;

DROP TABLE requests;
ALTER TABLE requests_old RENAME TO requests;

CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_tmdb_id ON requests(tmdb_id);
CREATE INDEX idx_requests_tvdb_id ON requests(tvdb_id);
CREATE INDEX idx_requests_media_type ON requests(media_type);

-- Slot assignments: drop new columns (requires table recreation)
CREATE TABLE movie_slot_assignments_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_id INTEGER NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    slot_id INTEGER NOT NULL REFERENCES version_slots(id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES movie_files(id) ON DELETE SET NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(movie_id, slot_id)
);

INSERT INTO movie_slot_assignments_old (id, movie_id, slot_id, file_id, monitored, created_at, updated_at)
SELECT id, movie_id, slot_id, file_id, monitored, created_at, updated_at
FROM movie_slot_assignments;

DROP TABLE movie_slot_assignments;
ALTER TABLE movie_slot_assignments_old RENAME TO movie_slot_assignments;

CREATE INDEX idx_movie_slot_assignments_movie ON movie_slot_assignments(movie_id);
CREATE INDEX idx_movie_slot_assignments_slot ON movie_slot_assignments(slot_id);
CREATE INDEX idx_movie_slot_assignments_file ON movie_slot_assignments(file_id);

CREATE TABLE episode_slot_assignments_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    slot_id INTEGER NOT NULL REFERENCES version_slots(id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES episode_files(id) ON DELETE SET NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(episode_id, slot_id)
);

INSERT INTO episode_slot_assignments_old (id, episode_id, slot_id, file_id, monitored, created_at, updated_at)
SELECT id, episode_id, slot_id, file_id, monitored, created_at, updated_at
FROM episode_slot_assignments;

DROP TABLE episode_slot_assignments;
ALTER TABLE episode_slot_assignments_old RENAME TO episode_slot_assignments;

CREATE INDEX idx_episode_slot_assignments_episode ON episode_slot_assignments(episode_id);
CREATE INDEX idx_episode_slot_assignments_slot ON episode_slot_assignments(slot_id);
CREATE INDEX idx_episode_slot_assignments_file ON episode_slot_assignments(file_id);

-- +goose StatementEnd
