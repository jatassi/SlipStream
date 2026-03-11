-- +goose Up
-- Remove naming columns from import_settings; data lives in module_naming_settings since migration 072.

CREATE TABLE import_settings_new (
    id INTEGER PRIMARY KEY CHECK (id = 1),

    -- Validation settings
    validation_level TEXT NOT NULL DEFAULT 'standard'
        CHECK(validation_level IN ('basic', 'standard', 'full')),
    minimum_file_size_mb INTEGER NOT NULL DEFAULT 100,
    video_extensions TEXT NOT NULL DEFAULT '.mkv,.mp4,.avi,.m4v,.mov,.wmv,.ts,.m2ts,.webm',

    -- Matching settings
    match_conflict_behavior TEXT NOT NULL DEFAULT 'trust_queue'
        CHECK(match_conflict_behavior IN ('trust_queue', 'trust_parse', 'fail')),
    unknown_media_behavior TEXT NOT NULL DEFAULT 'ignore'
        CHECK(unknown_media_behavior IN ('ignore', 'auto_add')),

    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO import_settings_new (id, validation_level, minimum_file_size_mb, video_extensions, match_conflict_behavior, unknown_media_behavior, updated_at)
SELECT id, validation_level, minimum_file_size_mb, video_extensions, match_conflict_behavior, unknown_media_behavior, updated_at
FROM import_settings;

DROP TABLE import_settings;
ALTER TABLE import_settings_new RENAME TO import_settings;

-- +goose Down
-- Recreate import_settings with naming columns, but data will be empty defaults.

CREATE TABLE import_settings_old (
    id INTEGER PRIMARY KEY CHECK (id = 1),

    -- Validation settings
    validation_level TEXT NOT NULL DEFAULT 'standard'
        CHECK(validation_level IN ('basic', 'standard', 'full')),
    minimum_file_size_mb INTEGER NOT NULL DEFAULT 100,
    video_extensions TEXT NOT NULL DEFAULT '.mkv,.mp4,.avi,.m4v,.mov,.wmv,.ts,.m2ts,.webm',

    -- Matching settings
    match_conflict_behavior TEXT NOT NULL DEFAULT 'trust_queue'
        CHECK(match_conflict_behavior IN ('trust_queue', 'trust_parse', 'fail')),
    unknown_media_behavior TEXT NOT NULL DEFAULT 'ignore'
        CHECK(unknown_media_behavior IN ('ignore', 'auto_add')),

    -- Renaming settings (TV)
    rename_episodes BOOLEAN NOT NULL DEFAULT 1,
    replace_illegal_characters BOOLEAN NOT NULL DEFAULT 1,
    colon_replacement TEXT NOT NULL DEFAULT 'smart'
        CHECK(colon_replacement IN ('delete', 'dash', 'space_dash', 'space_dash_space', 'smart', 'custom')),
    custom_colon_replacement TEXT DEFAULT NULL,

    -- Episode naming patterns
    standard_episode_format TEXT NOT NULL DEFAULT '{Series Title} - S{season:00}E{episode:00} - {Quality Title}',
    daily_episode_format TEXT NOT NULL DEFAULT '{Series Title} - {Air-Date} - {Episode Title} {Quality Full}',
    anime_episode_format TEXT NOT NULL DEFAULT '{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}',

    -- Folder patterns
    series_folder_format TEXT NOT NULL DEFAULT '{Series Title}',
    season_folder_format TEXT NOT NULL DEFAULT 'Season {season}',
    specials_folder_format TEXT NOT NULL DEFAULT 'Specials',

    -- Multi-episode style
    multi_episode_style TEXT NOT NULL DEFAULT 'extend'
        CHECK(multi_episode_style IN ('extend', 'duplicate', 'repeat', 'scene', 'range', 'prefixed_range')),

    -- Movie naming patterns
    rename_movies BOOLEAN NOT NULL DEFAULT 1,
    movie_folder_format TEXT NOT NULL DEFAULT '{Movie Title} ({Year})',
    movie_file_format TEXT NOT NULL DEFAULT '{Movie Title} ({Year}) - {Quality Title}',

    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO import_settings_old (id, validation_level, minimum_file_size_mb, video_extensions, match_conflict_behavior, unknown_media_behavior, updated_at)
SELECT id, validation_level, minimum_file_size_mb, video_extensions, match_conflict_behavior, unknown_media_behavior, updated_at
FROM import_settings;

DROP TABLE import_settings;
ALTER TABLE import_settings_old RENAME TO import_settings;
