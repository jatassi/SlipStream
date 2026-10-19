-- +goose Up
ALTER TABLE works_work ADD COLUMN gone_upstream_at INTEGER;
ALTER TABLE works_work ADD COLUMN ratings_read_at INTEGER;

CREATE TABLE works_rating (
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB REFERENCES works_item (id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source <> ''),
    value REAL NOT NULL CHECK (value >= 0),
    scale REAL NOT NULL CHECK (scale > 0 AND value <= scale),
    votes INTEGER CHECK (votes >= 0),
    fetched_at INTEGER NOT NULL
) STRICT;

CREATE UNIQUE INDEX works_rating_one ON works_rating (work_id, ifnull(item_id, x''), source);

CREATE INDEX works_rating_item_id ON works_rating (item_id);

CREATE TABLE works_artwork (
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB REFERENCES works_item (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind <> ''),
    source TEXT NOT NULL CHECK (source <> ''),
    path TEXT NOT NULL CHECK (path <> '')
) STRICT;

CREATE UNIQUE INDEX works_artwork_one ON works_artwork (work_id, ifnull(item_id, x''), kind);

CREATE INDEX works_artwork_item_id ON works_artwork (item_id);
