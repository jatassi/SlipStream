-- +goose Up
-- +goose StatementBegin

-- Add default information settings for entities
-- Settings keys will follow pattern: default_{entity_type}_{media_type}
-- media_type can be 'movie', 'tv', or 'both'
-- entity_type can be 'root_folder', 'quality_profile', 'download_client', 'indexer'

-- Example initial defaults (will be managed through UI)
INSERT INTO settings (key, value) VALUES 
('default_root_folder_movie', ''),
('default_root_folder_tv', ''),
('default_quality_profile_movie', ''),
('default_quality_profile_tv', ''),
('default_download_client_movie', ''),
('default_download_client_tv', ''),
('default_indexer_movie', ''),
('default_indexer_tv', '');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove default settings
DELETE FROM settings WHERE key LIKE 'default_%';

-- +goose StatementEnd