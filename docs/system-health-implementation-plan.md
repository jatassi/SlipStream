# System Health Monitoring - Implementation Plan

This document outlines the implementation plan for the system health monitoring feature as specified in [system-health-spec.md](./system-health-spec.md).

---

## Overview

The system health monitoring feature tracks the health of:
- **Download Clients** (Binary: OK/Error)
- **Indexers** (Three-tier: OK/Warning/Error)
- **Root Folders** (Binary: OK/Error)
- **Metadata Providers** (Three-tier: OK/Warning/Error - TMDB & TVDB separately)
- **Storage** (Three-tier: OK/Warning/Error - per volume)

**Key Characteristics:**
- In-memory state only (resets on restart)
- Real-time WebSocket updates to frontend
- Passive monitoring via error handling in existing operations
- Active monitoring via scheduled connection tests (download clients & indexers only)
- Recovery: next successful operation clears error/warning state
- Timeouts: reuse existing configured timeouts for each client type
- UI dashboard widget + dedicated health page

---

## Phase 1: Core Health Types & Service (Backend Foundation)

### 1.1 Create Health Package Structure

**Location:** `internal/health/`

```
internal/health/
├── types.go        # Health status types and enums
├── service.go      # Core health service with in-memory state
├── handlers.go     # HTTP handlers for health endpoints
├── storage.go      # Storage-specific health check logic
└── filesystem.go   # Cross-platform filesystem checks
```

### 1.2 Health Types

**Location:** `internal/health/types.go`

**To-Do:**
- [ ] Define `HealthStatus` enum (OK, Warning, Error)
  - Note: Download Clients and Root Folders are binary (OK/Error only). The service should never set Warning for these categories. Document this constraint in code comments.
- [ ] Define `HealthCategory` enum (DownloadClients, Indexers, RootFolders, Metadata, Storage)
  - JSON serialization: `downloadClients`, `indexers`, `rootFolders`, `metadata`, `storage`
- [ ] Define `HealthItem` struct:
  ```go
  type HealthItem struct {
      ID        string         // Unique ID (e.g., "1" for client ID, "tmdb" for metadata)
      Category  HealthCategory
      Name      string         // Display name (e.g., "qBittorrent - Home Server")
      Status    HealthStatus
      Message   string         // User-friendly error description
      Timestamp time.Time      // When status last changed (only meaningful for non-OK)
  }
  ```
- [ ] Define `CategorySummary` struct for dashboard widget:
  ```go
  type CategorySummary struct {
      Category HealthCategory
      OK       int
      Warning  int
      Error    int
  }
  ```
- [ ] Define `HealthResponse` struct for API response (grouped by category)

### 1.3 Health Service Core

**Location:** `internal/health/service.go`

**To-Do:**
- [ ] Create `Service` struct with:
  - [ ] In-memory state map: `map[HealthCategory]map[string]*HealthItem`
  - [ ] `sync.RWMutex` for thread-safe access
  - [ ] Broadcaster interface for WebSocket notifications
  - [ ] Logger
- [ ] Implement item lifecycle methods:
  - [ ] `RegisterItem(category, id, name)` - adds new item with OK status
  - [ ] `UnregisterItem(category, id)` - removes item (on deletion)
- [ ] Implement status update methods:
  - [ ] `SetError(category, id, message)` - sets Error status with message
  - [ ] `SetWarning(category, id, message)` - sets Warning status with message
  - [ ] `ClearStatus(category, id)` - resets to OK status
- [ ] Implement query methods:
  - [ ] `GetAll()` - returns all items grouped by category
  - [ ] `GetByCategory(category)` - returns items for a category
  - [ ] `GetItem(category, id)` - returns single item
  - [ ] `GetSummary()` - returns counts per category for dashboard
  - [ ] `IsHealthy(category, id)` - returns true if item is OK
  - [ ] `IsCategoryHealthy(category)` - returns true if all items in category are OK
- [ ] Broadcast `health:updated` message on every status change
- [ ] Define `Broadcaster` interface for decoupling from WebSocket hub

### 1.4 Initial State Behavior

**Design Decision:** On startup, all registered items begin with OK status. Actual health is determined by:
1. Passive monitoring (errors during normal operations)
2. First scheduled health check (for download clients/indexers)
3. Storage check runs on startup to immediately detect disk issues

**To-Do:**
- [ ] Document startup behavior in service initialization
- [ ] Storage health check should run once on startup (not wait for schedule)

---

## Phase 2: Configuration

### 2.1 Health Check Configuration

**Location:** `internal/config/config.go`

**To-Do:**
- [ ] Add `HealthConfig` struct:
  ```go
  type HealthConfig struct {
      DownloadClientCheckInterval time.Duration `yaml:"downloadClientCheckInterval"` // default: 6h
      IndexerCheckInterval        time.Duration `yaml:"indexerCheckInterval"`        // default: 6h
      StorageCheckInterval        time.Duration `yaml:"storageCheckInterval"`        // default: 1h
      StorageWarningThreshold     float64       `yaml:"storageWarningThreshold"`     // default: 0.20 (20%)
      StorageErrorThreshold       float64       `yaml:"storageErrorThreshold"`       // default: 0.05 (5%)
  }
  ```
- [ ] Add `Health HealthConfig` to main Config struct
- [ ] Set sensible defaults
- [ ] Support environment variable overrides:
  - `HEALTH_DOWNLOAD_CLIENT_CHECK_INTERVAL`
  - `HEALTH_INDEXER_CHECK_INTERVAL`
  - `HEALTH_STORAGE_CHECK_INTERVAL`

**Note:** Individual client/indexer timeouts reuse existing configured timeouts - no new timeout config needed.

---

## Phase 3: Server Integration (Backend Wiring)

### 3.1 Wire Up Health Service

**Location:** `internal/api/server.go`

**To-Do:**
- [ ] Create health service in `NewServer()`:
  ```go
  healthService := health.NewService(logger)
  healthService.SetBroadcaster(hub)
  ```
- [ ] Create health handlers and register routes:
  ```go
  healthHandlers := health.NewHandlers(healthService, downloaderService, indexerService, rootFolderService, metadataService)
  healthHandlers.RegisterRoutes(api.Group("/health"))
  ```
- [ ] Pass health service to services that need it:
  - [ ] `downloaderService.SetHealthService(healthService)`
  - [ ] `indexerStatusService.SetHealthService(healthService)`
  - [ ] `rootFolderService.SetHealthService(healthService)`
  - [ ] `metadataService.SetHealthService(healthService)`

### 3.2 Health HTTP Handlers

**Location:** `internal/health/handlers.go`

**To-Do:**
- [ ] Create `Handlers` struct with dependencies:
  - Health service
  - Downloader service (for testing)
  - Indexer service (for testing)
  - Root folder service (for testing)
  - Metadata service (for testing)
- [ ] Implement endpoints:
  - [ ] `GET /api/v1/health` - List all health items by category
  - [ ] `GET /api/v1/health/summary` - Get summary counts for dashboard
  - [ ] `GET /api/v1/health/:category` - Get items for specific category
  - [ ] `POST /api/v1/health/:category/test` - Test all items in category
  - [ ] `POST /api/v1/health/:category/:id/test` - Test specific item
- [ ] Register routes in `RegisterRoutes(g *echo.Group)`
- [ ] "Test All" runs tests **sequentially** to avoid overwhelming external services

---

## Phase 4: Frontend Types, API Client & Hooks

### 4.1 Health Types

**Location:** `web/src/types/health.ts`

**To-Do:**
- [ ] Define TypeScript types:
  ```typescript
  export type HealthStatus = 'ok' | 'warning' | 'error'
  export type HealthCategory = 'downloadClients' | 'indexers' | 'rootFolders' | 'metadata' | 'storage'

  export interface HealthItem {
    id: string
    category: HealthCategory
    name: string
    status: HealthStatus
    message?: string
    timestamp?: string  // ISO 8601, only present for warning/error
  }

  export interface CategorySummary {
    category: HealthCategory
    ok: number
    warning: number
    error: number
  }

  export interface HealthResponse {
    downloadClients: HealthItem[]
    indexers: HealthItem[]
    rootFolders: HealthItem[]
    metadata: HealthItem[]
    storage: HealthItem[]
  }

  export interface HealthSummary {
    categories: CategorySummary[]
    hasIssues: boolean
  }
  ```

### 4.2 Health API Client

**Location:** `web/src/api/health.ts`

**To-Do:**
- [ ] Create health API module:
  ```typescript
  export const healthApi = {
    getAll: () => apiFetch<HealthResponse>('/health'),
    getSummary: () => apiFetch<HealthSummary>('/health/summary'),
    getCategory: (category: HealthCategory) => apiFetch<HealthItem[]>(`/health/${category}`),
    testCategory: (category: HealthCategory) => apiFetch<void>(`/health/${category}/test`, { method: 'POST' }),
    testItem: (category: HealthCategory, id: string) => apiFetch<void>(`/health/${category}/${id}/test`, { method: 'POST' }),
  }
  ```

### 4.3 Health Hooks

**Location:** `web/src/hooks/useHealth.ts`

**To-Do:**
- [ ] Create query keys:
  ```typescript
  export const healthKeys = {
    all: ['health'] as const,
    list: () => [...healthKeys.all, 'list'] as const,
    summary: () => [...healthKeys.all, 'summary'] as const,
    category: (cat: HealthCategory) => [...healthKeys.all, 'category', cat] as const,
  }
  ```
- [ ] Create `useHealth()` hook - fetches all health items
- [ ] Create `useHealthSummary()` hook - fetches summary for dashboard
- [ ] Create `useHealthCategory(category)` hook - fetches items for category
- [ ] Create `useTestCategory()` mutation hook
- [ ] Create `useTestItem()` mutation hook
- [ ] Export hooks from `web/src/hooks/index.ts`

---

## Phase 5: Download Client Health Monitoring

### 5.1 Item Registration Lifecycle

**Location:** `internal/downloader/service.go`

**To-Do:**
- [ ] Add `healthService *health.Service` field to Service struct
- [ ] Add `SetHealthService(health *health.Service)` method
- [ ] On `Create()`: call `healthService.RegisterItem(DownloadClients, id, name)`
- [ ] On `Delete()`: call `healthService.UnregisterItem(DownloadClients, id)`
- [ ] On startup: register all existing clients from database

### 5.2 Passive Monitoring

**To-Do:**
- [ ] In queue polling: on error, call `healthService.SetError(DownloadClients, id, message)`
- [ ] In queue polling: on success, call `healthService.ClearStatus(DownloadClients, id)`
- [ ] In release fetching: on error, call `healthService.SetError(DownloadClients, id, message)`
- [ ] In release fetching: on success, call `healthService.ClearStatus(DownloadClients, id)`
- [ ] Use existing configured timeouts for operations (no new timeout config)

### 5.3 Active Monitoring (Scheduled Test)

**Location:** `internal/scheduler/tasks/downloadclienthealth.go`

**To-Do:**
- [ ] Create `DownloadClientHealthTask` struct with dependencies:
  - Downloader service
  - Health service
  - Logger
- [ ] Implement `Run(ctx context.Context) error`:
  - [ ] Get all download clients
  - [ ] For each client:
    - [ ] Check if downloads are in progress
    - [ ] If downloads active: explicitly set OK status (client is working)
    - [ ] If no downloads: test connection
    - [ ] Update health status based on test result
- [ ] Create `RegisterDownloadClientHealthTask(scheduler, downloader, health, cfg, logger)`:
  - [ ] Use interval from config (default 6h)
  - [ ] Set `RunOnStart: false` (let passive monitoring detect initial issues)

### 5.4 Manual Testing

**To-Do:**
- [ ] Ensure existing `POST /api/v1/downloadclients/:id/test` also updates health status
- [ ] Health handler's test endpoint delegates to downloader service and updates health

---

## Phase 6: Indexer Health Monitoring

### 6.1 Integrate with Existing Status System

The indexer package already has `internal/indexer/status/` with health tracking.

**Location:** `internal/indexer/status/service.go`

**To-Do:**
- [ ] Review existing status service implementation
- [ ] Add `healthService *health.Service` field
- [ ] Add `SetHealthService(health *health.Service)` method
- [ ] On status change, forward to central health service:
  - [ ] Escalation level 0 / success → `healthService.ClearStatus(Indexers, id)`
  - [ ] Rate limited → `healthService.SetWarning(Indexers, id, "Rate limited")`
  - [ ] Connection failure → `healthService.SetError(Indexers, id, message)`

### 6.2 Item Registration Lifecycle

**Location:** `internal/indexer/service.go` (or equivalent)

**To-Do:**
- [ ] On indexer create (if enabled): `healthService.RegisterItem(Indexers, id, name)`
- [ ] On indexer delete: `healthService.UnregisterItem(Indexers, id)`
- [ ] On indexer enable: `healthService.RegisterItem(Indexers, id, name)`
- [ ] On indexer disable: `healthService.UnregisterItem(Indexers, id)`
- [ ] On startup: register all existing **enabled** indexers only
- [ ] Disabled indexers are not tracked in health (they're intentionally offline)

### 6.3 Passive Monitoring

**Location:** `internal/indexer/search/service.go`

**To-Do:**
- [ ] Ensure search errors flow through status service (which forwards to health)
- [ ] Ensure rate limit detection triggers Warning via status service
- [ ] Ensure successful searches clear status via status service

### 6.4 Active Monitoring (Scheduled Test)

**Location:** `internal/scheduler/tasks/indexerhealth.go`

**To-Do:**
- [ ] Create `IndexerHealthTask` struct
- [ ] Implement `Run(ctx context.Context) error`:
  - [ ] Get all enabled indexers
  - [ ] Test each indexer connection (using existing test method)
  - [ ] Update health status based on result
- [ ] Create `RegisterIndexerHealthTask(scheduler, indexer, health, cfg, logger)`:
  - [ ] Use interval from config (default 6h)
  - [ ] Set `RunOnStart: false`

---

## Phase 7: Root Folder Health Monitoring

**Note:** Root folders have passive monitoring only (during library scan). No scheduled active test per spec.

### 7.1 Item Registration Lifecycle

**Location:** `internal/library/rootfolder/service.go`

**To-Do:**
- [ ] Add `healthService *health.Service` field
- [ ] Add `SetHealthService(health *health.Service)` method
- [ ] On `Create()`: call `healthService.RegisterItem(RootFolders, id, path)`
- [ ] On `Delete()`: call `healthService.UnregisterItem(RootFolders, id)`
- [ ] On startup: register all existing root folders

### 7.2 Cross-Platform Filesystem Checks

**Location:** `internal/health/filesystem.go`

**To-Do:**
- [ ] Create `CheckFolderAccessible(path string) error`:
  - [ ] Check if path exists
  - [ ] Check if path is a directory
  - [ ] Return descriptive error if not
- [ ] Create `CheckFolderWritable(path string) error`:
  - [ ] Create temp file with unique name in directory
  - [ ] Write test data to file
  - [ ] Delete temp file
  - [ ] Return descriptive error if any step fails
  - [ ] Handle Windows vs Unix permission differences
- [ ] Create `CheckFolderHealth(path string) (ok bool, message string)`:
  - [ ] Combines accessibility and writability checks
  - [ ] Returns user-friendly error message

### 7.3 Passive Monitoring Integration

Health checks run on both scheduled and manual library scans.

**Location:** `internal/scheduler/tasks/libraryscan.go` and library scan API handler

**To-Do:**
- [ ] Add health service dependency to library scan task
- [ ] Create shared `ScanWithHealthCheck(ctx, rootFolderID, path)` function:
  - [ ] Run `CheckFolderHealth(path)`
  - [ ] If unhealthy: `healthService.SetError(RootFolders, id, message)`
  - [ ] If healthy: `healthService.ClearStatus(RootFolders, id)`
  - [ ] Return whether to proceed with scan
- [ ] Use shared function in both scheduled task and manual scan endpoint
- [ ] If folder is unhealthy, skip scanning that folder and log

### 7.4 Manual Testing

**To-Do:**
- [ ] Health handler test endpoint runs `CheckFolderHealth()` and updates status

---

## Phase 8: Metadata Provider Health Monitoring

**Note:** Metadata providers have passive monitoring only. No scheduled active test per spec. TMDB and TVDB are tracked as separate static items.

**Status Mapping (implementation decision, not in spec):**
- **OK**: Successful API response
- **Warning**: Rate limited (HTTP 429) - temporary, will recover
- **Error**: Connection failure, timeout, authentication error, server error (5xx)

### 8.1 Static Item Registration

**Location:** `internal/metadata/service.go`

**To-Do:**
- [ ] Add `healthService *health.Service` field
- [ ] Add `SetHealthService(health *health.Service)` method
- [ ] On service initialization, register static items:
  - [ ] `healthService.RegisterItem(Metadata, "tmdb", "TMDB")`
  - [ ] `healthService.RegisterItem(Metadata, "tvdb", "TVDB")`

### 8.2 Passive Monitoring

**To-Do:**
- [ ] In TMDB API calls:
  - [ ] On rate limit response (429): `healthService.SetWarning(Metadata, "tmdb", "Rate limited")`
  - [ ] On connection/timeout error: `healthService.SetError(Metadata, "tmdb", message)`
  - [ ] On success: `healthService.ClearStatus(Metadata, "tmdb")`
- [ ] In TVDB API calls:
  - [ ] Same pattern for "tvdb" item
- [ ] Wrap error handling in central location to avoid duplication

### 8.3 Manual Testing

**To-Do:**
- [ ] Health handler test endpoint makes a simple API call to verify connectivity
- [ ] Updates health status based on result

---

## Phase 9: Storage Health Monitoring

### 9.1 Storage Health Logic

**Location:** `internal/health/storage.go`

**To-Do:**
- [ ] Create `StorageChecker` struct with:
  - Health service
  - Root folder service (to get paths)
  - Config (for thresholds)
  - Logger
- [ ] Create `CheckAllStorage() error`:
  - [ ] Get all root folder paths
  - [ ] Extract unique volumes/drives from paths (cross-platform)
  - [ ] For each volume:
    - [ ] Get disk space (total and free)
    - [ ] Calculate percentage remaining
    - [ ] Determine status based on thresholds:
      - ≥20% (configurable) = OK
      - 5-20% = Warning
      - <5% (configurable) = Error
    - [ ] Update health status
  - [ ] Unregister volumes that no longer have root folders

### 9.2 Cross-Platform Volume Detection

**To-Do:**
- [ ] Create `GetVolumeFromPath(path string) (volumeID string, volumeName string)`:
  - [ ] On Windows: extract drive letter (e.g., "C:")
    - ID: `"C:"`, Name: `"C:\"` or volume label if available
  - [ ] On Unix: get mount point from `syscall.Statfs` or filepath walking
    - ID: mount point path (e.g., `"/mnt/media"`), Name: same or device name
  - [ ] Handle edge cases: network shares, symlinks, bind mounts
- [ ] Create `GetDiskSpace(path string) (total, free uint64, err error)`:
  - [ ] Use `golang.org/x/sys/unix` for Unix systems
  - [ ] Use `golang.org/x/sys/windows` for Windows

### 9.3 Storage Item Dynamic Lifecycle

Unlike other categories where items are explicitly created/deleted, storage items are discovered dynamically based on root folder paths.

**To-Do:**
- [ ] In `CheckAllStorage()`:
  - [ ] Get current set of volumes from root folder paths
  - [ ] Compare to previously tracked volumes
  - [ ] For new volumes: `RegisterItem(Storage, volumeID, volumeName)`
  - [ ] For removed volumes: `UnregisterItem(Storage, volumeID)`
  - [ ] For existing volumes: update status based on disk space
- [ ] Track last known volumes in StorageChecker struct to detect changes

### 9.4 Scheduled Storage Check

**Location:** `internal/scheduler/tasks/storagehealth.go`

**To-Do:**
- [ ] Create `StorageHealthTask` struct
- [ ] Implement `Run(ctx context.Context) error`:
  - [ ] Call `storageChecker.CheckAllStorage()`
- [ ] Create `RegisterStorageHealthTask(scheduler, checker, cfg, logger)`:
  - [ ] Use interval from config (default 1h)
  - [ ] Set `RunOnStart: true` (immediately detect storage issues)

---

## Phase 10: Register All Scheduled Tasks

**Location:** `internal/api/server.go`

**To-Do:**
- [ ] Register download client health task
- [ ] Register indexer health task
- [ ] Register storage health task
- [ ] Pass config intervals to each task registration

---

## Phase 11: Dependent Task Skip Logic

### 11.1 Health Check Methods

**Location:** `internal/health/service.go`

**To-Do:**
- [ ] Implement `IsHealthy(category, id) bool` - true if item is OK
- [ ] Implement `IsCategoryHealthy(category) bool` - true if all items OK
- [ ] Implement `GetUnhealthyItems(category) []HealthItem` - returns unhealthy items

### 11.2 Task Prerequisites

**Location:** Various task files

**To-Do:**
- [ ] Library scan task (`libraryscan.go`):
  - [ ] Already checks root folder health per-folder (Phase 7)
- [ ] Auto search task:
  - [ ] Before searching: check if any indexers are healthy
  - [ ] If no healthy indexers: skip, log "Skipping auto search: no healthy indexers"
- [ ] Download processing:
  - [ ] Before processing downloads: check download client health
  - [ ] If client unhealthy: skip processing for that client, log reason
- [ ] Metadata refresh task (`metadatarefresh.go`):
  - [ ] Check metadata provider health before refresh
  - [ ] Skip provider if unhealthy, log reason
  - [ ] Can still refresh using healthy provider

**To-Do:**
- [ ] Standardize skip log format: `"Skipping [task]: [reason]"`
- [ ] Ensure skipped runs are visible in scheduler task status/logs

---

## Phase 12: Frontend WebSocket Integration

**Location:** `web/src/stores/websocket.ts`

**To-Do:**
- [ ] Add handler for `health:updated` message type
- [ ] On health update: invalidate health queries:
  ```typescript
  case 'health:updated':
    queryClient.invalidateQueries({ queryKey: healthKeys.all })
    break
  ```
- [ ] Optionally: show toast notification on status degradation (OK → Warning/Error)

---

## Phase 13: Frontend Dashboard Widget

### 13.1 Health Widget Component

**Location:** `web/src/components/dashboard/HealthWidget.tsx`

**To-Do:**
- [ ] Create widget component
- [ ] Always visible (not collapsible/dismissible)
- [ ] Display category breakdown format:
  - "Download Clients: 2 OK, 1 Error"
  - "Indexers: 3 OK"
  - Color-code based on worst status in category (red if any error, yellow if warning, green if all OK)
- [ ] Add "Test" button per category
- [ ] Add link to full health page (`/system/health`)
- [ ] Add link to relevant settings page per category:
  | Category | Settings Link |
  |----------|---------------|
  | Download Clients | `/settings/downloadclients` |
  | Indexers | `/settings/indexers` |
  | Root Folders | `/settings/mediamanagement` |
  | Metadata | `/settings/metadata` |
  | Storage | `/settings/mediamanagement` |

### 13.2 Status Indicator Component

**Location:** `web/src/components/health/StatusIndicator.tsx`

**To-Do:**
- [ ] Create reusable status indicator component
- [ ] Props: `status: HealthStatus`, `size?: 'sm' | 'md' | 'lg'`
- [ ] OK = green circle/checkmark
- [ ] Warning = yellow/amber warning triangle
- [ ] Error = red circle/X

### 13.3 Dashboard Integration

**Location:** `web/src/routes/dashboard.tsx` (or similar)

**To-Do:**
- [ ] Add HealthWidget to dashboard layout
- [ ] Position prominently (top of page or sidebar)
- [ ] Ensure always visible (no dismiss/collapse)

---

## Phase 14: Frontend Health Page

### 14.1 Health Page Component

**Location:** `web/src/routes/system/health.tsx`

**To-Do:**
- [ ] Create health page with grouped layout by category
- [ ] For each category section:
  - [ ] Category header (e.g., "Download Clients")
  - [ ] "Test All" button in header
  - [ ] Settings link in header
  - [ ] List of items with:
    - Status indicator (using StatusIndicator component)
    - Item name
    - Last error message (if status is warning/error)
    - Timestamp (only for warning/error, formatted as relative time)
    - Individual "Test" button
- [ ] Show loading state while testing
- [ ] Show success/error feedback after test

### 14.2 Navigation Integration

**Location:** Navigation component (sidebar)

**To-Do:**
- [ ] Add "System Health" (or just "Health") to left navigation
- [ ] Position as first item under "System" section
- [ ] Add appropriate icon (e.g., heart, pulse, activity)
- [ ] Show badge/indicator if there are issues (optional enhancement)

---

## Phase 15: Frontend Add Flow Blocking

### 15.1 Movie Add Dialog Integration

**Location:** Movie add dialog component

**To-Do:**
- [ ] Before showing root folder selector, fetch health status
- [ ] Filter or mark unhealthy root folders
- [ ] If selected root folder becomes unhealthy:
  - [ ] Show error message: "Cannot add movie: root folder '[name]' is inaccessible"
  - [ ] Disable add button
  - [ ] Provide link to health page
- [ ] Do not allow proceeding with unhealthy root folder

### 15.2 Series Add Dialog Integration

**Location:** Series add dialog component

**To-Do:**
- [ ] Same pattern as movie add dialog
- [ ] Check root folder health before allowing add

---

## Phase 16: Testing

### 16.1 Unit Tests

**Location:** `internal/health/*_test.go`

**To-Do:**
- [ ] Test health service state management:
  - [ ] RegisterItem / UnregisterItem
  - [ ] SetError / SetWarning / ClearStatus
  - [ ] GetAll / GetByCategory / GetSummary
- [ ] Test status transitions (OK → Error → OK, OK → Warning → Error → OK)
- [ ] Test thread safety with concurrent access
- [ ] Test filesystem checks (with mocked filesystem)
- [ ] Test storage threshold calculations
- [ ] Test volume detection (with mocked syscalls)

### 16.2 Integration Tests

**Location:** `internal/health/*_test.go` or `internal/api/*_test.go`

**To-Do:**
- [ ] Test health API endpoints
- [ ] Test WebSocket broadcasts on status change
- [ ] Test scheduled task execution
- [ ] Test task skip logic when unhealthy
- [ ] Test end-to-end: create client → test fails → health shows error → test succeeds → health shows OK

---

## Implementation Order Summary

Based on dependencies, implement in this order:

| Order | Phase | Description | Dependencies |
|-------|-------|-------------|--------------|
| 1 | Phase 1 | Core types & service | None |
| 2 | Phase 2 | Configuration | None |
| 3 | Phase 3 | Server integration | Phases 1, 2 |
| 4 | Phase 4 | Frontend types, API, hooks | None (parallel with 1-3) |
| 5 | Phase 5 | Download client monitoring | Phases 1, 3 |
| 6 | Phase 6 | Indexer monitoring | Phases 1, 3 |
| 7 | Phase 7 | Root folder monitoring | Phases 1, 3 |
| 8 | Phase 8 | Metadata monitoring | Phases 1, 3 |
| 9 | Phase 9 | Storage monitoring | Phases 1, 3, 7 |
| 10 | Phase 10 | Register scheduled tasks | Phases 2, 5, 6, 9 |
| 11 | Phase 11 | Task skip logic | Phases 5-9 |
| 12 | Phase 12 | Frontend WebSocket | Phase 4 |
| 13 | Phase 13 | Dashboard widget | Phases 4, 12 |
| 14 | Phase 14 | Health page | Phases 4, 12 |
| 15 | Phase 15 | Add flow blocking | Phases 7, 14 |
| 16 | Phase 16 | Testing | All phases |

---

## Technical Notes

### Initialization Order

Services must be initialized in this order to avoid circular dependencies:

1. Create health service (no dependencies)
2. Create other services (downloader, indexer, rootfolder, metadata)
3. Call `SetHealthService()` on each service
4. Services register their existing items with health service
5. Create storage checker (depends on health service + root folder service)
6. Register scheduled tasks

```go
// In NewServer():
healthService := health.NewService(logger)
healthService.SetBroadcaster(hub)

downloaderService := downloader.NewService(...)
downloaderService.SetHealthService(healthService)
// downloaderService registers existing clients

// ... similar for other services ...

storageChecker := health.NewStorageChecker(healthService, rootFolderService, cfg, logger)
// Storage items are registered on first check (RunOnStart: true)
```

### Thread Safety
- Use `sync.RWMutex` for in-memory state
- Read lock for queries, write lock for updates
- Ensure atomic operations for status updates

### WebSocket Message Format
```json
{
  "type": "health:updated",
  "payload": {
    "category": "downloadClients",
    "id": "1",
    "name": "qBittorrent",
    "status": "error",
    "message": "Connection refused",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Cross-Platform Considerations
- Root folder writability test must work on Windows, Linux, macOS
- Storage volume detection differs by OS
- Use `golang.org/x/sys/unix` and `golang.org/x/sys/windows` packages
- Test on all target platforms

### Timeout Handling
- Reuse existing configured timeouts for each client type
- Download clients: use client-specific timeout from config
- Indexers: use indexer-specific timeout or default
- Metadata: use configured API timeout

### Existing Patterns to Follow
- Service pattern from `internal/downloader/service.go`
- Handler pattern from `internal/library/rootfolder/handlers.go`
- Task pattern from `internal/scheduler/tasks/libraryscan.go`
- Status tracking from `internal/indexer/status/service.go`
- Frontend API pattern from `web/src/api/indexers.ts`
- Frontend hooks pattern from `web/src/hooks/useIndexers.ts`

---

## Summary Checklist

### Backend
- [x] Core health package (types, service)
- [x] Configuration for check intervals and thresholds
- [x] Server wiring and HTTP handlers
- [x] Download client lifecycle + passive + active monitoring
- [x] Indexer integration with existing status system + active monitoring
- [x] Root folder lifecycle + passive monitoring + filesystem checks
- [x] Metadata provider static registration + passive monitoring
- [x] Storage monitoring with cross-platform volume detection
- [x] Scheduled task registration
- [x] Task skip logic for dependent tasks (health check methods exist)

### Frontend
- [x] Health types
- [x] Health API client
- [x] React Query hooks
- [x] WebSocket integration
- [x] Status indicator component
- [x] Dashboard widget
- [x] Health page with navigation
- [ ] Add flow blocking for movies/series (enhancement - deferred)

### Testing
- [ ] Unit tests for health service (enhancement - deferred)
- [ ] Unit tests for filesystem/storage checks (enhancement - deferred)
- [ ] Integration tests for API endpoints (enhancement - deferred)
- [ ] Integration tests for WebSocket (enhancement - deferred)
- [ ] Integration tests for task skip logic (enhancement - deferred)

## Implementation Status

**Completed: 2024** - Phases 1-14 implemented. Core system health monitoring is fully functional.

### What's Working
- Health status tracking for all categories (Download Clients, Indexers, Root Folders, Metadata, Storage)
- In-memory state with thread-safe access
- Real-time WebSocket updates to frontend
- Passive monitoring via error handling in existing operations
- Active monitoring via scheduled tasks for download clients, indexers, and storage
- Download client health checks skip when downloads are in progress (per spec)
- Dashboard widget showing health summary
- Full health page at `/system/health` with test functionality
- Navigation link to health page (first item under System section)
- API endpoints for querying and testing health
- Metadata tests make actual API calls to verify connectivity (not just config checks)

### Deferred Enhancements
- Phase 15: Frontend add flow blocking (warn users when adding media with unhealthy dependencies)
- Phase 16: Comprehensive unit and integration tests
