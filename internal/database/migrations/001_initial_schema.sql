-- +goose Up
-- +goose StatementBegin

-- System settings
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Authentication
CREATE TABLE IF NOT EXISTS auth (
    id INTEGER PRIMARY KEY,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Quality profiles
CREATE TABLE IF NOT EXISTS quality_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    cutoff INTEGER NOT NULL DEFAULT 0,
    items TEXT NOT NULL DEFAULT '[]', -- JSON array of quality items
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Root folders (media library paths)
CREATE TABLE IF NOT EXISTS root_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv')),
    free_space INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Movies
CREATE TABLE IF NOT EXISTS movies (
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

CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);
CREATE INDEX IF NOT EXISTS idx_movies_monitored ON movies(monitored);

-- Movie files
CREATE TABLE IF NOT EXISTS movie_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_id INTEGER NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    quality TEXT,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_movie_files_movie_id ON movie_files(movie_id);

-- TV Series
CREATE TABLE IF NOT EXISTS series (
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

CREATE INDEX IF NOT EXISTS idx_series_tvdb_id ON series(tvdb_id);
CREATE INDEX IF NOT EXISTS idx_series_monitored ON series(monitored);

-- Seasons
CREATE TABLE IF NOT EXISTS seasons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    UNIQUE(series_id, season_number)
);

CREATE INDEX IF NOT EXISTS idx_seasons_series_id ON seasons(series_id);

-- Episodes
CREATE TABLE IF NOT EXISTS episodes (
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

CREATE INDEX IF NOT EXISTS idx_episodes_series_id ON episodes(series_id);
CREATE INDEX IF NOT EXISTS idx_episodes_air_date ON episodes(air_date);

-- Episode files
CREATE TABLE IF NOT EXISTS episode_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    quality TEXT,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_episode_files_episode_id ON episode_files(episode_id);

-- Indexers
CREATE TABLE IF NOT EXISTS indexers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('torznab', 'newznab')),
    url TEXT NOT NULL,
    api_key TEXT,
    categories TEXT DEFAULT '[]', -- JSON array
    supports_movies INTEGER NOT NULL DEFAULT 1,
    supports_tv INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 50,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Download clients
CREATE TABLE IF NOT EXISTS download_clients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('qbittorrent', 'transmission', 'deluge', 'rtorrent', 'sabnzbd', 'nzbget')),
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password TEXT,
    use_ssl INTEGER NOT NULL DEFAULT 0,
    category TEXT,
    priority INTEGER NOT NULL DEFAULT 50,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Download queue
CREATE TABLE IF NOT EXISTS downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER REFERENCES download_clients(id),
    external_id TEXT, -- ID from download client
    title TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'episode')),
    media_id INTEGER NOT NULL, -- movie_id or episode_id
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'downloading', 'paused', 'completed', 'failed', 'importing')),
    progress REAL NOT NULL DEFAULT 0,
    size INTEGER NOT NULL DEFAULT 0,
    download_url TEXT,
    output_path TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
CREATE INDEX IF NOT EXISTS idx_downloads_media ON downloads(media_type, media_id);

-- History/Activity log
CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'episode')),
    media_id INTEGER NOT NULL,
    source TEXT,
    quality TEXT,
    data TEXT, -- JSON for additional data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_history_media ON history(media_type, media_id);
CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at);

-- Scheduled jobs
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    last_run DATETIME,
    next_run DATETIME,
    interval_seconds INTEGER NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1
);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS history;
DROP TABLE IF EXISTS downloads;
DROP TABLE IF EXISTS download_clients;
DROP TABLE IF EXISTS indexers;
DROP TABLE IF EXISTS episode_files;
DROP TABLE IF EXISTS episodes;
DROP TABLE IF EXISTS seasons;
DROP TABLE IF EXISTS series;
DROP TABLE IF EXISTS movie_files;
DROP TABLE IF EXISTS movies;
DROP TABLE IF EXISTS root_folders;
DROP TABLE IF EXISTS quality_profiles;
DROP TABLE IF EXISTS auth;
DROP TABLE IF EXISTS settings;
