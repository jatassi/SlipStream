-- +goose Up
-- +goose StatementBegin

-- Req 1.1.6: Each slot has its own independent monitored status per movie
-- Track per-movie slot assignments for status determination and monitoring
CREATE TABLE IF NOT EXISTS movie_slot_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_id INTEGER NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    slot_id INTEGER NOT NULL REFERENCES version_slots(id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES movie_files(id) ON DELETE SET NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(movie_id, slot_id)
);

-- Req 16.1.1: Each episode independently tracks which slots are filled
-- Track per-episode slot assignments
CREATE TABLE IF NOT EXISTS episode_slot_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    slot_id INTEGER NOT NULL REFERENCES version_slots(id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES episode_files(id) ON DELETE SET NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(episode_id, slot_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_movie_slot_assignments_movie ON movie_slot_assignments(movie_id);
CREATE INDEX IF NOT EXISTS idx_movie_slot_assignments_slot ON movie_slot_assignments(slot_id);
CREATE INDEX IF NOT EXISTS idx_movie_slot_assignments_file ON movie_slot_assignments(file_id);
CREATE INDEX IF NOT EXISTS idx_episode_slot_assignments_episode ON episode_slot_assignments(episode_id);
CREATE INDEX IF NOT EXISTS idx_episode_slot_assignments_slot ON episode_slot_assignments(slot_id);
CREATE INDEX IF NOT EXISTS idx_episode_slot_assignments_file ON episode_slot_assignments(file_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_episode_slot_assignments_file;
DROP INDEX IF EXISTS idx_episode_slot_assignments_slot;
DROP INDEX IF EXISTS idx_episode_slot_assignments_episode;
DROP INDEX IF EXISTS idx_movie_slot_assignments_file;
DROP INDEX IF EXISTS idx_movie_slot_assignments_slot;
DROP INDEX IF EXISTS idx_movie_slot_assignments_movie;
DROP TABLE IF EXISTS episode_slot_assignments;
DROP TABLE IF EXISTS movie_slot_assignments;

-- +goose StatementEnd
