## Phase 1: Database Architecture

**Goal:** Migrate shared tables to the discriminator pattern and establish per-module migration tracks. This is the highest-risk phase — it fundamentally changes the schema.

**Spec sections covered:** §2.2, §2.3, §2.4, Appendix B (§2.1 universal columns is deferred — existing movie/TV tables already have the required columns; enforcing them as a framework constraint happens when module CRUD interfaces are built in later phases)

### Phase 1 Architecture Decisions

1. **Single migration file:** All shared table discriminator changes go in ONE migration file (`070_module_discriminators.sql`). SQLite requires table recreation for constraint changes, and each recreation must happen in order (some tables have FK relationships). A single migration ensures atomicity.

2. **Per-module migration tracks (deferred):** The spec calls for separate Goose version tables per module. We define the infrastructure for this in Phase 1, but existing movie/TV tables do NOT move to separate tracks yet — they stay in the main migration stream. Per-module tracks will be used when new modules are added. Moving existing tables to separate tracks is a larger refactor that introduces risk without immediate value.

3. **Column naming convention:** The discriminator pattern uses `module_type` (not `media_type`) for the module discriminator, and `entity_type` + `entity_id` for the entity discriminator. Existing `media_type` columns are renamed to align.

4. **Backward-compatible query migration:** sqlc queries are updated to use the new column names. The generated Go code will have different parameter names, requiring callers to update. This is done in a separate task from the migration.

5. **`DeleteEntity` hook:** Implemented as a function in `internal/module/` that accepts `(ctx, db, moduleType, entityType, entityID)` and cascades deletions to all shared tables. Accepts a `DBTX` interface (`*sql.DB` or `*sql.Tx`) so callers with an existing transaction can pass it through — see Task 1.4. The function does NOT create its own transaction; callers are responsible for transaction management when atomicity is needed (most delete paths already use `db.BeginTx`).

6. **Frontend JSON shape preservation:** Phase 1 is a backend-only change. All HTTP API response shapes MUST remain identical to avoid frontend breakage. Where existing API responses include `movieId`, `seriesId`, `episodeId` (e.g., in queue items, download mappings), derive these from the new discriminator columns at the Go-to-JSON serialization layer. Frontend JSON shape changes are deferred to Phase 10.

7. **ON DELETE CASCADE removal:** The current `download_mappings` table has `ON DELETE CASCADE` FKs to `movies(id)`, `series(id)`, and `episodes(id)`. The migration removes these FKs (the new discriminator columns are plain integers, not FKs). This means **all code paths that delete movies/series/episodes MUST call `DeleteEntity()` to clean up shared table records**. Previously, SQLite's CASCADE did this automatically. Task 1.5 must audit all delete paths and ensure `DeleteEntity` is called. The movie service's `Delete()`, TV service's `DeleteSeries()`/`DeleteSeason()`, and any bulk delete operations are the key call sites.

### Phase 1 Dependency Graph

```
Task 1.1 (migration SQL) → Task 1.2 (sqlc queries) → Task 1.3 (sqlc generate)
                                                              ↓
Task 1.4 (DeleteEntity hook) ← depends on Task 1.3 for generated types
                                                              ↓
Task 1.5 (update Go callers) ← depends on Task 1.3 for new param names
                                                              ↓
Task 1.6 (status aggregation helpers) — independent, can parallel with 1.5
                                                              ↓
Task 1.7 (per-module migration infrastructure) — independent, can parallel with 1.5
                                                              ↓
Task 1.8 (validation)
```

---

### ~~Task 1.1: Discriminator Migration SQL~~ ✅

**Create** `internal/database/migrations/070_module_discriminators.sql`

This migration:

1. **`download_mappings`**: Add `module_type`, `entity_type`, `entity_id` columns. Populate from existing nullable FKs. Drop the nullable FK columns.

2. **`queue_media`**: Add `module_type`, `entity_type`, `entity_id` columns. Populate from existing nullable FKs. Drop the nullable FK columns.

3. **`downloads`**: Rename `media_type` → `module_type` (with data mapping: 'movie'→'movie', 'episode'→'tv'), rename `media_id` → `entity_id`, add `entity_type` (populated from original `media_type` values: 'movie' or 'episode'). Widen the CHECK constraint to remove the hardcoded values (allow any text — modules self-validate).

4. **`history`**: Rename `media_type` → `entity_type` (it already contains entity-level values like 'movie', 'episode', 'season'). Rename `media_id` → `entity_id`. Add `module_type` column, populate from entity_type ('movie' → 'movie', 'episode'/'season' → 'tv').

5. **`autosearch_status`**: Rename `item_type` → `entity_type`, rename `item_id` → `entity_id`. Add `module_type`, populate from entity_type.

6. **`import_decisions`**: Rename `media_type` → `entity_type`, rename `media_id` → `entity_id`. Add `module_type`, populate from entity_type.

7. **`root_folders`**: Rename `media_type` → `module_type`. Update CHECK constraint (use `module_type` name, keep same values for now but widen to allow future modules: remove CHECK or make it TEXT).

8. **`requests`**: Rename `media_type` → `entity_type`. Add `module_type`, populate from entity_type mapping ('movie' → 'movie', 'series'/'season'/'episode' → 'tv').

**Detailed SQL for each table:**

```sql
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
```

**Critical details for the agent writing this migration:**

1. **`download_mappings` entity_id derivation:** When `movie_id IS NOT NULL`, entity is `module_type='movie'`, `entity_type='movie'`, `entity_id=movie_id`. When `episode_id IS NOT NULL`, entity is `module_type='tv'`, `entity_type='episode'`, `entity_id=episode_id`. When only `series_id` is set (season packs or complete series), entity is `module_type='tv'`, `entity_type='series'`, `entity_id=series_id`. We use `entity_type='series'` (not `'season'`) because download_mappings never stored a season_id — only `(series_id, season_number)` pairs. The `season_number`, `is_season_pack`, and `is_complete_series` columns are retained as supplementary TV-specific columns for sub-level targeting.

2. **`queue_media` has a target_slot_id column** (added in migration 022). Include it.

3. **`download_mappings` has import retry columns** (added in migrations 063/064). Include them.

4. **`history` media_type already contains entity-level types** ('movie', 'episode', 'season'). The new `module_type` derives from: 'movie' → 'movie', 'episode'/'season' → 'tv'.

5. **`downloads` table media_type contains `'movie'` and `'episode'`**. Map `'episode'` → module_type 'tv', entity_type 'episode'.

6. **`root_folders` currently uses `'movie'` and `'tv'`**. These map directly to module_type. No constraint needed (allow any text for future modules).

7. **Write the DOWN migration** for each table recreation — full reverse. This is important for development workflow.

**Verify:** Apply migration to a test database. Verify all data is preserved. `make test` should detect any issues with the schema change.

---

### ~~Task 1.2: Update sqlc Query Files for New Column Names~~ ✅

All sqlc query files that reference the changed tables must be updated. This is a bulk update affecting many files.

**Files to update:**

1. **`internal/database/queries/download_mappings.sql`** — Replace all `movie_id`, `series_id`, `episode_id` references with `module_type`, `entity_type`, `entity_id`. Update all queries:
   - `CreateDownloadMapping`: params change from nullable FKs to discriminator columns
   - `GetDownloadingMovieIDs` → `GetDownloadingEntities` with `module_type` filter
   - `GetDownloadingSeriesData` → replace with generic entity query
   - `IsMovieDownloading` → `IsEntityDownloading(module_type, entity_type, entity_id)`
   - `IsSeriesDownloading`, `IsSeasonDownloading`, `IsEpisodeDownloading` → consolidated or kept with wrapper queries
   - `DeleteDownloadMappingsByMovieID` → `DeleteDownloadMappingsByEntity(module_type, entity_type, entity_id)`
   - `DeleteDownloadMappingsBySeriesID` → same pattern

2. **`internal/database/queries/queue_media.sql`** — Replace `movie_id`, `episode_id` with discriminator columns. Update all queries. **Critical:** The JOIN queries `GetReadyQueueMediaWithMapping` and `GetFailedQueueMediaWithMapping` currently SELECT `dm.series_id` from download_mappings — that column no longer exists. These JOINs must be rewritten to use `dm.module_type`, `dm.entity_type`, `dm.entity_id` instead.

3. **`internal/database/queries/autosearch.sql`** — Replace `item_type`/`item_id` with `module_type`/`entity_type`/`entity_id`. All queries need the extra `module_type` param. Update:
   - `GetAutosearchStatus`: add `module_type` param
   - `UpsertAutosearchStatus`: add `module_type` param
   - `IncrementAutosearchFailure`: add `module_type` param
   - `ResetAutosearchFailure`: add `module_type` param
   - `ResetAllAutosearchFailuresForItem`: add `module_type` param
   - `MarkAutosearchSearched`: add `module_type` param
   - `DeleteAutosearchStatus`: add `module_type` param
   - `DeleteAutosearchStatusForSeriesEpisodes`: rename `item_type` → `entity_type` in the WHERE clause (`item_type = 'episode'` → `entity_type = 'episode'`). The JOIN to `episodes` table is still valid.

4. **`internal/database/queries/requests.sql`** — Replace `media_type` with `module_type` + `entity_type`. This file has ~20 queries referencing `media_type`, many with hardcoded string literals like `media_type = 'season'`. Rename column references to `entity_type` (keeping the same string values 'movie', 'series', 'season', 'episode'). Add `module_type` parameter to queries that need it (e.g., `CreateRequest`, `CountUserAutoApprovedInPeriod`, `ListRequestsByMediaType`).

5. **`internal/database/queries/import_decisions.sql`** — Replace `media_type`/`media_id` with `module_type`/`entity_type`/`entity_id`. Note: `DeleteImportDecisionsByMediaItem` currently takes 2 params (`media_type`, `media_id`) and will expand to 3 (`module_type`, `entity_type`, `entity_id`).

6. **`internal/database/queries/settings.sql`** — **This is the grab-bag query file containing history, downloads, and root_folders queries.** Update:
   - **Root folder queries:** `ListRootFoldersByMediaType`, `ListRootFoldersByType`, `CreateRootFolder` — rename `media_type` param to `module_type`
   - **Downloads queries:** `CreateDownload` — rename `media_type`/`media_id` params to `module_type`/`entity_type`/`entity_id`
   - **History queries:** `CreateHistoryEntry` — add `module_type` param, rename `media_type` to `entity_type`. `ListHistoryByMedia` — add `module_type` param, rename column. `ListHistoryFiltered`, `CountHistoryFiltered` — update `media_type` filter to `entity_type`. `ListHistoryByMediaType`, `CountHistoryByMediaType` — rename to use `entity_type`. `HasRecentGrab` — rename `media_type` to `entity_type`. `HasRecentSeasonGrab` — update the JOIN condition `h.media_type = 'episode'` to `h.entity_type = 'episode'`.

   **Note:** `SELECT *` queries on downloads and history tables (e.g., `ListDownloads`, `ListHistoryFiltered`) will produce different sqlc-generated Go struct field names after column renames (e.g., `MediaType` → `ModuleType`/`EntityType`). Task 1.5 must update all callers accordingly.

7. **`internal/database/queries/movies.sql`** and **`internal/database/queries/series.sql`** — These do NOT JOIN or filter on `root_folders.media_type`. They reference `root_folder_id` as a plain FK without touching the discriminator column. **No changes needed for the discriminator migration.** The media-type constraint is implicit (movies use movie root folders, series use TV root folders) and enforced at the service layer, not in SQL.

**Query design guidance:**
- Where existing queries filter by specific movie/episode IDs, the new queries should accept `module_type`, `entity_type`, `entity_id` parameters
- Keep the UNIQUE constraints aligned: `download_mappings` is `UNIQUE(client_id, download_id)`, `autosearch_status` is `UNIQUE(module_type, entity_type, entity_id, search_type)`
- `queue_media` UNIQUE changes from `UNIQUE(download_mapping_id, episode_id)` / `UNIQUE(download_mapping_id, movie_id)` to `UNIQUE(download_mapping_id, entity_id)`

**Verify:** All `.sql` files are syntactically valid. Run `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` — it should generate without errors.

---

### ~~Task 1.3: Run sqlc Generate and Fix Compilation~~ ✅

After Task 1.2, run:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

This regenerates `internal/database/sqlc/`. The generated code will have different struct field names and method signatures. **Do not manually edit the generated files.**

Also update `sqlc.yaml` overrides:
- Remove overrides for dropped columns (`download_mappings.is_season_pack` etc. if column names changed)
- Add any new boolean column overrides if needed

**Verify:** `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` completes without errors. `go build ./internal/database/sqlc/...` compiles.

---

### ~~Task 1.4: Implement DeleteEntity Framework Hook~~ ✅

**Create** `internal/module/delete.go`

This implements the mandatory `DeleteEntity` hook from spec §2.3:

```go
package module

import (
    "context"
    "database/sql"
    "fmt"
)

// DBTX is the common interface satisfied by both *sql.DB and *sql.Tx.
// This allows DeleteEntity to work within an existing transaction or create its own.
type DBTX interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DeleteEntity cascades entity deletion to all shared tables.
// Modules MUST call this when deleting entities. Modules must not delete shared table
// records directly — all shared-table cleanup flows through this function.
//
// The db parameter accepts *sql.DB or *sql.Tx. When called with *sql.DB, no wrapping
// transaction is created — the caller is responsible for transaction management if needed.
// When called with *sql.Tx, the deletes participate in the caller's transaction.
//
// IMPORTANT: This function replaces the ON DELETE CASCADE behavior that was previously
// provided by FKs on download_mappings (movie_id, series_id, episode_id) and
// queue_media (movie_id, episode_id). All code paths that delete movies, series,
// seasons, or episodes MUST call this function.
func DeleteEntity(ctx context.Context, db DBTX, moduleType Type, entityType EntityType, entityID int64) error {
    mt := string(moduleType)
    et := string(entityType)

    // Delete from all shared tables that use the discriminator pattern.
    // Note: queue_media rows also cascade-delete via FK when their parent
    // download_mapping is deleted, but we delete them explicitly here first
    // to catch any queue_media rows whose entity differs from the mapping's
    // entity (e.g., episode-level queue rows under a season-pack mapping).
    standardDeletes := []string{
        "DELETE FROM download_mappings WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
        "DELETE FROM queue_media WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
        "DELETE FROM downloads WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
        "DELETE FROM history WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
        "DELETE FROM autosearch_status WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
        "DELETE FROM import_decisions WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
    }

    for _, query := range standardDeletes {
        if _, err := db.ExecContext(ctx, query, mt, et, entityID); err != nil {
            return fmt.Errorf("delete shared data: %w", err)
        }
    }

    // Requests table uses (module_type, entity_type, media_id) — note media_id not entity_id.
    // media_id stores the library entity ID once a request is fulfilled.
    // For root entities (movies, series), also delete child-level requests.
    if _, err := db.ExecContext(ctx,
        "DELETE FROM requests WHERE module_type = ? AND entity_type = ? AND media_id = ?",
        mt, et, entityID,
    ); err != nil {
        return fmt.Errorf("delete requests: %w", err)
    }
    // For root entities, cascade to child request types too (e.g., deleting a series
    // should also delete season/episode requests linked to that series via media_id).
    // This is safe because media_id for TV requests always holds the series.id,
    // so this catches all season/episode requests associated with the deleted entity.
    if _, err := db.ExecContext(ctx,
        "DELETE FROM requests WHERE module_type = ? AND media_id = ?",
        mt, entityID,
    ); err != nil {
        return fmt.Errorf("delete child requests: %w", err)
    }

    return nil
}
```

**Note on `requests` table:** The requests table uses `media_id` (not `entity_id`) for the linked entity. For TV requests, `media_id` always holds the `series.id` regardless of whether the request is for a series, season, or episode. The broad `DELETE FROM requests WHERE module_type = ? AND media_id = ?` query handles cascading all child-level requests when a root entity is deleted. Verify against the actual migration output.

**Note on transaction management:** `DeleteEntity` no longer creates its own transaction. Callers that need atomicity (most delete paths already use `db.BeginTx`) should pass the `*sql.Tx` directly. This avoids nested transaction issues. Callers that don't need a transaction can pass `*sql.DB`.

**Verify:** `go build ./internal/module/...` compiles. Write `delete_test.go` that creates test records in shared tables and verifies cascade deletion.

---

### ~~Task 1.5: Update Go Callers for New sqlc Signatures~~ ✅

This is the largest task in Phase 1. Every Go file that calls sqlc-generated functions against the changed tables needs updating.

**Strategy:** The agent should use `go build ./...` iteratively — each compilation error points to a caller that needs updating. Fix one package at a time.

**Packages that will have compilation errors (in rough dependency order):**

1. **`internal/history/service.go`** — Calls sqlc-generated functions (`CreateHistoryEntry`, `ListHistoryPaginated`, `ListHistoryFiltered`, `CountHistoryFiltered`, `ListHistoryByMedia`, `CountHistory`, `DeleteAllHistory`, etc.). The regenerated sqlc code will have new param/field names (`MediaType` → `EntityType`, new `ModuleType` field). Update all call sites to pass the new params. The `CreateInput` struct in `internal/history/types.go` needs a `ModuleType` field, and the `Entry` struct needs both `ModuleType` and `EntityType` (renamed from `MediaType`). All callers of `Create()`, `LogAutoSearchDownload()`, `LogStatusChanged()`, etc. must pass the new `ModuleType`.

2. **`internal/autosearch/`** — Calls autosearch_status queries. Update all `item_type`/`item_id` params to include `module_type`. The `Service` methods need to accept or derive `module_type`.

3. **`internal/import/`** (importer) — **This is the most complex package to update.** It has its own wrapper types in `service.go`:
   - `DownloadMapping` struct (line ~168): has `MediaType`, `MovieID *int64`, `SeriesID *int64`, `EpisodeID *int64` → change to `ModuleType`, `EntityType`, `EntityID int64`. Keep `SeasonNumber`, `IsSeasonPack`, `IsCompleteSeries` (still needed for TV season pack logic).
   - `QueueMedia` struct (line ~184): has `EpisodeID *int64`, `MovieID *int64` → change to `ModuleType`, `EntityType`, `EntityID int64`.
   - `LibraryMatch` struct (line ~196): has `MediaType`, `MovieID`, `SeriesID`, `SeasonNum`, `EpisodeID` → these are **internal** match types, NOT stored in the database. They can keep their current shape or be updated. The executing agent should assess whether changing them improves clarity vs. creates churn. If kept as-is, the mapping from `DownloadMapping` → `LibraryMatch` just needs updating.
   - `matchFromMapping()` in `matching.go`: uses `mapping.MovieID`, `mapping.SeriesID` etc. — must be rewritten to dispatch on `mapping.ModuleType`/`mapping.EntityType`.
   - All sqlc calls to download_mappings, queue_media, import_decisions need param updates.

4. **`internal/downloader/`** — `queue.go` has a `QueueItem` struct (line ~22) that is **sent to the frontend as JSON** with fields `movieId`, `seriesId`, `episodeId`. **Preserve the JSON shape in Phase 1** to avoid frontend changes: derive `movieId`/`seriesId`/`episodeId` from the new discriminator columns in the mapping enrichment code. The mapping logic in `queue.go` (line ~170) and `completion.go` (line ~176) that reads `mapping.MovieID.Valid` / `mapping.SeriesID.Valid` must be updated to dispatch on `mapping.ModuleType` / `mapping.EntityType` / `mapping.EntityID` instead.

5. **`internal/indexer/grab/`** — Creates download_mappings. Update params to use `ModuleType`, `EntityType`, `EntityID`.

6. **`internal/library/movies/service.go`** — Calls download_mappings and autosearch queries. Update to pass `module_type: "movie"`, `entity_type: "movie"`.

7. **`internal/library/tv/service.go`** — Similar updates with `module_type: "tv"`.

8. **`internal/library/rootfolder/service.go`** — References `media_type` column (now `module_type`). Update queries, validation, and the `mediaTypeMovie` constant.

9. **`internal/portal/requests/`** — Uses `media_type` on requests table. Update to `module_type` + `entity_type`. This package likely has the most query call sites — the `StatusTracker`, `LibraryChecker`, and `RequestSearcher` all reference media types.

10. **`internal/portal/provisioner/`** — Creates requests and checks library. Update discriminator columns.

**Note:** The following packages do NOT need Phase 1 changes (verified by grep):
- `internal/defaults/service.go` — Has its own `MediaType` constants (`"movie"`, `"tv"`) but uses them only for settings table keys (e.g., `default_rootFolder_movie`), NOT for discriminator table queries. No schema changes affect it.
- `internal/missing/` — queries movies/series tables directly, not shared discriminator tables
- `internal/availability/` — no `media_type` references
- `internal/rsssync/` — uses `decisioning.MediaType` in-memory, not DB columns (updated in Phase 5)
- `internal/calendar/` — `MediaType` is a JSON response field, not a DB column
- `internal/library/scanner/` — no `media_type` or `download_mapping` references
- `internal/library/slots/` — no direct `download_mapping` or `media_type` references (verify at implementation time — this package interacts with `version_slots` which has `movie_root_folder_id`/`tv_root_folder_id` columns that may be affected in later phases)
- `internal/portal/quota/` — no `media_type` references

**Approach for the executing agent:**

1. Run `go build ./...` and collect all errors
2. Group errors by package
3. Launch parallel subagents for packages with no mutual dependencies
4. Each subagent gets: the compilation errors for its package, the new sqlc-generated types (read from `internal/database/sqlc/`), and instructions to fix all callers
5. After all subagents complete, run `go build ./...` again to catch any remaining issues

**Key transformation pattern:**

Where code previously did:
```go
queries.CreateDownloadMapping(ctx, sqlc.CreateDownloadMappingParams{
    MovieID:   sql.NullInt64{Int64: movieID, Valid: true},
    SeriesID:  sql.NullInt64{},
    EpisodeID: sql.NullInt64{},
    ...
})
```

It becomes:
```go
queries.CreateDownloadMapping(ctx, sqlc.CreateDownloadMappingParams{
    ModuleType: "movie",
    EntityType: "movie",
    EntityID:   movieID,
    ...
})
```

For TV episodes:
```go
queries.CreateDownloadMapping(ctx, sqlc.CreateDownloadMappingParams{
    ModuleType: "tv",
    EntityType: "episode",
    EntityID:   episodeID,
    ...
})
```

**MediaType constant consolidation:** During this task, existing `MediaType` definitions across packages should be updated to use `module.Type` and `module.EntityType` where appropriate. However, this is a SOFT goal — if it creates too many import cycles, keep the local constants and add TODO comments for Phase 2+ cleanup.

**Verify:** `go build ./...` compiles with zero errors. `make test` passes (existing tests may need fixture updates for the new column names).

---

### ~~Task 1.6: Status Aggregation Helpers~~ ✅

**Create** `internal/module/status.go`

Implement the status aggregation helpers from spec §2.4:

```go
package module

// Status constants for media entities.
const (
    StatusUnreleased  = "unreleased"
    StatusMissing     = "missing"
    StatusDownloading = "downloading"
    StatusFailed      = "failed"
    StatusUpgradable  = "upgradable"
    StatusAvailable   = "available"
)

// statusPriority defines the priority order for status aggregation.
// Higher value = higher priority (wins in aggregation).
var statusPriority = map[string]int{
    StatusUnreleased:  0,
    StatusAvailable:   1,
    StatusUpgradable:  2,
    StatusMissing:     3,
    StatusFailed:      4,
    StatusDownloading: 5,
}

// AggregateStatus computes a parent node's status from its children's statuses.
// Uses the priority rule: downloading > failed > missing > upgradable > available > unreleased.
// Returns StatusUnreleased if no children are provided.
func AggregateStatus(childStatuses []string) string {
    if len(childStatuses) == 0 {
        return StatusUnreleased
    }

    highest := StatusUnreleased
    highestPrio := statusPriority[highest]

    for _, s := range childStatuses {
        p, ok := statusPriority[s]
        if ok && p > highestPrio {
            highest = s
            highestPrio = p
        }
    }

    return highest
}
```

This function can be called by both movie and TV services when computing parent status from children. It consolidates the logic currently duplicated in both services.

**Also note:** The existing `internal/library/status/status.go` has the same constants. The new `module.Status*` constants should eventually replace them, but for Phase 1, both can coexist. Add a deprecation comment in `internal/library/status/status.go`:

```go
// Deprecated: Use module.Status* constants instead. Will be removed when movie/TV services
// are refactored to use the module framework (Phase 4+ when MonitoringPresets is implemented).
```

Existing code continues to import from `internal/library/status/`. The migration to `module.Status*` happens organically as services are refactored in later phases — no bulk rename in Phase 1.

**Verify:** `go build ./internal/module/...` compiles. Write `status_test.go` testing aggregation with various child status combinations.

---

### ~~Task 1.7: Per-Module Migration Infrastructure~~ ✅

**Create** `internal/module/migrate.go`

This provides helpers for modules to have their own migration tracks (spec §2.2). In Phase 1, this is infrastructure only — existing tables stay in the main migration stream.

```go
package module

import (
    "context"
    "database/sql"
    "embed"
    "fmt"
    "io/fs"

    "github.com/pressly/goose/v3"
)

// MigrateModule runs migrations for a specific module using its own
// embedded migration files and version tracking table.
//
// Each module gets a separate goose version table (e.g., goose_movie_version,
// goose_tv_version) so modules can migrate independently.
//
// Uses goose.NewProvider() (instance-based API) to avoid global state issues.
// This is safe to call for multiple modules without concurrency concerns.
//
// The dir parameter specifies the subdirectory within the embedded FS that
// contains the .sql files (e.g., "migrations").
func MigrateModule(db *sql.DB, moduleType Type, migrations embed.FS, dir string) error {
    tableName := fmt.Sprintf("goose_%s_version", moduleType)

    subFS, err := fs.Sub(migrations, dir)
    if err != nil {
        return fmt.Errorf("sub filesystem for module %s migrations: %w", moduleType, err)
    }

    provider, err := goose.NewProvider(
        goose.DialectSQLite3,
        db,
        subFS,
        goose.WithTableName(tableName),
    )
    if err != nil {
        return fmt.Errorf("create migration provider for module %s: %w", moduleType, err)
    }

    if _, err := provider.Up(context.Background()); err != nil {
        return fmt.Errorf("migrate module %s: %w", moduleType, err)
    }

    return nil
}
```

**Note:** This uses the instance-based `goose.NewProvider()` API (available since goose v3.20; we have v3.26.0). This avoids the fragile global state approach (`goose.SetTableName`, `goose.SetBaseFS`) that would require sequential calls and could leave bad state on panics.

**Verify:** `go build ./internal/module/...` compiles.

---

### ~~Task 1.8: Update Test Fixtures~~ ✅

After the migration, test helpers and fixtures need updating.

**Files likely affected:**

1. **`internal/library/movies/service_test.go`** and **`status_test.go`** — If tests create download_mappings or autosearch_status records, they need the new column names.

2. **`internal/library/tv/` test files** — Same.

3. **`internal/import/` test files** — Download mapping creation in tests.

4. **`internal/autosearch/` test files** — Autosearch status creation.

5. **`internal/portal/requests/` test files** — Request creation with new column names.

6. **`internal/testutil/`** — If shared test helpers exist for creating test data.

**Approach:** Run `make test` and fix all failures. The agent should run tests package-by-package to isolate failures.

**Verify:** `make test` passes with zero failures.

---

### ~~Task 1.9: Phase 1 Validation~~ ✅

**Run all of these after all Phase 1 tasks complete:**

1. `make build` — full project builds
2. `make test` — all tests pass
3. `make lint` — no new lint issues
4. Manual verification: check the migration applies cleanly to a fresh database AND to a database with existing data (use the dev mode database as a test)

**What exists after Phase 1:**
- Migration `070_module_discriminators.sql` that transforms all shared tables to use `(module_type, entity_type, entity_id)` discriminator pattern
- Updated sqlc queries and generated Go code for new column names
- All Go callers updated for new sqlc signatures
- `DeleteEntity` framework hook in `internal/module/`
- `AggregateStatus` helper in `internal/module/`
- Per-module migration infrastructure in `internal/module/`
- Updated `internal/library/status/status.go` with deprecation notice
- Zero functional regressions — the app behaves identically to before

---

## Appendix: File Inventory

### New files created in Phase 0
```
internal/module/schema.go
internal/module/schema_test.go
internal/module/types.go
internal/module/interfaces.go
internal/module/optional_interfaces.go
internal/module/search_types.go
internal/module/import_types.go
internal/module/calendar_types.go
internal/module/quality_types.go
internal/module/notification_types.go
internal/module/path_types.go
internal/module/task_types.go
internal/module/parse_types.go
internal/module/arr_types.go
internal/module/slot_types.go
internal/module/module.go
internal/module/module_test.go
internal/module/settings.go
internal/module/register.go
internal/modules/movie/module.go
internal/modules/tv/module.go
```

### New files created in Phase 1
```
internal/database/migrations/070_module_discriminators.sql
internal/module/delete.go
internal/module/delete_test.go
internal/module/status.go
internal/module/status_test.go
internal/module/migrate.go
```

### Existing files modified in Phase 1
```
sqlc.yaml (column override updates)
internal/database/queries/settings.sql (root_folders, downloads, history queries — all use media_type)
internal/database/queries/download_mappings.sql
internal/database/queries/queue_media.sql
internal/database/queries/autosearch.sql
internal/database/queries/requests.sql
internal/database/queries/import_decisions.sql
internal/database/sqlc/* (regenerated)
internal/history/service.go (sqlc call site updates for renamed columns)
internal/history/types.go (add ModuleType to CreateInput and Entry)
internal/autosearch/*.go (new params)
internal/import/*.go (new params)
internal/downloader/*.go (new params)
internal/indexer/grab/*.go (new params)
internal/library/movies/service.go (new params)
internal/library/tv/service.go (new params)
internal/library/rootfolder/service.go (media_type → module_type)
internal/library/status/status.go (deprecation comment)
internal/portal/requests/*.go (media_type → module_type + entity_type)
internal/portal/provisioner/*.go (new params)
Various test files (fixture updates)
```

---

