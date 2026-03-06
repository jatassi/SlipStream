-- +goose Up
-- +goose StatementBegin

-- Disable FK enforcement during migration (Goose wraps in transaction)
PRAGMA foreign_keys = OFF;

-- 1. Create new quality_profiles table with module_type column
CREATE TABLE quality_profiles_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    module_type TEXT NOT NULL DEFAULT 'movie',
    cutoff INTEGER NOT NULL DEFAULT 0,
    items TEXT NOT NULL DEFAULT '[]',
    hdr_settings TEXT,
    video_codec_settings TEXT,
    audio_codec_settings TEXT,
    audio_channel_settings TEXT,
    upgrades_enabled INTEGER NOT NULL DEFAULT 1,
    allow_auto_approve INTEGER NOT NULL DEFAULT 0,
    upgrade_strategy TEXT NOT NULL DEFAULT 'balanced',
    cutoff_overrides_strategy INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(module_type, name)
);

-- 2. Copy existing profiles as 'movie' type (preserving original IDs)
INSERT INTO quality_profiles_new (
    id, name, module_type, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
)
SELECT
    id, name, 'movie', cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
FROM quality_profiles;

-- 3. Create 'tv' copies (new auto-generated IDs)
INSERT INTO quality_profiles_new (
    name, module_type, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
)
SELECT
    name, 'tv', cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
FROM quality_profiles;

-- 4. Build temporary mapping: old profile ID → new TV copy ID
-- Join on name since movie copies have the original IDs and TV copies have new IDs
CREATE TEMPORARY TABLE profile_id_map AS
SELECT qp.id AS old_id, qpn.id AS new_id
FROM quality_profiles qp
JOIN quality_profiles_new qpn ON qp.name = qpn.name AND qpn.module_type = 'tv';

-- 5. Update series.quality_profile_id to point to new TV copies
UPDATE series SET quality_profile_id = (
    SELECT new_id FROM profile_id_map WHERE old_id = series.quality_profile_id
) WHERE quality_profile_id IS NOT NULL
  AND quality_profile_id IN (SELECT old_id FROM profile_id_map);

-- 6. Update portal_users.tv_quality_profile_id to point to new TV copies
UPDATE portal_users SET tv_quality_profile_id = (
    SELECT new_id FROM profile_id_map WHERE old_id = portal_users.tv_quality_profile_id
) WHERE tv_quality_profile_id IS NOT NULL
  AND tv_quality_profile_id IN (SELECT old_id FROM profile_id_map);

-- 7. Update portal_invitations.tv_quality_profile_id to point to new TV copies
UPDATE portal_invitations SET tv_quality_profile_id = (
    SELECT new_id FROM profile_id_map WHERE old_id = portal_invitations.tv_quality_profile_id
) WHERE tv_quality_profile_id IS NOT NULL
  AND tv_quality_profile_id IN (SELECT old_id FROM profile_id_map);

-- 8. Drop temp mapping table
DROP TABLE profile_id_map;

-- 9. Clear import decisions cache (profile IDs are now stale for TV decisions)
DELETE FROM import_decisions;

-- 10. Drop old table, rename new
-- NOTE: version_slots.quality_profile_id references quality_profiles(id).
-- After this migration, slots will reference "movie" copies (original IDs preserved).
-- This is intentional — Phase 8 will refactor slots with per-module profile references.
DROP TABLE quality_profiles;
ALTER TABLE quality_profiles_new RENAME TO quality_profiles;

-- Re-enable FK enforcement
PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- Build reverse mapping: TV copy ID → original (movie copy) ID
CREATE TEMPORARY TABLE profile_reverse_map AS
SELECT qp_movie.id AS movie_id, qp_tv.id AS tv_id
FROM quality_profiles qp_movie
JOIN quality_profiles qp_tv ON qp_movie.name = qp_tv.name
WHERE qp_movie.module_type = 'movie' AND qp_tv.module_type = 'tv';

-- Revert series FKs to original IDs
UPDATE series SET quality_profile_id = (
    SELECT movie_id FROM profile_reverse_map WHERE tv_id = series.quality_profile_id
) WHERE quality_profile_id IN (SELECT tv_id FROM profile_reverse_map);

-- Revert portal_users.tv_quality_profile_id
UPDATE portal_users SET tv_quality_profile_id = (
    SELECT movie_id FROM profile_reverse_map WHERE tv_id = portal_users.tv_quality_profile_id
) WHERE tv_quality_profile_id IN (SELECT tv_id FROM profile_reverse_map);

-- Revert portal_invitations.tv_quality_profile_id
UPDATE portal_invitations SET tv_quality_profile_id = (
    SELECT movie_id FROM profile_reverse_map WHERE tv_id = portal_invitations.tv_quality_profile_id
) WHERE tv_quality_profile_id IN (SELECT tv_id FROM profile_reverse_map);

DROP TABLE profile_reverse_map;

-- Recreate without module_type
CREATE TABLE quality_profiles_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    cutoff INTEGER NOT NULL DEFAULT 0,
    items TEXT NOT NULL DEFAULT '[]',
    hdr_settings TEXT,
    video_codec_settings TEXT,
    audio_codec_settings TEXT,
    audio_channel_settings TEXT,
    upgrades_enabled INTEGER NOT NULL DEFAULT 1,
    allow_auto_approve INTEGER NOT NULL DEFAULT 0,
    upgrade_strategy TEXT NOT NULL DEFAULT 'balanced',
    cutoff_overrides_strategy INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Keep only movie copies (they have the original IDs)
INSERT INTO quality_profiles_old (
    id, name, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
)
SELECT
    id, name, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
FROM quality_profiles WHERE module_type = 'movie';

DROP TABLE quality_profiles;
ALTER TABLE quality_profiles_old RENAME TO quality_profiles;

DELETE FROM import_decisions;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd
