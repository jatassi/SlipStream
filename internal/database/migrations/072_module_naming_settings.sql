-- +goose Up
-- +goose StatementBegin

CREATE TABLE module_naming_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    module_type TEXT NOT NULL,
    setting_key TEXT NOT NULL,
    setting_value TEXT NOT NULL DEFAULT '',
    UNIQUE(module_type, setting_key)
);

-- Migrate movie naming settings from import_settings
INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'rename_enabled', CASE WHEN rename_movies = 1 THEN 'true' ELSE 'false' END
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'colon_replacement', colon_replacement
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'custom_colon_replacement', COALESCE(custom_colon_replacement, '')
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'movie-folder', movie_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'movie-file', movie_file_format
FROM import_settings WHERE id = 1;

-- Migrate TV naming settings from import_settings
INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'rename_enabled', CASE WHEN rename_episodes = 1 THEN 'true' ELSE 'false' END
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'colon_replacement', colon_replacement
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'custom_colon_replacement', COALESCE(custom_colon_replacement, '')
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'multi_episode_style', multi_episode_style
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'series-folder', series_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'season-folder', season_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'specials-folder', specials_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'episode-file.standard', standard_episode_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'episode-file.daily', daily_episode_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'episode-file.anime', anime_episode_format
FROM import_settings WHERE id = 1;

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS module_naming_settings;
