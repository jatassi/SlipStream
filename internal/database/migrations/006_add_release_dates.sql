-- +goose Up
-- +goose StatementBegin

-- Add release date columns to movies table
ALTER TABLE movies ADD COLUMN release_date DATE;
ALTER TABLE movies ADD COLUMN digital_release_date DATE;
ALTER TABLE movies ADD COLUMN physical_release_date DATE;

-- Add network column to series table
ALTER TABLE series ADD COLUMN network TEXT;

-- Create indexes for calendar queries
CREATE INDEX IF NOT EXISTS idx_movies_release_date ON movies(release_date);
CREATE INDEX IF NOT EXISTS idx_movies_digital_release ON movies(digital_release_date);
CREATE INDEX IF NOT EXISTS idx_movies_physical_release ON movies(physical_release_date);
CREATE INDEX IF NOT EXISTS idx_series_network ON series(network);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN directly, so we recreate the movies table
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
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO movies_new (id, title, sort_title, year, tmdb_id, imdb_id, overview, runtime, path, root_folder_id, quality_profile_id, monitored, status, added_at, updated_at)
SELECT id, title, sort_title, year, tmdb_id, imdb_id, overview, runtime, path, root_folder_id, quality_profile_id, monitored, status, added_at, updated_at FROM movies;

DROP TABLE movies;
ALTER TABLE movies_new RENAME TO movies;

CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);
CREATE INDEX IF NOT EXISTS idx_movies_monitored ON movies(monitored);

-- Recreate the series table without network column
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
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO series_new (id, title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime, path, root_folder_id, quality_profile_id, monitored, season_folder, status, added_at, updated_at)
SELECT id, title, sort_title, year, tvdb_id, tmdb_id, imdb_id, overview, runtime, path, root_folder_id, quality_profile_id, monitored, season_folder, status, added_at, updated_at FROM series;

DROP TABLE series;
ALTER TABLE series_new RENAME TO series;

CREATE INDEX IF NOT EXISTS idx_series_tvdb_id ON series(tvdb_id);
CREATE INDEX IF NOT EXISTS idx_series_monitored ON series(monitored);

-- +goose StatementEnd
