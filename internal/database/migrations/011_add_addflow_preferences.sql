-- +goose Up
-- +goose StatementBegin

-- Add default settings for add-flow preferences
-- These store user's last selection for the add movie/series dialogs
INSERT OR IGNORE INTO settings (key, value) VALUES
('addflow_movie_search_on_add', 'false'),
('addflow_series_search_on_add', 'no'),
('addflow_series_monitor_on_add', 'future'),
('addflow_series_include_specials', 'false');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM settings WHERE key LIKE 'addflow_%';

-- +goose StatementEnd
