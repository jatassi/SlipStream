-- +goose Up
-- +goose StatementBegin

-- Req 2.1.1-2.1.4: Add profile-level attribute settings for HDR, video codec, audio codec, and audio channels
-- Each column stores JSON with structure: {"mode": "any"|"preferred"|"required", "values": [...]}
ALTER TABLE quality_profiles ADD COLUMN hdr_settings TEXT NOT NULL DEFAULT '{"mode":"any","values":[]}';
ALTER TABLE quality_profiles ADD COLUMN video_codec_settings TEXT NOT NULL DEFAULT '{"mode":"any","values":[]}';
ALTER TABLE quality_profiles ADD COLUMN audio_codec_settings TEXT NOT NULL DEFAULT '{"mode":"any","values":[]}';
ALTER TABLE quality_profiles ADD COLUMN audio_channel_settings TEXT NOT NULL DEFAULT '{"mode":"any","values":[]}';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
CREATE TABLE quality_profiles_backup (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    cutoff INTEGER NOT NULL DEFAULT 0,
    items TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO quality_profiles_backup (id, name, cutoff, items, created_at, updated_at)
SELECT id, name, cutoff, items, created_at, updated_at FROM quality_profiles;

DROP TABLE quality_profiles;

ALTER TABLE quality_profiles_backup RENAME TO quality_profiles;

-- +goose StatementEnd
