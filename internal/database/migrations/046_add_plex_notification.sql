-- +goose Up
-- Add 'plex' to the notifications type constraint
-- SQLite doesn't support ALTER TABLE to modify CHECK constraints, so we need to:
-- 1. Create new table with updated constraint
-- 2. Copy data
-- 3. Drop old table
-- 4. Rename new table

-- Create temp table with updated constraint
CREATE TABLE notifications_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'discord', 'telegram', 'webhook', 'email', 'slack', 'pushover',
        'gotify', 'ntfy', 'apprise', 'pushbullet', 'join', 'prowl',
        'simplepush', 'signal', 'custom_script', 'mock', 'plex'
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

-- Copy data from old table
INSERT INTO notifications_new
SELECT * FROM notifications;

-- Drop old table and rename
DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;

-- Recreate notification_status foreign key reference
-- (The existing data is preserved since we didn't touch notification_status)

-- Create the Plex refresh queue table
CREATE TABLE plex_refresh_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    server_id TEXT NOT NULL,
    section_key INTEGER NOT NULL,
    path TEXT,
    queued_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(notification_id, server_id, section_key, path)
);

-- Index for efficient queue processing
CREATE INDEX idx_plex_refresh_queue_notification ON plex_refresh_queue(notification_id);

-- +goose Down
DROP INDEX IF EXISTS idx_plex_refresh_queue_notification;
DROP TABLE IF EXISTS plex_refresh_queue;

-- Revert notifications table constraint (remove 'plex' and 'mock')
CREATE TABLE notifications_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'discord', 'telegram', 'webhook', 'email', 'slack', 'pushover',
        'gotify', 'ntfy', 'apprise', 'pushbullet', 'join', 'prowl',
        'simplepush', 'signal', 'custom_script'
    )),
    enabled INTEGER NOT NULL DEFAULT 1,
    settings TEXT NOT NULL DEFAULT '{}',

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

-- Copy data (excluding plex and mock notifications)
INSERT INTO notifications_old
SELECT * FROM notifications WHERE type NOT IN ('plex', 'mock');

DROP TABLE notifications;
ALTER TABLE notifications_old RENAME TO notifications;
