-- +goose Up
-- Expand download_clients: add api_key, url_base columns; expand type CHECK constraint for new torrent clients

-- SQLite doesn't support ALTER TABLE to modify CHECK constraints, so recreate the table
CREATE TABLE download_clients_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'qbittorrent', 'transmission', 'deluge', 'rtorrent',
        'vuze', 'aria2', 'flood', 'utorrent', 'hadouken',
        'downloadstation', 'freeboxdownload', 'rqbit', 'tribler',
        'sabnzbd', 'nzbget', 'mock'
    )),
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password TEXT,
    use_ssl INTEGER NOT NULL DEFAULT 0,
    api_key TEXT,
    category TEXT,
    url_base TEXT NOT NULL DEFAULT '',
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

-- +goose Down
-- Revert: remove api_key, url_base columns; revert type CHECK constraint
CREATE TABLE download_clients_old (
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
WHERE type IN ('qbittorrent', 'transmission', 'deluge', 'rtorrent', 'sabnzbd', 'nzbget', 'mock');

DROP TABLE download_clients;
ALTER TABLE download_clients_old RENAME TO download_clients;
