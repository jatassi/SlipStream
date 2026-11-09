-- +goose Up
-- A Content Type's naming switches. No row, or NULL, is the type's
-- default.
CREATE TABLE naming_type (
    content_type TEXT PRIMARY KEY NOT NULL,
    rename_files INTEGER CHECK (rename_files IN (0, 1)),
    colon TEXT CHECK (colon IN ('delete', 'dash', 'space_dash', 'space_dash_space', 'smart', 'custom')),
    colon_custom TEXT,
    multi_episode TEXT CHECK (multi_episode IN ('extend', 'duplicate', 'repeat', 'scene', 'range', 'prefixed_range'))
) STRICT;

-- A template an Admin changed. field is folder, levels.<n>, aside or
-- file.<variant>. No row is the type's default.
CREATE TABLE naming_template (
    content_type TEXT NOT NULL,
    field TEXT NOT NULL CHECK (field <> ''),
    template TEXT NOT NULL,
    PRIMARY KEY (content_type, field)
) STRICT;
