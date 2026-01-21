-- +goose Up
-- Portal invitations for magic link signup
CREATE TABLE portal_invitations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_portal_invitations_token ON portal_invitations(token);
CREATE INDEX idx_portal_invitations_email ON portal_invitations(email);

-- +goose Down
DROP INDEX IF EXISTS idx_portal_invitations_email;
DROP INDEX IF EXISTS idx_portal_invitations_token;
DROP TABLE IF EXISTS portal_invitations;
