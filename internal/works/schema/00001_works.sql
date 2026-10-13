-- +goose Up
CREATE TABLE works_work (
    id BLOB PRIMARY KEY NOT NULL,
    content_type TEXT NOT NULL,
    key TEXT NOT NULL CHECK (length(CAST(key AS BLOB)) BETWEEN 1 AND 200),
    label TEXT NOT NULL CHECK (label <> ''),
    sort_label TEXT NOT NULL CHECK (sort_label <> ''),
    added_at INTEGER NOT NULL,
    refreshed_at INTEGER NOT NULL,
    UNIQUE (content_type, key)
) STRICT;

CREATE TABLE works_level (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    parent_id BLOB REFERENCES works_level (id) ON DELETE CASCADE,
    depth INTEGER NOT NULL CHECK (depth >= 1),
    key TEXT NOT NULL CHECK (length(CAST(key AS BLOB)) BETWEEN 1 AND 200),
    label TEXT NOT NULL CHECK (label <> ''),
    position INTEGER NOT NULL CHECK (position >= 1),
    withdrawn_at INTEGER,
    UNIQUE (work_id, key),
    CHECK ((parent_id IS NULL) = (depth = 1))
) STRICT;

CREATE INDEX works_level_parent_id ON works_level (parent_id);

CREATE TABLE works_item (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    level_id BLOB REFERENCES works_level (id) ON DELETE CASCADE,
    key TEXT NOT NULL CHECK (length(CAST(key AS BLOB)) BETWEEN 1 AND 200),
    label TEXT NOT NULL CHECK (label <> ''),
    position INTEGER NOT NULL CHECK (position >= 1),
    withdrawn_at INTEGER,
    UNIQUE (work_id, key)
) STRICT;

CREATE INDEX works_item_level_id ON works_item (level_id);

CREATE TABLE works_disabled_type (
    content_type TEXT PRIMARY KEY NOT NULL,
    disabled_at INTEGER NOT NULL
) STRICT;
