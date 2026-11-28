-- +goose Up
-- S44 (Conversion). A Profile row is downloaded, made from a Target above
-- it, or downloaded and made from it after grace_seconds with no download.
CREATE TABLE targets_profile_target_next (
    profile_id BLOB NOT NULL REFERENCES targets_profile (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    target_id BLOB NOT NULL REFERENCES targets_target (id),
    sourcing TEXT NOT NULL DEFAULT 'download' CHECK (sourcing IN ('download', 'derive', 'download_then_derive')),
    from_target_id BLOB REFERENCES targets_target (id),
    grace_seconds INTEGER,
    PRIMARY KEY (profile_id, position),
    UNIQUE (profile_id, target_id),
    CHECK ((sourcing = 'download') = (from_target_id IS NULL)),
    CHECK ((sourcing = 'download_then_derive') = (grace_seconds IS NOT NULL)),
    CHECK (grace_seconds IS NULL OR grace_seconds BETWEEN 3600 AND 7776000),
    CHECK (from_target_id IS NULL OR from_target_id <> target_id)
) STRICT;

INSERT INTO targets_profile_target_next (profile_id, position, target_id, sourcing)
SELECT profile_id, position, target_id, sourcing FROM targets_profile_target;

DROP TABLE targets_profile_target;
ALTER TABLE targets_profile_target_next RENAME TO targets_profile_target;

CREATE INDEX targets_profile_target_target_id ON targets_profile_target (target_id);
CREATE INDEX targets_profile_target_from ON targets_profile_target (from_target_id) WHERE from_target_id IS NOT NULL;

-- A Target with Conversion settings has a row here. version rises each
-- time one of the settings changes, never for Run on.
CREATE TABLE targets_conversion (
    target_id BLOB PRIMARY KEY NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    video_codec TEXT NOT NULL CHECK (video_codec IN ('h264', 'hevc', 'av1')),
    max_height INTEGER NOT NULL CHECK (max_height IN (2160, 1080, 720, 480)),
    bitrate_kbps INTEGER NOT NULL CHECK (bitrate_kbps BETWEEN 100 AND 200000),
    bit_depth INTEGER NOT NULL CHECK (bit_depth IN (8, 10)),
    range TEXT NOT NULL CHECK (range IN ('sdr', 'hdr10', 'source')),
    keep_dv INTEGER NOT NULL CHECK (keep_dv IN (0, 1)),
    curve TEXT NOT NULL CHECK (curve IN ('bt2390', 'hable', 'mobius', 'reinhard')),
    audio TEXT NOT NULL CHECK (audio IN ('copy', 'stereo_aac', 'ac3_51_and_stereo_aac')),
    drop_commentary INTEGER NOT NULL CHECK (drop_commentary IN (0, 1)),
    container TEXT NOT NULL CHECK (container IN ('mkv', 'mp4')),
    run_on_node_id BLOB REFERENCES nodes_node (id) ON DELETE SET NULL,
    CHECK (bit_depth = 8 OR video_codec <> 'h264'),
    CHECK (keep_dv = 0 OR (range <> 'sdr' AND video_codec <> 'h264'))
) STRICT;
CREATE INDEX targets_conversion_run_on ON targets_conversion (run_on_node_id) WHERE run_on_node_id IS NOT NULL;
