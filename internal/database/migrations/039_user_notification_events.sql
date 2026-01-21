-- +goose Up
-- Add additional notification event columns for portal users
ALTER TABLE user_notifications ADD COLUMN on_approved INTEGER NOT NULL DEFAULT 1;
ALTER TABLE user_notifications ADD COLUMN on_denied INTEGER NOT NULL DEFAULT 1;

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
CREATE TABLE user_notifications_backup AS SELECT
    id, user_id, type, name, settings, on_available, enabled, created_at, updated_at
FROM user_notifications;

DROP TABLE user_notifications;

CREATE TABLE user_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN (
        'discord', 'telegram', 'webhook', 'email', 'slack', 'pushover',
        'gotify', 'ntfy', 'apprise', 'pushbullet', 'join', 'prowl',
        'simplepush', 'signal', 'custom_script'
    )),
    name TEXT NOT NULL,
    settings TEXT NOT NULL DEFAULT '{}',
    on_available INTEGER NOT NULL DEFAULT 1,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO user_notifications SELECT * FROM user_notifications_backup;
DROP TABLE user_notifications_backup;

CREATE INDEX idx_user_notifications_user_id ON user_notifications(user_id);
