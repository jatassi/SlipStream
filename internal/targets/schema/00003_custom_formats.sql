-- +goose Up
-- A Custom Format, in Sonarr's and Radarr's shape. name_folded is the name
-- case-folded, so names are unique case-insensitively. conditions is the
-- list in upstream's export form. source is the pasted document, verbatim,
-- and NULL for a format made here. revision counts saves, and timeouts
-- counts the Releases a pattern ran past its limit on since the last save.
CREATE TABLE targets_format (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL CHECK (name <> ''),
    name_folded TEXT NOT NULL UNIQUE,
    include_when_renaming INTEGER NOT NULL CHECK (include_when_renaming IN (0, 1)),
    dialect TEXT NOT NULL CHECK (dialect IN ('radarr', 'sonarr', 'slipstream')),
    conditions TEXT NOT NULL CHECK (json_valid(conditions) AND json_type(conditions) = 'array'),
    source BLOB,
    revision INTEGER NOT NULL CHECK (revision >= 1),
    timeouts INTEGER NOT NULL DEFAULT 0 CHECK (timeouts >= 0),
    created_at INTEGER NOT NULL
) STRICT;

-- A Target's score for a format. A format the Target doesn't score has no
-- row, so "no Target scores it" is a row count.
CREATE TABLE targets_format_score (
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    format_id BLOB NOT NULL REFERENCES targets_format (id) ON DELETE CASCADE,
    score INTEGER NOT NULL CHECK (score <> 0),
    PRIMARY KEY (target_id, format_id)
) STRICT;

CREATE INDEX targets_format_score_format_id ON targets_format_score (format_id);

ALTER TABLE targets_target ADD COLUMN min_score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE targets_target ADD COLUMN upgrade_until INTEGER NOT NULL DEFAULT 0;
ALTER TABLE targets_target ADD COLUMN score_increment INTEGER NOT NULL DEFAULT 1 CHECK (score_increment >= 1);
ALTER TABLE targets_target ADD COLUMN propers TEXT NOT NULL DEFAULT 'prefer_and_upgrade'
    CHECK (propers IN ('prefer_and_upgrade', 'prefer_no_upgrade', 'dont_prefer'));
