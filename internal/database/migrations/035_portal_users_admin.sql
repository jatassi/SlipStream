-- +goose Up
-- Add is_admin column to portal_users for admin privileges
ALTER TABLE portal_users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE portal_users DROP COLUMN is_admin;
