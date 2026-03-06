# Module System Architecture Specification

## Overview

SlipStream currently manages movies and TV shows as hard-coded media types. This specification defines a **module system** that extracts the concept of "media type" into a pluggable architecture, enabling contributors to add support for new media types (music, books, audiobooks, etc.) without modifying core framework code.

Movies and TV will be refactored to become the first two modules, validating the framework design against real, complex use cases from day one.

---

## Terminology

- **Module**: A self-contained media type implementation (e.g., Movie Module, TV Module, Music Module). Each module defines its data model, metadata providers, search strategies, and UI components.
- **Framework**: The shared infrastructure that modules plug into — database management, download pipeline, search routing, notification dispatch, portal, UI shell.
- **Node**: An entity in a module's hierarchy (e.g., a Movie, a Series, a Season, an Episode, an Artist, an Album, a Track).
- **Node Level**: A tier in a module's hierarchy tree. Each level has a name, properties, and relationships to adjacent levels.
- **Leaf Node**: The file-bearing node level — the deepest level that holds actual media files (movies, episodes, tracks).
- **Root Node**: The top-level node that represents the library entry (a movie, a series, an artist).

---

## 1. Module Declaration

### 1.1 Interface Bundle

A module is defined as a Go struct implementing a bundle of interfaces. The framework discovers modules through a registration function called at startup.

**Required interfaces** (every module must implement):

| Interface | Purpose |
|---|---|
| `ModuleDescriptor` | Identity: ID, name, icon, theme color, node schema declaration |
| `MetadataProvider` | Search for media, fetch metadata by external ID, get extended info |
| `SearchStrategy` | Define search granularity, category mappings, result filtering, title matching |
| `ImportHandler` | Match completed downloads to library entities, create file records, multi-file support |
| `PathGenerator` | Declare folder path templates with module-specific variables for on-disk organization |
| `NamingProvider` | Declare file naming templates, token contexts, format sub-variants, and renaming options |
| `CalendarProvider` | Return items with dates for the unified calendar view |
| `QualityDefinition` | Declare quality tiers, scoring logic, and upgrade rules |
| `WantedCollector` | Collect missing/upgradable items for auto-search and the wanted dashboard |
| `MonitoringPresets` | Define monitoring preset strategies (e.g., "all", "future only", "first season") |
| `FileParser` | Parse filenames to extract module-specific identifiers (season/episode numbers, track numbers, etc.) |
| `MockFactory` | Create mock metadata, sample library data, and test root folders for dev mode |
| `NotificationEvents` | Declare the module's notification event catalog |
| `ReleaseDateResolver` | Compute the framework `availability_date` from module-specific date fields |
| `RouteProvider` | Register module-specific REST API routes |
| `TaskProvider` | Declare scheduled tasks (autosearch, RSS sync, release date transitions, etc.) |

**Optional interfaces:**

| Interface | Purpose |
|---|---|
| `PortalProvisioner` | Enable portal request support: validate requests, provision media, check availability |
| `SlotSupport` | Enable version slot support for multi-quality file management |
| `ArrImportAdapter` | Enable import from external *arr applications (Radarr, Sonarr, Lidarr, etc.) |

### 1.2 Node Schema Declaration

Each module declares its node tree structure via `ModuleDescriptor.NodeSchema()`. The schema defines:

- **Levels**: An ordered list of node levels from root to leaf. Each level declares:
  - `Name` (string): e.g., "series", "season", "episode"
  - `PluralName` (string): e.g., "series", "seasons", "episodes"
  - `IsRoot` (bool): whether this is the top-level library entry
  - `IsLeaf` (bool): whether this level holds files
  - `Properties`: module-specific fields for this level (beyond the universal fields provided by the framework)
  - `HasMonitored` (bool): whether this level participates in monitoring cascade
  - `Searchable` (bool): whether items at this level can be individually searched
  - `IsSpecial` (bool): whether this level can have "special" instances with distinct path handling (e.g., TV Season 0 / Specials)
  - `SupportsMultiEntityFiles` (bool): whether a single file can map to multiple entities at this level (e.g., multi-episode files S01E01E02)
  - `FormatVariants` ([]string): sub-variants that affect naming templates (e.g., TV episode has "standard", "daily", "anime")

**Examples:**

```
Movie Module:   [Movie(root, leaf)]                          — flat, 1 level
TV Module:      [Series(root), Season, Episode(leaf)]        — 3 levels
Music Module:   [Artist(root), Album, Track(leaf)]           — 3 levels
Book Module:    [Author(root), Book(leaf)]                   — 2 levels
```

### 1.3 Module Loading

Modules are **compiled into the binary** and **user-togglable** via settings.

- All modules are registered at compile time through a central registration function.
- Users enable/disable modules from the Settings UI.
- Disabling a module:
  - Skips its routes, scheduled tasks, and background workers.
  - Hides its entries from the sidebar, wanted dashboard, and calendar.
  - Preserves all database tables and data (does not roll back migrations).
  - A separate "Delete module data" action in settings performs irreversible cleanup.
- Re-enabling a module restores full functionality with existing data intact.
- Module enabled/disabled state is stored in the `settings` database table as `module.<id>.enabled` keys (e.g., `module.movie.enabled`). This uses the existing settings infrastructure and is queryable by services without config file parsing.

### 1.4 Dependency Injection Integration

Modules integrate with the Wire DI system through a standardized registration pattern:

- Each module provides a `Wire() wire.ProviderSet` function that returns all its service constructors.
- The framework's `wire.go` aggregates module provider sets into the build.
- Module services that need framework infrastructure (broadcaster, health service, etc.) receive them via constructor injection using `contracts` interfaces.
- Modules that need late-binding (circular deps, dev mode switching) register with the framework's `SwitchableServices` via struct tags.
- The framework validates all module registrations at startup to catch configuration errors early.

---

## 2. Database Architecture

### 2.1 Per-Module Tables

Each module owns its own database tables. The framework provides migration helpers but does not impose a shared media schema.

**Universal columns** that every module's root/leaf tables should include (enforced by the framework's CRUD interface expectations):

| Column | Purpose |
|---|---|
| `id` | Primary key |
| `title` | Display title |
| `sort_title` | Sortable title (generated by shared `generateSortTitle` utility) |
| `path` | On-disk path |
| `root_folder_id` | FK to shared `root_folders` table |
| `quality_profile_id` | FK to module-scoped quality profile |
| `monitored` | Boolean monitoring flag |
| `status` | Enum: unreleased, missing, downloading, failed, upgradable, available |
| `active_download_id` | Currently active download |
| `status_message` | Human-readable status detail |
| `added_at` | Timestamp |
| `updated_at` | Timestamp |

Module-specific columns (e.g., `studio`, `network`, `season_folder`, `format_type`) are defined by the module on its own tables.

**Example — Movie Module tables:** `movies`, `movie_files`
**Example — TV Module tables:** `series`, `seasons`, `episodes`, `episode_files`
**Example — Music Module tables:** `artists`, `albums`, `tracks`, `track_files`

### 2.2 Per-Module Migration Tracks

Each module has its own Goose migration history with a separate version tracking table (e.g., `goose_movie_version`, `goose_tv_version`, `goose_music_version`).

- Modules can migrate independently.
- Enabling a module for the first time runs all its pending migrations.
- Module migrations are embedded in the module's Go package.
- The framework provides helpers for creating migration tracks and running them at startup.

### 2.3 Shared Tables — Discriminator Pattern

Shared/cross-cutting tables use a `(module_type, entity_type, entity_id)` discriminator pattern instead of foreign keys to module tables:

| Shared Table | Discriminator Columns | Notes |
|---|---|---|
| `root_folders` | `module_type` | One root folder per module type. Multiple root folders per module supported. |
| `downloads` | `module_type`, `entity_type`, `entity_id` | Replaces current `media_type` |
| `download_mappings` | `module_type`, `entity_type`, `entity_id` | Replaces nullable `movie_id`/`series_id`/`episode_id` FKs |
| `history` | `module_type`, `entity_type`, `entity_id` | Already uses discriminator pattern |
| `autosearch_status` | `module_type`, `entity_type`, `entity_id` | Already uses discriminator pattern |
| `requests` | `module_type`, `entity_type` | Portal requests |
| `queue_media` | `module_type`, `entity_type`, `entity_id` | Download queue |
| `import_decisions` | `module_type` | Import history |
| `notifications` | Dynamic event toggles per module (see Section 9) | |

The `history` table uses the discriminator pattern and the framework provides a `LogHistory(ctx, moduleType, entityType, entityID, eventType, data)` helper. Modules call this during import, grab, and upgrade operations. Event types are module-declared (e.g., `"grabbed"`, `"imported"`, `"upgraded"`, `"deleted"`). The history UI renders entries generically using module-provided display metadata.

**Integrity enforcement**: The framework provides a mandatory `DeleteEntity(ctx, moduleType, entityType, entityID)` hook that modules MUST call when deleting entities. This hook cascades deletions to all shared tables (downloads, download_mappings, history, autosearch_status, queue_media, requests) within a single transaction. Modules must not delete shared table records directly — all shared-table cleanup flows through this framework hook. This ensures referential integrity without cross-module SQLite foreign keys.

### 2.4 Status Model

**Status lives on every node level** that declares `HasMonitored: true` in its schema.

- **Leaf node status is authoritative** — directly reflects whether the file exists, is downloading, etc.
- **Parent node status is a denormalized aggregate** — updated whenever child statuses change.
- The framework provides aggregation helpers that compute parent status from children using the existing priority rules: `downloading > failed > missing > upgradable > available > unreleased`.
- Modules call the framework's status aggregation function after any child status change.

### 2.5 Version Slots (Optional)

The framework provides slot management as a **reusable mixin**. Modules that implement `SlotSupport` get:

- A framework-generated slot assignment table for their leaf entity type (e.g., `movie_slot_assignments`, `episode_slot_assignments`, `track_slot_assignments`).
- Shared slot CRUD operations (create, update, delete, list assignments).
- Slot-aware status computation.
- The framework manages the slot table schema and queries; the module just declares which entity type is slot-eligible.

**Per-module slot root folders** use a pivot table instead of per-module columns on `version_slots`:

| Table: `slot_root_folders` | |
|---|---|
| `slot_id` | FK to `version_slots` |
| `module_type` | String (module ID) |
| `root_folder_id` | FK to `root_folders` |

This replaces the current `movie_root_folder_id` / `tv_root_folder_id` columns and scales to N modules. One row per slot per module. The framework queries this table when resolving the target root folder during slot-based imports.

---

## 3. Metadata

### 3.1 Module-Provided Metadata Service

Each module provides its own metadata service implementation satisfying the `MetadataProvider` interface:

```
Search(query string, options) → []SearchResult
GetByID(externalID) → MediaMetadata
GetExtendedInfo(externalID) → ExtendedMetadata
RefreshMetadata(entityID) → updated entity
```

- The **Movie Module** uses TMDB (and optionally OMDb for supplementary data).
- The **TV Module** uses TVDB (with TMDB fallback).
- A **Music Module** would use MusicBrainz.
- Modules own their provider logic entirely, including API clients, rate limiting, and caching.

### 3.2 External IDs

Modules declare their own external ID columns on their tables. Each module chooses which ID systems it uses:

- Movie Module: `tmdb_id` (primary, unique), `imdb_id`, `tvdb_id`
- TV Module: `tvdb_id` (primary, unique), `tmdb_id`, `imdb_id`
- Music Module: `musicbrainz_id` (primary, unique), `discogs_id`

The `SearchableItem` interface (Section 5.3) includes a `GetExternalIDs() map[string]string` method, allowing shared systems to work with external IDs generically without knowing column names.

---

## 4. Quality System

### 4.1 Module-Defined Quality Schemas

Each module defines its own quality tiers, scoring, and upgrade logic via the `QualityDefinition` interface.

- **Quality items** are module-specific: video modules define resolution/codec tiers (HDTV-720p, Bluray-1080p, etc.), music modules define format/bitrate tiers (FLAC, MP3-320, AAC-256, etc.), book modules define format tiers (EPUB, PDF, MOBI, etc.).
- **Quality profiles** are module-scoped: a "HD-1080p" profile belongs to the Movie Module and cannot be assigned to a Music Module entity.
- The framework provides the **profile structure** — ordered tiers, cutoff point, upgrade-until logic, allow/deny lists — but the items within profiles are module-provided.
- Each module registers its quality items at startup. The framework generates the quality profile management UI and API dynamically.
- `quality_profiles` table gets a `module_type` column to scope profiles.

**Migration strategy**: Existing quality profiles are currently shared between movies and TV (no `module_type` column). During migration, each existing profile is **duplicated per module**: one copy gets `module_type='movie'`, another gets `module_type='tv'`. All movie entities' `quality_profile_id` FKs are updated to point to the movie copy; all TV entities' FKs point to the TV copy. Portal user settings (`movie_quality_profile_id`, `tv_quality_profile_id`) are updated to reference the correct module-scoped copies. After migration, profiles are fully independent per module.

### 4.2 Quality Scoring

Modules implement quality scoring as part of `QualityDefinition`:

```
ParseQuality(releaseTitle string) → QualityResult
ScoreQuality(item QualityItem) → int
IsUpgrade(current, candidate QualityItem, profile Profile) → bool
```

The framework's release selection logic calls these module-provided functions during the decisioning pipeline.

---

## 5. Search and Indexer System

### 5.1 Category Mappings

Each module declares its Newznab/Torznab category ranges via `SearchStrategy`:

- Movie Module: 2000–2999
- TV Module: 5000–5999
- Music Module: 3000–3999
- Book Module: 7000–7999

### 5.2 Indexer Capability Auto-Detection

Indexer support for a module is **inferred from the indexer's reported category list**. If an indexer reports categories in the 3000 range, it automatically supports the Music Module. No manual flags needed.

### 5.3 SearchableItem Interface

The current union-struct `SearchableItem` is replaced with an **interface**:

```
type SearchableItem interface {
    GetModuleType() string
    GetMediaType() string          // "movie", "episode", "track", etc.
    GetEntityID() int64
    GetTitle() string
    GetExternalIDs() map[string]string
    GetQualityProfile() *QualityProfile
    GetCurrentQuality() *QualityItem   // nil if no file
    GetSearchParams() SearchParams     // module-specific search parameters
}
```

Each module implements this interface with its own struct containing domain-specific fields. The framework's search router, scoring, and grab logic operate through this interface.

### 5.4 Custom Filter Functions

Modules provide release filter and title matching functions via `SearchStrategy`:

```
FilterRelease(release Release, item SearchableItem) (reject bool, reason string)
TitlesMatch(releaseTitle, mediaTitle string) bool
BuildSearchCriteria(item SearchableItem) SearchCriteria
```

`SearchCriteria` includes the **Newznab search type** and provider-specific parameters. Each module defines its own search type:

- Movie Module: type `"movie"` with IMDB/TMDB ID parameters
- TV Module: type `"tvsearch"` with TVDB ID + season/episode parameters
- Music Module: type `"audio"` with artist/album parameters
- Book Module: type `"book"` with author/title parameters

The framework handles indexer communication, HTTP/rate-limiting, and result aggregation. Modules handle domain-specific matching logic (e.g., TV needs S01E02 pattern matching, music needs artist/album matching).

### 5.6 RSS Sync

Modules participate in RSS sync through `SearchStrategy`. The framework periodically fetches RSS feeds from indexers using the module's declared category ranges. Each module's `FilterRelease` and `TitlesMatch` functions are called to match incoming RSS items against the module's wanted list. The framework handles feed polling, deduplication, and grab orchestration.

### 5.5 Group Search

The framework formalizes a **group search** concept for batch searching:

- Modules declare via `SearchStrategy` whether certain parent nodes can be searched as a unit (TV season packs, full album downloads).
- `IsGroupSearchEligible(parentNode) bool` — determines if a parent should be searched as a group.
- `SuppressChildSearches(parentNode, grabbedRelease) []entityID` — after a group search succeeds, returns which child entity searches should be suppressed.
- The framework handles the search pipeline: collect wanted items, attempt group searches first, fall back to individual leaf searches, suppress covered children.
- Upgrade group searches are also supported: `IsGroupUpgradeEligible(parentNode, currentQualities) bool`.

---

## 6. Monitoring

### 6.1 Framework Cascade + Module Presets

The framework provides **cascade infrastructure**:

- Every node level with `HasMonitored: true` has a `monitored` boolean column.
- Setting a parent node's `monitored` to false automatically propagates to all descendants.
- Setting a parent node's `monitored` to true propagates to children (subject to module preset logic).
- The framework provides bulk toggle mechanics at any level.

Modules define **monitoring presets** via `MonitoringPresets`:

- TV Module: `all`, `none`, `future`, `first_season`, `latest_season`, `existing`
- Music Module: `all`, `none`, `new_releases`, `specific_albums`
- Movie Module: no presets needed (flat structure, just a boolean)

Presets are strategies that, given a root node and its children, determine which descendants should be monitored. The framework applies the strategy; modules define the logic.

---

## 7. Release Dates and Availability

### 7.1 Framework Availability Date

The framework defines a single **availability date** concept — the date at which content becomes obtainable for download.

Each module implements `ReleaseDateResolver`:

```
ComputeAvailabilityDate(entity) → *time.Time
```

- **Movie Module**: returns the earliest of physical release date and digital release date (theatrical is irrelevant for downloads).
- **TV Module**: returns the episode's air date.
- **Music Module**: returns the album/track release date.

The framework uses this date for:
- **Unreleased → Missing transitions**: a scheduled task periodically calls each module's resolver and updates status for entities whose availability date has passed.
- **Calendar display**: the availability date is the canonical date shown on the unified calendar.
- **Search eligibility**: items with a future availability date are not searched.

Modules retain their own domain-specific date columns (e.g., `theatrical_release_date`, `physical_release_date` on movies) for display purposes. The framework only consumes the computed availability date.

---

## 8. Download and Import Pipeline

### 8.1 Download Categories

Download subdirectories are **auto-derived from the module name**: `SlipStream/{ModuleName}` (e.g., `SlipStream/Movies`, `SlipStream/Music`). No user configuration needed.

### 8.2 Download Mappings

The `download_mappings` table uses the discriminator pattern: `(module_type, entity_type, entity_id)`. When a download completes, the framework looks up the mapping and dispatches to the correct module's import handler.

### 8.3 Module-Provided Import Handler

Each module implements `ImportHandler`:

```
MatchDownload(download CompletedDownload) → []MatchedEntity
ImportFile(filePath string, entity MatchedEntity, qualityInfo QualityInfo) → ImportResult
SupportsMultiFileDownload() bool
MatchIndividualFile(filePath string, parentEntity MatchedEntity) → *MatchedEntity
IsGroupImportReady(parentEntity MatchedEntity, matchedFiles []MatchedEntity) bool
MediaInfoFields() []MediaInfoFieldDecl
```

- The **framework** handles: download monitoring, completion detection, file extraction, quality attribute detection (resolution, codec, format), file movement to library path, orchestration and progress tracking, `queue_media` progress tracking for multi-file downloads.
- The **module** handles: matching files to library entities (parsing filenames for episode numbers, track numbers, etc.) and creating file records in its module tables.

**Multi-file downloads** (season packs, full albums): When `SupportsMultiFileDownload()` returns true, the framework uses `MatchIndividualFile` to match each file in a multi-file download to a specific entity. The framework tracks per-file status in `queue_media` and calls `IsGroupImportReady` to determine when to proceed with batch import. Modules that return false (e.g., Movie Module) get the simpler single-file pipeline.

**Media info probing**: After import, the framework probes files for technical metadata. `MediaInfoFields()` declares which probed attributes are relevant to the module. Video modules declare resolution, codec, HDR, audio channels; music modules declare bitrate, sample rate, format; book modules may declare no fields (skip probing). The framework stores probed data and the module's file record can reference it.

### 8.4 File Parsing

Modules provide their own filename parsers via `FileParser`, using a **shared utility library** of common parsing helpers:

- `ExtractYear(filename) → int`
- `NormalizeTitle(title) → string`
- `DetectQualityAttributes(filename) → QualityAttributes`
- `ExtractEdition(filename) → string`

Modules compose these utilities into domain-specific parsers (e.g., TV parses `S01E02` patterns, music parses `01 - Track Name.flac` patterns).

**Orphan scan detection**: When the framework scans download directories for files without a download mapping (`ScanForPendingImports`), it needs to determine which module should handle each file. `FileParser` includes a `TryMatch(filename string) (confidence float64, match *ParseResult)` method. The framework calls `TryMatch` on all enabled modules and dispatches to the module with the highest confidence. A confidence of 0 means the module cannot parse the filename. This prevents modules from claiming files they don't understand.

### 8.5 On-Disk Path Templates

Modules declare **folder path templates** using variables from their node schema:

- Movie Module: `{Title} ({Year})` → `Inception (2010)/`
- TV Module: `{Title}/Season {SeasonNumber}` → `Breaking Bad/Season 01/`
- Music Module: `{Artist}/{Album} ({Year})` → `Pink Floyd/The Wall (1979)/`

The framework resolves templates using the module's variable definitions. Module path templates are user-customizable in settings.

**Conditional path segments**: Modules can declare optional path segments that are included based on a boolean setting. For example, TV declares `season_folder` as a conditional — when enabled, the path includes `Season {SeasonNumber}/`; when disabled, episodes go directly under the series folder. The template engine supports `{?ConditionName:segment}` syntax for this. Modules declare their conditions via `PathGenerator.ConditionalSegments()`.

**Special node paths**: Nodes marked as `IsSpecial` in the schema (e.g., TV Season 0) can have a dedicated path template. Modules declare a `SpecialPathTemplate` alongside the default. For TV, specials use `Specials/` instead of `Season 00/`.

### 8.6 File Naming (NamingProvider)

Each module implements `NamingProvider` to declare its file renaming configuration:

```
TokenContexts() → []TokenContext
DefaultFileTemplates() → map[string]string
FormatOptions() → []FormatOption
```

**Token contexts** define named scopes of available variables for the template editor. Each context is used for a specific naming template:

- Movie Module: `movie-folder` context (Title, Year, Quality, Edition, ImdbId), `movie-file` context (same variables plus file-specific ones)
- TV Module: `series-folder` context, `season-folder` context, `episode-file` context (with standard/daily/anime variants), `specials-folder` context
- Music Module: `artist-folder` context, `album-folder` context, `track-file` context

**Format sub-variants**: Modules with `FormatVariants` declared on their node schema can provide different default templates per variant. TV's episode level has variants `["standard", "daily", "anime"]`, each with a distinct default naming template. The framework renders variant-specific template editors in the settings UI.

**Format options**: Module-specific boolean/enum options that affect renaming behavior:
- TV Module: `colonReplacement` (enum: dash, space, smart), `multiEpisodeStyle` (enum: extend, duplicate, range)
- Movie Module: none currently
- Music Module: `padTrackNumbers` (bool)

**Naming settings storage**: Module naming settings (format templates, options, rename enabled toggle) are stored in a `module_naming_settings` table with `(module_type, setting_key, setting_value)` rows. This replaces the current hardcoded movie/TV columns on `import_settings`. The framework reads settings at startup and caches them for the renamer.

---

## 9. Notifications

### 9.1 Module-Declared Event Catalog

Each module declares its notification events at registration via `NotificationEvents`:

```
DeclareEvents() → []NotificationEvent{
    {ID: "movie:added", Label: "Movie Added", Description: "When a movie is added to the library"},
    {ID: "movie:available", Label: "Movie Available", Description: "When a movie file becomes available"},
    {ID: "movie:deleted", Label: "Movie Deleted", Description: "When a movie is removed"},
    ...
}
```

- The notification system **auto-generates config toggles** from the declared events.
- Users pick which events they care about per notification agent.
- The notification settings table stores event toggle state as a JSON column keyed by event ID (rather than hard-coded columns like `on_movie_added`).
- The notification settings UI groups events by module.

---

## 10. WebSocket / Real-Time Updates

### 10.1 Generic Entity Events

All WebSocket events follow a **standard shape**:

```json
{
    "module": "movie",
    "entityType": "movie",
    "entityId": 42,
    "action": "updated"
}
```

- One frontend handler maps `module + entityType` to TanStack Query keys using a registry.
- Each module's frontend code registers its query key invalidation rules with the framework.
- Standard actions: `added`, `updated`, `deleted`.
- The framework provides a `Broadcast(module, entityType, entityId, action)` helper that services call.

---

## 11. Portal (External Requests)

### 11.1 Generic Provision Interface

The portal request system dispatches to modules via a **generic provisioner**:

```
type PortalProvisioner interface {
    EnsureInLibrary(ctx context.Context, input *ProvisionInput) (entityID int64, err error)
    CheckAvailability(ctx context.Context, input *AvailabilityCheckInput) (*AvailabilityResult, error)
    ValidateRequest(ctx context.Context, input *RequestValidationInput) error
}
```

- The portal framework handles the request lifecycle: `pending → approved → searching → downloading → available` (or `denied`/`cancelled`).
- Modules implement media-specific provisioning logic.
- The `requests` table uses `module_type` and `entity_type` discriminators.

**Request completion inference**: The framework uses the module's `NodeSchema` hierarchy to determine when a request is satisfied. When an entity becomes available, the framework walks up the node tree: if all children of a requested parent node are available, the parent request is marked as available. For example, when a TV episode becomes available, the framework checks if all episodes in the requested season(s) are available — if so, the season/series request transitions to `available`. This generic hierarchy walk replaces the current TV-specific `StatusTracker` cascade logic.

**Duplicate detection**: The framework provides generic cross-entity-type duplicate detection using the module's external IDs. A request for a parent entity (series, album) covers child-level requests (season, track) for the same external ID. Modules do not need custom duplicate logic — the framework derives coverage from the node hierarchy.

### 11.2 Portal User Module Settings

A `portal_user_module_settings` table stores per-user, per-module configuration:

| Column | Type |
|---|---|
| `user_id` | FK to `portal_users` |
| `module_type` | String (module ID) |
| `quota_limit` | Integer (nullable = unlimited) |
| `quota_used` | Integer |
| `quality_profile_id` | FK to `quality_profiles` (nullable = use default) |
| `period_start` | Timestamp (quota reset tracking) |

One row per user per module. This table consolidates data currently spread across **three** source tables:

1. **`user_quotas`**: Currently stores `movies_limit`, `seasons_limit`, `episodes_limit`, `movies_used`, `seasons_used`, `episodes_used`, `period_start`. Migrated to per-module rows: movie quota → row with `module_type='movie'`, season/episode quotas → row with `module_type='tv'`.
2. **`portal_users`**: Currently stores `movie_quality_profile_id`, `tv_quality_profile_id`. Migrated to the `quality_profile_id` column on the corresponding module rows. These columns are dropped from `portal_users` after migration.
3. **`portal_invitations`**: Currently stores `movie_quality_profile_id`, `tv_quality_profile_id`. Migrated to a new `portal_invitation_module_settings` table with the same `(module_type, quality_profile_id)` pattern. When an invitation is accepted, the framework creates `portal_user_module_settings` rows from the invitation's module settings.

---

## 12. Arr Import

### 12.1 Framework Feature with Module Adapters

The import-from-external-*arr pipeline is a **framework feature**. The framework handles:

- Connecting to external SQLite databases
- Reading data and tracking progress
- UI for import configuration and progress

Modules provide **adapters** via the optional `ArrImportAdapter` interface:

```
type ArrImportAdapter interface {
    ExternalAppName() string                    // "Radarr", "Sonarr", "Lidarr"
    ReadExternalDB(db *sql.DB) ([]ImportItem, error)
    ConvertToCreateInput(item ImportItem) (*CreateInput, error)
}
```

The framework orchestrates the import pipeline; modules translate external DB schemas into their own domain types.

---

## 13. Frontend Architecture

### 13.1 Shared Shells + Module Detail Components

The frontend uses a **hybrid approach**:

- **List/grid pages** are generic, driven by the module's node schema. The existing `MediaListLayout` component is extended to work with any module. Filter options, sort options, and card rendering are parameterized per module.
- **Detail pages** have a shared shell (hero section, action bar, edit dialog) with module-provided detail content components (e.g., `SeasonsList` for TV, `TrackList` for music, `ChapterList` for books).
- **Add pages** use the shared `AddMediaConfigure` flow with module-specific configuration fields.

### 13.2 Navigation

The sidebar uses **framework-provided sections** with **module-contributed entries**:

| Section | Module Entries |
|---|---|
| **Library** | One entry per enabled module (Movies, Series, Music, etc.) |
| **Wanted** | Unified "Missing" page showing all modules |
| **Calendar** | Unified calendar with module-colored items |
| **Settings** | Module-specific settings pages |
| **Add** | One "Add" entry per enabled module |

### 13.3 Unified Views

All dashboard-style pages (Wanted, Calendar, Activity) show data from **all enabled modules** in a unified view with module-based grouping/filtering. Users can filter by module if desired.

### 13.4 Module Frontend Registration

Each module provides a frontend configuration object:

```typescript
interface ModuleConfig {
    id: string
    name: string
    pluralName: string
    icon: ComponentType
    themeColor: string              // e.g., "movie" → blue, "tv" → green, "music" → purple
    routes: RouteConfig[]
    queryKeys: QueryKeyConfig
    wsInvalidationRules: Record<string, string[]>
    detailComponent: ComponentType
    cardComponent: ComponentType
    addConfigFields: ComponentType
    filterOptions: FilterOption[]
    sortOptions: SortOption[]
}
```

The framework registers these configs and uses them to render navigation, route trees, and shared pages.

### 13.5 Naming Format Configuration

The framework provides a **format template engine and editor UI**. Modules register their available template variables:

- Movie Module: `{Title}`, `{Year}`, `{Quality}`, `{Edition}`, `{ImdbId}`, etc.
- TV Module: `{SeriesTitle}`, `{SeasonNumber}`, `{EpisodeNumber}`, `{EpisodeTitle}`, etc.
- Music Module: `{Artist}`, `{Album}`, `{TrackNumber}`, `{TrackTitle}`, `{Format}`, etc.

Users customize naming formats per module in settings. The framework handles template rendering and validation. Modules provide sensible defaults.

---

## 14. Dev Mode

### 14.1 Module-Provided Mock Factories

Each module implements `MockFactory`:

```
CreateMockMetadataProvider() → MetadataProvider
CreateSampleLibraryData(db *sql.DB) error
CreateTestRootFolders(virtualFS) error
```

When dev mode is activated:
1. The framework switches to the dev database.
2. Each enabled module's `MockFactory` is called to create mock providers and sample data.
3. The mock indexer and download client infrastructure remains framework-provided (shared across all modules).

---

## 15. Health Monitoring

The existing `HealthService` interface is **already generic** with string-based categories and IDs. No changes needed — modules call `RegisterItem`, `SetError`, `ClearStatus`, and `SetWarning` with their own category names (e.g., `"music-metadata"`, `"musicbrainz-api"`).

---

## 16. Indexer Compatibility

Indexer support for modules is **inferred from the indexer's reported Newznab/Torznab category list**:

- If an indexer reports categories in the 2000 range → supports Movie Module
- If an indexer reports categories in the 5000 range → supports TV Module
- If an indexer reports categories in the 3000 range → supports Music Module

The framework queries indexer capabilities at setup and stores the supported module list. The search router only sends searches to compatible indexers.

---

## 17. Migration Strategy

Movies and TV are refactored to become modules **from day one**. This validates the framework against the two most complex existing media types before any new modules are added.

### Phase approach (informational, not part of this spec):

The recommended approach is to **extract the framework iteratively from movies/TV code** — each refactoring step produces a working framework piece and refactored movie/TV code. This minimizes the risk of building abstractions that don't fit the actual use cases.

---

## 18. Edge Cases

### 18.1 Multi-Entity Files

Some modules support files that map to multiple leaf entities (e.g., TV multi-episode files like `S01E01E02.mkv`). Node levels with `SupportsMultiEntityFiles: true` in their schema enable this:

- `FileParser.ParseFilename()` returns a list of matched entity identifiers, not just one.
- `ImportHandler.ImportFile()` creates file records linked to all matched entities.
- The file record has a primary entity (first in the list) that determines the canonical filename.
- `NamingProvider` declares a `MultiEntityStyle` option (e.g., TV's `multiEpisodeStyle`: extend `S01E01-E02`, duplicate `S01E01.S01E02`, range `S01E01-E02`).
- Status computation: all linked entities get `available` status from the shared file.

### 18.2 Special Nodes

Node levels with `IsSpecial: true` can have "special" instances with distinct path and naming behavior. The canonical example is TV's Season 0 (Specials):

- `PathGenerator` declares a `SpecialPathTemplate` that overrides the default for special nodes. TV uses `Specials/` instead of `Season 00/`.
- `NamingProvider` can provide a variant template for specials if naming differs.
- `MonitoringPresets` can exclude specials by default (e.g., TV's `all` preset monitors all seasons except Season 0 unless `includeSpecials` is set).
- The framework identifies special nodes by calling `IsSpecialNode(node) bool` on the module's `PathGenerator`.

### 18.3 Format Sub-Variants

Modules with `FormatVariants` on a node level can have different naming and search behavior per variant. TV's episode level has `["standard", "daily", "anime"]`:

- **Naming**: Each variant can have a different default file naming template. Daily episodes use `{SeriesTitle} - {AirDate}` instead of `{SeriesTitle} - S{Season}E{Episode}`.
- **File parsing**: `FileParser` uses the variant to determine which parsing patterns apply. Daily episodes parse dates instead of S/E numbers.
- **Search**: `SearchStrategy.BuildSearchCriteria()` can vary search parameters by format variant.
- The variant is stored as a property on the root or intermediate node (e.g., `series.format_type = "daily"`).

### 18.4 Conditional Path Segments

Some modules need path segments that are conditionally included based on user settings. TV's `season_folder` toggle is the primary example:

- `PathGenerator.ConditionalSegments()` declares named conditions with a label and default value.
- Path templates use `{?ConditionName:template segment}` syntax: `{Title}/{?SeasonFolder:Season {SeasonNumber}/}`.
- When `SeasonFolder` is disabled, the framework omits the segment entirely.
- Conditions are stored as per-module settings. The framework provides the toggle UI.

### 18.5 Metadata Refresh with Existing Data

When metadata is refreshed for an entity that already has child nodes (e.g., refreshing a TV series adds new episodes from the metadata provider), the module must preserve existing state:

- New child entities inherit monitoring state from the parent's monitoring preset (or from the parent's current `monitored` value for flat cascading).
- Existing child entities retain their current monitoring state, status, and file associations.
- Removed child entities (no longer in metadata) are NOT auto-deleted — they are flagged for user review.
- `MetadataProvider.RefreshMetadata()` returns a structured diff (added, updated, removed entities) that the framework applies with these preservation rules.

---

## 19. Contributor Tooling

### 19.1 Module Scaffolding

The framework provides a `make new-module <id>` command that generates the full skeleton for a new module:

- Go package directory under `internal/modules/<id>/` with stub implementations of all required interfaces.
- Migration directory with an initial migration template.
- Frontend directory under `web/src/modules/<id>/` with `ModuleConfig` stub, route placeholders, and card/detail component skeletons.
- Test file with framework test helpers pre-configured.
- `TODO` markers in every stub method indicating what the contributor needs to implement.

### 19.2 Module Validation

A `slipstream validate-module <id>` CLI command checks a module implementation for correctness:

- All required interfaces are implemented (compile-time check via `var _ Interface = (*Type)(nil)` assertions).
- Node schema is valid: exactly one root, exactly one leaf, levels are ordered, names are unique.
- Quality items have unique IDs and weights across the module.
- Discriminator values (`module_type`, entity types) don't conflict with registered modules.
- Migration files exist, parse correctly, and are sequentially numbered.
- Frontend `ModuleConfig` matches backend schema (route IDs align, theme color is valid).
- Notification event IDs follow the `module:event` naming convention.

### 19.3 Module Testing Harness

A `moduletest` package provides integration test helpers that validate a module implementation against its declared interfaces:

- `moduletest.RunLifecycleTest(t, module)` — runs through the full lifecycle: add entity → search → grab → import → verify status transition. Uses mock infrastructure (indexer, download client) provided by the framework.
- `moduletest.RunParsingTest(t, module, fixtures)` — validates `FileParser` against a set of filename → expected parse result fixtures.
- `moduletest.RunNamingTest(t, module, fixtures)` — validates `NamingProvider` templates against expected output fixtures.
- `moduletest.RunSchemaTest(t, module)` — validates node schema consistency.

Contributors run these tests to verify their implementation before submitting.

### 19.4 Shared UI Component Catalog

The framework documents its reusable frontend components available to module authors:

| Component | Purpose |
|---|---|
| `MediaListLayout` | Generic list/grid page with filtering, sorting, bulk actions, card render prop |
| `MediaEditDialog` | Generic edit dialog for any entity with `{id, title, monitored, qualityProfileId}` |
| `AddMediaConfigure` | Shared add-media form with metadata search, root folder, quality profile selection |
| `MediaPreview` | Poster + metadata preview during the add flow |
| `MediaDeleteDialog` | Confirmation dialog for entity deletion |
| `MediaStatusBadge` | Status pill rendering for any status value |
| `QualityBadge` | Quality label with color coding |
| `PosterImage` / `BackdropImage` | Artwork rendering with cache-busting |
| `PatternEditor` | Token-aware template editor for naming format settings |

Module authors use these for consistent UI. Custom components are only needed for module-specific detail views (e.g., `SeasonsList`, `TrackList`).

---

## Appendix A: Interface Summary

### Required Module Interfaces

| Interface | Key Methods |
|---|---|
| `ModuleDescriptor` | `ID()`, `Name()`, `Icon()`, `ThemeColor()`, `NodeSchema()`, `Wire()` |
| `MetadataProvider` | `Search()`, `GetByID()`, `GetExtendedInfo()`, `RefreshMetadata()` |
| `SearchStrategy` | `Categories()`, `FilterRelease()`, `TitlesMatch()`, `BuildSearchCriteria()`, `IsGroupSearchEligible()`, `SuppressChildSearches()` |
| `ImportHandler` | `MatchDownload()`, `ImportFile()`, `SupportsMultiFileDownload()`, `MatchIndividualFile()`, `IsGroupImportReady()`, `MediaInfoFields()` |
| `PathGenerator` | `DefaultTemplates()`, `AvailableVariables()`, `ResolveTemplate()`, `ConditionalSegments()`, `IsSpecialNode()` |
| `NamingProvider` | `TokenContexts()`, `DefaultFileTemplates()`, `FormatOptions()` |
| `CalendarProvider` | `GetItemsInDateRange()` |
| `QualityDefinition` | `QualityItems()`, `ParseQuality()`, `ScoreQuality()`, `IsUpgrade()` |
| `WantedCollector` | `CollectMissing()`, `CollectUpgradable()` |
| `MonitoringPresets` | `AvailablePresets()`, `ApplyPreset()` |
| `FileParser` | `ParseFilename()`, `MatchToEntity()`, `TryMatch()` |
| `MockFactory` | `CreateMockMetadataProvider()`, `CreateSampleLibraryData()`, `CreateTestRootFolders()` |
| `NotificationEvents` | `DeclareEvents()` |
| `ReleaseDateResolver` | `ComputeAvailabilityDate()`, `CheckReleaseDateTransitions()` |
| `RouteProvider` | `RegisterRoutes()` |
| `TaskProvider` | `ScheduledTasks()` |

### Optional Module Interfaces

| Interface | Key Methods |
|---|---|
| `PortalProvisioner` | `EnsureInLibrary()`, `CheckAvailability()`, `ValidateRequest()` |
| `SlotSupport` | `SlotEntityType()`, `SlotTableSchema()` |
| `ArrImportAdapter` | `ExternalAppName()`, `ReadExternalDB()`, `ConvertToCreateInput()` |

---

## Appendix B: Shared Table Discriminator Values

Modules register their discriminator values at startup. The framework validates that no two modules claim the same value.

| Module | `module_type` | Entity Types |
|---|---|---|
| Movie | `movie` | `movie` |
| TV | `tv` | `series`, `season`, `episode` |
| Music | `music` | `artist`, `album`, `track` |
| Book | `book` | `author`, `book` |

---

## Appendix C: Current Duplication to Eliminate

The following code is currently duplicated between movie and TV and will be consolidated into the framework:

| Duplicated Code | Location | Framework Destination |
|---|---|---|
| `generateSortTitle()` | `movies/movie.go`, `tv/series.go` | Shared utility |
| `resolveField[T]()` | `movies/service.go`, `tv/service.go` | Shared utility |
| `rowToMovieFile` / `rowToEpisodeFile` | Both service files | Framework file record mapper |
| File removal → status transition | Both service files | Framework status transition helper |
| Service struct layout + constructor | Both service files | Framework base service |
| ~40 duplicated slot queries | `slots.sql` | Framework slot mixin |
| List/search query patterns | `movies.sql`, `series.sql` | Per-module but generated from patterns |
| Frontend API clients | `movies.ts`, `series.ts` | Generic module API client factory |
| TanStack Query hooks | `use-movies.ts`, `use-series.ts` | Generic module hook factory |
| List page components | Both route files | Single generic list page |
| WebSocket event handlers | `ws-message-handlers.ts` | Single generic handler |
