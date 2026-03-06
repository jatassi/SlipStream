-- +goose Up
-- +goose StatementBegin

-- ====================================================================
-- 1. download_mappings: nullable FKs → discriminator columns
-- ====================================================================

CREATE TABLE download_mappings_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER NOT NULL REFERENCES download_clients(id) ON DELETE CASCADE,
    download_id TEXT NOT NULL,
    module_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    -- TV-specific pack info (still needed for season/series pack detection)
    season_number INTEGER,
    is_season_pack INTEGER NOT NULL DEFAULT 0,
    is_complete_series INTEGER NOT NULL DEFAULT 0,
    -- NOTE: ON DELETE SET NULL is intentionally added here. The original ALTER TABLE
    -- (migration 021) omitted it, but requests and other recreated tables already have it.
    -- This aligns download_mappings with the intended behavior: nullify slot ref when slot is deleted.
    target_slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL,
    source TEXT NOT NULL DEFAULT 'auto-search',
    import_attempts INTEGER NOT NULL DEFAULT 0,
    last_import_error TEXT,
    next_import_retry_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(client_id, download_id)
);

INSERT INTO download_mappings_new (
    id, client_id, download_id, module_type, entity_type, entity_id,
    season_number, is_season_pack, is_complete_series,
    target_slot_id, source, import_attempts, last_import_error,
    next_import_retry_at, created_at
)
SELECT
    id, client_id, download_id,
    CASE
        WHEN movie_id IS NOT NULL THEN 'movie'
        ELSE 'tv'
    END,
    CASE
        WHEN movie_id IS NOT NULL THEN 'movie'
        WHEN episode_id IS NOT NULL THEN 'episode'
        -- Season packs and complete series use entity_type='series' with entity_id=series_id.
        -- The season_number and is_season_pack columns provide sub-level targeting info.
        -- We don't use entity_type='season' here because download_mappings never stored
        -- a season_id — only (series_id, season_number) pairs.
        WHEN series_id IS NOT NULL THEN 'series'
        ELSE 'movie' -- fallback, shouldn't happen
    END,
    COALESCE(movie_id, episode_id, series_id),
    season_number, is_season_pack, is_complete_series,
    target_slot_id, source, import_attempts, last_import_error,
    next_import_retry_at, created_at
FROM download_mappings;

DROP TABLE download_mappings;
ALTER TABLE download_mappings_new RENAME TO download_mappings;

CREATE INDEX idx_download_mappings_entity ON download_mappings(module_type, entity_type, entity_id);
CREATE INDEX idx_download_mappings_slot ON download_mappings(target_slot_id);

-- ====================================================================
-- 2. queue_media: nullable FKs → discriminator columns
-- ====================================================================

CREATE TABLE queue_media_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    download_mapping_id INTEGER NOT NULL REFERENCES download_mappings(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    file_path TEXT,
    file_status TEXT NOT NULL DEFAULT 'pending'
        CHECK(file_status IN ('pending', 'downloading', 'ready', 'importing', 'imported', 'failed')),
    error_message TEXT,
    import_attempts INTEGER NOT NULL DEFAULT 0,
    target_slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(download_mapping_id, entity_id)
);

INSERT INTO queue_media_new (
    id, download_mapping_id, module_type, entity_type, entity_id,
    file_path, file_status, error_message, import_attempts, target_slot_id,
    created_at, updated_at
)
SELECT
    id, download_mapping_id,
    CASE WHEN movie_id IS NOT NULL THEN 'movie' ELSE 'tv' END,
    CASE WHEN movie_id IS NOT NULL THEN 'movie' ELSE 'episode' END,
    COALESCE(movie_id, episode_id),
    file_path, file_status, error_message, import_attempts, target_slot_id,
    created_at, updated_at
FROM queue_media;

DROP TABLE queue_media;
ALTER TABLE queue_media_new RENAME TO queue_media;

CREATE INDEX idx_queue_media_status ON queue_media(file_status);
CREATE INDEX idx_queue_media_entity ON queue_media(module_type, entity_type, entity_id);
CREATE INDEX idx_queue_media_download_mapping ON queue_media(download_mapping_id);

-- ====================================================================
-- 3. downloads: rename columns, widen constraint
-- ====================================================================

CREATE TABLE downloads_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER REFERENCES download_clients(id),
    external_id TEXT,
    title TEXT NOT NULL,
    module_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'downloading', 'paused', 'completed', 'failed', 'importing')),
    progress REAL NOT NULL DEFAULT 0,
    size INTEGER NOT NULL DEFAULT 0,
    download_url TEXT,
    output_path TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

INSERT INTO downloads_new (
    id, client_id, external_id, title, module_type, entity_type, entity_id,
    status, progress, size, download_url, output_path, added_at, completed_at
)
SELECT
    id, client_id, external_id, title,
    CASE WHEN media_type = 'movie' THEN 'movie' ELSE 'tv' END,
    media_type,
    media_id,
    status, progress, size, download_url, output_path, added_at, completed_at
FROM downloads;

DROP TABLE downloads;
ALTER TABLE downloads_new RENAME TO downloads;

CREATE INDEX idx_downloads_status ON downloads(status);
CREATE INDEX idx_downloads_entity ON downloads(module_type, entity_type, entity_id);

-- ====================================================================
-- 4. history: add module_type, rename media_type → entity_type
-- ====================================================================

CREATE TABLE history_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    module_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    source TEXT,
    quality TEXT,
    data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- media_type → entity_type, media_id → entity_id
INSERT INTO history_new (
    id, event_type, module_type, entity_type, entity_id,
    source, quality, data, created_at
)
SELECT
    id, event_type,
    CASE WHEN media_type = 'movie' THEN 'movie' ELSE 'tv' END,
    media_type,  -- becomes entity_type
    media_id,    -- becomes entity_id
    source, quality, data, created_at
FROM history;

DROP TABLE history;
ALTER TABLE history_new RENAME TO history;

CREATE INDEX idx_history_entity ON history(module_type, entity_type, entity_id);
CREATE INDEX idx_history_created_at ON history(created_at);

-- ====================================================================
-- 5. autosearch_status: add module_type, rename columns
-- ====================================================================

CREATE TABLE autosearch_status_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    module_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    search_type TEXT NOT NULL DEFAULT 'missing' CHECK (search_type IN ('missing', 'upgrade')),
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_searched_at DATETIME,
    last_meta_change_at DATETIME,
    UNIQUE(module_type, entity_type, entity_id, search_type)
);

INSERT INTO autosearch_status_new (
    id, module_type, entity_type, entity_id, search_type,
    failure_count, last_searched_at, last_meta_change_at
)
SELECT
    id,
    CASE WHEN item_type = 'movie' THEN 'movie' ELSE 'tv' END,
    item_type,
    item_id,
    search_type,
    failure_count, last_searched_at, last_meta_change_at
FROM autosearch_status;

DROP TABLE autosearch_status;
ALTER TABLE autosearch_status_new RENAME TO autosearch_status;

CREATE INDEX idx_autosearch_status_entity ON autosearch_status(module_type, entity_type, entity_id, search_type);
CREATE INDEX idx_autosearch_status_failure ON autosearch_status(failure_count);

-- ====================================================================
-- 6. import_decisions: add module_type, rename columns
-- ====================================================================

CREATE TABLE import_decisions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_path TEXT NOT NULL UNIQUE,
    decision TEXT NOT NULL,
    module_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    slot_id INTEGER,
    candidate_quality_id INTEGER,
    existing_quality_id INTEGER,
    existing_file_id INTEGER,
    quality_profile_id INTEGER,
    reason TEXT,
    evaluated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO import_decisions_new (
    id, source_path, decision, module_type, entity_type, entity_id,
    slot_id, candidate_quality_id, existing_quality_id, existing_file_id,
    quality_profile_id, reason, evaluated_at
)
SELECT
    id, source_path, decision,
    CASE WHEN media_type = 'movie' THEN 'movie' ELSE 'tv' END,
    media_type,
    media_id,
    slot_id, candidate_quality_id, existing_quality_id, existing_file_id,
    quality_profile_id, reason, evaluated_at
FROM import_decisions;

DROP TABLE import_decisions;
ALTER TABLE import_decisions_new RENAME TO import_decisions;

CREATE INDEX idx_import_decisions_source ON import_decisions(source_path);
CREATE INDEX idx_import_decisions_entity ON import_decisions(module_type, entity_type, entity_id);
CREATE INDEX idx_import_decisions_profile ON import_decisions(quality_profile_id);
CREATE INDEX idx_import_decisions_existing_file ON import_decisions(existing_file_id);

-- ====================================================================
-- 7. root_folders: rename media_type → module_type, widen constraint
-- ====================================================================

CREATE TABLE root_folders_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    module_type TEXT NOT NULL,
    free_space INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO root_folders_new (id, path, name, module_type, free_space, created_at)
SELECT id, path, name, media_type, free_space, created_at
FROM root_folders;

DROP TABLE root_folders;
ALTER TABLE root_folders_new RENAME TO root_folders;

-- ====================================================================
-- 8. requests: add module_type, rename media_type → entity_type
-- ====================================================================

CREATE TABLE requests_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    tmdb_id INTEGER,
    tvdb_id INTEGER,
    title TEXT NOT NULL,
    year INTEGER,
    season_number INTEGER,
    episode_number INTEGER,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'denied', 'searching', 'downloading', 'failed', 'available')),
    monitor_type TEXT,
    denied_reason TEXT,
    approved_at DATETIME,
    approved_by INTEGER REFERENCES portal_users(id) ON DELETE SET NULL,
    media_id INTEGER,
    target_slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL,
    poster_url TEXT,
    requested_seasons TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO requests_new (
    id, user_id, module_type, entity_type, tmdb_id, tvdb_id, title, year,
    season_number, episode_number, status, monitor_type, denied_reason,
    approved_at, approved_by, media_id, target_slot_id, poster_url,
    requested_seasons, created_at, updated_at
)
SELECT
    id, user_id,
    CASE WHEN media_type = 'movie' THEN 'movie' ELSE 'tv' END,
    media_type,
    tmdb_id, tvdb_id, title, year,
    season_number, episode_number, status, monitor_type, denied_reason,
    approved_at, approved_by, media_id, target_slot_id, poster_url,
    requested_seasons, created_at, updated_at
FROM requests;

DROP TABLE requests;
ALTER TABLE requests_new RENAME TO requests;

CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_tmdb_id ON requests(tmdb_id);
CREATE INDEX idx_requests_tvdb_id ON requests(tvdb_id);
CREATE INDEX idx_requests_entity ON requests(module_type, entity_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse migration: recreate all tables with original column names and constraints.
-- For development use only — do not rely on this in production.

-- 8. requests: entity_type → media_type, drop module_type
CREATE TABLE requests_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'series', 'season', 'episode')),
    tmdb_id INTEGER,
    tvdb_id INTEGER,
    title TEXT NOT NULL,
    year INTEGER,
    season_number INTEGER,
    episode_number INTEGER,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'denied', 'searching', 'downloading', 'failed', 'available')),
    monitor_type TEXT,
    denied_reason TEXT,
    approved_at DATETIME,
    approved_by INTEGER REFERENCES portal_users(id) ON DELETE SET NULL,
    media_id INTEGER,
    target_slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL,
    poster_url TEXT,
    requested_seasons TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO requests_old SELECT id, user_id, entity_type, tmdb_id, tvdb_id, title, year,
    season_number, episode_number, status, monitor_type, denied_reason,
    approved_at, approved_by, media_id, target_slot_id, poster_url,
    requested_seasons, created_at, updated_at FROM requests;
DROP TABLE requests;
ALTER TABLE requests_old RENAME TO requests;
CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_tmdb_id ON requests(tmdb_id);
CREATE INDEX idx_requests_tvdb_id ON requests(tvdb_id);
CREATE INDEX idx_requests_media_type ON requests(media_type);

-- 7. root_folders: module_type → media_type, restore CHECK
CREATE TABLE root_folders_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv')),
    free_space INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO root_folders_old SELECT id, path, name, module_type, free_space, created_at FROM root_folders;
DROP TABLE root_folders;
ALTER TABLE root_folders_old RENAME TO root_folders;

-- 6. import_decisions: entity_type/entity_id → media_type/media_id, drop module_type
CREATE TABLE import_decisions_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_path TEXT NOT NULL UNIQUE,
    decision TEXT NOT NULL,
    media_type TEXT NOT NULL,
    media_id INTEGER NOT NULL,
    slot_id INTEGER,
    candidate_quality_id INTEGER,
    existing_quality_id INTEGER,
    existing_file_id INTEGER,
    quality_profile_id INTEGER,
    reason TEXT,
    evaluated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO import_decisions_old SELECT id, source_path, decision, entity_type, entity_id,
    slot_id, candidate_quality_id, existing_quality_id, existing_file_id,
    quality_profile_id, reason, evaluated_at FROM import_decisions;
DROP TABLE import_decisions;
ALTER TABLE import_decisions_old RENAME TO import_decisions;
CREATE INDEX idx_import_decisions_source ON import_decisions(source_path);
CREATE INDEX idx_import_decisions_media ON import_decisions(media_type, media_id);
CREATE INDEX idx_import_decisions_profile ON import_decisions(quality_profile_id);
CREATE INDEX idx_import_decisions_existing_file ON import_decisions(existing_file_id);

-- 5. autosearch_status: entity_type/entity_id → item_type/item_id, drop module_type
CREATE TABLE autosearch_status_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_type TEXT NOT NULL CHECK (item_type IN ('movie', 'episode', 'series')),
    item_id INTEGER NOT NULL,
    search_type TEXT NOT NULL DEFAULT 'missing' CHECK (search_type IN ('missing', 'upgrade')),
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_searched_at DATETIME,
    last_meta_change_at DATETIME,
    UNIQUE(item_type, item_id, search_type)
);
INSERT INTO autosearch_status_old SELECT id, entity_type, entity_id, search_type,
    failure_count, last_searched_at, last_meta_change_at FROM autosearch_status;
DROP TABLE autosearch_status;
ALTER TABLE autosearch_status_old RENAME TO autosearch_status;
CREATE INDEX idx_autosearch_status_entity ON autosearch_status(item_type, item_id, search_type);
CREATE INDEX idx_autosearch_status_failure ON autosearch_status(failure_count);

-- 4. history: entity_type/entity_id → media_type/media_id, drop module_type
CREATE TABLE history_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'episode', 'season')),
    media_id INTEGER NOT NULL,
    source TEXT,
    quality TEXT,
    data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO history_old SELECT id, event_type, entity_type, entity_id,
    source, quality, data, created_at FROM history;
DROP TABLE history;
ALTER TABLE history_old RENAME TO history;
CREATE INDEX idx_history_media ON history(media_type, media_id);
CREATE INDEX idx_history_created_at ON history(created_at);

-- 3. downloads: module_type/entity_type/entity_id → media_type/media_id
CREATE TABLE downloads_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER REFERENCES download_clients(id),
    external_id TEXT,
    title TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'episode')),
    media_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'downloading', 'paused', 'completed', 'failed', 'importing')),
    progress REAL NOT NULL DEFAULT 0,
    size INTEGER NOT NULL DEFAULT 0,
    download_url TEXT,
    output_path TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);
INSERT INTO downloads_old SELECT id, client_id, external_id, title,
    entity_type, entity_id,
    status, progress, size, download_url, output_path, added_at, completed_at FROM downloads;
DROP TABLE downloads;
ALTER TABLE downloads_old RENAME TO downloads;
CREATE INDEX idx_downloads_status ON downloads(status);

-- 2. queue_media: discriminator → nullable FKs
CREATE TABLE queue_media_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    download_mapping_id INTEGER NOT NULL REFERENCES download_mappings(id) ON DELETE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    file_path TEXT,
    file_status TEXT NOT NULL DEFAULT 'pending'
        CHECK(file_status IN ('pending', 'downloading', 'ready', 'importing', 'imported', 'failed')),
    error_message TEXT,
    import_attempts INTEGER NOT NULL DEFAULT 0,
    target_slot_id INTEGER REFERENCES version_slots(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(download_mapping_id, episode_id),
    UNIQUE(download_mapping_id, movie_id)
);
INSERT INTO queue_media_old (id, download_mapping_id, episode_id, movie_id,
    file_path, file_status, error_message, import_attempts, target_slot_id,
    created_at, updated_at)
SELECT id, download_mapping_id,
    CASE WHEN entity_type = 'episode' THEN entity_id ELSE NULL END,
    CASE WHEN entity_type = 'movie' THEN entity_id ELSE NULL END,
    file_path, file_status, error_message, import_attempts, target_slot_id,
    created_at, updated_at FROM queue_media;
DROP TABLE queue_media;
ALTER TABLE queue_media_old RENAME TO queue_media;
CREATE INDEX idx_queue_media_status ON queue_media(file_status);
CREATE INDEX idx_queue_media_download_mapping ON queue_media(download_mapping_id);

-- 1. download_mappings: discriminator → nullable FKs
CREATE TABLE download_mappings_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER NOT NULL REFERENCES download_clients(id) ON DELETE CASCADE,
    download_id TEXT NOT NULL,
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    series_id INTEGER REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    is_season_pack INTEGER NOT NULL DEFAULT 0,
    is_complete_series INTEGER NOT NULL DEFAULT 0,
    target_slot_id INTEGER REFERENCES version_slots(id),
    source TEXT NOT NULL DEFAULT 'auto-search',
    import_attempts INTEGER NOT NULL DEFAULT 0,
    last_import_error TEXT,
    next_import_retry_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(client_id, download_id)
);
INSERT INTO download_mappings_old (id, client_id, download_id,
    movie_id, series_id, season_number, episode_id,
    is_season_pack, is_complete_series, target_slot_id, source,
    import_attempts, last_import_error, next_import_retry_at, created_at)
SELECT id, client_id, download_id,
    CASE WHEN module_type = 'movie' THEN entity_id ELSE NULL END,
    CASE WHEN entity_type = 'series' THEN entity_id ELSE NULL END,
    season_number,
    CASE WHEN entity_type = 'episode' THEN entity_id ELSE NULL END,
    is_season_pack, is_complete_series, target_slot_id, source,
    import_attempts, last_import_error, next_import_retry_at, created_at
FROM download_mappings;
DROP TABLE download_mappings;
ALTER TABLE download_mappings_old RENAME TO download_mappings;
CREATE INDEX idx_download_mappings_movie ON download_mappings(movie_id) WHERE movie_id IS NOT NULL;
CREATE INDEX idx_download_mappings_series ON download_mappings(series_id) WHERE series_id IS NOT NULL;
CREATE INDEX idx_download_mappings_episode ON download_mappings(episode_id) WHERE episode_id IS NOT NULL;

-- +goose StatementEnd
