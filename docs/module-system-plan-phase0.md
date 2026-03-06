## Phase 0: Module Framework Foundation

**Goal:** Define all interfaces, types, and the registration/loading mechanism. Purely additive — no behavioral changes. Existing code continues to work unchanged.

**Spec sections covered:** §1.1, §1.2, §1.3, §1.4 (§1.4 partially — the `Wire()` method signature is defined on `Descriptor` but returns an empty provider set; actual DI wiring is deferred to the phase where module services are implemented)

### Phase 0 Architecture Decisions

1. **Package location:** New framework types go in `internal/module/` (new package). Module implementations go in `internal/modules/movie/` and `internal/modules/tv/`. This cleanly separates modules from existing service packages in `internal/library/`. Existing `internal/library/movies/` and `internal/library/tv/` packages remain untouched in this phase.

2. **Interface-only in Phase 0:** We define all interfaces and types but do NOT implement them yet. Movie and TV module packages get `ModuleDescriptor` stubs that declare their node schemas. No existing code changes.

3. **Registration is a simple Go map:** No dynamic loading, no plugin system. A `Registry` struct in `internal/module/` holds registered modules. Modules are added in a `RegisterAll()` function called at startup.

4. **Module enable/disable state:** Stored in the existing `settings` table as `module.<id>.enabled` keys (e.g., `module.movie.enabled`). This uses the existing settings infrastructure. **Spec deviation:** The spec §1.3 says "application config" — we use the settings DB table instead because it's queryable by services without config file parsing. The spec should be updated to match.

5. **Zero internal deps (with exceptions):** The `internal/module/` package should have zero imports from other SlipStream packages. The only exceptions are `context`, `time`, `database/sql`, `embed`, `fmt`, and `io/fs` from stdlib, plus `github.com/google/wire` for the `wire.ProviderSet` return type on `Descriptor`, and `github.com/pressly/goose/v3` in the migration helper. `RegisterAll()` in `register.go` accepts a `*zerolog.Logger` — this is the only zerolog import in the package and is confined to that one file.

---

### Task 0.1: Define Node Schema Types

**Create** `internal/module/schema.go`

Define the node schema types that every module uses to declare its hierarchy:

```go
package module

// NodeLevel defines a single tier in a module's hierarchy.
type NodeLevel struct {
    Name                    string            // e.g., "series", "season", "episode"
    PluralName              string            // e.g., "series", "seasons", "episodes"
    IsRoot                  bool              // true for the top-level library entry
    IsLeaf                  bool              // true for the file-bearing level
    HasMonitored            bool              // whether this level has a `monitored` column
    Searchable              bool              // whether items at this level can be individually searched
    IsSpecial               bool              // whether this level can have "special" instances (e.g., Season 0)
    SupportsMultiEntityFiles bool             // whether a single file can map to multiple entities
    FormatVariants          []string          // sub-variants affecting naming (e.g., "standard", "daily", "anime")
    Properties              map[string]string // module-specific fields beyond universal columns (spec §1.2)
}

// NodeSchema defines a module's complete hierarchy.
type NodeSchema struct {
    Levels []NodeLevel // ordered from root to leaf
}

// Validate checks schema invariants: exactly one root, exactly one leaf, ordered.
func (s NodeSchema) Validate() error { ... }

// Root returns the root-level node, or an error if none exists.
func (s NodeSchema) Root() (NodeLevel, error) { ... }

// Leaf returns the leaf-level node, or an error if none exists.
func (s NodeSchema) Leaf() (NodeLevel, error) { ... }

// Depth returns the number of levels.
func (s NodeSchema) Depth() int { return len(s.Levels) }
```

**Validation rules** (implement in `Validate()`):
- Exactly one level has `IsRoot: true`
- Exactly one level has `IsLeaf: true`
- Root must be the first level, leaf must be the last
- All level names must be unique
- All level plural names must be unique
- At least one level must exist
- If only one level, it must be both root and leaf (flat modules like Movie)

**Verify:** `go build ./internal/module/...` compiles. Write a `schema_test.go` with tests for valid schemas (movie flat, TV 3-level) and invalid schemas (no root, no leaf, duplicate names).

---

### Task 0.2: Define Module Descriptor Interface and Discriminator Types

**Create** `internal/module/types.go`

Define the core identity types and the discriminator constants:

```go
package module

import "github.com/google/wire"

// Type identifies a module (used as discriminator in shared tables).
type Type string

const (
    TypeMovie Type = "movie"
    TypeTV    Type = "tv"
)

// EntityType identifies an entity within a module (e.g., "movie", "series", "season", "episode").
type EntityType string

// Common entity types. Modules declare their own via NodeSchema.
const (
    EntityMovie   EntityType = "movie"
    EntitySeries  EntityType = "series"
    EntitySeason  EntityType = "season"
    EntityEpisode EntityType = "episode"
)

// Descriptor provides a module's identity and schema. (spec §1.1 ModuleDescriptor + §1.4)
type Descriptor interface {
    // ID returns the unique module identifier used as the discriminator value.
    ID() Type
    // Name returns the human-readable module name (e.g., "Movies").
    Name() string
    // PluralName returns the plural form (e.g., "Movies").
    PluralName() string
    // Icon returns the icon identifier for the UI.
    Icon() string
    // ThemeColor returns the CSS color used for this module in the UI.
    ThemeColor() string
    // NodeSchema returns the module's hierarchy definition.
    NodeSchema() NodeSchema
    // EntityTypes returns all entity types this module uses (derived from NodeSchema level names).
    EntityTypes() []EntityType
    // Wire returns the module's Wire DI provider set containing all its service constructors. (§1.4)
    // In Phase 0 this returns an empty provider set. Populated when module services are implemented.
    Wire() wire.ProviderSet
}
```

**Verify:** Compiles cleanly.

---

### Task 0.3: Define All Required Module Interfaces

**Create** `internal/module/interfaces.go`

Define ALL required interfaces from spec §1.1. These are **empty placeholder interfaces for now** — they carry documentation and method signatures but no implementations exist yet. The goal is to have the full contract surface area defined so that future phases can implement against stable interfaces.

Each interface maps to a spec section:

```go
package module

import (
    "context"
    "time"
)

// --- Required interfaces (§1.1) ---

// MetadataProvider handles search and metadata retrieval for a module. (§3.1)
type MetadataProvider interface {
    Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
    GetByID(ctx context.Context, externalID string) (*MediaMetadata, error)
    GetExtendedInfo(ctx context.Context, externalID string) (*ExtendedMetadata, error)
    RefreshMetadata(ctx context.Context, entityID int64) (*RefreshResult, error)
}

// SearchStrategy defines search behavior for a module. (§5.4)
type SearchStrategy interface {
    Categories() []int // Newznab/Torznab category ranges
    FilterRelease(release Release, item SearchableItem) (reject bool, reason string)
    TitlesMatch(releaseTitle, mediaTitle string) bool
    BuildSearchCriteria(item SearchableItem) SearchCriteria
    IsGroupSearchEligible(parentEntityType EntityType, parentID int64) bool
    SuppressChildSearches(parentEntityType EntityType, parentID int64, grabbedRelease Release) []int64
}

// ImportHandler handles download completion and file import. (§8.3)
type ImportHandler interface {
    MatchDownload(ctx context.Context, download CompletedDownload) ([]MatchedEntity, error)
    ImportFile(ctx context.Context, filePath string, entity MatchedEntity, qualityInfo QualityInfo) (*ImportResult, error)
    SupportsMultiFileDownload() bool
    MatchIndividualFile(ctx context.Context, filePath string, parentEntity MatchedEntity) (*MatchedEntity, error)
    IsGroupImportReady(ctx context.Context, parentEntity MatchedEntity, matchedFiles []MatchedEntity) bool
    MediaInfoFields() []MediaInfoFieldDecl
}

// PathGenerator declares folder path templates. (§8.5)
type PathGenerator interface {
    DefaultTemplates() map[string]string
    AvailableVariables(level string) []TemplateVariable
    ResolveTemplate(template string, data map[string]any) (string, error)
    ConditionalSegments() []ConditionalSegment
    IsSpecialNode(entityType EntityType, entityID int64) bool
}

// NamingProvider declares file naming configuration. (§8.6)
type NamingProvider interface {
    TokenContexts() []TokenContext
    DefaultFileTemplates() map[string]string
    FormatOptions() []FormatOption
}

// CalendarProvider returns items for the unified calendar. (§7.1)
type CalendarProvider interface {
    GetItemsInDateRange(ctx context.Context, start, end time.Time) ([]CalendarItem, error)
}

// QualityDefinition declares quality tiers and scoring. (§4.1, §4.2)
type QualityDefinition interface {
    QualityItems() []QualityItem
    ParseQuality(releaseTitle string) (*QualityResult, error)
    ScoreQuality(item QualityItem) int
    IsUpgrade(current, candidate QualityItem, profileID int64) (bool, error)
}

// WantedCollector collects missing/upgradable items. (§6.1)
type WantedCollector interface {
    CollectMissing(ctx context.Context) ([]SearchableItem, error)
    CollectUpgradable(ctx context.Context) ([]SearchableItem, error)
}

// MonitoringPresets defines monitoring strategies. (§6.1)
type MonitoringPresets interface {
    AvailablePresets() []MonitoringPreset
    ApplyPreset(ctx context.Context, rootEntityID int64, presetID string) error
}

// FileParser parses filenames to extract module-specific identifiers. (§8.4)
type FileParser interface {
    ParseFilename(filename string) (*ParseResult, error)
    MatchToEntity(ctx context.Context, parseResult *ParseResult) (*MatchedEntity, error)
    TryMatch(filename string) (confidence float64, match *ParseResult)
}

// MockFactory creates mock data for dev mode. (§14.1)
type MockFactory interface {
    CreateMockMetadataProvider() MetadataProvider
    CreateSampleLibraryData(ctx context.Context) error
    CreateTestRootFolders(ctx context.Context) error
}

// NotificationEvents declares notification event catalog. (§9.1)
type NotificationEvents interface {
    DeclareEvents() []NotificationEvent
}

// ReleaseDateResolver computes availability dates. (§7.1)
type ReleaseDateResolver interface {
    ComputeAvailabilityDate(ctx context.Context, entityID int64) (*time.Time, error)
    CheckReleaseDateTransitions(ctx context.Context) (transitioned int, err error)
}

// RouteProvider registers module-specific API routes. (§1.1)
type RouteProvider interface {
    RegisterRoutes(group RouteGroup)
}

// TaskProvider declares scheduled tasks. (§1.1)
type TaskProvider interface {
    ScheduledTasks() []ScheduledTask
}
```

**Create** `internal/module/optional_interfaces.go` for optional interfaces:

```go
package module

import "context"

// --- Optional interfaces (§1.1) ---

// PortalProvisioner enables portal request support. (§11.1)
type PortalProvisioner interface {
    EnsureInLibrary(ctx context.Context, input *ProvisionInput) (entityID int64, err error)
    CheckAvailability(ctx context.Context, input *AvailabilityCheckInput) (*AvailabilityResult, error)
    ValidateRequest(ctx context.Context, input *RequestValidationInput) error
}

// SlotSupport enables version slot support. (§2.5, Appendix A)
type SlotSupport interface {
    SlotEntityType() EntityType
    SlotTableSchema() SlotTableDecl // declares the slot assignment table schema for the framework to generate
}

// ArrImportAdapter enables import from external *arr apps. (§12.1)
type ArrImportAdapter interface {
    ExternalAppName() string
    ReadExternalDB(ctx context.Context, dbPath string) ([]ArrImportItem, error)
    ConvertToCreateInput(item ArrImportItem) (any, error)
}
```

**Verify:** Compiles cleanly. All types referenced in method signatures must be defined (see Task 0.4).

---

### Task 0.4: Define Supporting Types for Interfaces

**Create** `internal/module/search_types.go`, `internal/module/import_types.go`, `internal/module/calendar_types.go`, `internal/module/quality_types.go`, `internal/module/notification_types.go`, `internal/module/path_types.go`

These files define the struct types referenced by the interfaces. Keep them as minimal skeletons — just enough for the interfaces to compile. They will be fleshed out in later phases when implementations are written.

Key types to define (organized by file):

**`search_types.go`:**
- `SearchOptions` — query parameters for metadata search
- `SearchResult` — single search result from metadata provider
- `MediaMetadata` — full metadata for an entity
- `ExtendedMetadata` — additional metadata (credits, recommendations, etc.)
- `RefreshResult` — diff result from metadata refresh (added/updated/removed)
- `Release` — a release found by an indexer
- `SearchableItem` — interface for wanted items (replaces `decisioning.SearchableItem` in later phase)
- `SearchCriteria` — Newznab search parameters

**`import_types.go`:**
- `CompletedDownload` — download that finished and is ready for import
- `MatchedEntity` — an entity matched to a download/file
- `QualityInfo` — quality details of a file being imported
- `ImportResult` — result of importing a file
- `MediaInfoFieldDecl` — declares a media info field relevant to the module

**`calendar_types.go`:**
- `CalendarItem` — an item with a date for the calendar view

**`quality_types.go`:**
- `QualityItem` — a quality tier defined by a module
- `QualityResult` — result of parsing quality from a release title

**`notification_types.go`:**
- `NotificationEvent` — declares a notification event (ID, label, description)

**`path_types.go`:**
- `TemplateVariable` — a variable available for path templates
- `ConditionalSegment` — a conditional path segment declaration
- `TokenContext` — a named scope of template variables
- `FormatOption` — a formatting option (bool or enum)

**`task_types.go`:**
- `ScheduledTask` — configuration for a scheduled task
- `RouteGroup` — abstraction over the HTTP router for module route registration

**`parse_types.go`:**
- `ParseResult` — result of parsing a filename
- `MonitoringPreset` — defines a monitoring strategy

**`arr_types.go`:**
- `ArrImportItem` — an item from an external *arr DB
- `ProvisionInput`, `AvailabilityCheckInput`, `AvailabilityResult`, `RequestValidationInput` — portal provisioner types

**`slot_types.go`:**
- `SlotTableDecl` — declares a slot assignment table schema for the framework to generate (used by `SlotSupport`)

**Important:** All types should use Go primitives and standard library types. Do NOT import from existing SlipStream packages — the module package should have zero internal dependencies at this stage. Allowed external imports: stdlib (`context`, `time`, `database/sql`, `embed`, `fmt`, `sync`), `github.com/google/wire` (for `Descriptor.Wire()`), and `github.com/pressly/goose/v3` (in `migrate.go` only). See Architecture Decision #5.

**Verify:** `go build ./internal/module/...` compiles with zero errors.

---

### Task 0.5: Define Module Interface Bundle and Registry

**Create** `internal/module/module.go`

The `Module` type bundles all required interfaces. The `Registry` manages module registration and lookup.

```go
package module

import (
    "fmt"
    "sync"
)

// Module is the full interface bundle that a module implementation must satisfy.
// Every module implements Descriptor plus all required interfaces.
// Optional interfaces are detected via type assertion at runtime.
type Module interface {
    Descriptor
    MetadataProvider
    SearchStrategy
    ImportHandler
    PathGenerator
    NamingProvider
    CalendarProvider
    QualityDefinition
    WantedCollector
    MonitoringPresets
    FileParser
    MockFactory
    NotificationEvents
    ReleaseDateResolver
    RouteProvider
    TaskProvider
}

// Registry holds all registered modules and provides lookup.
type Registry struct {
    mu      sync.RWMutex
    modules map[Type]Module
    order   []Type // registration order for deterministic iteration
}

// NewRegistry creates an empty module registry.
func NewRegistry() *Registry {
    return &Registry{
        modules: make(map[Type]Module),
    }
}

// Register adds a module to the registry. Panics on duplicate ID or conflicting entity types.
func (r *Registry) Register(m Module) {
    r.mu.Lock()
    defer r.mu.Unlock()

    id := m.ID()
    if _, exists := r.modules[id]; exists {
        panic(fmt.Sprintf("module already registered: %s", id))
    }

    // Validate node schema
    schema := m.NodeSchema()
    if err := schema.Validate(); err != nil {
        panic(fmt.Sprintf("module %s has invalid schema: %v", id, err))
    }

    // Check for entity type conflicts with existing modules
    for _, existingMod := range r.modules {
        existingTypes := existingMod.EntityTypes()
        for _, newET := range m.EntityTypes() {
            for _, existingET := range existingTypes {
                if newET == existingET {
                    panic(fmt.Sprintf("entity type %q conflicts between modules %s and %s", newET, id, existingMod.ID()))
                }
            }
        }
    }

    r.modules[id] = m
    r.order = append(r.order, id)
}

// Get returns a module by type, or nil if not found.
func (r *Registry) Get(t Type) Module {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.modules[t]
}

// All returns all registered modules in registration order.
func (r *Registry) All() []Module {
    r.mu.RLock()
    defer r.mu.RUnlock()
    result := make([]Module, 0, len(r.order))
    for _, t := range r.order {
        result = append(result, r.modules[t])
    }
    return result
}

// Types returns all registered module types in registration order.
func (r *Registry) Types() []Type {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return append([]Type(nil), r.order...)
}

// ModuleForEntityType returns the module that owns the given entity type.
func (r *Registry) ModuleForEntityType(et EntityType) Module {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, m := range r.modules {
        for _, met := range m.EntityTypes() {
            if met == et {
                return m
            }
        }
    }
    return nil
}
```

**Verify:** Compiles. Write `module_test.go` testing: register two modules, duplicate ID panics, entity type conflict panics, `All()` returns in order, `ModuleForEntityType()` works.

---

### Task 0.6: Create Movie Module Descriptor Stub

**Create** `internal/modules/movie/module.go`

This is a stub that satisfies `module.Descriptor` and declares the movie node schema. It does NOT implement any other interfaces yet — that comes in later phases.

```go
package movie

import (
    "github.com/google/wire"
    "github.com/slipstream/slipstream/internal/module"
)

// Descriptor implements module.Descriptor for the Movie module.
type Descriptor struct{}

func (d *Descriptor) ID() module.Type          { return module.TypeMovie }
func (d *Descriptor) Name() string             { return "Movies" }
func (d *Descriptor) PluralName() string       { return "Movies" }
func (d *Descriptor) Icon() string             { return "film" }
func (d *Descriptor) ThemeColor() string        { return "blue" }

func (d *Descriptor) NodeSchema() module.NodeSchema {
    return module.NodeSchema{
        Levels: []module.NodeLevel{
            {
                Name:         "movie",
                PluralName:   "movies",
                IsRoot:       true,
                IsLeaf:       true,
                HasMonitored: true,
                Searchable:   true,
            },
        },
    }
}

func (d *Descriptor) EntityTypes() []module.EntityType {
    return []module.EntityType{module.EntityMovie}
}

func (d *Descriptor) Wire() wire.ProviderSet {
    return wire.NewSet() // empty until module services are implemented
}
```

This file only implements `Descriptor`. It does NOT embed or implement the full `module.Module` interface — that will happen incrementally as later phases add implementations for each interface.

**Verify:** `go build ./internal/modules/movie/...` compiles.

---

### Task 0.7: Create TV Module Descriptor Stub

**Create** `internal/modules/tv/module.go`

```go
package tv

import (
    "github.com/google/wire"
    "github.com/slipstream/slipstream/internal/module"
)

// Descriptor implements module.Descriptor for the TV module.
type Descriptor struct{}

func (d *Descriptor) ID() module.Type          { return module.TypeTV }
func (d *Descriptor) Name() string             { return "TV" }
func (d *Descriptor) PluralName() string       { return "Series" }
func (d *Descriptor) Icon() string             { return "tv" }
func (d *Descriptor) ThemeColor() string        { return "green" }

func (d *Descriptor) NodeSchema() module.NodeSchema {
    return module.NodeSchema{
        Levels: []module.NodeLevel{
            {
                Name:         "series",
                PluralName:   "series",
                IsRoot:       true,
                HasMonitored: true,
                Searchable:   true,
            },
            {
                Name:         "season",
                PluralName:   "seasons",
                HasMonitored: true,
                IsSpecial:    true, // Season 0 = Specials
            },
            {
                Name:                    "episode",
                PluralName:              "episodes",
                IsLeaf:                  true,
                HasMonitored:            true,
                Searchable:              true,
                SupportsMultiEntityFiles: true,
                FormatVariants:          []string{"standard", "daily", "anime"},
            },
        },
    }
}

func (d *Descriptor) EntityTypes() []module.EntityType {
    return []module.EntityType{module.EntitySeries, module.EntitySeason, module.EntityEpisode}
}

func (d *Descriptor) Wire() wire.ProviderSet {
    return wire.NewSet() // empty until module services are implemented
}
```

**Verify:** `go build ./internal/modules/tv/...` compiles.

---

### Task 0.8: Module Enable/Disable Setting and Registration Entry Point

**Create** `internal/module/settings.go`

Module enabled state is stored in the existing `settings` table using key format `module.<id>.enabled`:

```go
package module

import (
    "context"
    "database/sql"
    "fmt"
)

// IsModuleEnabled checks whether a module is enabled in settings.
// Returns true by default if no setting exists (all modules enabled out of the box).
func IsModuleEnabled(ctx context.Context, db *sql.DB, moduleType Type) (bool, error) {
    key := fmt.Sprintf("module.%s.enabled", moduleType)
    var value string
    err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
    if err == sql.ErrNoRows {
        return true, nil // enabled by default
    }
    if err != nil {
        return false, err
    }
    return value == "true", nil
}

// SetModuleEnabled sets the enabled state for a module.
func SetModuleEnabled(ctx context.Context, db *sql.DB, moduleType Type, enabled bool) error {
    key := fmt.Sprintf("module.%s.enabled", moduleType)
    value := "false"
    if enabled {
        value = "true"
    }
    _, err := db.ExecContext(ctx,
        `INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
        key, value,
    )
    return err
}
```

**Create** `internal/module/register.go`

The central registration point called at startup:

```go
package module

import (
    "github.com/rs/zerolog"
)

// RegisterAll registers all built-in modules with the registry.
// Called once at application startup.
func RegisterAll(registry *Registry, logger *zerolog.Logger) {
    // Module descriptors are registered here.
    // In Phase 0, we only register descriptors — full Module implementations come in later phases.
    //
    // For now, this function is a no-op placeholder. It will be populated
    // when module packages implement the full Module interface.
    //
    // Future:
    //   registry.Register(movie.New())
    //   registry.Register(tv.New())
    logger.Info().Msg("Module registry initialized (no full module implementations registered yet)")
}
```

**Note:** We do NOT wire the registry into the DI system in Phase 0. That happens in later phases when modules have meaningful implementations. The registry exists and can be constructed in tests, but it's not yet part of the startup path.

**Verify:** `go build ./internal/module/...` compiles.

---

### Task 0.9: Phase 0 Validation

**Run all of these after all Phase 0 tasks complete:**

1. `go build ./internal/module/...` — all new code compiles
2. `go build ./internal/modules/movie/...` — movie descriptor compiles
3. `go build ./internal/modules/tv/...` — TV descriptor compiles
4. `go test ./internal/module/...` — schema validation tests pass
5. `make build` — full project builds (no existing code broken)
6. `make test` — all existing tests pass
7. `make lint` — no new lint issues

**What exists after Phase 0:**
- `internal/module/` package with all interface definitions, types, registry, schema, settings helpers
- `internal/modules/movie/` with movie descriptor stub
- `internal/modules/tv/` with TV descriptor stub
- Zero changes to any existing code

---

