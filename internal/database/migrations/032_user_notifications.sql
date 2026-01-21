-- +goose Up
-- User notification configurations for portal users
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

CREATE INDEX idx_user_notifications_user_id ON user_notifications(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_user_notifications_user_id;
DROP TABLE IF EXISTS user_notifications;
