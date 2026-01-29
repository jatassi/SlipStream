-- +goose Up
-- Singleton table for import configuration settings
CREATE TABLE IF NOT EXISTS import_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),

    -- Validation settings
    validation_level TEXT NOT NULL DEFAULT 'standard'
        CHECK(validation_level IN ('basic', 'standard', 'full')),
    minimum_file_size_mb INTEGER NOT NULL DEFAULT 100,
    video_extensions TEXT NOT NULL DEFAULT '.mkv,.mp4,.avi,.m4v,.mov,.wmv,.ts,.m2ts,.webm',

    -- Matching settings
    match_conflict_behavior TEXT NOT NULL DEFAULT 'fail'
        CHECK(match_conflict_behavior IN ('trust_queue', 'trust_parse', 'fail')),
    unknown_media_behavior TEXT NOT NULL DEFAULT 'ignore'
        CHECK(unknown_media_behavior IN ('ignore', 'auto_add')),

    -- Renaming settings (TV)
    rename_episodes BOOLEAN NOT NULL DEFAULT 1,
    replace_illegal_characters BOOLEAN NOT NULL DEFAULT 1,
    colon_replacement TEXT NOT NULL DEFAULT 'smart'
        CHECK(colon_replacement IN ('delete', 'dash', 'space_dash', 'space_dash_space', 'smart', 'custom')),
    custom_colon_replacement TEXT DEFAULT NULL,

    -- Episode naming patterns (Sonarr-compatible defaults)
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

-- Ensure singleton row exists
INSERT OR IGNORE INTO import_settings (id) VALUES (1);

-- +goose Down
DROP TABLE IF EXISTS import_settings;
