-- +goose Up
-- Add "series" to autosearch_status item_type CHECK constraint for season pack backoff tracking.
-- Previously, season pack backoff was tracked under item_type='episode' with the series ID,
-- causing collisions with actual episode records and a check/write ID mismatch.

CREATE TABLE autosearch_status_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_type TEXT NOT NULL CHECK (item_type IN ('movie', 'episode', 'series')),
    item_id INTEGER NOT NULL,
    search_type TEXT NOT NULL DEFAULT 'missing' CHECK (search_type IN ('missing', 'upgrade')),
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_searched_at DATETIME,
    last_meta_change_at DATETIME,
    UNIQUE(item_type, item_id, search_type)
);

INSERT INTO autosearch_status_new (id, item_type, item_id, search_type, failure_count, last_searched_at, last_meta_change_at)
SELECT id, item_type, item_id, search_type, failure_count, last_searched_at, last_meta_change_at
FROM autosearch_status;

DROP TABLE autosearch_status;

ALTER TABLE autosearch_status_new RENAME TO autosearch_status;

CREATE INDEX idx_autosearch_status_item ON autosearch_status(item_type, item_id, search_type);
CREATE INDEX idx_autosearch_status_failure_count ON autosearch_status(failure_count);

-- +goose Down
CREATE TABLE autosearch_status_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_type TEXT NOT NULL CHECK (item_type IN ('movie', 'episode')),
    item_id INTEGER NOT NULL,
    search_type TEXT NOT NULL DEFAULT 'missing' CHECK (search_type IN ('missing', 'upgrade')),
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_searched_at DATETIME,
    last_meta_change_at DATETIME,
    UNIQUE(item_type, item_id, search_type)
);

INSERT INTO autosearch_status_old (id, item_type, item_id, search_type, failure_count, last_searched_at, last_meta_change_at)
SELECT id, item_type, item_id, search_type, failure_count, last_searched_at, last_meta_change_at
FROM autosearch_status
WHERE item_type IN ('movie', 'episode');

DROP TABLE autosearch_status;

ALTER TABLE autosearch_status_old RENAME TO autosearch_status;

CREATE INDEX idx_autosearch_status_item ON autosearch_status(item_type, item_id, search_type);
CREATE INDEX idx_autosearch_status_failure_count ON autosearch_status(failure_count);
