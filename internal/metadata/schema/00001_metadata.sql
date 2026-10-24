-- +goose Up
-- A NULL setting is its default.
CREATE TABLE metadata_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    language TEXT,
    region TEXT,
    unreleased_every_seconds INTEGER,
    growing_every_seconds INTEGER,
    others_every_seconds INTEGER,
    follow_changes INTEGER CHECK (follow_changes IN (0, 1)),
    ratings_every_seconds INTEGER,
    cache_for_seconds INTEGER,
    artwork_store_mb INTEGER
) STRICT;

INSERT INTO metadata_settings (id) VALUES (1);

CREATE TABLE metadata_key (
    provider TEXT PRIMARY KEY NOT NULL,
    api_key_sealed BLOB NOT NULL,
    set_at INTEGER NOT NULL
) STRICT;

CREATE TABLE metadata_type (
    content_type TEXT PRIMARY KEY NOT NULL,
    changes_through INTEGER,
    refresh_all_at INTEGER
) STRICT;

-- A Work is due while since is after its refreshed_at.
CREATE TABLE metadata_due (
    work_id BLOB PRIMARY KEY NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    reason TEXT NOT NULL CHECK (reason IN ('changed', 'requested', 'settings')),
    since INTEGER NOT NULL
) STRICT;

-- Works stored before metadata was whole hold six facts: the first refresh
-- reads each of them again.
INSERT INTO metadata_type (content_type, refresh_all_at)
SELECT DISTINCT content_type, CAST(strftime('%s', 'now') AS INTEGER) * 1000000 FROM works_work;
