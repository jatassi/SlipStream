-- +goose Up
-- Junction table for tracking individual files within a download (especially season packs)
-- Links download_mappings to specific episodes/movies with per-file status tracking
CREATE TABLE IF NOT EXISTS queue_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    download_mapping_id INTEGER NOT NULL REFERENCES download_mappings(id) ON DELETE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    file_path TEXT,
    file_status TEXT NOT NULL DEFAULT 'pending'
        CHECK(file_status IN ('pending', 'downloading', 'ready', 'importing', 'imported', 'failed')),
    error_message TEXT,
    import_attempts INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(download_mapping_id, episode_id),
    UNIQUE(download_mapping_id, movie_id)
);

CREATE INDEX idx_queue_media_status ON queue_media(file_status);
CREATE INDEX idx_queue_media_episode ON queue_media(episode_id);
CREATE INDEX idx_queue_media_movie ON queue_media(movie_id);
CREATE INDEX idx_queue_media_download_mapping ON queue_media(download_mapping_id);

-- +goose Down
DROP INDEX IF EXISTS idx_queue_media_download_mapping;
DROP INDEX IF EXISTS idx_queue_media_movie;
DROP INDEX IF EXISTS idx_queue_media_episode;
DROP INDEX IF EXISTS idx_queue_media_status;
DROP TABLE IF EXISTS queue_media;
