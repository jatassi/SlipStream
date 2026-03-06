## Phase 2: Quality System ✅ COMPLETED

**Goal:** Make quality profiles module-scoped. Add `module_type` to the `quality_profiles` table, duplicate existing profiles per module, update all FK references, and make the quality service + API + frontend module-aware. Implement `QualityDefinition` on module stubs.

**Spec sections covered:** §4.1 (module-scoped profiles, quality items registration, migration), §4.2 (QualityDefinition interface: `ParseQuality`, `ScoreQuality`, `IsUpgrade`)

**Depends on:** Phase 0 (module types, interfaces, registry) and Phase 1 (discriminator pattern on shared tables — specifically the `module_type` column convention and the `module.Type` constants).

### Phase 2 Architecture Decisions

1. **Migration strategy — duplicate and remap:** Each existing quality profile is duplicated: the original row (preserving its ID) becomes the `movie` copy, and a new row is inserted as the `tv` copy. This minimizes FK updates — only TV-related FKs need remapping to the new TV copy IDs. Movie FKs stay unchanged because the original IDs are preserved.

2. **UNIQUE constraint change:** The current `quality_profiles.name` has a `UNIQUE` constraint. After migration, uniqueness is scoped to `(module_type, name)` — the same name (e.g., "HD-1080p") can exist independently in both modules. SQLite requires table recreation for this constraint change.

3. **Version slots — deferred:** `version_slots.quality_profile_id` currently references the shared quality profile pool. After migration, these references will point to "movie" copies (since original IDs are preserved). This is acceptable as a temporary state — Phase 8 (Portal & Version Slots) will properly refactor the slot system with per-module quality profile references. Phase 2 does NOT touch the version_slots table or its quality profile FK.

4. **Slot recalculation queries — unused, left as-is:** The queries `ListMovieSlotAssignmentsWithFilesForProfile` and `ListEpisodeSlotAssignmentsWithFilesForProfile` exist in `slots.sql` but are not called by any Go code. They are left unchanged in Phase 2. Phase 8 will wire them into the slot service properly.

5. **Import decisions — cleared during migration:** `import_decisions.quality_profile_id` caches the profile ID active during import. After profile duplication, TV import decisions would reference the wrong (movie) copy. Rather than remapping, all import decisions are cleared during migration — they are regenerated automatically on the next import scan.

6. **QualityDefinition interface — implemented but not yet consumed by scoring pipeline:** Both movie and TV modules implement `QualityDefinition` returning the same 17 video quality tiers. The quality service gains a `RegisterModuleQualities()` method that stores per-module quality definitions at startup. The `/qualityprofiles/qualities` endpoint becomes module-aware. However, the scoring pipeline (`internal/indexer/scoring/`) continues to use `quality.PredefinedQualities` directly — wiring the scoring pipeline through module interfaces is deferred to Phase 5 (Search & Indexer), when the full `SearchStrategy` integration happens.

7. **API backward compatibility:** The `GET /qualityprofiles` endpoint returns ALL profiles (both modules) when no `moduleType` query parameter is provided, preserving backward compatibility. When `moduleType=movie` or `moduleType=tv` is specified, results are filtered. The `POST` endpoint requires `moduleType` in the request body. Response payloads include the new `moduleType` field.

8. **Frontend minimal changes:** The settings quality profiles page adds a module filter/grouping. Movie add/edit pages filter to movie profiles. TV add/edit pages filter to TV profiles. The `useQualityProfiles` hook gains an optional `moduleType` parameter. Changes are minimal — just enough to keep the app functional with module-scoped profiles.

9. **Deletion guard — module-aware + slots:** After Phase 2, a movie profile's deletion guard checks `CountMoviesUsingProfile` only (not series). A TV profile checks `CountSeriesUsingProfile` only. Both check `CountSlotsUsingProfile` since slots are module-agnostic until Phase 8. This also fixes a pre-existing gap where slot usage was not checked during profile deletion.

10. **EnsureDefaults — per-module:** Default profile creation (`EnsureDefaults`) becomes per-module. If no profiles exist for a module, three defaults are created for that module. This is called for each enabled module at startup.

### Phase 2 Dependency Graph

```
Task 2.1 (migration SQL) ─────────────────→ Task 2.3 (sqlc queries)
                                                       ↓
                                              Task 2.4 (sqlc generate)
                                                       ↓
Task 2.2 (QualityDefinition impl.) ──→ Task 2.5 (quality service changes)
  (can start in parallel with 2.1)                     ↓
                                              Task 2.6 (API handler changes)
                                                       ↓
                                              Task 2.7 (update Go callers)
                                                       ↓
                                              Task 2.8 (frontend changes)
                                                       ↓
                                              Task 2.9 (phase validation)
```

**Parallelism:** Tasks 2.1 and 2.2 can run in parallel (2.1 is SQL/queries, 2.2 is Go interface implementation — no file overlap).

---

### Task 2.1: Migration SQL — Module-Scoped Quality Profiles ✅

**Create** `internal/database/migrations/071_module_scoped_quality_profiles.sql`

This migration:
1. Recreates `quality_profiles` with a `module_type` column and `UNIQUE(module_type, name)` constraint
2. Preserves original IDs as "movie" copies, creates new IDs for "tv" copies
3. Remaps TV-related FKs to the new TV copy IDs
4. Clears the import decisions cache

**Critical consideration:** Multiple tables reference `quality_profiles(id)`: `movies.quality_profile_id`, `series.quality_profile_id`, `version_slots.quality_profile_id`, `portal_users.movie_quality_profile_id`, `portal_users.tv_quality_profile_id`, `portal_invitations.movie_quality_profile_id`, `portal_invitations.tv_quality_profile_id`, and `import_decisions.quality_profile_id` (cached, not a FK). Note that migration 068 already split portal_users and portal_invitations into separate `movie_quality_profile_id` / `tv_quality_profile_id` columns — there is no single `quality_profile_id` on those tables. The table recreation must handle these carefully. SQLite's `PRAGMA foreign_keys` must be OFF during the migration to allow the drop/rename.

```sql
-- +goose Up
-- +goose StatementBegin

-- Disable FK enforcement during migration (Goose wraps in transaction)
PRAGMA foreign_keys = OFF;

-- 1. Create new quality_profiles table with module_type column
CREATE TABLE quality_profiles_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    module_type TEXT NOT NULL DEFAULT 'movie',
    cutoff INTEGER NOT NULL DEFAULT 0,
    items TEXT NOT NULL DEFAULT '[]',
    hdr_settings TEXT,
    video_codec_settings TEXT,
    audio_codec_settings TEXT,
    audio_channel_settings TEXT,
    upgrades_enabled INTEGER NOT NULL DEFAULT 1,
    allow_auto_approve INTEGER NOT NULL DEFAULT 0,
    upgrade_strategy TEXT NOT NULL DEFAULT 'balanced',
    cutoff_overrides_strategy INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(module_type, name)
);

-- 2. Copy existing profiles as 'movie' type (preserving original IDs)
INSERT INTO quality_profiles_new (
    id, name, module_type, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
)
SELECT
    id, name, 'movie', cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
FROM quality_profiles;

-- 3. Create 'tv' copies (new auto-generated IDs)
INSERT INTO quality_profiles_new (
    name, module_type, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
)
SELECT
    name, 'tv', cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
FROM quality_profiles;

-- 4. Build temporary mapping: old profile ID → new TV copy ID
-- Join on name since movie copies have the original IDs and TV copies have new IDs
CREATE TEMPORARY TABLE profile_id_map AS
SELECT qp.id AS old_id, qpn.id AS new_id
FROM quality_profiles qp
JOIN quality_profiles_new qpn ON qp.name = qpn.name AND qpn.module_type = 'tv';

-- 5. Update series.quality_profile_id to point to new TV copies
UPDATE series SET quality_profile_id = (
    SELECT new_id FROM profile_id_map WHERE old_id = series.quality_profile_id
) WHERE quality_profile_id IS NOT NULL
  AND quality_profile_id IN (SELECT old_id FROM profile_id_map);

-- 6. Update portal_users.tv_quality_profile_id to point to new TV copies
UPDATE portal_users SET tv_quality_profile_id = (
    SELECT new_id FROM profile_id_map WHERE old_id = portal_users.tv_quality_profile_id
) WHERE tv_quality_profile_id IS NOT NULL
  AND tv_quality_profile_id IN (SELECT old_id FROM profile_id_map);

-- 7. Update portal_invitations.tv_quality_profile_id to point to new TV copies
UPDATE portal_invitations SET tv_quality_profile_id = (
    SELECT new_id FROM profile_id_map WHERE old_id = portal_invitations.tv_quality_profile_id
) WHERE tv_quality_profile_id IS NOT NULL
  AND tv_quality_profile_id IN (SELECT old_id FROM profile_id_map);

-- 8. Drop temp mapping table
DROP TABLE profile_id_map;

-- 9. Clear import decisions cache (profile IDs are now stale for TV decisions)
DELETE FROM import_decisions;

-- 10. Drop old table, rename new
-- NOTE: version_slots.quality_profile_id references quality_profiles(id).
-- After this migration, slots will reference "movie" copies (original IDs preserved).
-- This is intentional — Phase 8 will refactor slots with per-module profile references.
DROP TABLE quality_profiles;
ALTER TABLE quality_profiles_new RENAME TO quality_profiles;

-- Re-enable FK enforcement
PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- Build reverse mapping: TV copy ID → original (movie copy) ID
CREATE TEMPORARY TABLE profile_reverse_map AS
SELECT qp_movie.id AS movie_id, qp_tv.id AS tv_id
FROM quality_profiles qp_movie
JOIN quality_profiles qp_tv ON qp_movie.name = qp_tv.name
WHERE qp_movie.module_type = 'movie' AND qp_tv.module_type = 'tv';

-- Revert series FKs to original IDs
UPDATE series SET quality_profile_id = (
    SELECT movie_id FROM profile_reverse_map WHERE tv_id = series.quality_profile_id
) WHERE quality_profile_id IN (SELECT tv_id FROM profile_reverse_map);

-- Revert portal_users.tv_quality_profile_id
UPDATE portal_users SET tv_quality_profile_id = (
    SELECT movie_id FROM profile_reverse_map WHERE tv_id = portal_users.tv_quality_profile_id
) WHERE tv_quality_profile_id IN (SELECT tv_id FROM profile_reverse_map);

-- Revert portal_invitations.tv_quality_profile_id
UPDATE portal_invitations SET tv_quality_profile_id = (
    SELECT movie_id FROM profile_reverse_map WHERE tv_id = portal_invitations.tv_quality_profile_id
) WHERE tv_quality_profile_id IN (SELECT tv_id FROM profile_reverse_map);

DROP TABLE profile_reverse_map;

-- Recreate without module_type
CREATE TABLE quality_profiles_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    cutoff INTEGER NOT NULL DEFAULT 0,
    items TEXT NOT NULL DEFAULT '[]',
    hdr_settings TEXT,
    video_codec_settings TEXT,
    audio_codec_settings TEXT,
    audio_channel_settings TEXT,
    upgrades_enabled INTEGER NOT NULL DEFAULT 1,
    allow_auto_approve INTEGER NOT NULL DEFAULT 0,
    upgrade_strategy TEXT NOT NULL DEFAULT 'balanced',
    cutoff_overrides_strategy INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Keep only movie copies (they have the original IDs)
INSERT INTO quality_profiles_old (
    id, name, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
)
SELECT
    id, name, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled, allow_auto_approve,
    upgrade_strategy, cutoff_overrides_strategy, created_at, updated_at
FROM quality_profiles WHERE module_type = 'movie';

DROP TABLE quality_profiles;
ALTER TABLE quality_profiles_old RENAME TO quality_profiles;

DELETE FROM import_decisions;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd
```

**Implementation notes for the subagent:**
- The migration number (071) assumes Phase 1's migration is 070. If Phase 1 added additional migrations, adjust accordingly — check `ls internal/database/migrations/ | tail -5` to find the latest number.
- SQLite `PRAGMA foreign_keys = OFF` must be OUTSIDE the transaction in some Goose configurations. If the migration fails due to FK violations, the subagent should test whether wrapping the PRAGMA in `-- +goose NO TRANSACTION` is needed. Goose's SQLite driver typically disables FK checking during migrations, but verify.
- Test the migration against a database with existing profiles, movies, series, portal users, and invitations. Verify FK integrity after migration: `PRAGMA foreign_key_check;` should return no violations.

**Verify:**
- Migration runs cleanly: `go test ./internal/database/... -run TestMigrations` (or however migrations are tested)
- After migration: `SELECT module_type, COUNT(*) FROM quality_profiles GROUP BY module_type` returns equal counts for 'movie' and 'tv'
- After migration: all `series.quality_profile_id` values reference profiles with `module_type = 'tv'`
- After migration: all `movies.quality_profile_id` values reference profiles with `module_type = 'movie'`
- `PRAGMA foreign_key_check;` returns empty (no FK violations)

---

### Task 2.2: Implement QualityDefinition on Module Stubs ✅

**Can run in parallel with Task 2.1** (no file overlap).

**Modify** `internal/modules/movie/module.go` — add a `QualityDefinition` implementation:

The movie module's `QualityDefinition` provides the 17 video quality tiers. For Phase 2, the actual parsing/scoring logic delegates to existing code in `internal/library/quality/`. The module acts as a thin facade.

```go
// In internal/modules/movie/module.go (or a new quality.go file in the same package)

// QualityItems returns the video quality tiers for the Movie module.
func (d *Descriptor) QualityItems() []module.QualityItemDef {
    // Returns the same 17 video qualities used by movies.
    // In the future, non-video modules (music, books) would return different tiers.
    return VideoQualityItems()
}

func (d *Descriptor) ParseQuality(releaseTitle string) (*module.QualityResult, error) {
    // Stub — delegates to existing parser in Phase 5
    return nil, nil
}

func (d *Descriptor) ScoreQuality(item module.QualityItemDef) int {
    // Stub — delegates to existing scorer in Phase 5
    return item.Weight
}

func (d *Descriptor) IsUpgrade(current, candidate module.QualityItemDef, profileID int64) (bool, error) {
    // Stub — upgrade logic lives on Profile, not the module.
    // This method exists for future flexibility but delegates to profile logic.
    return false, nil
}
```

**Spec deviation note:** The spec §4.2 defines `IsUpgrade(current, candidate QualityItem, profile Profile) → bool`, passing a full `Profile` object. This plan uses `profileID int64` instead, since the stub does not need the full profile and the quality service already has profile lookup by ID. The Phase 0 `QualityDefinition` interface definition is the canonical source — Phase 2's implementation must match whatever signature Phase 0 defined. If Phase 0 used the spec's `Profile` parameter, update this stub accordingly.

**Create** `internal/modules/shared/video_qualities.go` — shared quality definitions for video modules:

```go
package shared

import "github.com/slipstream/slipstream/internal/module"

// VideoQualityItems returns the 17 standard video quality tiers used by
// both the Movie and TV modules. Future non-video modules (music, books)
// would define their own quality items.
func VideoQualityItems() []module.QualityItemDef {
    return []module.QualityItemDef{
        {ID: 1, Name: "SDTV", Source: "tv", Resolution: 480, Weight: 1},
        {ID: 2, Name: "DVD", Source: "dvd", Resolution: 480, Weight: 2},
        {ID: 3, Name: "WEBRip-480p", Source: "webrip", Resolution: 480, Weight: 3},
        {ID: 4, Name: "HDTV-720p", Source: "tv", Resolution: 720, Weight: 4},
        {ID: 5, Name: "WEBRip-720p", Source: "webrip", Resolution: 720, Weight: 5},
        {ID: 6, Name: "WEBDL-720p", Source: "webdl", Resolution: 720, Weight: 6},
        {ID: 7, Name: "Bluray-720p", Source: "bluray", Resolution: 720, Weight: 7},
        {ID: 8, Name: "HDTV-1080p", Source: "tv", Resolution: 1080, Weight: 8},
        {ID: 9, Name: "WEBRip-1080p", Source: "webrip", Resolution: 1080, Weight: 9},
        {ID: 10, Name: "WEBDL-1080p", Source: "webdl", Resolution: 1080, Weight: 10},
        {ID: 11, Name: "Bluray-1080p", Source: "bluray", Resolution: 1080, Weight: 11},
        {ID: 12, Name: "Remux-1080p", Source: "remux", Resolution: 1080, Weight: 12},
        {ID: 13, Name: "HDTV-2160p", Source: "tv", Resolution: 2160, Weight: 13},
        {ID: 14, Name: "WEBRip-2160p", Source: "webrip", Resolution: 2160, Weight: 14},
        {ID: 15, Name: "WEBDL-2160p", Source: "webdl", Resolution: 2160, Weight: 15},
        {ID: 16, Name: "Bluray-2160p", Source: "bluray", Resolution: 2160, Weight: 16},
        {ID: 17, Name: "Remux-2160p", Source: "remux", Resolution: 2160, Weight: 17},
    }
}
```

**Modify** `internal/modules/tv/module.go` — same pattern, delegates to `shared.VideoQualityItems()`.

**Modify** `internal/module/quality_types.go` — flesh out the skeleton `QualityItemDef` type (created in Phase 0 as a minimal placeholder):

```go
// QualityItemDef defines a quality tier registered by a module.
// This is the module-facing type; the quality service uses its own internal Profile/QualityItem types.
type QualityItemDef struct {
    ID         int    `json:"id"`
    Name       string `json:"name"`
    Source     string `json:"source"`     // "bluray", "webdl", "tv", etc.
    Resolution int    `json:"resolution"` // 480, 720, 1080, 2160
    Weight     int    `json:"weight"`     // Higher = better quality
}
```

**Implementation notes for the subagent:**
- The `module.QualityDefinition` interface was defined in Phase 0 (`internal/module/interfaces.go`). Read it first to ensure method signatures match exactly.
- The Phase 0 plan used `QualityItem` as the type name in the interface. If Phase 0 used a different name, match it. The key point is that the type must have `ID`, `Name`, `Source`, `Resolution`, `Weight` fields.
- `ParseQuality`, `ScoreQuality`, and `IsUpgrade` are stub implementations that return zero values. They will be properly wired in Phase 5.
- Do NOT import from `internal/library/quality/` in the module packages — the module package should only import from `internal/module/` and `internal/modules/shared/`. This preserves the dependency direction: modules → module framework. The quality service imports from modules, not the other way around.

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go build ./internal/modules/tv/...` compiles
- `go build ./internal/modules/shared/...` compiles
- Both module descriptors satisfy the `QualityDefinition` interface (add compile-time check: `var _ module.QualityDefinition = (*Descriptor)(nil)`)

---

### Task 2.3: Update sqlc Queries — Module-Scoped Quality Profiles ✅

**Depends on:** Task 2.1 (migration must be written first so the schema is known).

**Modify** `internal/database/queries/settings.sql` — update quality profile queries:

```sql
-- Quality Profiles (module-scoped after migration 071)

-- name: GetQualityProfile :one
SELECT * FROM quality_profiles WHERE id = ? LIMIT 1;

-- name: ListQualityProfiles :many
SELECT * FROM quality_profiles ORDER BY module_type, name;

-- name: ListQualityProfilesByModule :many
SELECT * FROM quality_profiles WHERE module_type = ? ORDER BY name;

-- name: CreateQualityProfile :one
INSERT INTO quality_profiles (
    name, module_type, cutoff, items, hdr_settings, video_codec_settings,
    audio_codec_settings, audio_channel_settings, upgrades_enabled,
    allow_auto_approve, upgrade_strategy, cutoff_overrides_strategy
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateQualityProfile :one
UPDATE quality_profiles SET
    name = ?,
    cutoff = ?,
    items = ?,
    hdr_settings = ?,
    video_codec_settings = ?,
    audio_codec_settings = ?,
    audio_channel_settings = ?,
    upgrades_enabled = ?,
    allow_auto_approve = ?,
    upgrade_strategy = ?,
    cutoff_overrides_strategy = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteQualityProfile :exec
DELETE FROM quality_profiles WHERE id = ?;

-- name: GetQualityProfileByName :one
SELECT * FROM quality_profiles WHERE name = ? AND module_type = ? LIMIT 1;

-- name: CountMoviesUsingProfile :one
SELECT COUNT(*) FROM movies WHERE quality_profile_id = ?;

-- name: CountSeriesUsingProfile :one
SELECT COUNT(*) FROM series WHERE quality_profile_id = ?;

-- name: CountQualityProfilesByModule :one
SELECT COUNT(*) FROM quality_profiles WHERE module_type = ?;
```

**Key changes from current queries:**
1. `ListQualityProfiles` — add `ORDER BY module_type, name` for consistent ordering
2. `ListQualityProfilesByModule` — new query, filters by `module_type`
3. `CreateQualityProfile` — add `module_type` parameter (second position after `name`)
4. `GetQualityProfileByName` — add `module_type` parameter (profiles are unique per `(module_type, name)`)
5. `CountQualityProfilesByModule` — new query for EnsureDefaults per-module check
6. `UpdateQualityProfile` — unchanged (module_type is immutable after creation; the column is not in the SET clause)
7. `CountMoviesUsingProfile` and `CountSeriesUsingProfile` — unchanged (they're correct: movie profiles should only have movie references, TV profiles only series references)

**Note:** The `settings.sql` file also contains root_folders, downloads, and history queries. Phase 1 may have already renamed columns in those queries (e.g., `media_type` → `module_type`). The subagent must read the file as it exists AFTER Phase 1 and only modify the quality profile queries. Do not touch the root_folders, downloads, or history queries.

**Verify:** The modified `settings.sql` has valid SQL syntax. All query names are unique. The `module_type` parameter position matches the migration schema.

---

### Task 2.4: Run sqlc Generate ✅

**Depends on:** Task 2.3 (query changes must be in place).

**Run:**
```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

**Verify:**
- `internal/database/sqlc/settings.sql.go` is regenerated with updated function signatures
- `CreateQualityProfileParams` struct now includes a `ModuleType string` field
- `GetQualityProfileByNameParams` struct now includes `ModuleType string`
- New functions `ListQualityProfilesByModule` and `CountQualityProfilesByModule` exist
- `QualityProfile` model in `models.go` includes `ModuleType string`
- `go build ./internal/database/sqlc/...` compiles

---

### Task 2.5: Update Quality Service — Module-Aware Operations ✅

**Depends on:** Task 2.4 (needs regenerated sqlc types) and Task 2.2 (needs `QualityItemDef` type).

**Modify** `internal/library/quality/service.go`:

1. **Add module quality registration:**

```go
// moduleQualities stores per-module quality item definitions registered at startup.
type moduleQualityRegistry struct {
    mu    sync.RWMutex
    items map[string][]QualityItemDef // module_type → quality items
}

// Service gains a new field:
type Service struct {
    queries               *sqlc.Queries
    logger                *zerolog.Logger
    importDecisionCleaner ImportDecisionCleaner
    moduleQualities       moduleQualityRegistry
}

// RegisterModuleQualities registers quality items for a module.
// Called at startup for each enabled module.
func (s *Service) RegisterModuleQualities(moduleType string, items []QualityItemDef) {
    s.moduleQualities.mu.Lock()
    defer s.moduleQualities.mu.Unlock()
    if s.moduleQualities.items == nil {
        s.moduleQualities.items = make(map[string][]QualityItemDef)
    }
    s.moduleQualities.items[moduleType] = items
    s.logger.Info().Str("moduleType", moduleType).Int("qualityCount", len(items)).
        Msg("Registered quality items for module")
}

// GetQualitiesForModule returns quality items for a specific module.
// Falls back to PredefinedQualities if no module-specific items registered.
func (s *Service) GetQualitiesForModule(moduleType string) []Quality {
    s.moduleQualities.mu.RLock()
    defer s.moduleQualities.mu.RUnlock()
    items, ok := s.moduleQualities.items[moduleType]
    if !ok || len(items) == 0 {
        return PredefinedQualities
    }
    // Convert QualityItemDef → Quality
    result := make([]Quality, len(items))
    for i, item := range items {
        result[i] = Quality{
            ID: item.ID, Name: item.Name, Source: item.Source,
            Resolution: item.Resolution, Weight: item.Weight,
        }
    }
    return result
}
```

The `QualityItemDef` type must be defined in the quality package or imported from the module package. To avoid a dependency on `internal/module/` (which could create import cycles), define a local copy:

```go
// QualityItemDef mirrors module.QualityItemDef for registration without importing the module package.
type QualityItemDef struct {
    ID         int
    Name       string
    Source     string
    Resolution int
    Weight     int
}
```

2. **Update `Create` — require moduleType:**

```go
func (s *Service) Create(ctx context.Context, input *CreateProfileInput) (*Profile, error) {
    if input.Name == "" {
        return nil, ErrInvalidProfile
    }
    if input.ModuleType == "" {
        return nil, fmt.Errorf("%w: moduleType is required", ErrInvalidProfile)
    }
    // ... existing serialization logic ...
    row, err := s.queries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
        Name:       input.Name,
        ModuleType: input.ModuleType, // NEW
        // ... rest unchanged ...
    })
    // ...
}
```

3. **Add `ModuleType` to `CreateProfileInput` and `Profile`:**

```go
type CreateProfileInput struct {
    Name       string `json:"name"`
    ModuleType string `json:"moduleType"` // NEW — required
    // ... rest unchanged ...
}

type Profile struct {
    ID         int64  `json:"id"`
    Name       string `json:"name"`
    ModuleType string `json:"moduleType"` // NEW
    // ... rest unchanged ...
}
```

`UpdateProfileInput` does NOT get a `ModuleType` field — module type is immutable after creation.

4. **Update `List` — add module-filtered variant:**

```go
// List returns all quality profiles, optionally filtered by module type.
func (s *Service) List(ctx context.Context) ([]*Profile, error) {
    rows, err := s.queries.ListQualityProfiles(ctx)
    // ... existing logic ...
}

// ListByModule returns quality profiles for a specific module.
func (s *Service) ListByModule(ctx context.Context, moduleType string) ([]*Profile, error) {
    rows, err := s.queries.ListQualityProfilesByModule(ctx, moduleType)
    if err != nil {
        return nil, fmt.Errorf("failed to list quality profiles for module %s: %w", moduleType, err)
    }
    profiles := make([]*Profile, 0, len(rows))
    for _, row := range rows {
        p, err := s.rowToProfile(row)
        if err != nil {
            s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to parse quality profile")
            continue
        }
        profiles = append(profiles, p)
    }
    return profiles, nil
}
```

5. **Update `Delete` — module-aware guard + slot check:**

```go
func (s *Service) Delete(ctx context.Context, id int64) error {
    // Get profile to determine module type
    profile, err := s.Get(ctx, id)
    if err != nil {
        return err
    }

    // Check module-specific usage
    switch profile.ModuleType {
    case "movie":
        count, err := s.queries.CountMoviesUsingProfile(ctx, sql.NullInt64{Int64: id, Valid: true})
        if err != nil {
            return fmt.Errorf("failed to check movie usage: %w", err)
        }
        if count > 0 {
            return ErrProfileInUse
        }
    case "tv":
        count, err := s.queries.CountSeriesUsingProfile(ctx, sql.NullInt64{Int64: id, Valid: true})
        if err != nil {
            return fmt.Errorf("failed to check series usage: %w", err)
        }
        if count > 0 {
            return ErrProfileInUse
        }
    }

    // Check slot usage (slots are module-agnostic until Phase 8)
    slotCount, err := s.queries.CountSlotsUsingProfile(ctx, sql.NullInt64{Int64: id, Valid: true})
    if err != nil {
        return fmt.Errorf("failed to check slot usage: %w", err)
    }
    if slotCount > 0 {
        return ErrProfileInUse
    }

    if err := s.queries.DeleteQualityProfile(ctx, id); err != nil {
        return fmt.Errorf("failed to delete quality profile: %w", err)
    }

    s.logger.Info().Int64("id", id).Str("moduleType", profile.ModuleType).Msg("Deleted quality profile")
    return nil
}
```

6. **Update `EnsureDefaults` — per-module:**

```go
// EnsureDefaults creates default profiles for a specific module if none exist.
func (s *Service) EnsureDefaults(ctx context.Context, moduleType string) error {
    count, err := s.queries.CountQualityProfilesByModule(ctx, moduleType)
    if err != nil {
        return fmt.Errorf("failed to count profiles for module %s: %w", moduleType, err)
    }
    if count > 0 {
        return nil
    }

    defaults := []struct {
        profile Profile
    }{
        {DefaultProfile()},
        {HD1080pProfile()},
        {Ultra4KProfile()},
    }

    for _, d := range defaults {
        p := d.profile
        upgradesEnabled := p.UpgradesEnabled
        _, err := s.Create(ctx, &CreateProfileInput{
            Name:            p.Name,
            ModuleType:      moduleType,
            Cutoff:          p.Cutoff,
            UpgradesEnabled: &upgradesEnabled,
            Items:           p.Items,
        })
        if err != nil {
            s.logger.Warn().Err(err).Str("name", p.Name).Str("moduleType", moduleType).
                Msg("Failed to create default profile")
        }
    }

    s.logger.Info().Str("moduleType", moduleType).Msg("Created default quality profiles for module")
    return nil
}
```

The caller at startup should call `EnsureDefaults` for each enabled module:
```go
for _, moduleType := range []string{"movie", "tv"} {
    if err := qualityService.EnsureDefaults(ctx, moduleType); err != nil {
        logger.Warn().Err(err).Str("moduleType", moduleType).Msg("Failed to ensure default profiles")
    }
}
```

7. **Update `GetByName` — require moduleType:**

```go
func (s *Service) GetByName(ctx context.Context, name string, moduleType string) (*Profile, error) {
    row, err := s.queries.GetQualityProfileByName(ctx, sqlc.GetQualityProfileByNameParams{
        Name:       name,
        ModuleType: moduleType,
    })
    // ...
}
```

8. **Update `RecalculateStatusForProfile` — module-aware:**

```go
func (s *Service) RecalculateStatusForProfile(ctx context.Context, profileID int64) (int, error) {
    profile, err := s.Get(ctx, profileID)
    if err != nil {
        return 0, err
    }

    var updated int
    switch profile.ModuleType {
    case "movie":
        updated, err = s.recalculateMovieStatuses(ctx, profileID, profile)
    case "tv":
        updated, err = s.recalculateEpisodeStatuses(ctx, profileID, profile)
    default:
        s.logger.Warn().Str("moduleType", profile.ModuleType).Msg("Unknown module type for status recalculation")
        return 0, nil
    }
    if err != nil {
        return updated, err
    }

    s.logger.Info().Int64("profileId", profileID).Str("moduleType", profile.ModuleType).
        Int("updated", updated).Msg("Recalculated media status for profile change")
    return updated, nil
}
```

9. **Update `rowToProfile` — include ModuleType:**

```go
func (s *Service) rowToProfile(row *sqlc.QualityProfile) (*Profile, error) {
    // ... existing deserialization ...
    p := &Profile{
        ID:         row.ID,
        Name:       row.Name,
        ModuleType: row.ModuleType, // NEW
        // ... rest unchanged ...
    }
    // ...
}
```

**Implementation notes for the subagent:**
- Read `internal/library/quality/service.go` and `internal/library/quality/profile.go` FIRST before making changes.
- The `CountSlotsUsingProfile` query already exists in `internal/database/queries/slots.sql`. The generated Go code is in `internal/database/sqlc/slots.sql.go`. Just call it — no query changes needed.
- Keep the `GetQualities() []Quality` method as-is for backward compatibility. Add `GetQualitiesForModule(moduleType string) []Quality` alongside it.
- The `PredefinedQualities` global variable stays as-is. It serves as the fallback when no module-specific qualities are registered.
- All callers of `EnsureDefaults` must be updated to pass a module type. Find them with: `grep -r "EnsureDefaults" --include="*.go"`.
- All callers of `GetByName` must be updated to pass a module type. Find them with: `grep -r "GetByName" --include="*.go" internal/`.
- Do NOT import `internal/module/` from the quality package. Use the local `QualityItemDef` copy to avoid dependency cycles.

**Verify:**
- `go build ./internal/library/quality/...` compiles
- All existing tests in `internal/library/quality/` still pass (may need `ModuleType` added to test fixtures)
- `go vet ./internal/library/quality/...` clean

---

### Task 2.6: Update API Handlers — Module-Aware Endpoints ✅

**Depends on:** Task 2.5 (service changes must be in place).

**Modify** `internal/library/quality/handlers.go`:

1. **Update `List` handler — accept optional `moduleType` query param:**

```go
func (h *Handlers) List(c echo.Context) error {
    moduleType := c.QueryParam("moduleType")
    var profiles []*Profile
    var err error

    if moduleType != "" {
        profiles, err = h.service.ListByModule(c.Request().Context(), moduleType)
    } else {
        profiles, err = h.service.List(c.Request().Context())
    }
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    return c.JSON(http.StatusOK, profiles)
}
```

2. **Update `Create` handler — moduleType from request body:**

The `CreateProfileInput` already includes `ModuleType` (from Task 2.5). The handler just binds it:

```go
func (h *Handlers) Create(c echo.Context) error {
    var input CreateProfileInput
    if err := c.Bind(&input); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    if input.ModuleType == "" {
        return echo.NewHTTPError(http.StatusBadRequest, "moduleType is required")
    }
    // ... rest unchanged ...
}
```

3. **Update `ListQualities` handler — accept optional `moduleType`:**

```go
func (h *Handlers) ListQualities(c echo.Context) error {
    moduleType := c.QueryParam("moduleType")
    if moduleType != "" {
        return c.JSON(http.StatusOK, h.service.GetQualitiesForModule(moduleType))
    }
    return c.JSON(http.StatusOK, h.service.GetQualities())
}
```

4. **No changes to `Get`, `Update`, `Delete`, `ListAttributes`, `CheckExclusivity`** — these operate by profile ID, which is already module-scoped after migration. The response payloads automatically include `moduleType` because `Profile` struct was updated in Task 2.5.

**Verify:**
- `go build ./internal/library/quality/...` compiles
- Manual test: `curl localhost:8080/api/v1/qualityprofiles` returns profiles with `moduleType` field
- Manual test: `curl localhost:8080/api/v1/qualityprofiles?moduleType=movie` returns only movie profiles
- Manual test: `curl localhost:8080/api/v1/qualityprofiles/qualities?moduleType=tv` returns quality items

---

### Task 2.7: Update Go Callers — Module-Aware Profile References ✅

**Depends on:** Task 2.5 (service API changes must be in place).

Audit all code that calls into the quality service and update for module-type awareness. Key call sites:

1. **`internal/api/startup.go` (or wherever `EnsureDefaults` is called):**

Find the existing `EnsureDefaults` call and replace with per-module calls:
```go
// Before:
qualityService.EnsureDefaults(ctx)

// After:
for _, mt := range []string{"movie", "tv"} {
    if err := qualityService.EnsureDefaults(ctx, mt); err != nil {
        logger.Warn().Err(err).Str("moduleType", mt).Msg("Failed to ensure default quality profiles")
    }
}
```

Search: `grep -r "EnsureDefaults" --include="*.go" internal/` to find all call sites.

2. **`internal/library/quality/service.go` — `GetByName` callers:**

Search: `grep -r "GetByName" --include="*.go" internal/` to find all call sites. Each must now pass a `moduleType` string. Common callers:
- arr-import quality mapping (`internal/arrimport/`) — when importing from Radarr, use `"movie"`; from Sonarr, use `"tv"`.
- Dev mode sample data creation — must specify module type.

3. **Portal provisioner — quality profile resolution:**

The portal provisioner (`internal/portal/provisioner/service.go`) resolves quality profiles when creating media from requests. Currently it uses a default profile fallback. After Phase 2, the fallback must be module-aware:

```go
// Before (in EnsureMovieInLibrary):
profile, _ := qualityService.Get(ctx, defaultProfileID)

// After — if no user profile set, get default for the correct module:
// The provisioner already knows whether it's creating a movie or series,
// so it just needs to pass the right module type to ListByModule.
```

The specific change: when the provisioner falls back to a default profile (because the user has no profile assigned), it should query `ListByModule("movie")` or `ListByModule("tv")` and pick the first one, rather than querying all profiles. However, since the portal user already has separate `movie_quality_profile_id` and `tv_quality_profile_id` fields, and these now correctly reference module-scoped profiles (thanks to the migration), no changes may be needed if the provisioner always uses the user's assigned profile. **Verify this by reading the provisioner code.**

4. **Arr import quality profile mapping (`internal/arrimport/`):**

When importing from Radarr/Sonarr, quality profiles are mapped. After Phase 2, the created profiles must specify `moduleType`:
- Radarr import → creates profiles with `moduleType: "movie"`
- Sonarr import → creates profiles with `moduleType: "tv"`

Search: `grep -r "CreateProfileInput" --include="*.go" internal/arrimport/` to find the exact call site.

5. **Dev mode setup:**

If dev mode creates quality profiles, ensure module type is specified. Search: `grep -r "quality" --include="*.go" internal/devmode/` or similar.

**Implementation notes for the subagent:**
- This task is a targeted audit. The subagent should:
  1. Search for all callers of `EnsureDefaults`, `GetByName`, `Create` (on the quality service)
  2. Update each to pass the correct module type
  3. Ensure no compilation errors
- Do NOT change the scoring pipeline (`internal/indexer/scoring/`) or autosearch (`internal/autosearch/`). Those use profile IDs, not module types, and continue to work as-is.
- Do NOT change the portal user handlers (`internal/portal/admin/`). They already use `movie_quality_profile_id` / `tv_quality_profile_id`, and the migration remapped these to the correct module-scoped profile IDs.

**Verify:**
- `go build ./...` compiles (full project build)
- `go vet ./...` clean
- `make test` — all existing tests pass

---

### Task 2.8: Frontend Changes — Module-Aware Quality Profiles ✅

**Depends on:** Tasks 2.5 and 2.6 (backend API must be stable).

1. **Update TypeScript types** — `web/src/types/quality-profile.ts`:

```typescript
export type QualityProfile = {
  id: number
  name: string
  moduleType: string  // NEW — "movie" or "tv"
  cutoff: number
  // ... rest unchanged ...
}

export type CreateQualityProfileInput = {
  name: string
  moduleType: string  // NEW — required
  cutoff: number
  // ... rest unchanged ...
}
```

`UpdateQualityProfileInput` does NOT get `moduleType` — it's immutable.

2. **Update API client** — `web/src/api/quality-profiles.ts`:

```typescript
export const qualityProfilesApi = {
  list: (moduleType?: string) => {
    const params = moduleType ? `?moduleType=${moduleType}` : ''
    return apiFetch<QualityProfile[]>(`/qualityprofiles${params}`)
  },
  // ... rest unchanged (create already sends full body including moduleType) ...
}
```

3. **Update hooks** — `web/src/hooks/use-quality-profiles.ts`:

```typescript
export function useQualityProfiles(moduleType?: string) {
  return useQuery({
    queryKey: moduleType
      ? [...qualityProfileKeys.list(), moduleType]
      : qualityProfileKeys.list(),
    queryFn: () => qualityProfilesApi.list(moduleType),
  })
}
```

4. **Update movie add/edit pages** — pass `moduleType: 'movie'` to `useQualityProfiles`:

Files to update (search for `useQualityProfiles()` calls in movie-related files):
- `web/src/routes/movies/use-add-movie.ts` → `useQualityProfiles('movie')`
- `web/src/routes/movies/use-movie-detail.ts` → `useQualityProfiles('movie')`
- `web/src/routes/movies/use-movie-list.ts` → `useQualityProfiles('movie')` (if used for filter/toolbar)

5. **Update TV add/edit pages** — pass `moduleType: 'tv'`:

- `web/src/routes/series/use-add-series.ts` → `useQualityProfiles('tv')`
- `web/src/routes/series/use-series-detail.ts` → `useQualityProfiles('tv')`
- `web/src/routes/series/use-series-list.ts` → `useQualityProfiles('tv')` (if used for filter/toolbar)

6. **Update shared components that call `useQualityProfiles()` directly:**

- `web/src/components/media/use-media-edit-dialog.ts` — calls `useQualityProfiles()`. Needs a `moduleType` parameter (inferred from whether editing a movie or series). The parent component must pass this context.
- `web/src/components/qualityprofiles/use-quality-profile-dialog.ts` — calls `useQualityProfiles()`. Used for the create/edit dialog. When creating from a module-filtered context, pass the module type. When creating from the settings "All" view, leave unfiltered or accept a `moduleType` prop.

7. **Update shared components that receive profiles as props (do NOT call hook directly):**

These components receive `qualityProfiles` as a prop from their parent. The parent is responsible for passing filtered data:
- `web/src/components/media/add-media-configure.tsx` — receives `qualityProfiles` prop. Parent (add movie/series pages) must pass module-filtered profiles.
- `web/src/components/media/media-list-toolbar.tsx` — receives `qualityProfiles` prop. Parent (movie/series list pages) must pass module-filtered profiles.
- `web/src/routes/requests-admin/profile-select.tsx` — generic select component receiving `qualityProfiles` prop. Used by invite-dialog and user-edit-dialog for per-module profile selection. No changes needed if parent passes the correct filtered list.
- `web/src/routes/requests-admin/invite-dialog.tsx` — receives `qualityProfiles` prop. Already has separate movie/TV profile selectors. Parent must pass module-filtered profiles per selector.
- `web/src/routes/requests-admin/user-edit-dialog.tsx` — same pattern as invite-dialog.

8. **Update quality profiles settings page** — `web/src/components/settings/sections/quality-profiles-section.tsx`:

The settings page shows ALL profiles. Add visual grouping or a tab/filter by module type:
- Option A (simpler): Show all profiles with a `moduleType` badge/label on each profile card.
- Option B (better UX): Add a filter dropdown at the top: "All", "Movies", "TV".

The create dialog must include a `moduleType` selector. Default to the currently filtered module type, or show a required dropdown if viewing "All".

9. **Update portal admin hook** — `web/src/routes/requests-admin/use-request-users-page.ts`:

This is the actual `useQualityProfiles()` caller for portal admin pages. It fetches all profiles and passes them down to `invite-dialog.tsx`, `user-edit-dialog.tsx`, and `profile-select.tsx` as props. After Phase 2, it should either:
- Call `useQualityProfiles('movie')` and `useQualityProfiles('tv')` separately, passing the filtered lists to child components for their movie/TV selectors.
- Or call `useQualityProfiles()` (all) and filter client-side by `moduleType` before passing to children.

10. **Update missing/upgradable and slots pages:**

These callers should continue showing ALL profiles (they operate across both modules):
- `web/src/routes/missing/use-missing-page.ts` → keep `useQualityProfiles()` (no module filter — builds quality name maps for both movie and TV items)
- `web/src/components/slots/profile-match-tester.tsx` → keep `useQualityProfiles()` (dev tool, shows all profiles)
- `web/src/components/slots/use-resolve-config-modal.ts` → keep `useQualityProfiles()` (slot conflicts span modules until Phase 8)
- `web/src/components/settings/sections/use-version-slots-section.ts` → keep `useQualityProfiles()` (slot config is module-agnostic until Phase 8)

11. **Update add-to-library hook** — `web/src/routes/requests-admin/use-add-to-library.ts`:

Currently takes the first profile as default. After Phase 2, filter by request media type: use the first movie profile for movie requests, first TV profile for TV requests.

12. **Update arr-import quality mapping:**

- `web/src/components/arr-import/mapping-step.tsx` — calls `useQualityProfiles()`. Pass module type based on import source: `'movie'` for Radarr, `'tv'` for Sonarr.
- `web/src/components/arr-import/quality-profile-mapping.tsx` — receives profiles as prop. No hook changes needed, but parent must pass module-filtered profiles.
- `web/src/components/arr-import/preview-step.tsx` — calls `useQualityProfiles()`. Same module filtering as mapping-step.

**Implementation notes for the subagent:**
- Search for ALL uses of `useQualityProfiles` in the codebase: `grep -r "useQualityProfiles" --include="*.{ts,tsx}" web/src/`
- Distinguish between **hook callers** (files that call `useQualityProfiles()` directly) and **prop receivers** (files that receive `qualityProfiles` as a prop from their parent). Hook callers need the `moduleType` parameter added. Prop receivers need no hook change — their parent must pass filtered data.
- For each hook caller, determine the context (movie page, TV page, settings page, shared component) and pass the appropriate `moduleType`.
- Pages that show ALL profiles (settings, missing/upgradable, slots) should keep `useQualityProfiles()` without a module filter.
- The create dialog must have a `moduleType` field. When creating from a module-filtered view, pre-fill and optionally hide the selector. When creating from the "All" view, show a required selector.
- Do NOT change the `PREDEFINED_QUALITIES` constant — the frontend hardcoded list stays as-is for now (Phase 2, Architecture Decision #6 — the `/qualityprofiles/qualities` endpoint is now module-aware, but the frontend doesn't yet dynamically load quality items. That's a Phase 10 frontend cleanup task).
- Run `cd web && bun run lint` after changes and fix any issues.

**Verify:**
- `cd web && bun run lint` — no new lint errors
- `cd web && bun run build` — builds successfully
- Manual test: movie add page shows only movie profiles in dropdown
- Manual test: series add page shows only TV profiles in dropdown
- Manual test: settings quality profiles page shows module type on each profile
- Manual test: creating a profile requires selecting a module type

---

### Task 2.9: Phase 2 Validation ✅

**Run all of these after all Phase 2 tasks complete:**

1. `make build` — full project builds (backend + frontend)
2. `make test` — all Go tests pass
3. `make lint` — no new Go lint issues
4. `cd web && bun run lint` — no new frontend lint issues
5. `cd web && bun run build` — frontend builds

**Integration verification (manual or via test):**
- Start the app with a fresh database → default profiles are created for both modules (3 movie + 3 TV = 6 total)
- Start the app with an existing database (pre-migration) → profiles are duplicated, FKs are correct
- Create a movie → can only assign movie quality profiles
- Create a series → can only assign TV quality profiles
- Edit a quality profile → status recalculation only runs for the correct module
- Delete a quality profile → deletion guard checks correct module's entities + slots
- Portal: create invitation with movie/TV quality profiles → profiles reference correct module-scoped copies

---

### Phase 2 File Inventory

### New files created in Phase 2
```
internal/database/migrations/071_module_scoped_quality_profiles.sql
internal/modules/shared/video_qualities.go
```

### Existing files modified in Phase 2

**Backend — module stubs and framework:**
```
internal/module/quality_types.go (flesh out QualityItemDef)
internal/modules/movie/module.go (QualityDefinition implementation)
internal/modules/tv/module.go (QualityDefinition implementation)
```

**Backend — database and queries:**
```
internal/database/queries/settings.sql (quality profile queries — module_type)
internal/database/sqlc/* (regenerated)
```

**Backend — quality service:**
```
internal/library/quality/service.go (module-aware CRUD, registration, EnsureDefaults, RecalculateStatus)
internal/library/quality/profile.go (ModuleType on Profile and CreateProfileInput)
internal/library/quality/handlers.go (moduleType query param, create validation)
internal/library/quality/profile_test.go (add ModuleType to test fixtures)
internal/library/quality/matcher_test.go (if affected)
internal/library/quality/status_test.go (if affected)
```

**Backend — callers:**
```
internal/arrimport/*.go (module type on profile creation)
cmd/slipstream/main.go (EnsureDefaults per-module loop)
internal/api/server.go (EnsureDefaults wrapper — update signature)
internal/api/devmode.go (EnsureDefaults per-module in dev mode sample data)
```

**Frontend — types, API, hooks:**
```
web/src/types/quality-profile.ts (moduleType field)
web/src/api/quality-profiles.ts (moduleType param on list)
web/src/hooks/use-quality-profiles.ts (moduleType param)
```

**Frontend — hook callers (pass moduleType to useQualityProfiles):**
```
web/src/routes/movies/use-add-movie.ts (moduleType: 'movie')
web/src/routes/movies/use-movie-detail.ts (moduleType: 'movie')
web/src/routes/movies/use-movie-list.ts (moduleType: 'movie')
web/src/routes/series/use-add-series.ts (moduleType: 'tv')
web/src/routes/series/use-series-detail.ts (moduleType: 'tv')
web/src/routes/series/use-series-list.ts (moduleType: 'tv')
web/src/components/media/use-media-edit-dialog.ts (moduleType from context)
web/src/components/qualityprofiles/use-quality-profile-dialog.ts (moduleType from context)
web/src/routes/requests-admin/use-request-users-page.ts (split into movie/tv calls or filter client-side)
web/src/routes/requests-admin/use-add-to-library.ts (moduleType from request media type)
web/src/components/arr-import/mapping-step.tsx (moduleType from import source)
web/src/components/arr-import/preview-step.tsx (moduleType from import source)
```

**Frontend — hook callers (keep unfiltered, show ALL profiles):**
```
web/src/routes/missing/use-missing-page.ts (cross-module quality name maps)
web/src/components/slots/profile-match-tester.tsx (dev tool, all profiles)
web/src/components/slots/use-resolve-config-modal.ts (slots are module-agnostic until Phase 8)
web/src/components/settings/sections/use-version-slots-section.ts (slots config, module-agnostic until Phase 8)
web/src/components/settings/sections/quality-profiles-section.tsx (module grouping + create moduleType)
```

**Frontend — prop receivers (no hook changes, parent passes filtered data):**
```
web/src/components/media/add-media-configure.tsx (receives filtered qualityProfiles prop)
web/src/components/media/media-list-toolbar.tsx (receives filtered qualityProfiles prop)
web/src/routes/requests-admin/invite-dialog.tsx (receives filtered qualityProfiles prop per module)
web/src/routes/requests-admin/user-edit-dialog.tsx (receives filtered qualityProfiles prop per module)
web/src/routes/requests-admin/profile-select.tsx (generic select, receives filtered prop)
web/src/components/arr-import/quality-profile-mapping.tsx (receives filtered prop from mapping-step)
```

---

