-- +goose Up
-- Add 'season' to history media_type CHECK constraint.
-- SQLite cannot ALTER constraints, so we must rebuild the table.

CREATE TABLE history_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'episode', 'season')),
    media_id INTEGER NOT NULL,
    source TEXT,
    quality TEXT,
    data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO history_new (id, event_type, media_type, media_id, source, quality, data, created_at)
SELECT id, event_type, media_type, media_id, source, quality, data, created_at
FROM history;

DROP TABLE history;
ALTER TABLE history_new RENAME TO history;

CREATE INDEX idx_history_media ON history(media_type, media_id);
CREATE INDEX idx_history_created_at ON history(created_at);

-- +goose Down
CREATE TABLE history_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'episode')),
    media_id INTEGER NOT NULL,
    source TEXT,
    quality TEXT,
    data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO history_old (id, event_type, media_type, media_id, source, quality, data, created_at)
SELECT id, event_type, media_type, media_id, source, quality, data, created_at
FROM history
WHERE media_type IN ('movie', 'episode');

DROP TABLE history;
ALTER TABLE history_old RENAME TO history;

CREATE INDEX idx_history_media ON history(media_type, media_id);
CREATE INDEX idx_history_created_at ON history(created_at);
