-- +goose Up
-- Portal users table for external request portal
CREATE TABLE portal_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
    auto_approve INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_portal_users_email ON portal_users(email);
CREATE INDEX idx_portal_users_enabled ON portal_users(enabled);

-- +goose Down
DROP INDEX IF EXISTS idx_portal_users_enabled;
DROP INDEX IF EXISTS idx_portal_users_email;
DROP TABLE IF EXISTS portal_users;
