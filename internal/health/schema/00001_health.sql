-- +goose Up
-- One row per open item: when it first appeared at any level and when its
-- level last changed. Subject and text are never stored, so they can't go
-- stale.
CREATE TABLE health_item (
    category TEXT NOT NULL,
    key TEXT NOT NULL CHECK (key <> ''),
    level TEXT NOT NULL CHECK (level IN ('warn', 'fail')),
    since INTEGER NOT NULL,
    changed_at INTEGER NOT NULL,
    PRIMARY KEY (category, key)
) STRICT, WITHOUT ROWID;

-- The global Storage levels. Each level is an amount and its unit, both
-- NULL at the default.
CREATE TABLE health_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    warn_amount INTEGER CHECK (warn_amount >= 0),
    warn_unit TEXT CHECK (warn_unit IN ('bytes', 'percent')),
    error_amount INTEGER CHECK (error_amount >= 0),
    error_unit TEXT CHECK (error_unit IN ('bytes', 'percent')),
    CHECK ((warn_amount IS NULL) = (warn_unit IS NULL)),
    CHECK ((error_amount IS NULL) = (error_unit IS NULL))
) STRICT;
INSERT INTO health_settings (id) VALUES (1);

-- A volume's own levels, keyed on the Node and the mount as the Node names
-- it, never the volume id: a filesystem id or a serial can change across a
-- remount. No foreign key to roots_volume, whose rows are replaced every
-- sample. A level left NULL follows the global one.
CREATE TABLE health_volume (
    node_id BLOB NOT NULL,
    mount TEXT NOT NULL CHECK (mount <> ''),
    warn_amount INTEGER CHECK (warn_amount >= 0),
    warn_unit TEXT CHECK (warn_unit IN ('bytes', 'percent')),
    error_amount INTEGER CHECK (error_amount >= 0),
    error_unit TEXT CHECK (error_unit IN ('bytes', 'percent')),
    CHECK ((warn_amount IS NULL) = (warn_unit IS NULL)),
    CHECK ((error_amount IS NULL) = (error_unit IS NULL)),
    CHECK (warn_amount IS NOT NULL OR error_amount IS NOT NULL),
    PRIMARY KEY (node_id, mount)
) STRICT, WITHOUT ROWID;
