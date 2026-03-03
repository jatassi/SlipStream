# Refactoring Plan: Server God Object, Adapter Graveyard, Dev Mode Minefield

## Context

Commit `907c318` claimed to address code review items #1-3 but only reorganized code cosmetically:
- 78 fields renamed into nested groups with no encapsulation
- 23 adapters split across 4 files but none eliminated
- Manual SetDB enumeration moved (not removed), introducing a 4-service regression
- Five identical copy functions untouched

This plan addresses the *spirit* of the review: reduce coupling, eliminate boilerplate, make the system safer to extend.

---

## Phase 1: Shared Contracts Package

**Goal:** Eliminate ~25 duplicate interface definitions across packages.

Create `internal/domain/contracts/contracts.go` with interfaces duplicated across 2+ packages:

```go
type Broadcaster interface {
    Broadcast(msgType string, payload interface{})
}

type HealthService interface {
    RegisterItemStr(category, id, name string)
    UnregisterItemStr(category, id string)
    SetErrorStr(category, id, message string)
    ClearStatusStr(category, id string)
    SetWarningStr(category, id, message string)
}

type StatusChangeLogger interface {
    LogStatusChanged(ctx context.Context, mediaType string, mediaID int64, from, to, reason string) error
}

type FileDeleteHandler interface {
    OnFileDeleted(ctx context.Context, mediaType string, fileID int64) error
}

type QueueTrigger interface {
    Trigger()
}
```

**Replace local definitions in:**
- `Broadcaster` (10 copies): `autosearch`, `grab`, `search`, `history`, `health`, `downloader`, `requests/events`, `logger`, `notification/mock`, `metadata/artwork` (named `ArtworkBroadcaster` — rename to use shared `Broadcaster`)
- `HealthService` (7 copies, 2 variants):
  - 4-method base (no `SetWarningStr`): `indexer`, `downloader`, `rootfolder`
  - 5-method full: `indexer/status`, `metadata`, `import`
  - 2-method subset: `librarymanager` — update to accept the full 5-method interface (unused methods are harmless)
- `StatusChangeLogger` (3 copies): `movies`, `tv`, `downloader`
- `FileDeleteHandler` (2 copies): `movies`, `tv`
- `QueueTrigger` (2 copies): `downloader/broadcaster`, `grab`

Add compile-time checks in each concrete type's package:
```go
var _ contracts.Broadcaster = (*Hub)(nil)       // in websocket package
var _ contracts.HealthService = (*Service)(nil)  // in health package
var _ contracts.StatusChangeLogger = (*Service)(nil)  // in history package
```

**Note on HealthService variants:** The shared interface uses the 5-method version (including `SetWarningStr`). Packages that consumed the 4-method variant (`indexer`, `downloader`, `rootfolder`) now accept a wider interface — this is backward compatible since they only call the methods they need. The `librarymanager` 2-method subset is updated to accept the full interface.

**Files modified:** ~22 service files (replace local interface with import), 1 new file created.

---

## Phase 2: Eliminate Adapters via Direct Interface Satisfaction

**Goal:** Remove 6 of 23 adapter types by making concrete services directly satisfy consumer interfaces. After elimination, reorganize remaining adapters by domain.

### 2a. `statusTrackerSeriesLookup` → eliminated
`tv.Service` already has `GetSeriesIDByTvdbID` and `AreSeasonsComplete` with matching signatures. Pure pass-through wrapper.
- **Change:** Pass `s.library.TV` directly as `requests.SeriesLookup` in server init.
- **Verify:** `var _ requests.SeriesLookup = (*tv.Service)(nil)`

### 2b. `arrImportQualityAdapter` → eliminated
arrimport **already imports** `library/quality` (for `quality.GetQualityByName` in `quality_map.go`). The adapter strips `quality.Profile` down to just ID and Name, but arrimport's config import logic (`flattenQualityProfileItems`) needs the full Profile with Items, Cutoff, HDR settings, and codec settings. The adapter is actively harmful — it discards data arrimport needs elsewhere.
- **Change:** Change `arrimport.QualityService` interface to return `[]*quality.Profile`. Delete `arrimport.QualityProfile` struct. Update arrimport code to use `quality.Profile` fields directly.
- **Verify:** `var _ arrimport.QualityService = (*quality.Service)(nil)` (if `List()` signatures align after change)

### 2c. `arrImportRootFolderAdapter` → eliminated
No circular dependency risk — `rootfolder` imports only `database/sqlc`, `defaults`, `filesystem/mock`, `zerolog`. arrimport uses 4 of 7 fields (ID, Name, Path, MediaType); unused fields (FreeSpace, CreatedAt, IsDefault) are harmless to carry.
- **Change:** Change `arrimport.RootFolderService` to return `[]*rootfolder.RootFolder`. Delete `arrimport.RootFolder` struct. Update arrimport code to use `rootfolder.RootFolder` fields.
- **Verify:** `var _ arrimport.RootFolderService = (*rootfolder.Service)(nil)`

### 2d. `slotRootFolderAdapter` → eliminated
Same logic as 2c. `slots` gains a dependency on `rootfolder` (lightweight, no cycle risk).
- **Change:** Change `slots.RootFolderProvider` to return `*rootfolder.RootFolder`. Delete `slots.RootFolder` struct. Update slots code.
- **Verify:** `var _ slots.RootFolderProvider = (*rootfolder.Service)(nil)`

### 2e. `arrImportMetadataRefresherAdapter` → eliminated
The adapter discards the return values from `RefreshMovieMetadata` / `RefreshSeriesMetadata`. Callers use `_ , err :=` pattern already.
- **Change:** Update `arrimport.MetadataRefresher` to return `(*movies.Movie, error)` / `(*tv.Series, error)`. Pass `librarymanager.Service` directly.
- **Verify:** `var _ arrimport.MetadataRefresher = (*librarymanager.Service)(nil)`

### 2f. `arrImportHubAdapter` → eliminated
The adapter hardcodes `"arrImportProgress"` as the WebSocket message type. arrimport should own its own message type constant and use the shared `contracts.Broadcaster` directly.
- **Change:** Add `const WsArrImportProgress = "arrImportProgress"` in arrimport package. Change arrimport to accept `contracts.Broadcaster` and call `hub.Broadcast(WsArrImportProgress, v)`. Remove the custom `BroadcastJSON` interface.
- **Verify:** Compile-time: `var _ contracts.Broadcaster = (*websocket.Hub)(nil)` (already done in Phase 1)

### Adapters NOT eliminated (signature mismatches too deep)
- **`portalRequestSearcherAdapter`**: Consumer `admin.RequestSearcher` has `SearchForRequestAsync(requestID int64)` (no context), concrete has `SearchForRequestAsync(ctx, requestID)`. Adapter bridges context. Changing the admin interface would ripple through all admin handlers for marginal gain.
- **`portalAutoApproveAdapter`**: Three differences — adds `ctx`, removes `*users.User` param, changes return from `error` to `(*AutoApproveResult, error)`. Too many interface changes for the lines saved.

### Post-elimination adapter reorganization

After eliminating 6 adapters, reorganize the remaining 17 adapters by domain. Delete `adapters.go` (344 lines). Create focused files:

| File | Contents | Adapters |
|------|----------|----------|
| `notification_bridge.go` | Notification event mapping | `movieNotificationAdapter`, `tvNotificationAdapter`, `importNotificationAdapter`, `grabNotificationAdapter` |
| `portal_bridge.go` | Portal-specific bridges | `portalAutoApproveAdapter`, `portalRequestSearcherAdapter`, `portalQueueGetterAdapter`, `portalMediaLookupAdapter`, `portalEnabledChecker`, `portalUserQualityProfileAdapter`, `adminRequestLibraryCheckerAdapter`, `statusTrackerMovieLookup`, `statusTrackerEpisodeLookup` |
| `history_bridge.go` | History event bridges | `statusChangeLoggerAdapter`, `importHistoryAdapter` |
| `slot_bridge.go` | Slot operation bridges | `slotFileDeleterAdapter` |

Move contents of `notification_adapters.go` and `grab_notifications.go` into `notification_bridge.go`. Delete the originals.

**Result:** 23 adapters → 17 adapters. ~150 lines removed. 4 adapter files → 4 domain-organized bridge files.

**Regression risk:** Each eliminated adapter has a compile-time assertion (listed above) that fails if the concrete type drifts from the consumer interface. Additionally, the callers/call sites that change for each elimination:
- 2a: `initPortalServices` — `StatusTracker.SetSeriesLookup(...)` call
- 2b: `initAutomationServices` — `arrimport.NewService(...)` constructor arg
- 2c: `initAutomationServices` — `arrimport.NewService(...)` constructor arg
- 2d: `initLibraryServices` — `Slots.SetRootFolderProvider(...)` call
- 2e: `initAutomationServices` — `ArrImport.SetMetadataRefresher(...)` call
- 2f: `initAutomationServices` — `arrimport.NewService(...)` constructor arg

---

## Phase 3: Promote `portalMediaProvisionerAdapter` to a Service

**Goal:** The 306-line "adapter" is actually a full service with business logic (media provisioning, monitoring strategies, root folder resolution). It doesn't belong in `internal/api/`.

- Create `internal/portal/provisioner/service.go`
- Move all logic from `internal/api/media_provisioner.go`
- It satisfies `requests.MediaProvisioner` interface directly
- Constructor: `NewService(queries, movieSvc, tvSvc, libraryManager, logger)`
- Delete `internal/api/media_provisioner.go`

### SetDB and registry registration

The current adapter has a `SetDB(*sql.DB)` method that updates its internal `queries` field:
```go
func (a *portalMediaProvisionerAdapter) SetDB(db *sql.DB) {
    a.queries = sqlc.New(db)
}
```

The new `provisioner.Service` must preserve this. It must also be registered with the service registry (Phase 5's `SwitchableServices` struct) so that dev mode toggle switches it to the dev database. Without this, dev mode toggle would silently leave the provisioner pointing at the production database — exactly the class of regression that item #3 in the review warns about.

**Files:** 1 new package, 1 deleted file, wiring updates in server init, registry registration.

---

## Phase 4: Wire Code Generation + Constructor Refactoring

**Goal:** Replace hand-written init methods with `google/wire` dependency injection. Refactor setter injection to constructor parameters wherever possible, reducing the cost of adding a new service from "edit 4 files" to "add a provider, run `wire`."

### Current state

67 total `SetX()` setter calls in `server.go` and `routes.go`:
- **42 CONSTRUCTOR-SAFE**: Dependency is available at construction time, no circular dependency. Can move to constructor params.
- **13 CIRCULAR**: Moving to constructor would create a dependency cycle between groups (e.g., Movies ↔ Slots, Quality ↔ Import, LibraryManager ↔ Autosearch, Notification → many consumers).
- **12 LATE-BINDING**: Dependency unavailable at construction time (WebSocket hub callbacks, lambda functions, config-derived values).

### Strategy: Hybrid Wire + Setter File

Wire handles all constructor injection. A small manual `setters.go` file handles the ~25 circular/late-binding calls that can't be expressed as constructor params.

### Step 1: Refactor 42 CONSTRUCTOR-SAFE setters to constructor params

For each CONSTRUCTOR-SAFE setter, add the dependency as a constructor parameter and remove the `SetX` method. Example:

**Before:**
```go
// movies/service.go
func NewService(db *sql.DB, hub Broadcaster, logger *zerolog.Logger) *Service { ... }
func (s *Service) SetQualityService(qs QualityService) { s.qualityService = qs }
func (s *Service) SetStatusChangeLogger(l StatusChangeLogger) { s.statusChangeLogger = l }

// server.go
s.library.Movies = movies.NewService(db, hub, logger)
s.library.Movies.SetQualityService(s.library.Quality)
s.library.Movies.SetStatusChangeLogger(&statusChangeLoggerAdapter{s.system.History})
```

**After:**
```go
// movies/service.go
func NewService(db *sql.DB, hub contracts.Broadcaster, logger *zerolog.Logger,
    qualityService QualityService, statusChangeLogger contracts.StatusChangeLogger,
) *Service {
    return &Service{
        db: db, hub: hub, logger: logger,
        qualityService: qualityService, statusChangeLogger: statusChangeLogger,
    }
}
// SetQualityService and SetStatusChangeLogger methods deleted
```

### Key constructor refactoring targets

**Within-group (trivial, no ordering issues):**
- `Search.SetStatusService(Status)`, `Search.SetRateLimiter(RateLimiter)` → constructor params
- `Router.SetProwlarrSearcher(ProwlarrSearch)`, `Router.SetModeProvider(ProwlarrMode)` → constructor params
- `Grab.SetIndexerService(ProwlarrGrab)`, `Grab.SetStatusService(Status)`, `Grab.SetRateLimiter(RateLimiter)` → constructor params

**Cross-group (safe because dependency group is created first):**
- `RootFolder.SetHealthService(Health)` → constructor param (System created before Library)
- `Download.Service.SetHealthService(Health)` → constructor param (System created before Download)
- `Import.SetHealthService(Health)`, `Import.SetQualityService(Quality)`, `Import.SetSlotsService(Slots)` → constructor params
- `Autosearch.SetHistoryService(History)`, `Autosearch.SetGrabLock(GrabLock)` → constructor params
- All `StatusTracker.Set*Lookup(...)` calls → constructor params
- All `PortalRequests.SetWatchersService(...)`, `SetNotificationDispatcher(...)` → constructor params

### Step 2: Wire configuration

```go
// wire.go
//go:build wireinject

package api

func BuildServices(
    dbManager *database.Manager,
    hub *websocket.Hub,
    cfg *config.Config,
    logger *zerolog.Logger,
) *ServiceContainer {
    wire.Build(
        // Infrastructure
        provideDB,           // extracts *sql.DB from dbManager
        provideQueries,      // sqlc.New(db)

        // System providers
        health.NewService,
        defaults.NewService,
        history.NewService,
        progress.NewManager,
        calendar.NewService,
        availability.NewService,
        missing.NewService,
        preferences.NewService,

        // Metadata providers
        metadata.NewService,
        metadata.NewArtworkDownloader,
        metadata.NewSQLNetworkLogoStore,

        // Library providers
        scanner.NewService,
        movies.NewService,
        tv.NewService,
        quality.NewService,
        slots.NewService,
        rootfolder.NewService,
        organizer.NewService,
        mediainfo.NewService,
        librarymanager.NewService,

        // Filesystem providers
        filesystem.NewService,
        filesystem.NewStorageService,

        // Download providers
        downloader.NewService,
        downloader.NewQueueBroadcaster,

        // Search providers
        cardigann.NewManager,
        indexer.NewService,
        status.NewService,
        indexer.NewCookieStore,
        ratelimit.NewLimiter,
        prowlarr.NewService,
        prowlarr.NewModeManager,
        search.NewService,
        prowlarr.NewSearchAdapter,
        prowlarr.NewGrabProvider,
        search.NewRouter,
        grab.NewService,
        decisioning.NewGrabLock,

        // Automation providers
        importer.NewService,
        arrimport.NewService,
        autosearch.NewService,
        autosearch.NewScheduledSearcher,
        rsssync.NewService,
        scheduler.New,

        // Notification providers
        notification.NewService,
        plex.NewClient,
        plex.NewHandlers,

        // Portal providers
        users.NewService,
        invitations.NewService,
        quota.NewService,
        requests.NewService,
        requests.NewWatchersService,
        requests.NewStatusTracker,
        requests.NewLibraryChecker,
        requests.NewRequestSearcher,
        requests.NewEventBroadcaster,
        portalnotifs.NewService,
        autoapprove.NewService,
        auth.NewService,
        auth.NewPasskeyService,
        portalmw.NewAuthMiddleware,
        portalratelimit.NewSearchLimiter,
        provisioner.NewService,

        // Security
        authratelimit.NewAuthLimiter,

        // Interface bindings
        wire.Bind(new(contracts.Broadcaster), new(*websocket.Hub)),
        wire.Bind(new(contracts.HealthService), new(*health.Service)),
        wire.Bind(new(contracts.StatusChangeLogger), new(*history.Service)),
        wire.Bind(new(contracts.FileDeleteHandler), new(*slots.Service)),
        wire.Bind(new(contracts.QueueTrigger), new(*downloader.QueueBroadcaster)),

        // Group struct assembly
        wire.Struct(new(ServiceContainer), "*"),
        wire.Struct(new(LibraryGroup), "*"),
        wire.Struct(new(MetadataGroup), "*"),
        wire.Struct(new(FilesystemGroup), "*"),
        wire.Struct(new(DownloadGroup), "*"),
        wire.Struct(new(SearchGroup), "*"),
        wire.Struct(new(AutomationGroup), "*"),
        wire.Struct(new(SystemGroup), "*"),
        wire.Struct(new(NotificationGroup), "*"),
        wire.Struct(new(PortalGroup), "*"),
        wire.Struct(new(SecurityGroup), "*"),
    )
    return nil
}
```

Run `wire` to generate `wire_gen.go` (~300-400 lines) that calls constructors in dependency order.

### Step 3: Manual setter file for circular and late-binding deps

```go
// setters.go (~80 lines)
func wireCircularDeps(c *ServiceContainer) {
    // Circular: Movies ↔ Slots (Movies deletes files via Slots, Slots deletes media via Movies/TV)
    c.Library.Movies.SetFileDeleteHandler(c.Library.Slots)
    c.Library.TV.SetFileDeleteHandler(c.Library.Slots)
    c.Library.Slots.SetFileDeleter(&slotFileDeleterAdapter{
        movieSvc: c.Library.Movies, tvSvc: c.Library.TV,
    })

    // Circular: Quality ↔ Import
    c.Library.Quality.SetImportDecisionCleaner(c.Automation.Import)

    // Circular: LibraryManager ↔ Autosearch
    c.Library.LibraryManager.SetAutosearchService(c.Automation.Autosearch)
    c.Automation.ScheduledSearcher.SetSeriesRefresher(c.Library.LibraryManager)
    c.Automation.ArrImport.SetMetadataRefresher(c.Library.LibraryManager)

    // Circular: Notification → many consumers
    c.System.Health.SetNotifier(c.Notification.Service)
    c.Search.Grab.SetNotificationService(&grabNotificationAdapter{
        svc: c.Notification.Service, movieSvc: c.Library.Movies, tvSvc: c.Library.TV,
    })
    c.Library.Movies.SetNotificationDispatcher(&movieNotificationAdapter{svc: c.Notification.Service})
    c.Library.TV.SetNotificationDispatcher(&tvNotificationAdapter{svc: c.Notification.Service})
    c.Automation.Import.SetNotificationDispatcher(&importNotificationAdapter{svc: c.Notification.Service})
    c.Automation.ArrImport.SetConfigImportServices(
        c.Download.Service, c.Search.Indexer, c.Notification.Service, c.Library.Quality, c.Automation.Import,
    )
}

func wireLateBindings(c *ServiceContainer, hub *websocket.Hub, dbManager *database.Manager) {
    hub.SetDevModeHandler(func(enabled bool) { /* ... */ })
    hub.SetTokenValidator(func(token string) bool { /* ... */ })
    c.Download.QueueBroadcaster.SetCompletionHandler(c.Automation.Import)
    c.Portal.SearchLimiter.StartCleanup(5 * time.Minute)
}
```

### Step 4: Move all service creation out of routes.go

Currently `routes.go` creates `RssSyncSettings`, `ImportSettings`, `RequestSearcher`, `AdminSettings`, `MediaProvisioner`. All of these become Wire providers. After this change, `setupRoutes` purely maps URLs to handlers — zero service construction, zero `SetX()` calls except on handler instances.

### NewServer becomes thin:

```go
func NewServer(dbManager, hub, cfg, logger, restartChan) *Server {
    services := BuildServices(dbManager, hub, cfg, logger)  // Wire-generated
    wireCircularDeps(services)
    wireLateBindings(services, hub, dbManager)
    s := &Server{echo: newEcho(), services: services, restartChan: restartChan}
    s.setupRoutes()
    return s
}
```

### Adding a new service after this change

1. Write the service package with a constructor accepting all dependencies as params
2. Add the constructor to `wire.Build()` in `wire.go`
3. Add the field to the appropriate group struct
4. Run `wire` — it resolves the dependency graph and regenerates `wire_gen.go`
5. If circular dependency: add one setter call to `setters.go`
6. Add route in `routes.go`

Cost: 2-3 files (service, wire.go, routes.go) instead of 4+ (server.go init, service_groups.go, service_registry.go, routes.go, possibly adapters.go).

**Build dependency:** `google/wire` (code generation tool, not a runtime dependency). Add to `tools.go`:
```go
//go:build tools
import _ "github.com/google/wire/cmd/wire"
```

Add `make wire` target:
```makefile
wire:
	cd internal/api && wire
```

**Files:** New `wire.go` and `wire_gen.go`. Delete 9 `init*` methods from `server.go`. New `setters.go` (~80 lines). `server.go` shrinks from ~848 to ~150 lines. `service_groups.go` kept (group structs still needed).

---

## Phase 5: Reflection-Based Service Registry

**Goal:** Replace the runtime slice-based `ServiceRegistry` with a struct-tagged registry that auto-discovers switchable services via reflection. Eliminates the manual enumeration that caused the 4-service regression.

### New `SwitchableServices` struct with tags:

```go
// switchable.go — replaces service_registry.go
type SwitchableServices struct {
    // SetDB(*sql.DB) services
    Defaults        *defaults.Service       `switchable:"db"`
    Calendar        *calendar.Service       `switchable:"db"`
    Availability    *availability.Service   `switchable:"db"`
    Missing         *missing.Service        `switchable:"db"`
    History         *history.Service        `switchable:"db"`
    Movies          *movies.Service         `switchable:"db"`
    TV              *tv.Service             `switchable:"db"`
    Quality         *quality.Service        `switchable:"db"`
    Slots           *slots.Service          `switchable:"db"`
    RootFolder      *rootfolder.Service     `switchable:"db"`
    Metadata        *metadata.Service       `switchable:"db"`
    Download        *downloader.Service     `switchable:"db"`
    Indexer         *indexer.Service         `switchable:"db"`
    Status          *status.Service         `switchable:"db"`
    RateLimiter     *ratelimit.Limiter      `switchable:"db"`
    Prowlarr        *prowlarr.Service       `switchable:"db"`
    Grab            *grab.Service           `switchable:"db"`
    Import          *importer.Service       `switchable:"db"`
    Autosearch      *autosearch.Service     `switchable:"db"`
    Notification    *notification.Service   `switchable:"db"`
    ImportSettings  *importer.SettingsHandlers `switchable:"db"`
    RequestSearcher *requests.RequestSearcher  `switchable:"db"`
    Provisioner     *provisioner.Service    `switchable:"db"`

    // SetDB(*sqlc.Queries) services
    RssSync         *rsssync.Service            `switchable:"queries"`
    RssSyncSettings *rsssync.SettingsHandler    `switchable:"queries"`
    PortalUsers     *users.Service              `switchable:"queries"`
    Invitations     *invitations.Service        `switchable:"queries"`
    Quota           *quota.Service              `switchable:"queries"`
    Requests        *requests.Service           `switchable:"queries"`
    Watchers        *requests.WatchersService   `switchable:"queries"`
    LibraryChecker  *requests.LibraryChecker    `switchable:"queries"`
    Notifications   *portalnotifs.Service       `switchable:"queries"`
    AdminSettings   *admin.SettingsHandlers     `switchable:"queries"`
    StatusTracker   *requests.StatusTracker     `switchable:"queries"`
}
```

### Reflection-based UpdateAll:

```go
func (sw *SwitchableServices) UpdateAll(db *sql.DB) {
    queries := sqlc.New(db)
    v := reflect.ValueOf(sw).Elem()
    t := v.Type()
    for i := 0; i < v.NumField(); i++ {
        field := v.Field(i)
        tag := t.Field(i).Tag.Get("switchable")
        if tag == "" || field.IsNil() {
            continue
        }
        method := field.MethodByName("SetDB")
        if !method.IsValid() {
            panic(fmt.Sprintf("switchable field %s has no SetDB method", t.Field(i).Name))
        }
        switch tag {
        case "db":
            method.Call([]reflect.Value{reflect.ValueOf(db)})
        case "queries":
            method.Call([]reflect.Value{reflect.ValueOf(queries)})
        }
    }
}
```

### Reflection-based Validate:

```go
func (sw *SwitchableServices) Validate() error {
    v := reflect.ValueOf(sw).Elem()
    t := v.Type()
    var missing []string
    for i := 0; i < v.NumField(); i++ {
        tag := t.Field(i).Tag.Get("switchable")
        if tag == "" {
            continue
        }
        if v.Field(i).IsNil() {
            missing = append(missing, t.Field(i).Name)
        }
    }
    if len(missing) > 0 {
        return fmt.Errorf("switchable services not registered: %s", strings.Join(missing, ", "))
    }
    return nil
}
```

### Why this works:
- Adding a field with a `switchable` tag but forgetting to assign it → caught by `Validate()` at startup
- Adding a field without a `switchable` tag → field is skipped by `UpdateAll()` (no silent failure — it simply isn't switched, which is the correct behavior for non-switchable fields)
- `UpdateAll()` never needs manual updating when services are added/removed — the reflection loop handles it
- The 4-service regression from commit `907c318` becomes impossible: if you add the field, reflection finds it; if you forget the field, Wire can't assign it (compile error)

### Startup test:
```go
func TestAllSwitchableServicesRegistered(t *testing.T) {
    tdb := testutil.NewTestDB(t)
    defer tdb.Close()
    cfg := &config.Config{...}
    container := BuildServices(tdb.Manager, nil, cfg, &tdb.Logger)
    wireCircularDeps(container)
    if err := container.Switchable.Validate(); err != nil {
        t.Fatal(err)
    }
}
```

### Performance note:
Reflection-based `UpdateAll` is ~10µs for 30 fields. Dev mode toggle happens at most once per user action. Performance is irrelevant here.

**Files:** New `switchable.go` replaces `service_registry.go`. Wire assigns to `c.Switchable.X` fields as part of the generated code.

---

## Phase 6: Dev Mode Manager Extraction

**Goal:** Extract dev mode logic into a dedicated `DevModeManager` type, removing ~20 methods from Server and grouping all dev mode concerns together. Eliminate shared boilerplate with a `prodAndDevQueries()` helper.

### Extract DevModeManager:

```go
// devmode.go
type DevModeManager struct {
    container *ServiceContainer
    dbManager *database.Manager
    hub       *websocket.Hub
    cfg       *config.Config
    logger    *zerolog.Logger
}

func NewDevModeManager(container *ServiceContainer, dbManager *database.Manager,
    hub *websocket.Hub, cfg *config.Config, logger *zerolog.Logger) *DevModeManager {
    return &DevModeManager{container: container, dbManager: dbManager, hub: hub, cfg: cfg, logger: logger}
}

func (d *DevModeManager) OnToggle(enabled bool) error { ... }
func (d *DevModeManager) switchMetadataClients() { ... }
func (d *DevModeManager) switchIndexer() { ... }
// etc.
```

### Helper: `prodAndDevQueries()`

```go
func (d *DevModeManager) prodAndDevQueries() (*sqlc.Queries, *sqlc.Queries) {
    return sqlc.New(d.dbManager.ProdConn()), sqlc.New(d.dbManager.Conn())
}
```

Replaces the repeated `sqlc.New(s.dbManager.ProdConn())` / `sqlc.New(s.dbManager.Conn())` pair at the top of each copy function. Saves ~10 lines total.

### What this phase does NOT do

The 5 copy functions (`copyJWTSecretToDevDB`, `copySettingsToDevDB`, `copyPortalUsersToDevDB`, `copyPortalUserNotificationsToDevDB`, `copyQualityProfilesToDevDB`) are **not consolidated with a generic helper**. Analysis shows they are structurally too different:
- `copyJWTSecretToDevDB`: Single item, no loop
- `copySettingsToDevDB`: Whitelist keys, individual fetches per key
- `copyPortalUsersToDevDB`: Existence check, ID preservation, quality profile ID remapping (depends on output of `copyQualityProfilesToDevDB`)
- `copyPortalUserNotificationsToDevDB`: Nested loop (users → notifications per user), no dedup
- `copyQualityProfilesToDevDB`: Early return if dev already has profiles, fallback to defaults, returns ID mapping for other functions

A generic `copyRows[T]` helper would need ~15 optional config fields to express these variations, producing code harder to read and more error-prone than the current explicit functions.

**Files:** `devmode.go` refactored (expected ~880 lines from ~923 — savings are modest because the value is structural clarity, not line reduction).

---

## Execution Strategy

This plan is designed to be executed by a single orchestrating Claude instance (the **main agent**) that delegates mechanical code changes to parallelized, backgrounded Sonnet subagents. The goal is to complete all phases within one main context window, avoiding compaction.

### Principles

1. **Main agent orchestrates; subagents execute.** The main agent reads this plan, launches subagents with self-contained prompts, verifies results, and handles cross-cutting concerns (server.go, wire.go, verification). It should avoid reading large files that subagents are modifying — trust their output and verify via `make build` / `make test`.
2. **Subagents touch disjoint file sets.** No two concurrent subagents should modify the same file. When a file must be touched by multiple phases (e.g., `server.go`), the main agent handles it directly or serializes the work.
3. **Background by default.** All subagents launch in background mode. The main agent proceeds with independent work (or launches more subagents) while waiting.
4. **Self-contained prompts.** Each subagent prompt includes the full context it needs — file paths, exact interface signatures, before/after examples. Subagents do NOT have access to this plan or the conversation history.
5. **Verification gates.** After each phase completes, the main agent runs verification before proceeding. This catches issues early rather than compounding them across phases.
6. **No git stash.** Every subagent prompt must include: *"You are forbidden from using git stash or any commands that affect the entire worktree."*

### Phase 1: Shared Contracts (~4 parallel subagents)

**Main agent does:**
- Create `internal/domain/contracts/contracts.go` (the source file — small, write directly)

**Subagents (launch all 4 in background simultaneously):**

| Agent | Task | Files touched |
|-------|------|--------------|
| 1A | Replace `Broadcaster` interface in 10 packages + rename `ArtworkBroadcaster` in `metadata/artwork.go` | `autosearch/service.go`, `indexer/grab/service.go`, `indexer/search/service.go`, `history/service.go`, `health/service.go`, `downloader/broadcaster.go`, `portal/requests/events.go`, `logger/broadcaster.go`, `notification/mock/notifier.go`, `metadata/artwork.go` |
| 1B | Replace `HealthService` interface in 7 packages (handle 4-method, 5-method, and 2-method variants) | `indexer/service.go`, `indexer/status/service.go`, `metadata/service.go`, `downloader/service.go`, `import/service.go`, `library/rootfolder/service.go`, `library/librarymanager/service.go` |
| 1C | Replace `StatusChangeLogger` (3 packages), `FileDeleteHandler` (2 packages), `QueueTrigger` (2 packages) | `library/movies/service.go`, `library/tv/service.go`, `downloader/service.go`, `downloader/broadcaster.go`, `indexer/grab/service.go` |
| 1D | Add compile-time assertions (`var _ contracts.X = (*Y)(nil)`) in each concrete implementor's package | `websocket/hub.go`, `health/service.go`, `history/service.go`, `library/slots/fileops.go`, `downloader/broadcaster.go` |

**Each subagent prompt must include:**
- The exact contents of `contracts.go` (all 5 interfaces with full signatures)
- The exact file paths and line numbers where the local interface is defined
- Instruction: delete the local interface definition, add `import "internal/domain/contracts"`, replace all references to the local type name with `contracts.X`
- Instruction for 1B: packages using the 4-method variant now accept the 5-method interface — this is backward compatible, just change the type name
- Instruction for 1D: list each concrete type and which interface it implements

**Main agent verifies:** `make build && go test -race ./internal/... && make lint`

Note: Agent 1C touches `downloader/service.go` for `StatusChangeLogger` and Agent 1B touches it for `HealthService`. These are **different interfaces in different locations in the same file**. If this causes merge conflicts, the main agent resolves them (trivial — both are import additions + interface replacements). Alternatively, merge 1B and 1C into a single agent to avoid the conflict.

### Phases 2+3: Adapter Elimination + Media Provisioner (~4 parallel subagents)

**These run after Phase 1 verification passes.** Phases 2 and 3 are independent and run simultaneously.

| Agent | Task | Files touched |
|-------|------|--------------|
| 2A | Eliminate arrImport adapters: 2b (`QualityService`), 2c (`RootFolderService`), 2e (`MetadataRefresher`), 2f (`BroadcastJSON` → `contracts.Broadcaster`) | `arrimport/service.go`, `arrimport/*.go` (uses of deleted types), `api/adapters.go` (delete 4 adapter types) |
| 2B | Eliminate `statusTrackerSeriesLookup` (2a) + `slotRootFolderAdapter` (2d). Update consumer interfaces in `requests/` and `slots/` | `portal/requests/status_tracker.go`, `library/slots/service.go`, `library/slots/*.go`, `api/adapters.go` (delete 2 adapter types) |
| 2C | **Phase 3:** Promote `portalMediaProvisionerAdapter` to `internal/portal/provisioner/service.go`. Create the new package, move logic, add `SetDB` method | `portal/provisioner/service.go` (new), `api/media_provisioner.go` (delete) |
| 2D | Reorganize remaining adapters: create `notification_bridge.go`, `portal_bridge.go`, `history_bridge.go`, `slot_bridge.go`. Merge `notification_adapters.go` + `grab_notifications.go` into `notification_bridge.go` | `api/notification_bridge.go` (new), `api/portal_bridge.go` (new), `api/history_bridge.go` (new), `api/slot_bridge.go` (new), `api/notification_adapters.go` (delete), `api/grab_notifications.go` (delete) |

**Conflict note:** Agents 2A and 2B both modify `api/adapters.go` (deleting different adapter types). Two options:
- **Option A (preferred):** Agent 2A deletes its 4 adapters. Agent 2B deletes its 2 adapters. Agent 2D then reorganizes what remains. Since 2D depends on 2A+2B completing, launch 2D after 2A+2B finish.
- **Option B:** Combine 2A+2B into one agent, then launch 2D after.

**Main agent does after all 4 complete:**
- Update `server.go` init methods to remove adapter wiring for the 6 eliminated adapters (replace with direct service references)
- Add compile-time assertions per Phase 2 verification section
- Run verification: `make build && go test -race ./internal/... && make lint`

### Phase 4: Wire + Constructor Refactoring (~6 parallel subagents, then main agent)

This is the highest-effort phase. Split into two stages:

**Stage 4a: Refactor constructors (~5 parallel subagents)**

Each subagent refactors one service group. The task: for each CONSTRUCTOR-SAFE setter listed below, add the dependency as a constructor parameter and delete the `SetX` method. The subagent must update both the service's `NewService` function signature AND the `Service` struct initialization.

| Agent | Group | Services to refactor | Setter → constructor conversions |
|-------|-------|---------------------|--------------------------------|
| 4A | System | `health`, `history` | `Health.SetBroadcaster(hub)`, `History.SetBroadcaster(hub)` — add `hub contracts.Broadcaster` to constructors |
| 4B | Library | `movies`, `tv`, `rootfolder`, `librarymanager` | `Movies.SetQualityService`, `Movies.SetStatusChangeLogger`, `TV.SetQualityService`, `TV.SetStatusChangeLogger`, `RootFolder.SetHealthService`, `LibraryManager.SetPreferencesService`, `LibraryManager.SetSlotsService`, `LibraryManager.SetHealthService` |
| 4C | Search | `search/service`, `search/router`, `grab`, `indexer`, `indexer/status` | `Search.SetStatusService`, `Search.SetRateLimiter`, `Search.SetBroadcaster`, `Router.SetProwlarrSearcher`, `Router.SetModeProvider`, `Grab.SetIndexerService`, `Grab.SetStatusService`, `Grab.SetRateLimiter`, `Grab.SetBroadcaster`, `Grab.SetQueueTrigger`, `Grab.SetPortalStatusTracker`, `Indexer.SetHealthService`, `Status.SetHealthService` |
| 4D | Automation + Download | `import`, `autosearch`, `downloader`, `downloader/broadcaster`, `arrimport` | `Import.SetHealthService`, `Import.SetHistoryService`, `Import.SetQualityService`, `Import.SetSlotsService`, `Import.SetStatusTracker`, `Autosearch.SetHistoryService`, `Autosearch.SetGrabLock`, `Autosearch.SetBroadcaster`, `Download.SetHealthService`, `Download.SetBroadcaster`, `Download.SetStatusChangeLogger`, `Download.SetPortalStatusTracker`, `QueueBroadcaster.SetCompletionHandler`, `ArrImport.SetSlotsService` |
| 4E | Portal + Notification + Metadata | `requests`, `requests/status_tracker`, `portalmw`, `metadata/service`, `metadata/artwork` | `Requests.SetWatchersService`, `Requests.SetNotificationDispatcher`, `StatusTracker.SetMovieLookup`, `StatusTracker.SetEpisodeLookup`, `StatusTracker.SetSeriesLookup`, `StatusTracker.SetNotificationDispatcher`, `AuthMiddleware.SetEnabledChecker`, `Metadata.SetHealthService`, `Metadata.SetNetworkLogoStore`, `ArtworkDownloader.SetBroadcaster` |

**Each subagent prompt must include:**
- The exact before/after pattern (shown in Phase 4 Step 1 of this plan)
- The specific setter methods to convert, with current signatures
- The interface types to use for new params (from `contracts` package or local interfaces)
- Instruction: update the `NewService`/`New` function, update the struct literal in the constructor, delete the `SetX` method, update any tests that call `NewService` to pass the new params (tests can pass `nil` for interface params if not exercised)
- Instruction: do NOT modify `server.go` or `routes.go` — the main agent handles those

**Main agent does after Stage 4a subagents complete:**
- Run `make build` to identify any compilation errors from constructor changes (expected: `server.go` won't compile because init methods still use old signatures)

**Stage 4b: Main agent writes Wire infrastructure**

The main agent directly:
1. Installs Wire: `go install github.com/google/wire/cmd/wire@latest`
2. Creates `internal/api/wire.go` (the full `wire.Build()` config from this plan)
3. Creates `internal/api/setters.go` (the `wireCircularDeps` + `wireLateBindings` functions)
4. Rewrites `internal/api/server.go` to the thin version (delete all `init*` methods, new `NewServer` calls `BuildServices` + `wireCircularDeps` + `wireLateBindings`)
5. Updates `internal/api/service_groups.go` if needed (field types may have changed)
6. Removes service creation from `routes.go` (the 5 services created there become Wire providers)
7. Runs `cd internal/api && wire` to generate `wire_gen.go`
8. Iterates on compilation errors until `make build` passes
9. Runs full verification

**Why the main agent handles Stage 4b:** These files (`server.go`, `wire.go`, `setters.go`, `routes.go`, `service_groups.go`) are all interdependent and reference every service. A single agent with full context resolves Wire's type-matching errors faster than multiple agents conflicting on the same files.

**Context window note:** Stage 4b is the most context-intensive part. The main agent should avoid unnecessary file reads — write `wire.go` from the template in this plan, and only read files to debug specific compilation errors.

### Phases 5+6: Registry + DevMode (~2 parallel subagents)

**Launch both in background after Phase 4 verification passes.**

| Agent | Task | Files touched |
|-------|------|--------------|
| 5A | Create `switchable.go` with the `SwitchableServices` struct (all tagged fields), `UpdateAll()`, `Validate()`. Delete `service_registry.go`. | `api/switchable.go` (new), `api/service_registry.go` (delete) |
| 6A | Extract `DevModeManager` from `devmode.go`. Change receiver from `*Server` to `*DevModeManager`. Add `prodAndDevQueries()` helper. Update `NewDevModeManager` constructor. | `api/devmode.go` |

**Main agent does after both complete:**
- Update `server.go` to use `DevModeManager` and `SwitchableServices` (replace `s.registry.UpdateAll(db)` with `s.services.Switchable.UpdateAll(db)`, create `DevModeManager` in `NewServer`)
- Update `wire.go` to include `SwitchableServices` in `ServiceContainer`
- Regenerate: `cd internal/api && wire`
- Write `TestAllSwitchableServicesRegistered` and `TestDevModeToggleDBSwitching` tests
- Run verification: `make build && go test -race ./internal/... && make lint`

### Phase 7: Documentation (~1 subagent)

**Single backgrounded subagent** with the full list of doc changes from this plan's Phase 7 section. It modifies 4 files:
- `CLAUDE.md`, `AGENTS.md` (root)
- `internal/CLAUDE.md`, `internal/AGENTS.md`

The subagent prompt must include the exact before/after text for each change.

### Context Window Budget

| Phase | Main agent context cost | Subagent count | Blocking? |
|-------|------------------------|----------------|-----------|
| Read plan | Low | 0 | — |
| Phase 1 | Low (create 1 file, launch 4 agents, verify) | 4 | Wait for all |
| Phases 2+3 | Medium (update server.go wiring after agents) | 4 | Wait for all |
| Phase 4 Stage 4a | Low (launch 5 agents) | 5 | Wait for all |
| Phase 4 Stage 4b | **High** (write wire.go, setters.go, rewrite server.go, iterate on errors) | 0 | Main agent works |
| Phases 5+6 | Medium (integrate results, write tests) | 2 | Wait for both |
| Phase 7 | Low (launch 1 agent) | 1 | Wait |
| **Total** | | **~16 subagents** | |

**Critical path for context:** Stage 4b is the only high-cost segment for the main agent. Everything else is orchestration (launching subagents with copy-pasted prompts from this plan) and verification (running `make build` / `make test`).

### Failure Recovery

- **Subagent produces broken code:** The main agent catches this at the verification gate. Fix directly if trivial (1-2 lines), or relaunch the subagent with a more specific prompt referencing the error.
- **Wire can't resolve dependencies:** Common causes: ambiguous interface bindings (two types satisfy the same interface), missing providers. Wire's error messages are specific — the main agent reads the error and adds `wire.Bind()` or adjusts constructor signatures.
- **Merge conflict between subagents:** Only possible if two agents touch the same file. The plan avoids this by design. If it happens (e.g., Agents 1B and 1C both touching `downloader/service.go`), the main agent resolves manually — it's always a trivial merge of independent import additions.
- **Tests fail after Phase 4:** Most likely cause: a test calls `NewService()` with the old argument count. The subagent prompt should instruct updating test calls, but if missed, the main agent runs `grep -r 'NewService(' internal/package/` to find the call sites and fix them. Alternatively, launch a targeted subagent: *"Fix all compilation errors in `internal/X/` test files caused by the new `NewService` constructor signature."*

---

## Execution Order

```
Phase 1 (shared contracts)
  │
  ├── Phase 2 (eliminate 6 adapters + reorganize remaining) — depends on Phase 1 for Broadcaster
  │
  └── Phase 3 (promote media provisioner) — independent, can parallel with 2
       │
       Phase 4 (Wire code generation + constructor refactoring) — after 2+3 so service signatures are final
       │
       ├── Phase 5 (reflection-based registry) — depends on Phase 4 for ServiceContainer
       │
       └── Phase 6 (DevModeManager extraction) — depends on Phase 4 for ServiceContainer
            │
            Phase 7 (documentation updates) — after all code changes and verification
```

Phases 2 and 3 can run in parallel. Phases 5 and 6 can run in parallel after Phase 4. Phase 7 runs last, after all verification passes.

**Critical path:** Phase 4 is the highest-effort phase and the bottleneck. It requires refactoring 42 service constructors across ~30 packages. Phases before it (1-3) should be completed first to minimize merge conflicts during the constructor refactoring.

---

## Expected Outcome

| Metric | Before | After |
|--------|--------|-------|
| Adapter types | 23 | 17 |
| Adapter lines | 940 | ~750 |
| Duplicate interface definitions | ~25 | 0 |
| Server struct direct fields | 19 | 4 (`echo`, `services`, `restartChan`, `configuredPort`) |
| `server.go` lines | 848 | ~150 |
| Setter calls | 67 | ~25 (circular + late-binding only) |
| Files to add a new service | 4+ | 2-3 (service, wire.go, routes.go) |
| Registry safety | Runtime (silent failures) | Reflection + startup validation |
| `devmode.go` lines | 923 | ~880 |
| Service creation locations | server.go + routes.go | Wire-generated |
| Build dependency | None | `google/wire` (codegen only, not runtime) |

---

## Verification

### After each phase:
1. `make build` — confirms compilation
2. `go test -race ./internal/...` — confirms no regressions AND no race conditions
3. `make lint` — confirms no lint issues
4. `go vet ./...` — confirms no import cycles

### Phase-specific regression tests:

**Phase 2 — Compile-time interface assertions:**
```go
// In respective test files:
var _ requests.SeriesLookup = (*tv.Service)(nil)
var _ arrimport.QualityService = (*quality.Service)(nil)
var _ arrimport.RootFolderService = (*rootfolder.Service)(nil)
var _ slots.RootFolderProvider = (*rootfolder.Service)(nil)
var _ arrimport.MetadataRefresher = (*librarymanager.Service)(nil)
var _ contracts.Broadcaster = (*websocket.Hub)(nil)
```

**Phase 4 — Wire generation validation:**
```bash
cd internal/api && wire check  # validates wire.go without generating
```

**Phase 5 — Switchable registry completeness:**
```go
func TestAllSwitchableServicesRegistered(t *testing.T) {
    tdb := testutil.NewTestDB(t)
    defer tdb.Close()
    cfg := &config.Config{...}
    container := BuildServices(tdb.Manager, nil, cfg, &tdb.Logger)
    wireCircularDeps(container)
    if err := container.Switchable.Validate(); err != nil {
        t.Fatal(err)
    }
}
```

**Phase 5+6 — Dev mode toggle integration:**
```go
func TestDevModeToggleDBSwitching(t *testing.T) {
    tdb := testutil.NewTestDB(t)
    defer tdb.Close()
    container := BuildServices(tdb.Manager, nil, cfg, &tdb.Logger)
    wireCircularDeps(container)

    // Write a sentinel value through one service
    container.System.Defaults.SetDB(devDB)
    // Verify the write is visible through the service
    // Switch back to prod
    container.Switchable.UpdateAll(prodDB)
    // Verify the sentinel is NOT visible (it's in devDB)
}

func TestDevModeCopyFunctions(t *testing.T) {
    // Create prod DB with known data:
    //   JWT secret, settings, quality profiles, portal users, notifications
    // Toggle to dev mode (triggers copy functions)
    // Verify:
    //   - JWT secret exists in dev DB
    //   - Expected settings copied
    //   - Quality profiles copied
    //   - Portal users copied with correct quality profile ID remapping
    //   - User notifications copied
}
```

---

## Phase 7: Documentation Updates

**Goal:** Update all project documentation to reflect the new architecture. Without this, agents and contributors will follow stale instructions that reference deleted files, wrong paths, and pre-refactoring patterns.

All changes below apply to **both** the `CLAUDE.md` and `AGENTS.md` in each directory (per the "Making Changes" header in each file).

### Root `CLAUDE.md` + `AGENTS.md`

**Fix existing inaccuracy — API routes path (line 81):**
```
Before: Route definitions in `internal/server/routes.go`.
After:  Route definitions in `internal/api/routes.go`.
```

**Update Project Structure to include new packages:**
```
- `internal/domain/contracts/` — Shared service interfaces (Broadcaster, HealthService, etc.)
- `internal/portal/provisioner/` — Media provisioning service (portal request fulfillment)
```

**Add Wire codegen to Common Commands:**
```bash
make wire                 # Regenerate Wire dependency injection (after changing wire.go)
```

### Internal `CLAUDE.md` + `AGENTS.md`

**Fix Architecture section:**
```
Before: Handlers -> Services -> sqlc queries. Services are injected during server setup in `internal/server/`.
After:  Handlers -> Services -> sqlc queries. Services are wired via google/wire in `internal/api/`.
        Wire handles constructor injection (`wire.go` → `wire_gen.go`).
        Circular and late-binding deps are in `setters.go`.
        Route setup in `routes.go` is purely declarative (no service construction).
```

**Add new section: Dependency Injection**
```
## Dependency Injection (Wire)

Service wiring uses `google/wire` for code generation. Key files in `internal/api/`:
- `wire.go` — Provider declarations and interface bindings (the source of truth)
- `wire_gen.go` — Auto-generated constructor call chain (DO NOT EDIT)
- `setters.go` — Manual setter calls for circular deps and late-binding (~25 calls)
- `service_groups.go` — Group structs (LibraryGroup, SearchGroup, etc.)
- `switchable.go` — Dev mode DB-switching registry with struct tags

### Adding a new service
1. Write the service package with a constructor accepting ALL dependencies as params
2. Add the constructor to `wire.Build()` in `wire.go`
3. Add the field to the appropriate group struct in `service_groups.go`
4. Run `make wire` to regenerate `wire_gen.go`
5. If the service needs dev mode DB switching: add a tagged field to `SwitchableServices` in `switchable.go`
6. If circular dependency: add one setter call to `setters.go`
7. Add route in `routes.go`

### Shared interfaces
Common interfaces live in `internal/domain/contracts/`. Use these instead of defining
local single-use copies:
- `contracts.Broadcaster` — WebSocket broadcast
- `contracts.HealthService` — Health status registration (5 methods)
- `contracts.StatusChangeLogger` — Media status change logging
- `contracts.FileDeleteHandler` — File deletion callbacks
- `contracts.QueueTrigger` — Download queue trigger

Add compile-time checks when implementing: `var _ contracts.Broadcaster = (*MyService)(nil)`
```

**Add new section: Dev Mode Switching**
```
## Dev Mode Switching

Dev mode toggles all services to a separate database. Managed by `DevModeManager` in
`internal/api/devmode.go`.

- `SwitchableServices` in `switchable.go` uses struct tags (`switchable:"db"` or
  `switchable:"queries"`) and reflection-based `UpdateAll()` to switch all registered services.
- Adding a new switchable service: add a field with the appropriate tag to `SwitchableServices`.
  The reflection loop handles the rest — no manual `SetDB` enumeration needed.
- `Validate()` checks for nil fields at startup. Forgetting to assign a tagged field is caught
  immediately, not silently at runtime.
```

### Bridge files reference

Add to internal docs for navigating adapter code:
```
### Adapter bridge files (`internal/api/`)
- `notification_bridge.go` — Notification event mapping adapters
- `portal_bridge.go` — Portal-specific interface bridges
- `history_bridge.go` — History event bridges
- `slot_bridge.go` — Slot operation bridges
```
