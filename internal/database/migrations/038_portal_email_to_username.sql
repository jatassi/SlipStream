-- +goose Up
-- Change portal_users and portal_invitations to use username instead of email

-- Rename email to username in portal_users
ALTER TABLE portal_users RENAME COLUMN email TO username;

-- Drop old index and create new one
DROP INDEX IF EXISTS idx_portal_users_email;
CREATE INDEX idx_portal_users_username ON portal_users(username);

-- Rename email to username in portal_invitations
ALTER TABLE portal_invitations RENAME COLUMN email TO username;

-- Drop old index and create new one
DROP INDEX IF EXISTS idx_portal_invitations_email;
CREATE INDEX idx_portal_invitations_username ON portal_invitations(username);

-- +goose Down
-- Revert username back to email in portal_invitations
DROP INDEX IF EXISTS idx_portal_invitations_username;
ALTER TABLE portal_invitations RENAME COLUMN username TO email;
CREATE INDEX idx_portal_invitations_email ON portal_invitations(email);

-- Revert username back to email in portal_users
DROP INDEX IF EXISTS idx_portal_users_username;
ALTER TABLE portal_users RENAME COLUMN username TO email;
CREATE INDEX idx_portal_users_email ON portal_users(email);
