-- name: GetImportSettings :one
SELECT * FROM import_settings WHERE id = 1;

-- name: UpdateImportSettings :one
UPDATE import_settings SET
    validation_level = ?,
    minimum_file_size_mb = ?,
    video_extensions = ?,
    match_conflict_behavior = ?,
    unknown_media_behavior = ?,
    rename_episodes = ?,
    replace_illegal_characters = ?,
    colon_replacement = ?,
    custom_colon_replacement = ?,
    standard_episode_format = ?,
    daily_episode_format = ?,
    anime_episode_format = ?,
    series_folder_format = ?,
    season_folder_format = ?,
    specials_folder_format = ?,
    multi_episode_style = ?,
    rename_movies = ?,
    movie_folder_format = ?,
    movie_file_format = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: UpdateImportValidationSettings :one
UPDATE import_settings SET
    validation_level = ?,
    minimum_file_size_mb = ?,
    video_extensions = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: UpdateImportMatchingSettings :one
UPDATE import_settings SET
    match_conflict_behavior = ?,
    unknown_media_behavior = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: UpdateImportTVNamingSettings :one
UPDATE import_settings SET
    rename_episodes = ?,
    replace_illegal_characters = ?,
    colon_replacement = ?,
    custom_colon_replacement = ?,
    standard_episode_format = ?,
    daily_episode_format = ?,
    anime_episode_format = ?,
    series_folder_format = ?,
    season_folder_format = ?,
    specials_folder_format = ?,
    multi_episode_style = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: UpdateImportMovieNamingSettings :one
UPDATE import_settings SET
    rename_movies = ?,
    movie_folder_format = ?,
    movie_file_format = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: EnsureImportSettingsExist :exec
INSERT OR IGNORE INTO import_settings (id) VALUES (1);
