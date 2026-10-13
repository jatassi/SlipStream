-- +goose Up
CREATE TABLE targets_target (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK (name <> ''),
    created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE targets_resolution (
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    resolution TEXT NOT NULL CHECK (resolution IN ('2160p', '1080p', '720p', '576p', '480p')),
    PRIMARY KEY (target_id, position),
    UNIQUE (target_id, resolution)
) STRICT;

CREATE TABLE targets_profile (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK (name <> ''),
    content_type TEXT NOT NULL,
    is_default INTEGER NOT NULL CHECK (is_default IN (0, 1))
) STRICT;

CREATE UNIQUE INDEX targets_profile_default ON targets_profile (content_type) WHERE is_default = 1;

CREATE TABLE targets_profile_target (
    profile_id BLOB NOT NULL REFERENCES targets_profile (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    target_id BLOB NOT NULL REFERENCES targets_target (id),
    PRIMARY KEY (profile_id, position),
    UNIQUE (profile_id, target_id)
) STRICT;

CREATE INDEX targets_profile_target_target_id ON targets_profile_target (target_id);
