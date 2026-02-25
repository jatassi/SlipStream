-- +goose Up
-- Remove the display_name column from portal_users (username is sufficient)
ALTER TABLE portal_users DROP COLUMN display_name;

-- +goose Down
ALTER TABLE portal_users ADD COLUMN display_name TEXT;
UPDATE portal_users SET display_name = username;
