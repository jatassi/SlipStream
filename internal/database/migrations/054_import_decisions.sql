-- +goose Up
ALTER TABLE download_mappings ADD COLUMN source TEXT NOT NULL DEFAULT 'auto-search';

CREATE TABLE import_decisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_path TEXT NOT NULL UNIQUE,
    decision TEXT NOT NULL,
    media_type TEXT NOT NULL,
    media_id INTEGER NOT NULL,
    slot_id INTEGER,
    candidate_quality_id INTEGER,
    existing_quality_id INTEGER,
    existing_file_id INTEGER,
    quality_profile_id INTEGER,
    reason TEXT,
    evaluated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_import_decisions_source ON import_decisions(source_path);
CREATE INDEX idx_import_decisions_media ON import_decisions(media_type, media_id);
CREATE INDEX idx_import_decisions_profile ON import_decisions(quality_profile_id);
CREATE INDEX idx_import_decisions_existing_file ON import_decisions(existing_file_id);

-- +goose Down
DROP INDEX IF EXISTS idx_import_decisions_existing_file;
DROP INDEX IF EXISTS idx_import_decisions_profile;
DROP INDEX IF EXISTS idx_import_decisions_media;
DROP INDEX IF EXISTS idx_import_decisions_source;
DROP TABLE IF EXISTS import_decisions;
