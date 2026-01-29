-- +goose Up
-- Add quality profile and auto-approve settings to invitations
ALTER TABLE portal_invitations ADD COLUMN quality_profile_id INTEGER;
ALTER TABLE portal_invitations ADD COLUMN auto_approve INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE portal_invitations DROP COLUMN auto_approve;
ALTER TABLE portal_invitations DROP COLUMN quality_profile_id;
