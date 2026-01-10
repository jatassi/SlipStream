-- +goose Up
-- +goose StatementBegin

-- Movies: released if release_date is in the past
ALTER TABLE movies ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Episodes: released if air_date is in the past
ALTER TABLE episodes ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Seasons: released if ALL episodes in season have aired
ALTER TABLE seasons ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Series: released if ALL seasons are released
ALTER TABLE series ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Indexes for efficient availability queries
CREATE INDEX idx_movies_released ON movies(released);
CREATE INDEX idx_episodes_released ON episodes(released);
CREATE INDEX idx_seasons_released ON seasons(released);
CREATE INDEX idx_series_released ON series(released);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite requires table recreation to drop columns
-- Movies table
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
    status TEXT NOT NULL DEFAULT 'missing' CHECK (status IN ('missing', 'downloading', 'available')),
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    release_date DATE,
    digital_release_date DATE,
    physical_release_date DATE
);

INSERT INTO movies_new SELECT id, title, sort_title, year, tmdb_id, imdb_id, overview, runtime, path, root_folder_id, quality_profile_id, monitored, status, added_at, updated_at, release_date, digital_release_date, physical_release_date FROM movies;
DROP TABLE movies;
ALTER TABLE movies_new RENAME TO movies;

CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);
CREATE INDEX IF NOT EXISTS idx_movies_monitored ON movies(monitored);
CREATE INDEX IF NOT EXISTS idx_movies_release_date ON movies(release_date);
CREATE INDEX IF NOT EXISTS idx_movies_digital_release ON movies(digital_release_date);
CREATE INDEX IF NOT EXISTS idx_movies_physical_release ON movies(physical_release_date);

-- Episodes table
CREATE TABLE episodes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    episode_number INTEGER NOT NULL,
    title TEXT,
    overview TEXT,
    air_date DATE,
    monitored INTEGER NOT NULL DEFAULT 1,
    UNIQUE(series_id, season_number, episode_number)
);

INSERT INTO episodes_new SELECT id, series_id, season_number, episode_number, title, overview, air_date, monitored FROM episodes;
DROP TABLE episodes;
ALTER TABLE episodes_new RENAME TO episodes;

CREATE INDEX IF NOT EXISTS idx_episodes_series_id ON episodes(series_id);
CREATE INDEX IF NOT EXISTS idx_episodes_air_date ON episodes(air_date);

-- Seasons table
CREATE TABLE seasons_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    overview TEXT,
    poster_url TEXT,
    UNIQUE(series_id, season_number)
);

INSERT INTO seasons_new SELECT id, series_id, season_number, monitored, overview, poster_url FROM seasons;
DROP TABLE seasons;
ALTER TABLE seasons_new RENAME TO seasons;

CREATE INDEX IF NOT EXISTS idx_seasons_series_id ON seasons(series_id);

-- Series table
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
    status TEXT NOT NULL DEFAULT 'continuing' CHECK (status IN ('continuing', 'ended', 'upcoming')),
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    network TEXT
);

INSERT INTO series_new SELECT id, title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime, path, root_folder_id, quality_profile_id, monitored, season_folder, status, added_at, updated_at, network FROM series;
DROP TABLE series;
ALTER TABLE series_new RENAME TO series;

CREATE INDEX IF NOT EXISTS idx_series_tvdb_id ON series(tvdb_id);
CREATE INDEX IF NOT EXISTS idx_series_monitored ON series(monitored);
CREATE INDEX IF NOT EXISTS idx_series_network ON series(network);

-- +goose StatementEnd
