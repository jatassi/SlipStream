-- +goose Up
-- Notifications table for external notification providers (Discord, Telegram, etc.)
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'discord', 'telegram', 'webhook', 'email', 'slack', 'pushover',
        'gotify', 'ntfy', 'apprise', 'pushbullet', 'join', 'prowl',
        'simplepush', 'signal', 'custom_script'
    )),
    enabled INTEGER NOT NULL DEFAULT 1,
    settings TEXT NOT NULL DEFAULT '{}',

    -- Event subscriptions
    on_grab INTEGER NOT NULL DEFAULT 0,
    on_download INTEGER NOT NULL DEFAULT 0,
    on_upgrade INTEGER NOT NULL DEFAULT 0,
    on_movie_added INTEGER NOT NULL DEFAULT 0,
    on_movie_deleted INTEGER NOT NULL DEFAULT 0,
    on_series_added INTEGER NOT NULL DEFAULT 0,
    on_series_deleted INTEGER NOT NULL DEFAULT 0,
    on_health_issue INTEGER NOT NULL DEFAULT 0,
    on_health_restored INTEGER NOT NULL DEFAULT 0,
    on_app_update INTEGER NOT NULL DEFAULT 0,
    include_health_warnings INTEGER NOT NULL DEFAULT 0,

    tags TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Track notification failures for backoff/retry logic
CREATE TABLE notification_status (
    notification_id INTEGER PRIMARY KEY REFERENCES notifications(id) ON DELETE CASCADE,
    initial_failure DATETIME,
    most_recent_failure DATETIME,
    escalation_level INTEGER NOT NULL DEFAULT 0,
    disabled_till DATETIME
);

-- +goose Down
DROP TABLE IF EXISTS notification_status;
DROP TABLE IF EXISTS notifications;
