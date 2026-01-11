-- +goose Up
-- download_mappings tracks which library items are being downloaded
-- This allows us to show download status on library items
CREATE TABLE IF NOT EXISTS download_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER NOT NULL REFERENCES download_clients(id) ON DELETE CASCADE,
    download_id TEXT NOT NULL,
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    series_id INTEGER REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    is_season_pack INTEGER NOT NULL DEFAULT 0,
    is_complete_series INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(client_id, download_id)
);

CREATE INDEX idx_download_mappings_movie ON download_mappings(movie_id) WHERE movie_id IS NOT NULL;
CREATE INDEX idx_download_mappings_series ON download_mappings(series_id) WHERE series_id IS NOT NULL;
CREATE INDEX idx_download_mappings_episode ON download_mappings(episode_id) WHERE episode_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_download_mappings_episode;
DROP INDEX IF EXISTS idx_download_mappings_series;
DROP INDEX IF EXISTS idx_download_mappings_movie;
DROP TABLE IF EXISTS download_mappings;
