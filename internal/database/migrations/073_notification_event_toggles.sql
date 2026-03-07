-- +goose Up
-- Replace individual event toggle columns with a JSON event_toggles column.
-- This makes the notification event set extensible for the module system.

-- Step 1: Create new table without event columns, with event_toggles JSON
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
    event_toggles TEXT NOT NULL DEFAULT '{}',
    include_health_warnings INTEGER NOT NULL DEFAULT 0,
    tags TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Copy data, converting boolean columns to JSON event_toggles
INSERT INTO notifications_new (
    id, name, type, enabled, settings, event_toggles,
    include_health_warnings, tags, created_at, updated_at
)
SELECT
    id, name, type, enabled, settings,
    json_object(
        'grab',             json(CASE WHEN on_grab = 1 THEN 'true' ELSE 'false' END),
        'import',           json(CASE WHEN on_import = 1 THEN 'true' ELSE 'false' END),
        'upgrade',          json(CASE WHEN on_upgrade = 1 THEN 'true' ELSE 'false' END),
        'movie:added',      json(CASE WHEN on_movie_added = 1 THEN 'true' ELSE 'false' END),
        'movie:deleted',    json(CASE WHEN on_movie_deleted = 1 THEN 'true' ELSE 'false' END),
        'tv:added',         json(CASE WHEN on_series_added = 1 THEN 'true' ELSE 'false' END),
        'tv:deleted',       json(CASE WHEN on_series_deleted = 1 THEN 'true' ELSE 'false' END),
        'health_issue',     json(CASE WHEN on_health_issue = 1 THEN 'true' ELSE 'false' END),
        'health_restored',  json(CASE WHEN on_health_restored = 1 THEN 'true' ELSE 'false' END),
        'app_update',       json(CASE WHEN on_app_update = 1 THEN 'true' ELSE 'false' END)
    ),
    include_health_warnings, tags, created_at, updated_at
FROM notifications;

-- Step 3: Drop old table and rename
DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;


-- +goose Down
-- Reverse: recreate table with individual columns
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
    on_grab INTEGER NOT NULL DEFAULT 0,
    on_import INTEGER NOT NULL DEFAULT 0,
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

INSERT INTO notifications_new (
    id, name, type, enabled, settings,
    on_grab, on_import, on_upgrade,
    on_movie_added, on_movie_deleted,
    on_series_added, on_series_deleted,
    on_health_issue, on_health_restored, on_app_update,
    include_health_warnings, tags, created_at, updated_at
)
SELECT
    id, name, type, enabled, settings,
    COALESCE(json_extract(event_toggles, '$.grab'), 0),
    COALESCE(json_extract(event_toggles, '$.import'), 0),
    COALESCE(json_extract(event_toggles, '$.upgrade'), 0),
    COALESCE(json_extract(event_toggles, '$."movie:added"'), 0),
    COALESCE(json_extract(event_toggles, '$."movie:deleted"'), 0),
    COALESCE(json_extract(event_toggles, '$."tv:added"'), 0),
    COALESCE(json_extract(event_toggles, '$."tv:deleted"'), 0),
    COALESCE(json_extract(event_toggles, '$.health_issue'), 0),
    COALESCE(json_extract(event_toggles, '$.health_restored'), 0),
    COALESCE(json_extract(event_toggles, '$.app_update'), 0),
    include_health_warnings, tags, created_at, updated_at
FROM notifications;

DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;
