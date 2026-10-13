-- +goose Up
CREATE TABLE music_artist (
    work_id BLOB PRIMARY KEY NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    country TEXT NOT NULL,
    disambiguation TEXT NOT NULL
) STRICT;

CREATE TABLE music_album (
    level_id BLOB PRIMARY KEY NOT NULL REFERENCES works_level (id) ON DELETE CASCADE,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    released TEXT NOT NULL,
    primary_type TEXT NOT NULL CHECK (primary_type <> ''),
    release_id TEXT NOT NULL CHECK (release_id <> '')
) STRICT;

CREATE INDEX music_album_work_id ON music_album (work_id);

CREATE TABLE music_track (
    item_id BLOB PRIMARY KEY NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    disc INTEGER NOT NULL CHECK (disc >= 1),
    number INTEGER NOT NULL CHECK (number >= 1),
    length_ms INTEGER CHECK (length_ms >= 0)
) STRICT;

CREATE INDEX music_track_work_id ON music_track (work_id);
