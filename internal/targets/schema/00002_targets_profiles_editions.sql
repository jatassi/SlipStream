-- +goose Up
-- A Target's qualities replace its resolutions. Each accepted quality is
-- one row: its tier (0 is best), its place in the Target's order, and its
-- size range in MiB per minute. A size can name only an accepted quality
-- because it is that quality's row.
CREATE TABLE targets_quality (
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    quality TEXT NOT NULL CHECK (quality IN (
        'sdtv', 'dvd', 'webrip-480p', 'hdtv-720p', 'webrip-720p', 'webdl-720p', 'bluray-720p',
        'hdtv-1080p', 'webrip-1080p', 'webdl-1080p', 'bluray-1080p', 'remux-1080p',
        'hdtv-2160p', 'webrip-2160p', 'webdl-2160p', 'bluray-2160p', 'remux-2160p', 'cam')),
    tier INTEGER NOT NULL CHECK (tier >= 0),
    position INTEGER NOT NULL CHECK (position >= 0),
    min_mib_per_minute REAL NOT NULL CHECK (min_mib_per_minute >= 0),
    max_mib_per_minute REAL CHECK (max_mib_per_minute IS NULL OR max_mib_per_minute >= min_mib_per_minute),
    PRIMARY KEY (target_id, quality),
    UNIQUE (target_id, position)
) STRICT;

-- No row for an attribute is Any.
CREATE TABLE targets_filter (
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    attribute TEXT NOT NULL CHECK (attribute IN ('hdr', 'video_codec', 'audio_codec', 'channels', 'language')),
    mode TEXT NOT NULL CHECK (mode IN ('only', 'not')),
    PRIMARY KEY (target_id, attribute)
) STRICT;

CREATE TABLE targets_filter_value (
    target_id BLOB NOT NULL,
    attribute TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    value TEXT NOT NULL CHECK (value <> ''),
    PRIMARY KEY (target_id, attribute, position),
    UNIQUE (target_id, attribute, value),
    FOREIGN KEY (target_id, attribute) REFERENCES targets_filter (target_id, attribute) ON DELETE CASCADE
) STRICT;

ALTER TABLE targets_target ADD COLUMN cutoff_tier INTEGER NOT NULL DEFAULT 0 CHECK (cutoff_tier >= 0);

-- Each resolution row becomes one tier of every non-cinema quality at that
-- resolution (576p is SDTV and DVD), in the quality table's order, at the
-- floors of release.Qualities, with the cutoff at the top tier. A quality
-- two rows name stays in the better one. Only a 576p row after a 480p row
-- adds nothing, so it takes no tier.
INSERT OR IGNORE INTO targets_quality (target_id, quality, tier, position, min_mib_per_minute)
SELECT r.target_id, m.quality,
    (SELECT count(*) FROM targets_resolution e
        WHERE e.target_id = r.target_id AND e.position < r.position
            AND NOT (e.resolution = '576p' AND EXISTS (
                SELECT 1 FROM targets_resolution f
                WHERE f.target_id = e.target_id AND f.position < e.position AND f.resolution = '480p'))),
    (r.position - 1) * 100 + m.ord, m.floor
FROM targets_resolution r JOIN (
    SELECT '2160p' AS resolution, 13 AS ord, 'hdtv-2160p' AS quality, 85.0 AS floor
    UNION ALL SELECT '2160p', 14, 'webrip-2160p', 34.5
    UNION ALL SELECT '2160p', 15, 'webdl-2160p', 34.5
    UNION ALL SELECT '2160p', 16, 'bluray-2160p', 102.0
    UNION ALL SELECT '2160p', 17, 'remux-2160p', 187.4
    UNION ALL SELECT '1080p', 8, 'hdtv-1080p', 33.8
    UNION ALL SELECT '1080p', 9, 'webrip-1080p', 12.5
    UNION ALL SELECT '1080p', 10, 'webdl-1080p', 12.5
    UNION ALL SELECT '1080p', 11, 'bluray-1080p', 50.8
    UNION ALL SELECT '1080p', 12, 'remux-1080p', 102.0
    UNION ALL SELECT '720p', 4, 'hdtv-720p', 17.1
    UNION ALL SELECT '720p', 5, 'webrip-720p', 12.5
    UNION ALL SELECT '720p', 6, 'webdl-720p', 12.5
    UNION ALL SELECT '720p', 7, 'bluray-720p', 25.7
    UNION ALL SELECT '576p', 1, 'sdtv', 0.0
    UNION ALL SELECT '576p', 2, 'dvd', 0.0
    UNION ALL SELECT '480p', 1, 'sdtv', 0.0
    UNION ALL SELECT '480p', 2, 'dvd', 0.0
    UNION ALL SELECT '480p', 3, 'webrip-480p', 0.0) m ON m.resolution = r.resolution
ORDER BY r.target_id, r.position, m.ord;

DROP TABLE targets_resolution;

-- Each Profile entry says how its Target is sourced. Conversion adds
-- derive and download-then-derive.
ALTER TABLE targets_profile_target ADD COLUMN sourcing TEXT NOT NULL DEFAULT 'download' CHECK (sourcing IN ('download'));

-- The starter Target, made once per library: targets.Open makes it on a
-- library with no row here. A library that already held Targets records
-- that it needs none, so deleting every Target never brings it back.
CREATE TABLE targets_starter (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL
) STRICT;

INSERT INTO targets_starter (id) SELECT 1 WHERE EXISTS (SELECT 1 FROM targets_target);

-- One row per Work: the Profile it wants from.
CREATE TABLE targets_wants (
    work_id BLOB PRIMARY KEY NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    profile_id BLOB NOT NULL REFERENCES targets_profile (id)
) STRICT;

CREATE INDEX targets_wants_profile_id ON targets_wants (profile_id);

-- Every existing Work takes its type's default Profile. targets.Open gives
-- a row to each Work whose type had none yet.
INSERT INTO targets_wants (work_id, profile_id)
SELECT w.id, p.id FROM works_work w
JOIN targets_profile p ON p.content_type = w.content_type AND p.is_default = 1;

-- A Work's own Edition list, which replaces its Profile's: any cut, or
-- the cuts targets_wants_cut names. NULL is the Profile's.
ALTER TABLE targets_wants ADD COLUMN editions TEXT CHECK (editions IN ('any', 'cuts'));

-- A Profile's Edition list: any cut, or the cuts targets_profile_cut
-- names in order. Every Profile that existed wants any cut.
ALTER TABLE targets_profile ADD COLUMN editions TEXT NOT NULL DEFAULT 'any' CHECK (editions IN ('any', 'cuts'));

-- A named list's cuts, in list order. A cut is a key: built-in and
-- Content Type cuts live in code, so only targets_cut rows can be checked
-- here, and targets refuses to delete a custom cut a list names.
CREATE TABLE targets_profile_cut (
    profile_id BLOB NOT NULL REFERENCES targets_profile (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    cut TEXT NOT NULL CHECK (cut NOT IN ('', 'any')),
    PRIMARY KEY (profile_id, position),
    UNIQUE (profile_id, cut)
) STRICT;

CREATE INDEX targets_profile_cut_cut ON targets_profile_cut (cut);

CREATE TABLE targets_wants_cut (
    work_id BLOB NOT NULL REFERENCES targets_wants (work_id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    cut TEXT NOT NULL CHECK (cut NOT IN ('', 'any')),
    PRIMARY KEY (work_id, position),
    UNIQUE (work_id, cut)
) STRICT;

CREATE INDEX targets_wants_cut_cut ON targets_wants_cut (cut);

-- The Admin's own cuts. The key is the name's slug, fixed at creation;
-- the name is matched in titles, as is each alias.
CREATE TABLE targets_cut (
    key TEXT PRIMARY KEY NOT NULL CHECK (key NOT IN ('', 'any')),
    name TEXT NOT NULL UNIQUE CHECK (name <> ''),
    created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE targets_cut_alias (
    cut TEXT NOT NULL REFERENCES targets_cut (key) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    alias TEXT NOT NULL CHECK (alias <> ''),
    PRIMARY KEY (cut, position),
    UNIQUE (cut, alias)
) STRICT;

-- A release of a cut an Indexer listed covering an Item. release is its
-- info hash in hex, or its title when the Indexer sent none. A file of a
-- cut counts as seen too, read from files and never stored here.
CREATE TABLE targets_sighting (
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    cut TEXT NOT NULL CHECK (cut NOT IN ('', 'any')),
    release TEXT NOT NULL CHECK (release <> ''),
    seen_at INTEGER NOT NULL,
    PRIMARY KEY (item_id, cut, release)
) STRICT;

CREATE INDEX targets_sighting_cut ON targets_sighting (cut);
CREATE INDEX targets_sighting_release ON targets_sighting (release);
