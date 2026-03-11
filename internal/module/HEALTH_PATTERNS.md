# Health Monitoring for Modules

The health monitoring system (`internal/health/`) is already fully generic.
Modules interact with it via the `contracts.HealthService` interface, which uses
string-based categories and IDs. No code changes are needed to support new modules.

## Interface

Modules receive `contracts.HealthService` via constructor injection:

```go
type HealthService interface {
    RegisterItemStr(category, id, name string)
    UnregisterItemStr(category, id string)
    SetErrorStr(category, id, message string)
    ClearStatusStr(category, id string)
    SetWarningStr(category, id, message string)
}
```

## Categories

Current categories (defined in `health/types.go`):
- `downloadClients` — download client connectivity
- `indexers` — indexer connectivity and search health
- `prowlarr` — Prowlarr integration health
- `rootFolders` — root folder accessibility
- `metadata` — metadata provider connectivity
- `storage` — disk space warnings
- `import` — import pipeline health

Modules may use existing categories or define new ones. New categories are
automatically created when `RegisterItemStr` is called with an unknown category.
However, the `GetAll()` response struct (`HealthResponse`) has hard-coded fields
per category — a future phase should make this dynamic (use a map instead of
named fields) when a 3rd module adds a new category.

## Usage Patterns

### Metadata provider health
```go
// On service startup
s.health.RegisterItemStr("metadata", "musicbrainz", "MusicBrainz")

// On API call failure
s.health.SetErrorStr("metadata", "musicbrainz", "API returned 503")

// On successful API call after failure
s.health.ClearStatusStr("metadata", "musicbrainz")
```

### Root folder health
```go
// Root folder health is checked by the storage health scheduled task
// (`scheduler/tasks/storagehealth.go`). It iterates all root folders and
// checks accessibility. No module-specific code needed — the root folder's
// `media_type` (module_type after Phase 1) associates it with a module.
```

### Import pipeline health
```go
// The import service uses dedicated convenience methods:
s.health.RegisterImportItem("music-import", "Music Import Pipeline")
s.health.SetImportError("music-import", "FFprobe not found")
s.health.ClearImportStatus("music-import")
```

## Notifications

Health status changes automatically trigger notifications via the
`NotificationDispatcher` interface. When a health item transitions from
OK → Error/Warning, `DispatchHealthIssue` is called. When it transitions
back to OK, `DispatchHealthRestored` is called. Modules don't need to handle
notification dispatch — the health service does it automatically.
