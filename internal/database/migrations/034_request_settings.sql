-- +goose Up
-- Insert default request settings
INSERT OR IGNORE INTO settings (key, value) VALUES ('requests_default_movie_quota', '5');
INSERT OR IGNORE INTO settings (key, value) VALUES ('requests_default_season_quota', '3');
INSERT OR IGNORE INTO settings (key, value) VALUES ('requests_default_episode_quota', '10');
INSERT OR IGNORE INTO settings (key, value) VALUES ('requests_default_root_folder_id', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('requests_admin_notify_new', 'false');
INSERT OR IGNORE INTO settings (key, value) VALUES ('requests_search_rate_limit', '30');

-- +goose Down
DELETE FROM settings WHERE key IN (
    'requests_default_movie_quota',
    'requests_default_season_quota',
    'requests_default_episode_quota',
    'requests_default_root_folder_id',
    'requests_admin_notify_new',
    'requests_search_rate_limit'
);
