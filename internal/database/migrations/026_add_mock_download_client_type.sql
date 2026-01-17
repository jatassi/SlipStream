-- +goose Up
-- Add 'mock' to download_clients and notifications type CHECK constraints for developer mode

-- SQLite doesn't support ALTER TABLE to modify CHECK constraints,
-- so we need to recreate the tables with updated constraints

-- ============================================
-- DOWNLOAD CLIENTS
-- ============================================

-- Create new table with updated constraint
CREATE TABLE download_clients_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('qbittorrent', 'transmission', 'deluge', 'rtorrent', 'sabnzbd', 'nzbget', 'mock')),
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password TEXT,
    use_ssl INTEGER NOT NULL DEFAULT 0,
    category TEXT,
    priority INTEGER NOT NULL DEFAULT 50,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    import_delay_seconds INTEGER NOT NULL DEFAULT 0,
    cleanup_mode TEXT NOT NULL DEFAULT 'leave',
    seed_ratio_target REAL
);

-- Copy data from old table
INSERT INTO download_clients_new (
    id, name, type, host, port, username, password, use_ssl, category,
    priority, enabled, created_at, updated_at, import_delay_seconds,
    cleanup_mode, seed_ratio_target
)
SELECT
    id, name, type, host, port, username, password, use_ssl, category,
    priority, enabled, created_at, updated_at, import_delay_seconds,
    cleanup_mode, seed_ratio_target
FROM download_clients;

-- Drop old table
DROP TABLE download_clients;

-- Rename new table
ALTER TABLE download_clients_new RENAME TO download_clients;

-- ============================================
-- NOTIFICATIONS
-- ============================================

-- Create new table with updated constraint (adding 'mock')
CREATE TABLE notifications_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'discord', 'telegram', 'webhook', 'email', 'slack', 'pushover',
        'gotify', 'ntfy', 'apprise', 'pushbullet', 'join', 'prowl',
        'simplepush', 'signal', 'custom_script', 'mock'
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

-- Copy data from old table
INSERT INTO notifications_new (
    id, name, type, enabled, settings, on_grab, on_download, on_upgrade,
    on_movie_added, on_movie_deleted, on_series_added, on_series_deleted,
    on_health_issue, on_health_restored, on_app_update, include_health_warnings,
    tags, created_at, updated_at
)
SELECT
    id, name, type, enabled, settings, on_grab, on_download, on_upgrade,
    on_movie_added, on_movie_deleted, on_series_added, on_series_deleted,
    on_health_issue, on_health_restored, on_app_update, include_health_warnings,
    tags, created_at, updated_at
FROM notifications;

-- Drop old table (notification_status has ON DELETE CASCADE)
DROP TABLE notifications;

-- Rename new table
ALTER TABLE notifications_new RENAME TO notifications;

-- +goose Down
-- Revert to original constraint (without 'mock')
CREATE TABLE download_clients_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('qbittorrent', 'transmission', 'deluge', 'rtorrent', 'sabnzbd', 'nzbget')),
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password TEXT,
    use_ssl INTEGER NOT NULL DEFAULT 0,
    category TEXT,
    priority INTEGER NOT NULL DEFAULT 50,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    import_delay_seconds INTEGER NOT NULL DEFAULT 0,
    cleanup_mode TEXT NOT NULL DEFAULT 'leave',
    seed_ratio_target REAL
);

-- Copy non-mock data
INSERT INTO download_clients_old (
    id, name, type, host, port, username, password, use_ssl, category,
    priority, enabled, created_at, updated_at, import_delay_seconds,
    cleanup_mode, seed_ratio_target
)
SELECT
    id, name, type, host, port, username, password, use_ssl, category,
    priority, enabled, created_at, updated_at, import_delay_seconds,
    cleanup_mode, seed_ratio_target
FROM download_clients
WHERE type != 'mock';

DROP TABLE download_clients;
ALTER TABLE download_clients_old RENAME TO download_clients;

-- Revert notifications table (without 'mock')
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

-- Copy non-mock data
INSERT INTO notifications_old (
    id, name, type, enabled, settings, on_grab, on_download, on_upgrade,
    on_movie_added, on_movie_deleted, on_series_added, on_series_deleted,
    on_health_issue, on_health_restored, on_app_update, include_health_warnings,
    tags, created_at, updated_at
)
SELECT
    id, name, type, enabled, settings, on_grab, on_download, on_upgrade,
    on_movie_added, on_movie_deleted, on_series_added, on_series_deleted,
    on_health_issue, on_health_restored, on_app_update, include_health_warnings,
    tags, created_at, updated_at
FROM notifications
WHERE type != 'mock';

DROP TABLE notifications;
ALTER TABLE notifications_old RENAME TO notifications;
