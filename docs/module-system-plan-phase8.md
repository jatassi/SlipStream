## Phase 8: Portal & Version Slots

**Goal:** Replace hard-coded movie/TV branching in the portal system with module-dispatched provisioning, generic hierarchy-based request completion tracking, and module-scoped user settings. Replace per-module FK columns on `version_slots` with a `slot_root_folders` pivot table.

**Spec sections covered:** §11.1 (PortalProvisioner interface, request completion inference via node hierarchy), §11.2 (portal_user_module_settings consolidating 3 source tables, portal_invitation_module_settings), §2.5 (SlotSupport interface, slot_root_folders pivot table replacing per-module columns)

**Spec items deferred from Phase 8:**
- §11.1 duplicate detection via external IDs — deferred; the framework currently handles duplicates via per-media-type request queries (`GetActiveRequestByTmdbID`, `GetActiveRequestByTvdbIDAndSeason`, etc.). Generic cross-entity-type duplicate detection using module external IDs will be added in Phase 10+ when a 3rd module makes the current approach insufficient.
- §2.3 `module_type` column on `requests` table — deferred; the existing `media_type` column already serves as `entity_type`, and `module_type` is derivable from it via the registry. Adding a redundant column is deferred until a 3rd module makes the derivation ambiguous or a query performance need arises.
- §11.1 `ValidateRequest` uses bare params (`entityType string, externalIDs map[string]int64`) instead of the spec's `*RequestValidationInput` — this is simpler for the current two modules. Can be wrapped in an input struct later if the param list grows.

**Deferred from Phase 2 (Quality System):**
- `version_slots.quality_profile_id` currently points to "movie" profile copies (Phase 2 preserved original IDs as movie copies). After the `version_slots` table recreation in this phase, verify slot quality profile assignments remain correct. If slots should support per-module quality profiles, add a `slot_quality_profiles` pivot table (analogous to `slot_root_folders`). If one profile per slot is sufficient (the slot is inherently module-scoped), no change needed — just verify FK integrity.
- The queries `ListMovieSlotAssignmentsWithFilesForProfile` and `ListEpisodeSlotAssignmentsWithFilesForProfile` in `internal/database/queries/slots.sql` exist but are never called by Go code. Wire them into the slot service for profile-based recalculation, or remove them if the slot service doesn't need them.

**Depends on:** Phase 0 (module types, interfaces, registry), Phase 1 (discriminator pattern — `module_type` column convention), Phase 2 (module-scoped quality profiles — `quality_profiles.module_type`), Phase 3 (MetadataProvider — modules own their own metadata lookup), Phase 5 (SearchStrategy — modules own search dispatch), Phase 6 (ImportHandler — modules own import), Phase 7 (BroadcastEntity — entity event broadcasting)

### Phase 8 Context Management

**Parallelism map** (tasks with no dependency arrow can run concurrently):

```
Group A (parallel):  8.1, 8.5, 8.9
Group B (after A):   8.2 (needs 8.1), 8.6 (needs 8.5), 8.10 (needs 8.9)
Group C (after B):   8.3 (needs 8.2), 8.7 (needs 8.6)
Group D (after C):   8.4 (needs 8.3), 8.8 (needs 8.7), 8.11 (needs 8.10)
Group E (after D):   8.12 (needs 8.4, 8.8, 8.11)
Group F (after E):   8.13 (needs 8.12)
Group G:             8.14 (validation — needs all)
```

**Delegation notes:**
- Tasks 8.1 + 8.2 (migration + sqlc for portal user module settings) should be one subagent — the sqlc queries depend on the new schema.
- Tasks 8.5 + 8.6 (migration + sqlc for slot_root_folders) should be one subagent.
- Tasks 8.9 + 8.10 (PortalProvisioner interface + module implementations) should be one subagent — the implementations depend on the interface.
- Task 8.12 (StatusTracker rewrite) is complex — dedicate a single subagent with access to the current `status_tracker.go` and the new interfaces.
- Task 8.13 (frontend) is large — dedicate a single subagent.

**Validation checkpoints:** After Groups B, D, and E, run `go build ./...` to catch compile errors early. Full validation in 8.14.

---

### Phase 8 Architecture Decisions

1. **`portal_user_module_settings` consolidates three source tables.** The spec (§11.2) calls for a single table replacing per-module columns spread across `portal_users` (quality profiles), `user_quotas` (limits/usage), and `portal_invitations` (quality profiles). The new table has one row per user per module with columns: `user_id`, `module_type`, `quota_limit` (nullable = use global default), `quota_used`, `quality_profile_id` (nullable = use module default), `period_start`.

   **Note — quota_limit NULL semantics deviate from the spec.** The spec (§11.2) says `quota_limit` is "nullable = unlimited". However, the current `user_quotas` table already uses NULL to mean "use global default" and 0 to mean "unlimited". We preserve this semantic to maintain backward compatibility with existing user data and admin expectations. The plan's semantics: `NULL` = use global default (`requests_default_{module_type}_quota` setting), `0` = unlimited, `>0` = explicit limit.

   **Migration mapping from current schema:**
   | Source | Destination |
   |---|---|
   | `portal_users.movie_quality_profile_id` | row `module_type='movie'`, `quality_profile_id` |
   | `portal_users.tv_quality_profile_id` | row `module_type='tv'`, `quality_profile_id` |
   | `user_quotas.movies_limit` / `movies_used` | row `module_type='movie'`, `quota_limit` / `quota_used` |
   | `user_quotas.seasons_limit` / `seasons_used` | row `module_type='tv'`, `quota_limit` / `quota_used` (seasons and episodes collapse into one TV quota) |
   | `user_quotas.episodes_limit` / `episodes_used` | **dropped** — TV quota is unified at the module level (see decision 2) |
   | `user_quotas.period_start` | copied to both movie and TV rows' `period_start` |

   After migration, `portal_users.movie_quality_profile_id` and `portal_users.tv_quality_profile_id` columns are dropped. The `user_quotas` table is dropped entirely.

2. **Quota simplification — one quota per module.** The current system has three separate quotas: movies, seasons, episodes. The module system unifies this to **one quota per module per user**. For the TV module, a single `quota_limit` replaces both `seasons_limit` and `episodes_limit`. The migration uses the *lower* of `seasons_limit` and `episodes_limit` as the initial TV quota, and the *higher* of `seasons_used` and `episodes_used` as the initial used count. This is a reasonable default since most users have similar limits for both.

   The global default settings also simplify: `requests_default_movie_quota` remains, `requests_default_season_quota` and `requests_default_episode_quota` merge into `requests_default_tv_quota`. A new setting key pattern `requests_default_{module_type}_quota` scales to N modules.

3. **`portal_invitation_module_settings` table.** Invitations currently store `movie_quality_profile_id` and `tv_quality_profile_id` directly on the `portal_invitations` table. These columns are migrated to a new `portal_invitation_module_settings` table with `(invitation_id, module_type, quality_profile_id)` rows. When an invitation is accepted, the framework creates `portal_user_module_settings` rows from the invitation's module settings.

4. **`PortalProvisioner` interface — module-provided.** The current `provisioner.Service` has hard-coded `EnsureMovieInLibrary` and `EnsureSeriesInLibrary` methods with direct imports of `movies.Service` and `tv.Service`. The spec (§11.1) defines `PortalProvisioner` with 3 methods: `EnsureInLibrary`, `CheckAvailability`, `ValidateRequest`. This plan extends the interface with 2 additional methods beyond the spec: `CheckRequestCompletion` (needed for the module-dispatched StatusTracker pattern — see decision 6) and `SupportedEntityTypes` (needed for entity-type-to-module routing without hardcoded mappings). The framework dispatches to the correct module's provisioner based on request `media_type` → `module_type` mapping via `SupportedEntityTypes`.

   **Current `provisioner.Service` → decomposition:**
   - `EnsureMovieInLibrary` → `movie.Module.EnsureInLibrary`
   - `EnsureSeriesInLibrary` (+ monitoring logic) → `tv.Module.EnsureInLibrary`
   - `getDefaultSettings` → framework helper (shared across modules)
   - `resolveRootFolderID` → framework helper

5. **`MediaProvisioner` interface replacement.** The current `requests.MediaProvisioner` interface in `searcher.go` has `EnsureMovieInLibrary` and `EnsureSeriesInLibrary`. This is replaced with a single `EnsureInLibrary(ctx, moduleType, input) (entityID, error)` method. The `RequestSearcher.ensureMediaInLibrary` method dispatches to the correct module by looking up the module type from the request's `media_type`. The mapping is: `"movie"` → module `"movie"`, `"series"/"season"/"episode"` → module `"tv"`.

6. **StatusTracker genericization via node hierarchy.** The current `StatusTracker` has movie-specific and TV-specific methods (`OnMovieAvailable`, `OnEpisodeAvailable`, `OnSeasonComplete`, `OnSeriesComplete`) with hard-coded lookup interfaces (`MovieLookup`, `EpisodeLookup`, `SeriesLookup`). The spec (§11.1) replaces this with a generic hierarchy walk.

   **New approach:** The `StatusTracker` receives a single `OnEntityAvailable(ctx, moduleType, entityType, entityID)` call. It:
   1. Finds active requests matching this entity (by external IDs via the module's provisioner).
   2. If the entity is a leaf node, marks direct leaf-level requests as available.
   3. Walks up the node hierarchy: for each ancestor level, checks if all children of a requested parent are available (via the module's `CheckAvailability`).
   4. If all children are available, marks the parent request as available.

   This replaces the three lookup interfaces with a single `PortalProvisioner.CheckAvailability` call per module. The existing TV-specific `AreSeasonsComplete` logic moves into the TV module's `CheckAvailability` implementation.

   **Transitional approach:** Rather than a full rewrite to generic hierarchy walking (which requires node schema introspection at runtime), Phase 8 implements a **module-dispatched** pattern: `StatusTracker.OnEntityAvailable` dispatches to the module's `PortalProvisioner.CheckAvailability` which returns whether the request should be marked available. This is simpler and sufficient for movie and TV. The fully generic hierarchy walk (spec §11.1 "framework walks up the node tree") is deferred to Phase 10+ when a 3rd module with a new hierarchy is added.

7. **`slot_root_folders` pivot table replaces per-module FK columns.** The current `version_slots` table has `movie_root_folder_id` and `tv_root_folder_id` columns. The spec (§2.5) replaces these with a `slot_root_folders` pivot table: `(slot_id, module_type, root_folder_id)`. One row per slot per module. This scales to N modules without schema changes.

   **Migration:** For each existing slot, if `movie_root_folder_id IS NOT NULL`, insert a row `(slot_id, 'movie', movie_root_folder_id)`. Same for `tv_root_folder_id`. Then drop the two columns from `version_slots` (requires SQLite table recreation).

8. **`LibraryChecker` refactoring.** The current `LibraryChecker` in `requests/library_check.go` has separate `CheckMovieAvailability`, `CheckSeriesAvailability`, `CheckSeasonAvailability`, `CheckEpisodeAvailability` methods with hard-coded sqlc queries. This is refactored to dispatch to each module's `PortalProvisioner.CheckAvailability`. The `LibraryChecker` becomes a thin framework orchestrator that:
   - Maps `media_type` to `module_type`
   - Calls the module's `CheckAvailability`
   - Checks for existing requests (framework responsibility, unchanged)

9. **Request `media_type` values remain unchanged (spec deviation — see deferred items above).** The `requests` table already uses `media_type IN ('movie', 'series', 'season', 'episode')`. These are entity types within modules. The framework maps them to module types: `'movie'` → module `'movie'`, `'series'/'season'/'episode'` → module `'tv'`. The spec (§2.3) calls for explicit `module_type` + `entity_type` columns on the `requests` table, but `media_type` already serves as `entity_type` and `module_type` is derivable from it via the registry's `SupportedEntityTypes`. Adding a redundant `module_type` column is deferred.

10. **`SlotSupport` interface — minimal Phase 8 scope.** The spec (§2.5) defines `SlotSupport` as an optional module interface with `SlotEntityType()` and `SlotTableSchema()`. In Phase 8, we implement the `slot_root_folders` pivot table and update the slot service to use it. The full `SlotSupport` interface (framework-generated slot assignment tables) is **deferred** — the existing `movie_slot_assignments` and `episode_slot_assignments` tables continue to work. The slot queries in `slots.sql` that reference `movie_files`/`episode_files`/`movies`/`episodes` directly are not changed — they will be genericized when a 3rd module adds slot support.

    Phase 8 scope for slots is limited to:
    - `slot_root_folders` pivot table migration
    - `SlotSupport` interface definition on module framework
    - Movie and TV modules declare `SlotSupport` (returning their entity types)
    - Slot service `GetRootFolderForSlot` reads from pivot table instead of FK columns
    - Slot service `SetRootFolders` writes to pivot table
    - Slot `UpdateSlotInput` and handlers updated for module-keyed root folders
    - Frontend slot settings updated for dynamic module root folders

11. **`auto_approve` column stays on `portal_users`.** The `auto_approve` boolean is a user-level setting (not per-module). A user who has auto-approve gets it for all modules. This is simpler than per-module auto-approve and matches the current UX. The column remains on `portal_users`.

12. **Quota consumption by entity type → module type mapping.** The current `autoapprove.Service.getQuotaMediaType` maps `"movie"` → `"movie"`, `"series"/"season"` → `"season"`, `"episode"` → `"episode"`. With unified per-module quotas, this simplifies to: `"movie"` → module `"movie"`, everything else → module `"tv"`. The quota service's `CheckQuota` and `ConsumeQuota` methods take `moduleType` instead of `mediaType`.

---

### Phase 8 Dependency Graph

```
Task 8.1 (migration: portal_user_module_settings) ──→ Task 8.2 (sqlc queries)
                                                                    ↓
                                                           Task 8.3 (quota + users service)
                                                                    ↓
                                                           Task 8.4 (handlers + bridge updates)

Task 8.5 (migration: slot_root_folders) ─────────────→ Task 8.6 (sqlc queries for slots)
                                                                    ↓
                                                           Task 8.7 (slot service refactor)
                                                                    ↓
                                                           Task 8.8 (slot handler + bridge updates)

Task 8.9 (PortalProvisioner interface) ──────────────→ Task 8.10 (movie + TV provisioner impls)
                                                                    ↓
                                                           Task 8.11 (RequestSearcher + provisioner dispatch)

                          8.4, 8.8, 8.11 ──→ Task 8.12 (StatusTracker rewrite)
                                                       ↓
                                              Task 8.13 (frontend updates)
                                                       ↓
                                              Task 8.14 (phase validation)
```

**Parallelism:** Tasks 8.1, 8.5, and 8.9 can all run in parallel (migration SQL, slot migration SQL, and interface definitions — no file overlap).

---

### Task 8.1: Migration — Portal User Module Settings & Invitation Module Settings

**Create** `internal/database/migrations/0XX_portal_user_module_settings.sql` (use next available migration number)

This migration:
1. Creates `portal_user_module_settings` with `(user_id, module_type)` composite PK
2. Migrates data from `portal_users` (quality profiles) and `user_quotas` (limits/usage)
3. Creates `portal_invitation_module_settings` with `(invitation_id, module_type)` composite PK
4. Migrates data from `portal_invitations` (quality profiles)
5. Drops `user_quotas` table
6. Recreates `portal_users` without `movie_quality_profile_id` / `tv_quality_profile_id`
7. Recreates `portal_invitations` without `movie_quality_profile_id` / `tv_quality_profile_id`
8. Migrates global quota settings keys

```sql
-- +goose Up
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- 1. Create portal_user_module_settings
CREATE TABLE portal_user_module_settings (
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    quota_limit INTEGER,          -- NULL = use global default, 0 = unlimited
    quota_used INTEGER NOT NULL DEFAULT 0,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
    period_start DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, module_type)
);

-- 2. Migrate quality profiles from portal_users + quotas from user_quotas
-- Movie rows: quality profile from portal_users, quota from user_quotas
INSERT INTO portal_user_module_settings (user_id, module_type, quota_limit, quota_used, quality_profile_id, period_start)
SELECT
    pu.id,
    'movie',
    uq.movies_limit,
    COALESCE(uq.movies_used, 0),
    pu.movie_quality_profile_id,
    COALESCE(uq.period_start, CURRENT_TIMESTAMP)
FROM portal_users pu
LEFT JOIN user_quotas uq ON pu.id = uq.user_id
WHERE pu.is_admin = 0;

-- TV rows: quality profile from portal_users, quota = min of seasons/episodes limits
INSERT INTO portal_user_module_settings (user_id, module_type, quota_limit, quota_used, quality_profile_id, period_start)
SELECT
    pu.id,
    'tv',
    CASE
        WHEN uq.seasons_limit IS NULL AND uq.episodes_limit IS NULL THEN NULL
        WHEN uq.seasons_limit IS NULL THEN uq.episodes_limit
        WHEN uq.episodes_limit IS NULL THEN uq.seasons_limit
        ELSE MIN(uq.seasons_limit, uq.episodes_limit)
    END,
    CASE
        WHEN uq.seasons_used IS NULL AND uq.episodes_used IS NULL THEN 0
        ELSE MAX(COALESCE(uq.seasons_used, 0), COALESCE(uq.episodes_used, 0))
    END,
    pu.tv_quality_profile_id,
    COALESCE(uq.period_start, CURRENT_TIMESTAMP)
FROM portal_users pu
LEFT JOIN user_quotas uq ON pu.id = uq.user_id
WHERE pu.is_admin = 0;

-- Also create rows for users who have quality profiles but no quota record
-- (the LEFT JOIN above handles this — uq columns are NULL, which is correct)

-- 3. Create portal_invitation_module_settings
CREATE TABLE portal_invitation_module_settings (
    invitation_id INTEGER NOT NULL REFERENCES portal_invitations(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
    PRIMARY KEY (invitation_id, module_type)
);

-- 4. Migrate invitation quality profiles
INSERT INTO portal_invitation_module_settings (invitation_id, module_type, quality_profile_id)
SELECT id, 'movie', movie_quality_profile_id
FROM portal_invitations
WHERE movie_quality_profile_id IS NOT NULL;

INSERT INTO portal_invitation_module_settings (invitation_id, module_type, quality_profile_id)
SELECT id, 'tv', tv_quality_profile_id
FROM portal_invitations
WHERE tv_quality_profile_id IS NOT NULL;

-- 5. Drop user_quotas table
DROP TABLE IF EXISTS user_quotas;

-- 6. Recreate portal_users without per-module quality profile columns
CREATE TABLE portal_users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    auto_approve INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO portal_users_new (id, username, password_hash, auto_approve, enabled, is_admin, created_at, updated_at)
SELECT id, username, password_hash, auto_approve, enabled, is_admin, created_at, updated_at
FROM portal_users;

DROP TABLE portal_users;
ALTER TABLE portal_users_new RENAME TO portal_users;

CREATE INDEX idx_portal_users_username ON portal_users(username);
CREATE INDEX idx_portal_users_enabled ON portal_users(enabled);

-- 7. Recreate portal_invitations without per-module quality profile columns
CREATE TABLE portal_invitations_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    auto_approve INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO portal_invitations_new (id, username, token, expires_at, used_at, auto_approve, created_at)
SELECT id, username, token, expires_at, used_at, auto_approve, created_at
FROM portal_invitations;

DROP TABLE portal_invitations;
ALTER TABLE portal_invitations_new RENAME TO portal_invitations;

CREATE INDEX idx_portal_invitations_token ON portal_invitations(token);
CREATE INDEX idx_portal_invitations_username ON portal_invitations(username);

-- 8. Migrate global quota settings keys
-- Merge season + episode defaults into a single TV quota default
-- Use the lower of the two as the new TV default
INSERT OR REPLACE INTO settings (key, value)
SELECT
    'requests_default_tv_quota',
    CAST(MIN(
        CAST((SELECT value FROM settings WHERE key = 'requests_default_season_quota') AS INTEGER),
        CAST((SELECT value FROM settings WHERE key = 'requests_default_episode_quota') AS INTEGER)
    ) AS TEXT)
WHERE EXISTS (SELECT 1 FROM settings WHERE key = 'requests_default_season_quota')
   OR EXISTS (SELECT 1 FROM settings WHERE key = 'requests_default_episode_quota');

DELETE FROM settings WHERE key IN ('requests_default_season_quota', 'requests_default_episode_quota');

PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Down migration is destructive — cannot fully reverse quota split
-- Recreate user_quotas table and portal_users columns for rollback

PRAGMA foreign_keys = OFF;

CREATE TABLE user_quotas (
    user_id INTEGER PRIMARY KEY REFERENCES portal_users(id) ON DELETE CASCADE,
    movies_limit INTEGER,
    seasons_limit INTEGER,
    episodes_limit INTEGER,
    movies_used INTEGER NOT NULL DEFAULT 0,
    seasons_used INTEGER NOT NULL DEFAULT 0,
    episodes_used INTEGER NOT NULL DEFAULT 0,
    period_start DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Restore user_quotas from portal_user_module_settings
INSERT INTO user_quotas (user_id, movies_limit, movies_used, seasons_limit, seasons_used, episodes_limit, episodes_used, period_start)
SELECT
    m.user_id,
    m.quota_limit,
    m.quota_used,
    t.quota_limit,
    t.quota_used,
    t.quota_limit,
    t.quota_used,
    m.period_start
FROM portal_user_module_settings m
LEFT JOIN portal_user_module_settings t ON m.user_id = t.user_id AND t.module_type = 'tv'
WHERE m.module_type = 'movie';

-- Recreate portal_users with quality profile columns
ALTER TABLE portal_users ADD COLUMN movie_quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;
ALTER TABLE portal_users ADD COLUMN tv_quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;

UPDATE portal_users SET
    movie_quality_profile_id = (SELECT quality_profile_id FROM portal_user_module_settings WHERE user_id = portal_users.id AND module_type = 'movie'),
    tv_quality_profile_id = (SELECT quality_profile_id FROM portal_user_module_settings WHERE user_id = portal_users.id AND module_type = 'tv');

-- Recreate portal_invitations with quality profile columns
ALTER TABLE portal_invitations ADD COLUMN movie_quality_profile_id INTEGER;
ALTER TABLE portal_invitations ADD COLUMN tv_quality_profile_id INTEGER;

UPDATE portal_invitations SET
    movie_quality_profile_id = (SELECT quality_profile_id FROM portal_invitation_module_settings WHERE invitation_id = portal_invitations.id AND module_type = 'movie'),
    tv_quality_profile_id = (SELECT quality_profile_id FROM portal_invitation_module_settings WHERE invitation_id = portal_invitations.id AND module_type = 'tv');

DROP TABLE IF EXISTS portal_user_module_settings;
DROP TABLE IF EXISTS portal_invitation_module_settings;

-- Restore old settings keys
INSERT OR REPLACE INTO settings (key, value)
SELECT 'requests_default_season_quota', value FROM settings WHERE key = 'requests_default_tv_quota';
INSERT OR REPLACE INTO settings (key, value)
SELECT 'requests_default_episode_quota', value FROM settings WHERE key = 'requests_default_tv_quota';
DELETE FROM settings WHERE key = 'requests_default_tv_quota';

PRAGMA foreign_keys = ON;

-- +goose StatementEnd
```

**Critical considerations:**
- `PRAGMA foreign_keys = OFF` is required during table drops/renames because `portal_user_module_settings` references `portal_users(id)` and we're recreating that table.
- Admin users (`is_admin = 1`) are excluded from `portal_user_module_settings` — they don't have quotas or quality profile overrides.
- The `quality_profiles(id)` FK on `portal_user_module_settings` references module-scoped profiles (from Phase 2). Movie rows should reference movie profiles, TV rows should reference TV profiles. Since Phase 2 preserved original IDs as movie copies and created new IDs for TV copies, the migrated data is already correct — `portal_users.movie_quality_profile_id` already points to a movie profile, `tv_quality_profile_id` already points to a TV profile.
- `request_watchers` table has a FK to `portal_users(id) ON DELETE CASCADE`. The table recreation must preserve IDs so this FK remains valid. `PRAGMA foreign_keys = OFF` prevents cascading deletes during the DROP/RENAME.
- `portal_user_module_settings` (being created in the same migration) references `portal_users(id)`. Since it's created before the portal_users recreation and data is migrated before the DROP, its rows' `user_id` values will match the recreated table's IDs.
- Passkey credentials table (`passkey_credentials`) references `portal_users(id)` via its `user_id` column (application-level reference, no formal FK constraint in the schema). The table recreation must preserve IDs so existing passkey credentials remain valid.
- `requests` table has `user_id` referencing `portal_users(id)` — same concern, handled by ID preservation.

**Verify:**
- Migration applies cleanly on a fresh DB and on a DB with existing portal users, quotas, and invitations
- `portal_user_module_settings` has correct rows for existing users
- `portal_invitation_module_settings` has correct rows for existing invitations
- `user_quotas` table no longer exists
- `portal_users` no longer has `movie_quality_profile_id` / `tv_quality_profile_id`
- `portal_invitations` no longer has `movie_quality_profile_id` / `tv_quality_profile_id`

---

### Task 8.2: sqlc Queries for Portal User Module Settings & Invitation Module Settings

**Create** `internal/database/queries/portal_user_module_settings.sql`

```sql
-- name: GetUserModuleSettings :one
SELECT * FROM portal_user_module_settings
WHERE user_id = ? AND module_type = ? LIMIT 1;

-- name: ListUserModuleSettings :many
SELECT * FROM portal_user_module_settings
WHERE user_id = ?
ORDER BY module_type;

-- name: UpsertUserModuleSettings :one
INSERT INTO portal_user_module_settings (
    user_id, module_type, quota_limit, quota_used, quality_profile_id, period_start
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, module_type) DO UPDATE SET
    quota_limit = excluded.quota_limit,
    quota_used = excluded.quota_used,
    quality_profile_id = excluded.quality_profile_id,
    period_start = excluded.period_start,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateUserModuleQuotaLimit :one
UPDATE portal_user_module_settings SET
    quota_limit = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: UpdateUserModuleQualityProfile :one
UPDATE portal_user_module_settings SET
    quality_profile_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: IncrementUserModuleQuota :one
UPDATE portal_user_module_settings SET
    quota_used = quota_used + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: ResetUserModuleQuota :one
UPDATE portal_user_module_settings SET
    quota_used = 0,
    period_start = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND module_type = ?
RETURNING *;

-- name: ResetAllModuleQuotas :exec
UPDATE portal_user_module_settings SET
    quota_used = 0,
    period_start = ?,
    updated_at = CURRENT_TIMESTAMP;

-- name: DeleteUserModuleSettings :exec
DELETE FROM portal_user_module_settings WHERE user_id = ?;

-- name: DeleteUserModuleSettingsForModule :exec
DELETE FROM portal_user_module_settings WHERE user_id = ? AND module_type = ?;
```

**Create** `internal/database/queries/portal_invitation_module_settings.sql`

```sql
-- name: GetInvitationModuleSettings :many
SELECT * FROM portal_invitation_module_settings
WHERE invitation_id = ?
ORDER BY module_type;

-- name: UpsertInvitationModuleSetting :one
INSERT INTO portal_invitation_module_settings (invitation_id, module_type, quality_profile_id)
VALUES (?, ?, ?)
ON CONFLICT(invitation_id, module_type) DO UPDATE SET
    quality_profile_id = excluded.quality_profile_id
RETURNING *;

-- name: DeleteInvitationModuleSettings :exec
DELETE FROM portal_invitation_module_settings WHERE invitation_id = ?;
```

**Modify** `internal/database/queries/portal_users.sql` — remove all references to `movie_quality_profile_id` and `tv_quality_profile_id`:

- `CreatePortalUser`: Remove `movie_quality_profile_id, tv_quality_profile_id` from INSERT columns and VALUES
- `UpdatePortalUserQualityProfiles`: **Delete** this query entirely
- `CreatePortalUserWithID`: Remove `movie_quality_profile_id, tv_quality_profile_id` from INSERT columns and VALUES
- All other queries use `SELECT *` and are unaffected (the columns no longer exist)

**Modify** `internal/database/queries/portal_invitations.sql` — remove all references to `movie_quality_profile_id` and `tv_quality_profile_id`:

- `CreatePortalInvitation`: Remove `movie_quality_profile_id, tv_quality_profile_id` from INSERT columns and VALUES

**Delete** `internal/database/queries/user_quotas.sql` — the table no longer exists.

**Run** `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` to regenerate `internal/database/sqlc/`.

**Verify:**
- `sqlc generate` completes without errors
- `go build ./internal/database/...` compiles
- Generated `PortalUser` struct no longer has `MovieQualityProfileID` / `TvQualityProfileID` fields
- Generated `PortalInvitation` struct no longer has `MovieQualityProfileID` / `TvQualityProfileID` fields
- `UserQuota` type no longer exists in generated code
- New types `PortalUserModuleSetting` and `PortalInvitationModuleSetting` exist

---

### Task 8.3: Quota Service & Users Service Refactoring

**Rewrite** `internal/portal/quota/service.go`

The quota service switches from per-media-type columns to per-module rows in `portal_user_module_settings`.

Key changes:
- `QuotaStatus` struct: Replace `MoviesLimit`/`SeasonsLimit`/`EpisodesLimit`/`MoviesUsed`/`SeasonsUsed`/`EpisodesUsed` with a map or slice of `ModuleQuotaStatus` entries:
  ```go
  type ModuleQuotaStatus struct {
      ModuleType  string    `json:"moduleType"`
      QuotaLimit  int64     `json:"quotaLimit"`  // 0 = unlimited
      QuotaUsed   int64     `json:"quotaUsed"`
      PeriodStart time.Time `json:"periodStart"`
  }

  type QuotaStatus struct {
      UserID  int64               `json:"userId"`
      Modules []ModuleQuotaStatus `json:"modules"`
  }
  ```
- `CheckQuota(ctx, userID, moduleType)` — reads `portal_user_module_settings` for the given module
- `ConsumeQuota(ctx, userID, moduleType)` — calls `IncrementUserModuleQuota`
- `GetGlobalDefaults(ctx)` — returns map of module_type → default quota. Reads `requests_default_movie_quota` and `requests_default_tv_quota` settings. For unknown modules, uses a fallback default of 5.
- `SetGlobalDefaults(ctx, moduleType, limit)` — sets `requests_default_{moduleType}_quota`
- `SetUserOverride(ctx, userID, moduleType, limit)` — calls `UpdateUserModuleQuotaLimit`
- `ClearUserOverride(ctx, userID, moduleType)` — sets `quota_limit = NULL` on the module row
- `ResetAllQuotas(ctx)` — calls `ResetAllModuleQuotas`
- `initializeUserQuota` — creates rows for all enabled modules using `UpsertUserModuleSettings`

Settings constants change:
```go
const (
    SettingDefaultQuotaPrefix = "requests_default_%s_quota"
    DefaultQuota              = 5
)
```

**Modify** `internal/portal/users/service.go`

- `User` struct: Remove `MovieQualityProfileID` and `TVQualityProfileID` fields. These are now in `portal_user_module_settings`.
- Remove `QualityProfileIDFor(mediaType)` method from `User` — callers will query `portal_user_module_settings` directly.
- `CreateInput` struct: Remove `MovieQualityProfileID` and `TVQualityProfileID`. Add `ModuleSettings map[string]*ModuleSettingInput` where `ModuleSettingInput` has `QualityProfileID *int64`.
- `Create` method: After creating the portal user, insert `portal_user_module_settings` rows for each module in `ModuleSettings`.
- Remove `SetQualityProfiles` method — replaced by direct updates to `portal_user_module_settings`.
- Add `GetModuleSettings(ctx, userID) ([]ModuleSettingRow, error)` that queries `ListUserModuleSettings`.
- Add `SetModuleQualityProfile(ctx, userID, moduleType string, profileID *int64)` that calls `UpdateUserModuleQualityProfile`.
- `toUser` function: No longer populates quality profile fields.

**Modify** `internal/portal/invitations/service.go`

- `Invitation` struct: Remove `MovieQualityProfileID` and `TVQualityProfileID`. Add `ModuleSettings []InvitationModuleSetting` where each entry has `ModuleType string` and `QualityProfileID *int64`.
- `CreateInput` struct: Replace `MovieQualityProfileID` / `TVQualityProfileID` with `ModuleSettings map[string]*int64` (module_type → quality_profile_id).
- `Create` method: After creating the invitation, insert `portal_invitation_module_settings` rows.
- `Delete` method: The CASCADE on `portal_invitation_module_settings.invitation_id` handles cleanup.
- `toInvitation` function: No longer reads quality profile columns. Must join or separately query `portal_invitation_module_settings`. The simplest approach: after building the base invitation, call `queries.GetInvitationModuleSettings(ctx, inv.ID)` and attach results.

**Modify** `internal/portal/autoapprove/service.go`

- `ProcessAutoApprove`: Instead of calling `user.QualityProfileIDFor(request.MediaType)`, query `portal_user_module_settings` to get the quality profile for the request's module type.
- `getQuotaMediaType` → `getModuleType`: Map request media types to module types:
  ```go
  func getModuleType(requestMediaType string) string {
      switch requestMediaType {
      case "movie":
          return "movie"
      default:
          return "tv"
      }
  }
  ```
- Pass module type (not media type) to `quotaService.CheckQuota` and `ConsumeQuota`.

**Modify** `internal/portal/library/handlers.go`

- Line 183 calls `user.QualityProfileIDFor(mediaType)` — replace with a query to `portal_user_module_settings` to get the quality profile for the module type. The handler needs access to the users service's new `GetModuleSettings` method or direct sqlc queries.

**Modify** `internal/portal/search/handlers.go`

- Line 130 calls `user.QualityProfileIDFor(mediaType)` — same change as above.

**Verify:**
- `go build ./internal/portal/...` compiles
- All callers of removed `QualityProfileIDFor` method are updated (autoapprove, library handlers, search handlers, portal_bridge)

---

### Task 8.4: Portal Handlers & Bridge Updates for User Module Settings

**Modify** `internal/portal/admin/users_handlers.go`

- User list/detail responses: Include `moduleSettings` array (from `portal_user_module_settings`) instead of `movieQualityProfileId` / `tvQualityProfileId`.
- User create handler: Accept `moduleSettings` in request body.
- Add/modify endpoint for updating a user's module settings (quality profile, quota override) — `PUT /api/v1/portal/admin/users/:id/module-settings/:moduleType`.

**Modify** `internal/portal/admin/settings_handlers.go`

- Quota defaults endpoint: Return/accept per-module defaults instead of per-media-type. Response shape: `{ "movie": { "defaultQuota": 5 }, "tv": { "defaultQuota": 3 } }`.

**Modify** `internal/portal/admin/invitations_handlers.go`

- Create invitation handler: Accept `moduleSettings` map instead of `movieQualityProfileId` / `tvQualityProfileId`.
- List invitations response: Include `moduleSettings` array.

**Modify** `internal/api/portal_bridge.go` (also touched in Task 8.11 for provisioner changes)

- `portalUserQualityProfileAdapter`: Rewrite `GetQualityProfileID` to query `portal_user_module_settings` instead of calling `user.QualityProfileIDFor`. The adapter queries `GetUserModuleSettings(ctx, userID, moduleType)` to get the quality profile ID.
- The adapter needs access to `sqlc.Queries` (or the users service's new `GetModuleSettings` method). Update the adapter struct and its construction in `wire.go`.

**Modify** `internal/api/setters.go` and `internal/api/switchable.go`

- Ensure the quota service, users service, and invitation service receive the updated `*sqlc.Queries` on dev mode switch.

**Verify:**
- `go build ./...` compiles
- API responses for portal users and invitations use new `moduleSettings` shape

---

### Task 8.5: Migration — Slot Root Folders Pivot Table

**Create** `internal/database/migrations/0XX_slot_root_folders_pivot.sql` (use next available migration number)

This migration:
1. Creates `slot_root_folders` pivot table
2. Migrates data from `version_slots.movie_root_folder_id` and `version_slots.tv_root_folder_id`
3. Recreates `version_slots` without the two FK columns

```sql
-- +goose Up
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- 1. Create pivot table
CREATE TABLE slot_root_folders (
    slot_id INTEGER NOT NULL REFERENCES version_slots(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    root_folder_id INTEGER NOT NULL REFERENCES root_folders(id) ON DELETE CASCADE,
    PRIMARY KEY (slot_id, module_type)
);

-- 2. Migrate existing data
INSERT INTO slot_root_folders (slot_id, module_type, root_folder_id)
SELECT id, 'movie', movie_root_folder_id
FROM version_slots
WHERE movie_root_folder_id IS NOT NULL;

INSERT INTO slot_root_folders (slot_id, module_type, root_folder_id)
SELECT id, 'tv', tv_root_folder_id
FROM version_slots
WHERE tv_root_folder_id IS NOT NULL;

-- 3. Recreate version_slots without per-module columns
CREATE TABLE version_slots_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slot_number INTEGER NOT NULL CHECK (slot_number >= 1 AND slot_number <= 3),
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
    display_order INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(slot_number)
);

INSERT INTO version_slots_new (id, slot_number, name, enabled, quality_profile_id, display_order, created_at, updated_at)
SELECT id, slot_number, name, enabled, quality_profile_id, display_order, created_at, updated_at
FROM version_slots;

DROP TABLE version_slots;
ALTER TABLE version_slots_new RENAME TO version_slots;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

ALTER TABLE version_slots ADD COLUMN movie_root_folder_id INTEGER REFERENCES root_folders(id) ON DELETE SET NULL;
ALTER TABLE version_slots ADD COLUMN tv_root_folder_id INTEGER REFERENCES root_folders(id) ON DELETE SET NULL;

UPDATE version_slots SET
    movie_root_folder_id = (SELECT root_folder_id FROM slot_root_folders WHERE slot_id = version_slots.id AND module_type = 'movie'),
    tv_root_folder_id = (SELECT root_folder_id FROM slot_root_folders WHERE slot_id = version_slots.id AND module_type = 'tv');

DROP TABLE IF EXISTS slot_root_folders;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd
```

**Critical considerations:**
- `version_slots` is referenced by FKs from: `movie_files.slot_id`, `episode_files.slot_id`, `movie_slot_assignments.slot_id`, `episode_slot_assignments.slot_id`, `requests.target_slot_id`, `queue_media.target_slot_id`, `download_mappings.target_slot_id`. Table recreation must preserve IDs.
- The `slot_root_folders.root_folder_id` FK has `ON DELETE CASCADE` — if a root folder is deleted, the slot root folder mapping is removed (slot falls back to media item's root folder).

**Verify:**
- Migration applies cleanly
- `slot_root_folders` has correct rows for slots that had root folders
- `version_slots` no longer has `movie_root_folder_id` / `tv_root_folder_id`
- All FK references to `version_slots(id)` still work

---

### Task 8.6: sqlc Queries for Slot Root Folders

**Create** `internal/database/queries/slot_root_folders.sql`

```sql
-- name: GetSlotRootFolder :one
SELECT * FROM slot_root_folders
WHERE slot_id = ? AND module_type = ? LIMIT 1;

-- name: ListSlotRootFolders :many
SELECT * FROM slot_root_folders
WHERE slot_id = ?
ORDER BY module_type;

-- name: UpsertSlotRootFolder :one
INSERT INTO slot_root_folders (slot_id, module_type, root_folder_id)
VALUES (?, ?, ?)
ON CONFLICT(slot_id, module_type) DO UPDATE SET
    root_folder_id = excluded.root_folder_id
RETURNING *;

-- name: DeleteSlotRootFolder :exec
DELETE FROM slot_root_folders WHERE slot_id = ? AND module_type = ?;

-- name: DeleteSlotRootFolders :exec
DELETE FROM slot_root_folders WHERE slot_id = ?;
```

**Modify** `internal/database/queries/slots.sql`

- `UpdateVersionSlot`: Remove `movie_root_folder_id = ?, tv_root_folder_id = ?` from the UPDATE — root folders are now managed via the pivot table. The query becomes:
  ```sql
  -- name: UpdateVersionSlot :one
  UPDATE version_slots SET
      name = ?,
      enabled = ?,
      quality_profile_id = ?,
      display_order = ?,
      updated_at = CURRENT_TIMESTAMP
  WHERE id = ?
  RETURNING *;
  ```
- `UpdateVersionSlotRootFolders`: **Delete** this query — replaced by `UpsertSlotRootFolder` / `DeleteSlotRootFolder`.

**Run** `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`

**Verify:**
- `sqlc generate` completes without errors
- `go build ./internal/database/...` compiles
- `VersionSlot` struct no longer has `MovieRootFolderID` / `TvRootFolderID` fields
- New types for `SlotRootFolder` exist

---

### Task 8.7: Slot Service Refactoring

**Modify** `internal/library/slots/slot.go`

- `Slot` struct: Remove `MovieRootFolderID *int64` and `TVRootFolderID *int64` fields. Add `RootFolders map[string]*int64` (module_type → root_folder_id, populated from pivot table).
- Remove `MovieRootFolder` and `TVRootFolder` display fields. Replace with `RootFolderDetails map[string]*SlotRootFolder` (populated when loading with root folder info).
- `UpdateSlotInput` struct: Remove `MovieRootFolderID` and `TVRootFolderID`. Add `RootFolders map[string]*int64` (module_type → root_folder_id, nil value = remove).

**Modify** `internal/library/slots/service.go`

- `rowToSlot`: No longer reads `MovieRootFolderID` / `TvRootFolderID` from the row (they don't exist). After building the base slot, query `ListSlotRootFolders` to populate `RootFolders` map.
- `Update`: After updating the slot row, sync root folders: for each entry in `input.RootFolders`, call `UpsertSlotRootFolder` (if value is non-nil) or `DeleteSlotRootFolder` (if value is nil/to remove).
- `SetRootFolders` method: Rewrite to iterate `moduleType → rootFolderID` map and call `UpsertSlotRootFolder` / `DeleteSlotRootFolder`.
- `GetRootFolderForSlot(ctx, slotID, mediaType)`: The `mediaType` parameter maps to `module_type` (`"movie"` → `"movie"`, `"episode"/"tv"` → `"tv"`). Query `GetSlotRootFolder(ctx, slotID, moduleType)`. If found, resolve the root folder path via `rootFolderProvider.Get`. If not found, return empty string (caller falls back).
- `List` and `ListEnabled`: After building slot list, batch-load root folders for all slots (or load per-slot — simpler).

**Note:** The `mediaTypeMovie` and `mediaTypeEpisode` constants are defined in `internal/library/slots/status.go` (not `slot.go`) and are used extensively across `status.go`, `migration.go`, `search.go`, `fileops.go`, `assignment.go`, and `service.go`. These constants remain valid — the `GetRootFolderForSlot` method in `service.go` already maps `"tv"` to the TV root folder alongside `mediaTypeEpisode`. No constant changes needed.

**Verify:**
- `go build ./internal/library/slots/...` compiles
- Slot list API returns `rootFolders` map instead of `movieRootFolderId` / `tvRootFolderId`

---

### Task 8.8: Slot Handlers & Bridge Updates

**Modify** `internal/library/slots/handlers.go`

- Update slot API request/response payloads to use `rootFolders` map instead of `movieRootFolderId` / `tvRootFolderId`.
- The update slot endpoint accepts `rootFolders: { "movie": 1, "tv": 2 }` in the request body.
- The list/get slot endpoints return `rootFolders: { "movie": 1, "tv": 2 }` with optional `rootFolderDetails` for display.

**Note:** There is no `internal/api/slot_bridge.go` file. Slot-related bridge/adapter code does not exist — the slot service is used directly. No bridge updates needed for slots.

**Modify** `internal/api/switchable.go`

- Ensure `slot_root_folders` queries are available through the switchable queries on dev mode switch.

**Verify:**
- `go build ./...` compiles
- Slot API endpoints return correct data shape

---

### Task 8.9: Define PortalProvisioner Interface on Module Framework

**Create** `internal/module/portal_provisioner.go`

Define the `PortalProvisioner` optional interface and its input/output types:

```go
package module

import "context"

// ProvisionInput contains the data needed to ensure a media item exists in the library.
type ProvisionInput struct {
    ExternalIDs      map[string]int64 // e.g., {"tmdb": 12345, "tvdb": 67890}
    Title            string
    Year             int
    QualityProfileID *int64  // User's assigned quality profile (nil = use default)
    AddedBy          *int64  // Portal user ID who triggered the add
    // Module-specific fields for TV-like modules
    RequestedSeasons []int64 // Specific seasons to monitor (empty = all)
    MonitorFuture    bool    // Monitor future/unaired content
}

// AvailabilityCheckInput contains the data needed to check if an entity is available.
type AvailabilityCheckInput struct {
    ModuleType string
    EntityType string // e.g., "movie", "series", "season", "episode"
    ExternalIDs map[string]int64
    SeasonNumber  *int64
    EpisodeNumber *int64
    UserQualityProfileID *int64
}

// AvailabilityResult describes the availability state of an entity in the library.
type AvailabilityResult struct {
    InLibrary    bool
    EntityID     *int64
    CanRequest   bool
    IsComplete   bool // All children available (for parent nodes)
    // Module-specific detail (opaque to framework, passed to frontend)
    Detail any
}

// RequestCompletionCheckInput contains the data for checking if a request should
// transition to "available" based on an entity becoming available.
type RequestCompletionCheckInput struct {
    // The entity that just became available
    AvailableEntityType string
    AvailableEntityID   int64
    // The request to check
    RequestEntityType    string
    RequestEntityID      *int64 // media_id on the request (may be nil)
    RequestExternalIDs   map[string]int64
    RequestedSeasons     []int64
}

// RequestCompletionResult indicates whether a request should be marked available.
type RequestCompletionResult struct {
    ShouldMarkAvailable bool
}

// PortalProvisioner is an optional module interface for portal request support.
// Modules that implement this interface enable their media types to be requested
// through the external requests portal.
// (spec §11.1)
type PortalProvisioner interface {
    // EnsureInLibrary finds or creates the requested media in the library.
    // Returns the root entity ID (e.g., movie ID, series ID).
    EnsureInLibrary(ctx context.Context, input *ProvisionInput) (entityID int64, err error)

    // CheckAvailability checks whether an entity exists in the library and
    // whether it can still be requested.
    CheckAvailability(ctx context.Context, input *AvailabilityCheckInput) (*AvailabilityResult, error)

    // CheckRequestCompletion determines whether a request should transition to
    // "available" because an entity just became available. Called by the
    // StatusTracker when an entity's status changes to available.
    CheckRequestCompletion(ctx context.Context, input *RequestCompletionCheckInput) (*RequestCompletionResult, error)

    // ValidateRequest validates a request before creation (e.g., checks that
    // the external IDs are valid, the media type is supported).
    ValidateRequest(ctx context.Context, entityType string, externalIDs map[string]int64) error

    // MapEntityTypeToModuleType returns the module type for a given entity type.
    // E.g., "movie" → "movie", "series"/"season"/"episode" → "tv".
    // Used by the framework to route requests to the correct module.
    SupportedEntityTypes() []string
}
```

**Modify** `internal/module/interfaces.go` — add `PortalProvisioner` to the optional interface list comment.

**Modify** `internal/module/registry.go` — add a method to look up provisioners:

```go
// GetProvisioner returns the PortalProvisioner for the given module type, or nil.
func (r *Registry) GetProvisioner(moduleType string) PortalProvisioner {
    mod, ok := r.modules[Type(moduleType)]
    if !ok {
        return nil
    }
    if pp, ok := mod.(PortalProvisioner); ok {
        return pp
    }
    return nil
}

// GetProvisionerForEntityType returns the PortalProvisioner for a given entity type
// (e.g., "movie" → movie module's provisioner, "episode" → tv module's provisioner).
func (r *Registry) GetProvisionerForEntityType(entityType string) PortalProvisioner {
    for _, mod := range r.modules {
        pp, ok := mod.(PortalProvisioner)
        if !ok {
            continue
        }
        for _, et := range pp.SupportedEntityTypes() {
            if et == entityType {
                return pp
            }
        }
    }
    return nil
}
```

**Define** `SlotSupport` interface in `internal/module/slot_support.go`:

```go
package module

// SlotSupport is an optional module interface for multi-version slot support.
// Modules that implement this interface can have files assigned to version slots.
// (spec §2.5)
type SlotSupport interface {
    // SlotEntityType returns the leaf entity type name used in slot assignment tables.
    // E.g., "movie" for movies, "episode" for TV episodes.
    SlotEntityType() string
}
```

**Verify:**
- `go build ./internal/module/...` compiles
- Interfaces are well-defined and consistent with the spec

---

### Task 8.10: Movie & TV Module PortalProvisioner Implementations

**Modify** `internal/modules/movie/module.go` — add `PortalProvisioner` and `SlotSupport` implementations:

```go
var _ module.PortalProvisioner = (*Module)(nil)
var _ module.SlotSupport = (*Module)(nil)
```

The movie module's `EnsureInLibrary` should contain the logic currently in `provisioner.Service.EnsureMovieInLibrary`:
- Look up existing movie by TMDB ID
- If not found, create it using the movie service with default or user-provided quality profile
- Refresh metadata via library manager
- Return movie ID

The movie module's `CheckAvailability` should contain the logic currently in `LibraryChecker.CheckMovieAvailability` (without the existing-request check, which stays in the framework).

The movie module's `CheckRequestCompletion` is simple — a movie request is complete when the movie entity itself has a file:
```go
func (m *Module) CheckRequestCompletion(ctx context.Context, input *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) {
    // For movies: the entity becoming available IS the request completion
    if input.AvailableEntityType == "movie" {
        return &module.RequestCompletionResult{ShouldMarkAvailable: true}, nil
    }
    return &module.RequestCompletionResult{ShouldMarkAvailable: false}, nil
}
```

The movie module's `SupportedEntityTypes` returns `[]string{"movie"}`.

The movie module's `SlotEntityType` returns `"movie"`.

**Modify** `internal/modules/tv/module.go` — add `PortalProvisioner` and `SlotSupport` implementations:

The TV module's `EnsureInLibrary` should contain the logic currently in `provisioner.Service.EnsureSeriesInLibrary` + `applyPortalRequestMonitoring`:
- Look up existing series by TVDB ID
- If found, apply additive monitoring for requested seasons
- If not found, create series, refresh metadata, apply monitoring preset
- Return series ID

The TV module's `CheckAvailability` should contain the logic currently in `LibraryChecker.CheckSeriesAvailability` / `CheckSeasonAvailability` / `CheckEpisodeAvailability`.

The TV module's `CheckRequestCompletion` contains the season-completion logic currently in `StatusTracker`:
```go
func (m *Module) CheckRequestCompletion(ctx context.Context, input *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) {
    switch input.RequestEntityType {
    case "episode":
        // Episode request: complete when this exact episode is available
        return &module.RequestCompletionResult{ShouldMarkAvailable: true}, nil
    case "season":
        // Season request: complete when all episodes in the season are available
        // (uses existing AreSeasonsComplete logic)
        ...
    case "series":
        // Series request: complete when all requested seasons are available
        // (if RequestedSeasons is empty, complete when ALL seasons are available)
        ...
    }
}
```

The TV module's `SupportedEntityTypes` returns `[]string{"series", "season", "episode"}`.

The TV module's `SlotEntityType` returns `"episode"`.

**Important:** The module implementations need access to their respective services (movie service, TV service, library manager, sqlc queries). These are injected via the module's Wire provider set. The Phase 0 `Wire()` method returns the module's service constructors — by Phase 8, the module struct should already have these services wired in from earlier phases.

**Verify:**
- `go build ./internal/modules/...` compiles
- Compile-time assertions pass
- No circular imports

---

### Task 8.11: RequestSearcher & Provisioner Dispatch Refactoring

**Modify** `internal/portal/requests/searcher.go`

- Replace `MediaProvisioner` interface (which has `EnsureMovieInLibrary` / `EnsureSeriesInLibrary`) with a reference to the module `Registry` (or a simpler dispatch interface):
  ```go
  type ModuleProvisioner interface {
      EnsureInLibrary(ctx context.Context, moduleType string, input *module.ProvisionInput) (int64, error)
  }
  ```
  The concrete implementation looks up the module's `PortalProvisioner` from the registry and delegates.

- `ensureMediaInLibrary`: Instead of switching on `request.MediaType` and calling different methods, build a `module.ProvisionInput` and call `provisioner.EnsureInLibrary(ctx, moduleType, input)`:
  ```go
  func (s *RequestSearcher) ensureMediaInLibrary(ctx context.Context, request *Request) (int64, error) {
      moduleType := s.getModuleType(request.MediaType)
      input := &module.ProvisionInput{
          Title:            request.Title,
          Year:             intFromPtr(request.Year),
          RequestedSeasons: request.RequestedSeasons,
          MonitorFuture:    isMonitorFuture(request.MonitorType),
          AddedBy:          &request.UserID,
      }
      // Set external IDs
      if request.TmdbID != nil {
          input.ExternalIDs = map[string]int64{"tmdb": *request.TmdbID}
      }
      if request.TvdbID != nil {
          if input.ExternalIDs == nil {
              input.ExternalIDs = make(map[string]int64)
          }
          input.ExternalIDs["tvdb"] = *request.TvdbID
      }
      // Resolve quality profile
      s.resolveQualityProfile(ctx, input, request.UserID, moduleType)
      return s.provisioner.EnsureInLibrary(ctx, moduleType, input)
  }
  ```

- `resolveQualityProfile`: Change to accept `moduleType` instead of `mediaType` and pass it to `userGetter.GetQualityProfileID(ctx, userID, moduleType)`.

- `executeSearch`: The search dispatch (`searchMovie`, `searchSeries`, etc.) remains media-type-specific for now — search is already modularized in Phase 5 via `SearchStrategy`. No change needed here.

- `getModuleType` helper: Map request media types to module types:
  ```go
  func (s *RequestSearcher) getModuleType(mediaType string) string {
      switch mediaType {
      case MediaTypeMovie:
          return "movie"
      default:
          return "tv"
      }
  }
  ```

**Delete** `internal/portal/provisioner/service.go` — its logic has been split into movie and TV module `PortalProvisioner` implementations (Task 8.10). The package is no longer needed.

**Modify** `internal/api/wire.go` — remove the provisioner service provider and replace with the module-dispatching adapter.

**Modify** `internal/api/portal_bridge.go`:
- `portalUserQualityProfileAdapter.GetQualityProfileID`: Now queries `portal_user_module_settings` using `moduleType` instead of calling `user.QualityProfileIDFor(mediaType)`. The adapter needs access to `sqlc.Queries` (or the users service's new `GetModuleSettings` method). Update the adapter struct and its construction in `wire.go`.
- `statusTrackerMovieLookup` and `statusTrackerEpisodeLookup` — keep these for now; they are removed in Task 8.12.

**Modify** `internal/api/interface_assertions.go`:
- Remove `_ requests.MediaProvisioner = (*provisioner.Service)(nil)` — the provisioner package is being deleted.
- Remove the `provisioner` import.

**Verify:**
- `go build ./...` compiles
- `internal/portal/provisioner/` package is deleted
- No remaining imports of `provisioner.Service`

---

### Task 8.12: StatusTracker Rewrite — Module-Dispatched Completion Checking

**Rewrite** `internal/portal/requests/status_tracker.go`

The current StatusTracker has:
- `MovieLookup`, `EpisodeLookup`, `SeriesLookup` interfaces — **remove all three**
- `OnMovieAvailable`, `OnEpisodeAvailable`, `OnSeasonComplete`, `OnSeriesComplete` — **replace with single `OnEntityAvailable`**
- Complex TV-specific cascade logic — **moves into TV module's `CheckRequestCompletion`**

New `StatusTracker`:

```go
type ModuleProvisionerLookup interface {
    GetProvisionerForEntityType(entityType string) module.PortalProvisioner
}

type StatusTracker struct {
    queries            *sqlc.Queries
    requestsService    *Service
    watchersService    *WatchersService
    notifDispatcher    NotificationDispatcher
    provisionerLookup  ModuleProvisionerLookup
    logger             *zerolog.Logger
}

func NewStatusTracker(
    queries *sqlc.Queries,
    requestsService *Service,
    watchersService *WatchersService,
    logger *zerolog.Logger,
    provisionerLookup ModuleProvisionerLookup,
    notifDispatcher NotificationDispatcher,
) *StatusTracker { ... }
```

The key method:

```go
func (t *StatusTracker) OnEntityAvailable(ctx context.Context, moduleType, entityType string, entityID int64) error {
    // 1. Find requests directly linked to this entity
    reqs := t.findDirectRequests(ctx, entityType, entityID)

    // 2. For leaf entities, also find parent-level requests that may be satisfied
    provisioner := t.provisionerLookup.GetProvisionerForEntityType(entityType)
    if provisioner != nil {
        parentReqs := t.findParentRequests(ctx, moduleType, entityType, entityID)
        reqs = append(reqs, parentReqs...)
    }

    // 3. For each candidate request, check if it should be marked available
    for _, req := range reqs {
        t.processRequestCompletion(ctx, req, provisioner, entityType, entityID)
    }

    return nil
}
```

Where `processRequestCompletion`:
```go
func (t *StatusTracker) processRequestCompletion(ctx context.Context, req *Request, provisioner module.PortalProvisioner, availableEntityType string, availableEntityID int64) {
    if req.Status != StatusDownloading && req.Status != StatusApproved && req.Status != StatusSearching {
        return
    }

    // For direct entity matches (e.g., movie request + movie available), mark available immediately
    if req.MediaType == availableEntityType {
        t.markAvailable(ctx, req)
        return
    }

    // For parent requests (e.g., series request + episode available), delegate to module
    if provisioner == nil {
        return
    }

    input := &module.RequestCompletionCheckInput{
        AvailableEntityType: availableEntityType,
        AvailableEntityID:   availableEntityID,
        RequestEntityType:   req.MediaType,
        RequestEntityID:     req.MediaID,
        RequestedSeasons:    req.RequestedSeasons,
    }
    // Populate external IDs from request
    if req.TmdbID != nil {
        input.RequestExternalIDs = map[string]int64{"tmdb": *req.TmdbID}
    }
    if req.TvdbID != nil {
        if input.RequestExternalIDs == nil {
            input.RequestExternalIDs = make(map[string]int64)
        }
        input.RequestExternalIDs["tvdb"] = *req.TvdbID
    }

    result, err := provisioner.CheckRequestCompletion(ctx, input)
    if err != nil {
        t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to check request completion")
        return
    }

    if result.ShouldMarkAvailable {
        t.markAvailable(ctx, req)
    }
}
```

`findDirectRequests` — uses `ListRequestsByMediaID` (existing query) to find requests where `media_id` matches and `media_type` matches.

`findParentRequests` — for leaf entities (e.g., episodes), finds parent-level requests (e.g., series/season requests) that might be satisfied. This uses the existing request queries: `ListActiveSeriesRequestsByTvdbID`, `FindRequestsCoveringSeasons`. The key difference: instead of hardcoding TV-specific TVDB lookups, the framework calls the module provisioner to get the external IDs needed for request lookup.

**Migrate callers:** All call sites currently calling `StatusTracker.OnMovieAvailable`, `OnEpisodeAvailable` must switch to `OnEntityAvailable(ctx, moduleType, entityType, entityID)`:

- Import service (`internal/import/service.go:416`): `OnMovieAvailable(ctx, *result.Match.MovieID)` → `OnEntityAvailable(ctx, "movie", "movie", *result.Match.MovieID)`
- Import pipeline (`internal/import/pipeline.go:1284`): `OnMovieAvailable(ctx, *mapping.MovieID)` → `OnEntityAvailable(ctx, "movie", "movie", *mapping.MovieID)`
- Import pipeline (`internal/import/pipeline.go:1468,1604`): `OnEpisodeAvailable(ctx, *mapping.EpisodeID)` / `OnEpisodeAvailable(ctx, episodeID)` → `OnEntityAvailable(ctx, "tv", "episode", episodeID)`
- **Note:** `OnSeasonComplete` and `OnSeriesComplete` have **zero external callers** in the codebase — they exist as methods but are never called from outside StatusTracker. No migration needed for these.

**Update consumer interfaces that bind to StatusTracker:**

The following interfaces define the contract that external packages use to call StatusTracker. They must be updated to match the new method signatures:

- `internal/import/service.go` — `StatusTrackerService` interface: Replace `OnMovieAvailable(ctx, movieID) error` and `OnEpisodeAvailable(ctx, episodeID) error` with `OnEntityAvailable(ctx, moduleType, entityType string, entityID int64) error`. Keep `OnDownloadFailed` unchanged for now (it already takes `mediaType` and `mediaID`).
- `internal/downloader/service.go` — `PortalStatusTracker` interface: Has `OnDownloadFailed(ctx, mediaType, mediaID) error` — unchanged.
- `internal/indexer/grab/service.go` — `PortalStatusTracker` interface: Has `OnDownloadStarted(ctx, mediaType, mediaID) error` and `OnDownloadFailed(ctx, mediaType, mediaID) error` — unchanged.
- `internal/api/wire.go` — Bindings: `wire.Bind(new(importer.StatusTrackerService), new(*requests.StatusTracker))` remains, but the interface it binds to has changed. Also `wire.Bind(new(downloader.PortalStatusTracker), ...)` and `wire.Bind(new(grab.PortalStatusTracker), ...)` remain unchanged.

**`OnDownloadStarted` and `OnDownloadFailed` remain largely unchanged.** These methods already take `mediaType string` and `mediaID int64` which is close to the module-dispatched pattern. The only change needed is replacing the internal `episodeLookup.GetEpisodeInfo` call (used to find parent series requests for episode events) with a call to the module's provisioner. The method signatures stay the same since their consumer interfaces (`downloader.PortalStatusTracker` and `grab.PortalStatusTracker`) don't need to change.

**Update** `internal/portal/requests/status_tracker_test.go` — rewrite tests to use the new interface. Replace mock `MovieLookup`, `EpisodeLookup`, `SeriesLookup` with a mock `ModuleProvisionerLookup`.

**Modify** `internal/api/portal_bridge.go`:
- Remove `statusTrackerMovieLookup` and `statusTrackerEpisodeLookup` adapter structs
- The `StatusTracker` constructor now takes `ModuleProvisionerLookup` (which wraps the module `Registry`)

**Modify** `internal/api/providers.go`:
- Remove `provideMovieLookup` function (line 180-182)
- Remove `provideEpisodeLookup` function (line 184-186)

**Modify** `internal/api/wire.go`:
- Remove `provideMovieLookup` and `provideEpisodeLookup` from the `wire.Build()` call
- Remove `wire.Bind(new(requests.SeriesLookup), new(*tv.Service))` binding
- Update `StatusTracker` provider to pass the `Registry` (wrapped as `ModuleProvisionerLookup`) instead of the three lookup adapters

**Modify** `internal/api/interface_assertions.go`:
- Remove `_ requests.SeriesLookup = (*tv.Service)(nil)` assertion
- Remove `_ requests.MediaProvisioner = (*provisioner.Service)(nil)` assertion (provisioner package is being deleted)
- Remove `provisioner` import

**Verify:**
- `go build ./...` compiles
- Status tracker tests pass with new interface
- No remaining references to `MovieLookup`, `EpisodeLookup`, `SeriesLookup` interfaces

---

### Task 8.13: Frontend Updates

**Modify** portal user types and API client:

- `web/src/types/portal.ts` (or equivalent): Update `PortalUser` type to replace `movieQualityProfileId` / `tvQualityProfileId` with `moduleSettings: { moduleType: string, qualityProfileId: number | null, quotaLimit: number | null, quotaUsed: number }[]`.
- Update `PortalInvitation` type similarly.
- Update `QuotaStatus` type to use per-module structure.

**Modify** portal admin user management pages:

- User detail/edit: Replace separate movie/TV quality profile dropdowns with a dynamic list of module settings. For each enabled module, show a quality profile dropdown filtered to that module's profiles.
- User quota display: Show per-module quota (used/limit) instead of per-media-type.

**Modify** portal admin invitation pages:

- Create invitation form: Replace `movieQualityProfileId` / `tvQualityProfileId` fields with a per-module quality profile selector.
- Invitation list: Show `moduleSettings` instead of individual quality profile fields.

**Modify** portal admin settings page:

- Quota defaults section: Show per-module defaults instead of movie/season/episode defaults.

**Modify** slot settings pages:

- Slot edit form: Replace `movieRootFolderId` / `tvRootFolderId` fields with a dynamic list keyed by module type. For each enabled module, show a root folder dropdown filtered to that module's root folders.
- Slot list display: Show root folders as a map/list per module instead of two fixed fields.

**Verify:**
- Frontend builds without TypeScript errors: `cd web && bun run lint`
- Portal user CRUD works in the UI
- Invitation CRUD works in the UI
- Slot root folder management works in the UI
- Quota display shows per-module values

---

### Task 8.14: Phase 8 Validation

**Run full build and test suite:**
```bash
make build && make test && make lint
cd web && bun run lint
```

**Manual verification checklist:**

1. **Portal user lifecycle:**
   - Create a portal user with movie quality profile = HD-1080p, TV quality profile = SD
   - Verify `portal_user_module_settings` has two rows (movie + tv)
   - User's quality profile shows correctly in admin UI

2. **Quota lifecycle:**
   - Set global default quotas (movie: 5, tv: 3)
   - Create a request as a portal user
   - Verify quota is consumed against the correct module
   - Verify quota resets on period expiry

3. **Invitation lifecycle:**
   - Create invitation with per-module quality profiles
   - Accept invitation
   - Verify `portal_user_module_settings` created from invitation settings

4. **Slot root folders:**
   - Set root folders for a slot via the pivot table
   - Verify import resolves the correct root folder for the slot

5. **Request provisioning:**
   - Submit a movie request → provisioner creates movie in library
   - Submit a TV request → provisioner creates series with monitoring

6. **Request completion tracking:**
   - Movie request → import a movie file → request transitions to available
   - Series request with specific seasons → import all episodes for those seasons → request transitions to available

7. **Migration on existing data:**
   - Apply migration to a database with existing portal users, quotas, invitations, and slot root folders
   - Verify all data migrated correctly
   - Verify rollback migration works

**Verify all expected files are touched/created:**

**New files:**
```
internal/database/migrations/0XX_portal_user_module_settings.sql
internal/database/migrations/0XX_slot_root_folders_pivot.sql
internal/database/queries/portal_user_module_settings.sql
internal/database/queries/portal_invitation_module_settings.sql
internal/database/queries/slot_root_folders.sql
internal/module/portal_provisioner.go
internal/module/slot_support.go
```

**Deleted files:**
```
internal/portal/provisioner/service.go (logic moved to module implementations)
internal/database/queries/user_quotas.sql (table dropped)
```

**Modified files:**
```
internal/database/queries/portal_users.sql (remove quality profile columns from queries)
internal/database/queries/portal_invitations.sql (remove quality profile columns from queries)
internal/database/queries/slots.sql (remove root folder columns from UpdateVersionSlot)
internal/database/sqlc/* (regenerated)
internal/portal/quota/service.go (per-module quotas)
internal/portal/users/service.go (remove quality profile fields, add module settings methods)
internal/portal/invitations/service.go (module settings instead of per-media quality profiles)
internal/portal/autoapprove/service.go (module type mapping, query module settings for quality profile)
internal/portal/library/handlers.go (replace QualityProfileIDFor with module settings query)
internal/portal/search/handlers.go (replace QualityProfileIDFor with module settings query)
internal/portal/requests/searcher.go (module-dispatched provisioning)
internal/portal/requests/status_tracker.go (module-dispatched completion checking)
internal/portal/requests/status_tracker_test.go (new mock interfaces)
internal/portal/requests/library_check.go (delegate to module CheckAvailability)
internal/portal/admin/users_handlers.go (moduleSettings in request/response)
internal/portal/admin/settings_handlers.go (per-module quota defaults)
internal/portal/admin/invitations_handlers.go (moduleSettings in request/response)
internal/import/service.go (update StatusTrackerService interface: OnMovieAvailable/OnEpisodeAvailable → OnEntityAvailable)
internal/import/pipeline.go (update OnMovieAvailable/OnEpisodeAvailable call sites to OnEntityAvailable)
internal/api/portal_bridge.go (remove lookup adapters, update quality profile adapter)
internal/api/providers.go (remove provideMovieLookup, provideEpisodeLookup)
internal/api/interface_assertions.go (remove SeriesLookup and MediaProvisioner assertions)
internal/api/wire.go (remove provisioner.Service, remove lookup providers/bindings, update StatusTracker construction)
internal/api/setters.go (update setter chains for refactored services)
internal/modules/movie/module.go (PortalProvisioner + SlotSupport implementation)
internal/modules/tv/module.go (PortalProvisioner + SlotSupport implementation)
internal/module/registry.go (GetProvisioner, GetProvisionerForEntityType methods)
internal/module/interfaces.go (PortalProvisioner + SlotSupport in optional interface list)
internal/library/slots/slot.go (RootFolders map replaces per-module fields)
internal/library/slots/service.go (pivot table queries, module-keyed root folders)
internal/library/slots/handlers.go (updated request/response payloads)
web/src/types/portal.ts (moduleSettings types)
web/src/hooks/use-portal-*.ts (updated API shapes)
web/src/routes/portal/admin/* (per-module UI for quality profiles, quotas, invitations)
web/src/routes/settings/slots/* (module-keyed root folders)
```

- Zero changes to: download client services (`downloader/`, `grab/`), search pipeline, notification system, quality profiles table, requests table schema, existing slot assignment tables
- `OnDownloadStarted`/`OnDownloadFailed` interfaces in `downloader.PortalStatusTracker` and `grab.PortalStatusTracker` are unchanged (they already take `mediaType` + `mediaID`)
- Import pipeline (`import/service.go`, `import/pipeline.go`) only changes are updating StatusTracker call sites and the `StatusTrackerService` interface

---

