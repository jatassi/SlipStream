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
