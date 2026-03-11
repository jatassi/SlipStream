# Adding a New Module to SlipStream

This guide walks through everything needed to add a new media module (e.g., "music", "audiobook", "game") to SlipStream. A module is the top-level unit that teaches SlipStream how to manage a category of media — from metadata lookup and quality scoring to file import and search.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Module Registry                         │
│  Holds all modules, validates schemas, provides lookups     │
├──────────────────────┬──────────────────────────────────────┤
│    Movie Module      │         TV Module                    │
│  (flat: movie)       │  (hierarchical: series→season→ep)    │
├──────────────────────┴──────────────────────────────────────┤
│                  Module Interface                            │
│  16 required sub-interfaces + optional capabilities         │
└─────────────────────────────────────────────────────────────┘
```

A module is a single Go struct that satisfies the composite `module.Module` interface — a bundle of 16 required sub-interfaces, each responsible for a distinct concern:

```
                        ┌──────────────┐
                        │   Module     │
                        │  Interface   │
                        └──────┬───────┘
           ┌───────────────────┼───────────────────┐
           │                   │                   │
    ┌──────┴──────┐    ┌──────┴──────┐    ┌───────┴──────┐
    │  Identity   │    │  Content    │    │  Operations  │
    ├─────────────┤    ├─────────────┤    ├──────────────┤
    │ Descriptor  │    │ Metadata    │    │ ImportHandler│
    │ NodeSchema  │    │ Provider    │    │ FileParser   │
    │ Quality     │    │ Calendar    │    │ Wanted       │
    │ Definition  │    │ Provider    │    │ Collector    │
    └─────────────┘    └─────────────┘    └──────────────┘
           │                   │                   │
    ┌──────┴──────┐    ┌──────┴──────┐    ┌───────┴──────┐
    │  Search     │    │  Naming     │    │  Lifecycle   │
    ├─────────────┤    ├─────────────┤    ├──────────────┤
    │ Search      │    │ Path        │    │ Monitoring   │
    │ Strategy    │    │ Generator   │    │ Presets      │
    │ Notification│    │ Naming      │    │ MockFactory  │
    │ Events      │    │ Provider    │    │ ReleaseDates │
    └─────────────┘    └─────────────┘    │ RouteProvider│
                                          │ TaskProvider │
                                          └──────────────┘
```

## Prerequisites

- Go 1.23+, Wire, sqlc
- Node.js / Bun for frontend
- Familiarity with `internal/module/interfaces.go` (all 16 interfaces defined there)

## Step 1: Scaffold the Module

Run the scaffolding tool:

```bash
go run ./scripts/new-module <module_id>

# Example:
go run ./scripts/new-module music
```

This generates the full file tree:

```
internal/modules/music/
├── module.go            # Module struct + Descriptor + delegation methods
├── migrate.go           # Embedded migration FS
├── metadata.go          # MetadataProvider implementation
├── search.go            # SearchStrategy implementation
├── fileparser.go        # FileParser implementation
├── importhandler.go     # ImportHandler implementation
├── monitoring.go        # MonitoringPresets implementation
├── notifications.go     # NotificationEvents implementation
├── calendar.go          # CalendarProvider implementation
├── wanted.go            # WantedCollector implementation
├── quality.go           # QualityDefinition (on Descriptor)
├── path_naming.go       # PathGenerator + NamingProvider
├── release_dates.go     # ReleaseDateResolver implementation
├── mock_factory.go      # MockFactory for dev mode
├── module_test.go       # Basic test template
└── migrations/
    └── 00001_initial.sql  # Module-specific migration

internal/database/queries/
└── musics.sql             # sqlc query file

web/src/modules/music/
└── index.ts               # Frontend ModuleConfig
```

Every generated Go file contains `TODO` stubs ready for implementation.

## Step 2: Register Type Constants

Add your module type and entity types in `internal/module/types.go`:

```go
const (
    TypeMovie Type = "movie"
    TypeTV    Type = "tv"
    TypeMusic Type = "music"  // <-- add
)

const (
    EntityMovie   EntityType = "movie"
    EntitySeries  EntityType = "series"
    EntitySeason  EntityType = "season"
    EntityEpisode EntityType = "episode"
    EntityMusic   EntityType = "music"  // <-- add (for flat module)
)
```

Entity types must be **globally unique** across all modules — the registry enforces this at startup.

## Step 3: Define the Node Schema

The schema declares your module's entity hierarchy. This is the single most important design decision for your module.

### Flat Module (single entity type)

For media with no hierarchy (like movies or individual tracks):

```go
func (d *Descriptor) NodeSchema() module.NodeSchema {
    return module.NodeSchema{
        Levels: []module.NodeLevel{
            {
                Name:         "music",
                PluralName:   "musics",
                IsRoot:       true,   // first level = root
                IsLeaf:       true,   // last level = leaf
                HasMonitored: true,
                Searchable:   true,
            },
        },
    }
}
```

### Hierarchical Module (multiple levels)

For media with parent/child relationships (like TV's series → season → episode):

```go
func (d *Descriptor) NodeSchema() module.NodeSchema {
    return module.NodeSchema{
        Levels: []module.NodeLevel{
            {
                Name:         "artist",
                PluralName:   "artists",
                IsRoot:       true,
                HasMonitored: true,
                Searchable:   true,
            },
            {
                Name:         "album",
                PluralName:   "albums",
                HasMonitored: true,
                IsSpecial:    true,  // e.g., "singles" album = special
            },
            {
                Name:         "track",
                PluralName:   "tracks",
                IsLeaf:       true,
                HasMonitored: true,
                Searchable:   true,
                SupportsMultiEntityFiles: true,
            },
        },
    }
}
```

**Schema rules** (enforced by `NodeSchema.Validate()`):
- At least one level
- Exactly one `IsRoot: true` (must be the first level)
- Exactly one `IsLeaf: true` (must be the last level)
- Single-level schemas must set both `IsRoot` and `IsLeaf`
- All `Name` and `PluralName` values must be unique

```
┌──────────────────────────────────────────────────────┐
│                Schema Validation                      │
│                                                      │
│  Flat:       [root+leaf]                             │
│  2-level:    [root] → [leaf]                         │
│  3-level:    [root] → [middle] → [leaf]              │
│                                                      │
│  ✗ No root          ✗ Root not first                 │
│  ✗ No leaf           ✗ Leaf not last                  │
│  ✗ Duplicate names   ✗ Single-level missing root+leaf │
└──────────────────────────────────────────────────────┘
```

## Step 4: Implement the Module Struct

The generated `module.go` uses the **delegation pattern** — the top-level `Module` struct delegates each interface to a focused sub-handler:

```go
type Module struct {
    descriptor       *Descriptor
    metadataProvider *metadataProvider
    importHandler    *importHandler
    fileParser       *fileParser
    pathGenerator    *pathGenerator
    namingProvider   *namingProvider
    // ... domain services, DB, logger
    db     *sql.DB
    logger *zerolog.Logger
}

func NewModule(db *sql.DB, logger *zerolog.Logger) *Module {
    return &Module{
        descriptor:       &Descriptor{},
        metadataProvider: newMetadataProvider(logger),
        importHandler:    newImportHandler(logger),
        fileParser:       newFileParser(logger),
        pathGenerator:    &pathGenerator{},
        namingProvider:   &namingProvider{},
        db:               db,
        logger:           logger,
    }
}
```

Each method on `Module` delegates to the appropriate sub-handler:

```go
// MetadataProvider delegation
func (m *Module) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
    return m.metadataProvider.Search(ctx, query, opts)
}

// PathGenerator delegation
func (m *Module) DefaultTemplates() map[string]string {
    return m.pathGenerator.DefaultTemplates()
}
```

Add constructor parameters for any services your module depends on (e.g., a `musicService`, `metadataSvc`).

## Step 5: Implement the 16 Required Interfaces

Each interface lives in its own file. Here's what each one does and what you need to implement:

### 5.1 Descriptor (`module.go`)

Identity, schema, and entity types. Already generated — customize `Name()`, `Icon()`, `ThemeColor()`.

### 5.2 MetadataProvider (`metadata.go`)

Connect to your external metadata source (e.g., MusicBrainz, TMDB, TVDB):

```go
type metadataProvider struct { /* ... */ }

func (p *metadataProvider) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
    // Query external API, return structured results
}

func (p *metadataProvider) GetByID(ctx context.Context, externalID string) (*module.MediaMetadata, error) {
    // Fetch full metadata by external ID
}

func (p *metadataProvider) GetExtendedInfo(ctx context.Context, externalID string) (*module.ExtendedMetadata, error) {
    // Detailed metadata including cast/credits
}

func (p *metadataProvider) RefreshMetadata(ctx context.Context, entityID int64) (*module.RefreshResult, error) {
    // Re-fetch and update cached metadata for a library entity
}
```

### 5.3 SearchStrategy (`search.go`)

Define how indexer searches work for your media type:

```go
func (s *searchStrategy) Categories() []int {
    // Newznab/Torznab category IDs (e.g., 3000-3999 for music)
    return []int{3000, 3010, 3020, 3030, 3040}
}

func (s *searchStrategy) FilterRelease(ctx context.Context, release *module.ReleaseForFilter, item module.SearchableItem) (bool, string) {
    // Return (true, "reason") to reject a release, (false, "") to accept
}

func (s *searchStrategy) BuildSearchCriteria(item module.SearchableItem) module.SearchCriteria {
    // Construct indexer query parameters from a wanted item
}
```

### 5.4 ImportHandler (`importhandler.go`)

Handle completed downloads — match them to library entities and organize files:

```go
func (h *importHandler) MatchDownload(ctx context.Context, dl *module.CompletedDownload) ([]module.MatchedEntity, error) {
    // Match a completed download to one or more library entities
}

func (h *importHandler) ImportFile(ctx context.Context, path string, entity *module.MatchedEntity, qi *module.QualityInfo) (*module.ImportResult, error) {
    // Move/copy/hardlink file to library folder, create file record
}
```

### 5.5 QualityDefinition (`quality.go`)

Define quality tiers. For video-based modules, reuse `shared.VideoQualityItems()`:

```go
func (d *Descriptor) QualityItems() []module.QualityItem {
    return shared.VideoQualityItems()  // Reuse shared video qualities
}

func (d *Descriptor) ParseQuality(releaseTitle string) (*module.QualityResult, error) {
    return shared.ParseVideoQuality(releaseTitle)
}
```

For non-video modules, define your own quality tiers:

```go
func (d *Descriptor) QualityItems() []module.QualityItem {
    return []module.QualityItem{
        {ID: 1, Name: "MP3-128",  Resolution: "128kbps",  Source: "mp3",  Weight: 1},
        {ID: 2, Name: "MP3-320",  Resolution: "320kbps",  Source: "mp3",  Weight: 2},
        {ID: 3, Name: "FLAC",     Resolution: "lossless",  Source: "flac", Weight: 3},
    }
}
```

### 5.6 PathGenerator + NamingProvider (`path_naming.go`)

File path templates and naming tokens:

```go
func (p *pathGenerator) DefaultTemplates() map[string]string {
    return map[string]string{
        "music": "{Artist Name} ({Year})/{Album}/{Track Number} - {Track Title}",
    }
}

func (p *pathGenerator) AvailableVariables(level string) []module.TemplateVariable {
    return []module.TemplateVariable{
        {Token: "{Artist Name}", Description: "Artist name"},
        {Token: "{Year}", Description: "Release year"},
        // ...
    }
}
```

### 5.7 Remaining Interfaces

| Interface | File | Purpose |
|---|---|---|
| `CalendarProvider` | `calendar.go` | Return items for the unified calendar view |
| `WantedCollector` | `wanted.go` | Collect missing/upgradable items for auto search |
| `MonitoringPresets` | `monitoring.go` | Pre-built monitoring strategies ("monitor all", "monitor future", etc.) |
| `FileParser` | `fileparser.go` | Parse filenames to extract metadata (title, year, quality) |
| `MockFactory` | `mock_factory.go` | Generate sample data for dev mode |
| `NotificationEvents` | `notifications.go` | Declare module-specific notification events |
| `ReleaseDateResolver` | `release_dates.go` | Compute availability dates and detect transitions |
| `RouteProvider` | `module.go` | Register custom API endpoints (can be a no-op stub) |
| `TaskProvider` | `module.go` | Declare scheduled tasks (can return nil) |

## Step 6: Add Module-Specific Migrations

Each module has its own migration track with an independent goose version table (`goose_db_version_<module_id>`):

```
internal/modules/music/
└── migrations/
    ├── 00001_initial.sql    # Created by scaffolding
    ├── 00002_add_genre.sql  # Future migration
    └── ...
```

The `migrate.go` file embeds the `migrations/` directory:

```go
//go:embed all:migrations
var moduleMigrations embed.FS

func (m *Module) MigrationFS() (migrations embed.FS, dir string) {
    return moduleMigrations, "migrations"
}
```

```
┌──────────────────────────────────────────────────────────┐
│                  Migration Execution Order                │
│                                                          │
│  1. Framework migrations (internal/database/migrations)  │
│       └─ goose_db_version (shared table)                 │
│                                                          │
│  2. Per-module migrations (run during registry setup)    │
│       ├─ goose_db_version_movie                          │
│       ├─ goose_db_version_tv                             │
│       └─ goose_db_version_music  ← your module           │
└──────────────────────────────────────────────────────────┘
```

Migrations run automatically via `module.MigrateAll()` during app startup, after the module is registered.

## Step 7: Wire the Module into DI

### 7.1 Add the constructor to `internal/api/wire.go`

```go
wire.Build(
    // ...

    // --- Module constructors ---
    moviemod.NewModule,
    tvmod.NewModule,
    musicmod.NewModule,  // <-- add your module constructor
    provideRegistry,

    // ...
)
```

### 7.2 Update the registry provider in `internal/api/providers.go`

```go
import musicmod "github.com/slipstream/slipstream/internal/modules/music"

func provideRegistry(
    db *sql.DB,
    movieMod *moviemod.Module,
    tvMod *tvmod.Module,
    musicMod *musicmod.Module,  // <-- add parameter
) *module.Registry {
    reg := module.NewRegistry()
    reg.Register(movieMod)
    reg.Register(tvMod)
    reg.Register(musicMod)  // <-- register

    if err := module.MigrateAll(db, reg); err != nil {
        panic("failed to run module migrations: " + err.Error())
    }

    scanner.SetGlobalRegistry(module.NewScannerRegistryAdapter(reg))
    return reg
}
```

### 7.3 Regenerate Wire

```bash
make wire
```

This updates `internal/api/wire_gen.go` with the generated dependency injection code.

## Step 8: Register the Frontend Module

### 8.1 Create the module config (`web/src/modules/music/index.ts`)

The scaffolding generates a starter file. Customize it:

```typescript
import { lazy } from 'react'
import { Music } from 'lucide-react'
import { musicApi } from '@/api/music'
import { musicKeys } from '@/hooks/use-music'
import { missingKeys } from '@/hooks/use-missing'
import type { ModuleConfig } from '../types'

const MusicListPage = lazy(() =>
  import('@/routes/music').then((m) => ({ default: m.MusicPage }))
)

export const musicModuleConfig: ModuleConfig = {
  id: 'music',
  name: 'Music',
  singularName: 'Track',
  pluralName: 'Music',
  icon: Music,
  themeColor: 'music',     // Define CSS variables in index.css
  basePath: '/music',

  routes: [
    { path: '/', id: 'music' },
    { path: '/$id', id: 'musicDetail' },
    { path: '/add', id: 'addMusic' },
  ],

  queryKeys: musicKeys,

  wsInvalidationRules: [
    {
      pattern: 'music:(added|updated|deleted)',
      queryKeys: [musicKeys.all],
      alsoInvalidate: [missingKeys.counts()],
    },
  ],

  filterOptions: [
    // { value: 'monitored', label: 'Monitored', icon: Eye },
    // ...
  ],

  sortOptions: [
    // { value: 'title', label: 'Title' },
    // ...
  ],

  tableColumns: { static: [], defaults: [] },

  listComponent: MusicListPage,
  cardComponent: () => null,      // Implement card component
  detailComponent: () => null,    // Implement detail component

  missingTabValue: 'music',
  missingCountKey: 'musicCount',

  api: {
    list: (options) => musicApi.list(options),
    get: (id) => musicApi.get(id),
    update: (id, data) => musicApi.update(id, data),
    delete: (id, deleteFiles) => musicApi.delete(id, deleteFiles),
    bulkDelete: (ids, deleteFiles) => musicApi.bulkDelete(ids, deleteFiles).then(() => undefined),
    bulkUpdate: (ids, data) => musicApi.bulkUpdate(ids, data),
    bulkMonitor: (ids, monitored) => musicApi.bulkMonitor(ids, monitored),
    search: (id) => musicApi.search(id),
    refresh: (id) => musicApi.refresh(id),
    refreshAll: () => musicApi.refreshAll(),
  },
}
```

### 8.2 Register in `web/src/modules/setup.ts`

```typescript
import { movieModuleConfig } from './movie'
import { musicModuleConfig } from './music'
import { registerModule } from './registry'
import { tvModuleConfig } from './tv'

export function setupModules(): void {
  registerModule(movieModuleConfig)
  registerModule(tvModuleConfig)
  registerModule(musicModuleConfig)  // <-- add
}
```

## Step 9: Optional Interfaces

Beyond the 16 required interfaces, modules can opt into additional capabilities via Go type assertions. The registry checks for these at runtime — no registration needed.

### Migrator (per-module migrations)

Already handled by `migrate.go` — implement `MigrationFS()` (generated by scaffolding).

### MonitoringCascader (hierarchical monitoring)

Only needed for multi-level schemas. Propagates monitored state from parent to children:

```go
var _ module.MonitoringCascader = (*Module)(nil)

func (m *Module) CascadeMonitored(ctx context.Context, entityType module.EntityType, entityID int64, monitored bool) error {
    // Set monitored on all descendants of the given entity
}
```

### PortalProvisioner (external request support)

Enables the requests portal to add media from this module:

```go
var _ module.PortalProvisioner = (*Module)(nil)

func (m *Module) SupportedEntityTypes() []string { return []string{"music"} }
func (m *Module) ValidateRequest(ctx context.Context, entityType string, externalIDs map[string]int64) error { /* ... */ }
func (m *Module) EnsureInLibrary(ctx context.Context, input *module.ProvisionInput) (int64, error) { /* ... */ }
func (m *Module) CheckAvailability(ctx context.Context, input *module.AvailabilityCheckInput) (*module.AvailabilityResult, error) { /* ... */ }
func (m *Module) CheckRequestCompletion(ctx context.Context, input *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) { /* ... */ }
```

### SlotSupport (multi-version files)

For modules that support multiple file versions of the same entity:

```go
var _ module.SlotSupport = (*Module)(nil)

func (m *Module) SlotEntityType() string { return "music" }
```

### ArrImportAdapter (import from external apps)

For importing libraries from external *arr apps:

```go
var _ module.ArrImportAdapter = (*Module)(nil)

func (m *Module) ExternalAppName() string { return "Lidarr" }
func (m *Module) ReadExternalDB(ctx context.Context, dbPath string) ([]module.ArrImportItem, error) { /* ... */ }
func (m *Module) ConvertToCreateInput(item module.ArrImportItem) (any, error) { /* ... */ }
```

## Step 10: Verify Everything

```bash
# Regenerate sqlc queries (if you added SQL)
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

# Regenerate Wire DI
make wire

# Build and lint backend
make build
make lint

# Lint frontend
cd web && bun run lint
```

## Startup Lifecycle

Understanding when your code runs:

```
Application Boot
       │
       ▼
┌─────────────────────┐
│  Framework DB        │  Core goose migrations (shared tables)
│  Migrations          │
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Wire DI Build      │  NewModule() constructors called
│                     │  Creates Module struct with sub-handlers
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  provideRegistry()  │  Registry created
│                     │  reg.Register(mod) — schema validated,
│                     │  entity type conflicts checked
│                     │  module.MigrateAll() — per-module migrations
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Late Binding       │  Setter injection for circular deps
│  (setters.go)       │  e.g., SetSlotsService()
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  LoadEnabledState() │  Read module enabled/disabled from DB
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Route Registration │  RegisterRoutes() called per module
│  Task Scheduling    │  ScheduledTasks() registered with scheduler
└──────────┬──────────┘
           ▼
       Ready to serve
```

## Frontend Lifecycle

```
App Initialization
       │
       ▼
┌─────────────────────┐
│  setupModules()     │  registerModule() for each config
│                     │  Populates in-memory registry
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  System API Call     │  GET /api/v1/system → returns enabled modules
│                     │  setModuleEnabledState() filters registry
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Route Setup        │  Lazy-loaded routes from module configs
│                     │  WebSocket invalidation rules active
│                     │  TanStack Query keys ready
└──────────┬──────────┘
           ▼
       UI ready
```

## Checklist

- [ ] Run scaffolding: `go run ./scripts/new-module <id>`
- [ ] Add type/entity constants to `internal/module/types.go`
- [ ] Design and implement `NodeSchema` (flat vs hierarchical)
- [ ] Write initial migration SQL in `migrations/00001_initial.sql`
- [ ] Implement all 16 interface methods (resolve every `TODO`)
- [ ] Add compile-time interface checks: `var _ module.Module = (*Module)(nil)`
- [ ] Wire constructor into `internal/api/wire.go`
- [ ] Register in `provideRegistry()` in `internal/api/providers.go`
- [ ] Run `make wire`
- [ ] Add sqlc queries and run `sqlc generate`
- [ ] Create frontend `ModuleConfig` in `web/src/modules/<id>/index.ts`
- [ ] Register in `web/src/modules/setup.ts`
- [ ] Create API client, query keys, and route components
- [ ] Add theme color CSS variables to `web/src/index.css`
- [ ] Implement optional interfaces as needed (Migrator, PortalProvisioner, etc.)
- [ ] Run `make lint` and `cd web && bun run lint`
- [ ] Verify app starts and module appears in UI

## Reference: File Map

```
internal/
├── module/                      # Framework (don't modify)
│   ├── interfaces.go            # 16 required interface definitions
│   ├── module.go                # Module composite interface + Registry
│   ├── types.go                 # Type/EntityType constants ← EDIT
│   ├── schema.go                # NodeSchema + validation
│   ├── migrate.go               # Per-module migration runner
│   ├── optional_interfaces.go   # ArrImportAdapter
│   ├── portal_provisioner.go    # PortalProvisioner interface
│   └── slot_support.go          # SlotSupport interface
│
├── modules/
│   ├── shared/                  # Shared utilities (video qualities, etc.)
│   ├── movie/                   # Reference: flat module
│   ├── tv/                      # Reference: hierarchical module
│   └── <your_module>/           # ← YOUR CODE
│       ├── module.go
│       ├── migrate.go
│       ├── metadata.go
│       ├── search.go
│       ├── fileparser.go
│       ├── importhandler.go
│       ├── monitoring.go
│       ├── notifications.go
│       ├── calendar.go
│       ├── wanted.go
│       ├── quality.go
│       ├── path_naming.go
│       ├── release_dates.go
│       ├── mock_factory.go
│       └── migrations/
│           └── 00001_initial.sql
│
├── api/
│   ├── wire.go                  # ← EDIT (add constructor)
│   ├── providers.go             # ← EDIT (add to provideRegistry)
│   └── wire_gen.go              # Generated — run `make wire`
│
└── database/queries/
    └── <your_module>.sql        # ← EDIT (sqlc queries)

web/src/modules/
├── types.ts                     # ModuleConfig type definition
├── registry.ts                  # registerModule / getModule
├── setup.ts                     # ← EDIT (register your config)
├── movie/index.ts               # Reference: movie config
├── tv/index.ts                  # Reference: TV config
└── <your_module>/
    └── index.ts                 # ← YOUR CODE
```
