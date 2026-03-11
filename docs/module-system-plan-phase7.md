## Phase 7: Notifications & Real-Time ✅ COMPLETED

**Goal:** Replace hard-coded notification event columns with a module-declared event catalog stored as JSON. Add structured entity fields to WebSocket broadcasts and a frontend registry for module-aware query invalidation.

**Spec sections covered:** §9.1 (notification event catalog, JSON toggles, grouped UI), §10.1 (generic entity event shape, `BroadcastEntity` helper, frontend invalidation registry)

### Phase 7 Context Management

**Parallelism map** (tasks with no dependency arrow can run concurrently):

```
Group A (parallel):  7.1, 7.3, 7.9
Group B (after A):   7.2 (needs 7.1), 7.4 (needs 7.3)
Group C (after B):   7.5 (needs 7.4), 7.10 (needs 7.9)
Group D (after C):   7.6 (needs 7.5), 7.11 (needs 7.10)
Group E (after D):   7.7 + 7.8 (parallel, both need 7.6)
Group F (after E):   7.12 (needs 7.8, 7.11)
Group G:             7.13 (validation — needs all)
```

**Delegation notes:**
- Tasks 7.3 + 7.4 (migration + sqlc) should be one subagent — the sqlc queries depend on the new schema.
- Tasks 7.5 + 7.6 (service types + dispatch) should be one subagent — they touch the same files.
- Tasks 7.7 + 7.8 can be two parallel subagents (bridges vs API handler).
- Task 7.12 (frontend) is large — dedicate a single subagent.
- Tasks 7.9 + 7.10 + 7.11 (all WebSocket) can be one subagent if run sequentially, or split 7.9 as standalone and 7.10+7.11 as a pair.

**Validation checkpoints:** After Groups B, D, and F, run `go build ./...` to catch compile errors early. Full validation in 7.13.

---

### Phase 7 Architecture Decisions

1. **Event toggle storage**: Replace the 10 hard-coded boolean columns (`on_grab`, `on_import`, `on_upgrade`, `on_movie_added`, `on_movie_deleted`, `on_series_added`, `on_series_deleted`, `on_health_issue`, `on_health_restored`, `on_app_update`) on the `notifications` table with a single `event_toggles TEXT NOT NULL DEFAULT '{}'` column storing a JSON object `{"event_id": true/false}`. This makes the event set extensible — new modules add new event IDs without schema changes.

2. **Event ID convention**: Framework events use flat underscore strings (`grab`, `import`, `upgrade`, `health_issue`, `health_restored`, `app_update`). Module events use `module_type:event` format (`movie:added`, `movie:deleted`, `tv:added`, `tv:deleted`). The colon separator distinguishes module events from framework events and enables grouping by module in the UI.

   **Migration mapping** (old column → new JSON key):
   | Old Column | New Event ID |
   |---|---|
   | `on_grab` | `grab` |
   | `on_import` (was `on_download`, renamed in migration 047) | `import` |
   | `on_upgrade` | `upgrade` |
   | `on_movie_added` | `movie:added` |
   | `on_movie_deleted` | `movie:deleted` |
   | `on_series_added` | `tv:added` |
   | `on_series_deleted` | `tv:deleted` |
   | `on_health_issue` | `health_issue` |
   | `on_health_restored` | `health_restored` |
   | `on_app_update` | `app_update` |

3. **Notifier interface unchanged in Phase 7**: The `Notifier` interface in `internal/notification/types/types.go` keeps all 10 typed methods (`OnGrab`, `OnImport`, `OnMovieAdded`, etc.). The service routes events to these typed methods using the event ID string. This preserves all existing notifier implementations (Discord, Telegram, Slack, etc.) without modification. A generic `OnModuleEvent` method for events that don't map to typed methods will be added in a future phase when a 3rd module introduces new event types.

4. **`include_health_warnings` stays as a column**: It's a behavior modifier for `health_issue` dispatch, not an event toggle. Keeping it as a dedicated column avoids mixing configuration options into the event toggles JSON.

5. **Event catalog API**: A new `GET /api/v1/notifications/events` endpoint returns the full event catalog grouped by source (framework and each enabled module). The frontend renders this dynamically, replacing the hardcoded `adminEventTriggers` array. The handler queries the module `Registry` to collect module-declared events, prepending framework events.

6. **WebSocket entity events**: Entity lifecycle broadcasts (movie/series add, update, delete) gain structured fields (`module`, `entityType`, `entityId`, `action`) on the `Message` struct alongside the existing `type` field. The `type` field is preserved as `entityType:action` (e.g., `movie:updated`, `series:deleted`) for backward compatibility. Non-entity events (progress, queue, health, scheduler, autosearch, etc.) are unchanged — their new fields are omitted via `omitempty`.

7. **`BroadcastEntity` on the `Broadcaster` interface**: A new method `BroadcastEntity(moduleType, entityType string, entityID int64, action string, payload any)` is added to `contracts.Broadcaster`. The Hub fills both the structured fields and the legacy `type` field. Entity broadcast call sites in movie/TV services, import pipeline, autosearch, and grab service migrate to `BroadcastEntity`. The existing `Broadcast(msgType, payload)` method remains for non-entity events.

8. **Frontend entity invalidation registry**: A data-driven registry maps module type → query key arrays for invalidation. The generic entity handler checks for the `module` field on incoming messages, looks up query keys in the registry, and invalidates them. This replaces the hardcoded `libraryHandler` that uses `type.startsWith('movie:')`. New modules register their keys without touching the handler code.

9. **Portal user notifications unchanged**: Portal user notification events (`on_available`, `on_approved`, `on_denied` on `user_notifications`) are request-lifecycle events, not module events. They remain as columns in this phase. The `portal_notifications` inbox table is also unchanged.

---

### Task 7.1: Define Notification Event Types in Module Framework ✅

**Create** `internal/module/notification_events.go`

Define the notification event types and the `NotificationEvents` interface:

```go
package module

// NotificationEvent describes a single event a module can emit.
type NotificationEvent struct {
    ID          string // e.g., "movie:added", "tv:deleted"
    Label       string // e.g., "Movie Added"
    Description string // e.g., "When a movie is added to the library"
}

// NotificationEventGroup is a labeled group of events for UI rendering.
type NotificationEventGroup struct {
    ID     string              // "framework" or module type ID
    Label  string              // "General" or module display name
    Events []NotificationEvent
}

// NotificationEvents is implemented by modules to declare their notification event catalog.
// (spec §9.1)
type NotificationEvents interface {
    DeclareNotificationEvents() []NotificationEvent
}
```

**Modify** `internal/module/interfaces.go` — add `NotificationEvents` to the required interface list comment (it was already declared in Phase 0's interface bundle but this file provides the concrete definition).

**Create** `internal/module/framework_events.go`

Define the framework's own notification events (media-agnostic events like grab, import, upgrade, health, app_update):

```go
package module

// Framework notification event IDs.
const (
    EventGrab           = "grab"
    EventImport         = "import"
    EventUpgrade        = "upgrade"
    EventHealthIssue    = "health_issue"
    EventHealthRestored = "health_restored"
    EventAppUpdate      = "app_update"
)

// FrameworkNotificationEvents returns the events declared by the framework itself
// (not tied to any specific module).
func FrameworkNotificationEvents() []NotificationEvent {
    return []NotificationEvent{
        {ID: EventGrab, Label: "On Grab", Description: "When a release is grabbed from an indexer"},
        {ID: EventImport, Label: "On Import", Description: "When a file is imported to the library"},
        {ID: EventUpgrade, Label: "On Upgrade", Description: "When a quality upgrade is imported"},
        {ID: EventHealthIssue, Label: "On Health Issue", Description: "When a health check fails"},
        {ID: EventHealthRestored, Label: "On Health Restored", Description: "When a health issue is resolved"},
        {ID: EventAppUpdate, Label: "On App Update", Description: "When the application is updated"},
    }
}
```

**Modify** `internal/module/registry.go` — add a method to collect the full event catalog:

```go
// CollectNotificationEvents returns all notification events grouped by source.
// Framework events come first, then each enabled module's events.
func (r *Registry) CollectNotificationEvents() []NotificationEventGroup {
    groups := []NotificationEventGroup{
        {ID: "framework", Label: "General", Events: FrameworkNotificationEvents()},
    }
    for _, mod := range r.modules {
        if ne, ok := mod.(NotificationEvents); ok {
            events := ne.DeclareNotificationEvents()
            if len(events) > 0 {
                groups = append(groups, NotificationEventGroup{
                    ID:     string(mod.ID()),
                    Label:  mod.Name(),
                    Events: events,
                })
            }
        }
    }
    return groups
}
```

Note: `r.modules` iterates in registration order (use a slice, not just the map, to ensure deterministic ordering — verify the Phase 0 `Registry` stores modules in order).

**Verify:**
- `go build ./internal/module/...` compiles
- No existing tests broken

---

### Task 7.2: Implement NotificationEvents on Movie and TV Modules ✅

**Modify** `internal/modules/movie/module.go` — add `NotificationEvents` implementation:

```go
// Movie module notification event IDs.
const (
    EventMovieAdded   = "movie:added"
    EventMovieDeleted = "movie:deleted"
)

func (m *Module) DeclareNotificationEvents() []module.NotificationEvent {
    return []module.NotificationEvent{
        {ID: EventMovieAdded, Label: "Movie Added", Description: "When a movie is added to the library"},
        {ID: EventMovieDeleted, Label: "Movie Deleted", Description: "When a movie is removed"},
    }
}
```

Add compile-time assertion: `var _ module.NotificationEvents = (*Module)(nil)`

**Modify** `internal/modules/tv/module.go` — add `NotificationEvents` implementation:

```go
// TV module notification event IDs.
const (
    EventTVAdded   = "tv:added"
    EventTVDeleted = "tv:deleted"
)

func (m *Module) DeclareNotificationEvents() []module.NotificationEvent {
    return []module.NotificationEvent{
        {ID: EventTVAdded, Label: "Series Added", Description: "When a series is added to the library"},
        {ID: EventTVDeleted, Label: "Series Deleted", Description: "When a series is removed"},
    }
}
```

Add compile-time assertion: `var _ module.NotificationEvents = (*Module)(nil)`

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go build ./internal/modules/tv/...` compiles

---

### Task 7.3: Database Migration — Event Toggles JSON Column ✅

**Create** `internal/database/migrations/0XX_notification_event_toggles.sql`

(Use the next available migration number after the highest existing one. As of this writing, the highest is 069.)

This migration replaces the 10 individual event boolean columns with a single `event_toggles` JSON column. SQLite requires table recreation to drop columns.

```sql
-- +goose Up
-- Replace individual event toggle columns with a JSON event_toggles column.
-- This makes the notification event set extensible for the module system.

-- Step 1: Create new table without event columns, with event_toggles JSON
CREATE TABLE notifications_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'discord', 'telegram', 'webhook', 'email', 'slack', 'pushover',
        'gotify', 'ntfy', 'apprise', 'pushbullet', 'join', 'prowl',
        'simplepush', 'signal', 'custom_script', 'mock', 'plex'
    )),
    enabled INTEGER NOT NULL DEFAULT 1,
    settings TEXT NOT NULL DEFAULT '{}',
    event_toggles TEXT NOT NULL DEFAULT '{}',
    include_health_warnings INTEGER NOT NULL DEFAULT 0,
    tags TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Copy data, converting boolean columns to JSON event_toggles
INSERT INTO notifications_new (
    id, name, type, enabled, settings, event_toggles,
    include_health_warnings, tags, created_at, updated_at
)
SELECT
    id, name, type, enabled, settings,
    json_object(
        'grab',             json(CASE WHEN on_grab = 1 THEN 'true' ELSE 'false' END),
        'import',           json(CASE WHEN on_import = 1 THEN 'true' ELSE 'false' END),
        'upgrade',          json(CASE WHEN on_upgrade = 1 THEN 'true' ELSE 'false' END),
        'movie:added',      json(CASE WHEN on_movie_added = 1 THEN 'true' ELSE 'false' END),
        'movie:deleted',    json(CASE WHEN on_movie_deleted = 1 THEN 'true' ELSE 'false' END),
        'tv:added',         json(CASE WHEN on_series_added = 1 THEN 'true' ELSE 'false' END),
        'tv:deleted',       json(CASE WHEN on_series_deleted = 1 THEN 'true' ELSE 'false' END),
        'health_issue',     json(CASE WHEN on_health_issue = 1 THEN 'true' ELSE 'false' END),
        'health_restored',  json(CASE WHEN on_health_restored = 1 THEN 'true' ELSE 'false' END),
        'app_update',       json(CASE WHEN on_app_update = 1 THEN 'true' ELSE 'false' END)
    ),
    include_health_warnings, tags, created_at, updated_at
FROM notifications;

-- Step 3: Drop old table and rename
DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;

-- notification_status FK (notification_id REFERENCES notifications(id) ON DELETE CASCADE)
-- survives because the id column is preserved and the table name is restored.
-- Goose runs with foreign_keys pragma off during migrations.


-- +goose Down
-- Reverse: recreate table with individual columns
CREATE TABLE notifications_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'discord', 'telegram', 'webhook', 'email', 'slack', 'pushover',
        'gotify', 'ntfy', 'apprise', 'pushbullet', 'join', 'prowl',
        'simplepush', 'signal', 'custom_script', 'mock', 'plex'
    )),
    enabled INTEGER NOT NULL DEFAULT 1,
    settings TEXT NOT NULL DEFAULT '{}',
    on_grab INTEGER NOT NULL DEFAULT 0,
    on_import INTEGER NOT NULL DEFAULT 0,
    on_upgrade INTEGER NOT NULL DEFAULT 0,
    on_movie_added INTEGER NOT NULL DEFAULT 0,
    on_movie_deleted INTEGER NOT NULL DEFAULT 0,
    on_series_added INTEGER NOT NULL DEFAULT 0,
    on_series_deleted INTEGER NOT NULL DEFAULT 0,
    on_health_issue INTEGER NOT NULL DEFAULT 0,
    on_health_restored INTEGER NOT NULL DEFAULT 0,
    on_app_update INTEGER NOT NULL DEFAULT 0,
    include_health_warnings INTEGER NOT NULL DEFAULT 0,
    tags TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO notifications_new (
    id, name, type, enabled, settings,
    on_grab, on_import, on_upgrade,
    on_movie_added, on_movie_deleted,
    on_series_added, on_series_deleted,
    on_health_issue, on_health_restored, on_app_update,
    include_health_warnings, tags, created_at, updated_at
)
SELECT
    id, name, type, enabled, settings,
    COALESCE(json_extract(event_toggles, '$.grab'), 0),
    COALESCE(json_extract(event_toggles, '$.import'), 0),
    COALESCE(json_extract(event_toggles, '$.upgrade'), 0),
    COALESCE(json_extract(event_toggles, '$."movie:added"'), 0),
    COALESCE(json_extract(event_toggles, '$."movie:deleted"'), 0),
    COALESCE(json_extract(event_toggles, '$."tv:added"'), 0),
    COALESCE(json_extract(event_toggles, '$."tv:deleted"'), 0),
    COALESCE(json_extract(event_toggles, '$.health_issue'), 0),
    COALESCE(json_extract(event_toggles, '$.health_restored'), 0),
    COALESCE(json_extract(event_toggles, '$.app_update'), 0),
    include_health_warnings, tags, created_at, updated_at
FROM notifications;

DROP TABLE notifications;
ALTER TABLE notifications_new RENAME TO notifications;
```

**Important notes for implementer:**
- Verify Goose migration runner disables `PRAGMA foreign_keys` around migrations (it does by default). The `notification_status` table has a FK to `notifications(id)` — the table recreation preserves IDs so the FK data remains valid after rename.
- The `json()` wrapper around `'true'`/`'false'` strings ensures proper JSON booleans in the output, not quoted strings.
- The down migration uses `json_extract` with quoted keys for event IDs containing colons (e.g., `$."movie:added"`).
- **Remove sqlc.yaml column overrides** for all 10 dropped event columns. Lines 58-77 of `sqlc.yaml` have `go_type: "bool"` overrides for `notifications.on_grab`, `notifications.on_import`, `notifications.on_upgrade`, `notifications.on_movie_added`, `notifications.on_movie_deleted`, `notifications.on_series_added`, `notifications.on_series_deleted`, `notifications.on_health_issue`, `notifications.on_health_restored`, and `notifications.on_app_update`. Remove all 10 — these columns no longer exist after the migration. Keep the overrides for `notifications.enabled` and `notifications.include_health_warnings` (those columns survive).

**Verify:**
- Create a test DB and run migrations up/down/up without errors
- After migration, `SELECT event_toggles FROM notifications` returns valid JSON with boolean values

---

### Task 7.4: Update sqlc Queries for JSON Event Storage ✅

**Modify** `internal/database/queries/notifications.sql`

Rewrite all queries to use the `event_toggles` column instead of the 10 individual columns:

```sql
-- name: GetNotification :one
SELECT * FROM notifications WHERE id = ? LIMIT 1;

-- name: ListNotifications :many
SELECT * FROM notifications ORDER BY name;

-- name: ListEnabledNotifications :many
SELECT * FROM notifications WHERE enabled = 1 ORDER BY name;

-- name: CreateNotification :one
INSERT INTO notifications (
    name, type, enabled, settings,
    event_toggles, include_health_warnings, tags
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateNotification :one
UPDATE notifications SET
    name = ?,
    type = ?,
    enabled = ?,
    settings = ?,
    event_toggles = ?,
    include_health_warnings = ?,
    tags = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteNotification :exec
DELETE FROM notifications WHERE id = ?;

-- name: CountNotifications :one
SELECT COUNT(*) FROM notifications;
```

The `ListNotificationsForEvent` query is no longer needed — event filtering is done in Go code by checking the JSON map. Remove it.

The status queries (`GetNotificationStatus`, `UpsertNotificationStatus`, `ClearNotificationStatus`, `ListDisabledNotifications`) are unchanged — they operate on `notification_status`, not `notifications`.

**Run:** `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`

**Verify after sqlc generate:**
- `internal/database/sqlc/models.go` — the `Notification` struct now has `EventToggles string` instead of 10 boolean fields. `IncludeHealthWarnings` remains as a bool.
- `internal/database/sqlc/notifications.sql.go` — `CreateNotificationParams` has `EventToggles string` instead of 10 booleans. Same for `UpdateNotificationParams`.
- `go build ./internal/database/sqlc/...` compiles

---

### Task 7.5: Update Notification Types and Event Constants ✅ ✅

**Modify** `internal/notification/events.go`

Update event ID constants to use the new convention. Import constants from the module framework where applicable:

```go
package notification

import (
    "github.com/slipstream/slipstream/internal/module"
    moviemod "github.com/slipstream/slipstream/internal/modules/movie"
    tvmod "github.com/slipstream/slipstream/internal/modules/tv"
)

// EventType identifies the type of notification event.
type EventType = string

// Framework event IDs (re-exported from module package for convenience).
const (
    EventGrab           EventType = module.EventGrab
    EventImport         EventType = module.EventImport
    EventUpgrade        EventType = module.EventUpgrade
    EventHealthIssue    EventType = module.EventHealthIssue
    EventHealthRestored EventType = module.EventHealthRestored
    EventAppUpdate      EventType = module.EventAppUpdate
)

// Module event IDs (re-exported from module packages for convenience).
const (
    EventMovieAdded    EventType = moviemod.EventMovieAdded
    EventMovieDeleted  EventType = moviemod.EventMovieDeleted
    EventSeriesAdded   EventType = tvmod.EventTVAdded
    EventSeriesDeleted EventType = tvmod.EventTVDeleted
)
```

Note: `EventType` is changed to a type alias (`= string`) rather than a named type. This allows seamless use of string event IDs from modules without casting. The constant names (`EventSeriesAdded`, etc.) are preserved for use in the notification service dispatch, even though the underlying values change.

**Modify** `internal/notification/types/types.go`

Remove the duplicate `EventType` definition and constants (lines 54-68). These are now defined in `internal/module/framework_events.go` and the module packages. Keep all the event struct types (`GrabEvent`, `ImportEvent`, etc.) unchanged.

**Modify** `internal/notification/types.go`

Replace the 10 boolean event fields on `Config`, `CreateInput`, and `UpdateInput` with a `map[string]bool`:

```go
// Config represents a notification configuration stored in the database.
type Config struct {
    ID       int64           `json:"id"`
    Name     string          `json:"name"`
    Type     NotifierType    `json:"type"`
    Enabled  bool            `json:"enabled"`
    Settings json.RawMessage `json:"settings"`

    EventToggles          map[string]bool `json:"eventToggles"`
    IncludeHealthWarnings bool            `json:"includeHealthWarnings"`
    Tags                  []int64         `json:"tags,omitempty"`

    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

// CreateInput is used when creating a new notification.
type CreateInput struct {
    Name     string          `json:"name"`
    Type     NotifierType    `json:"type"`
    Enabled  bool            `json:"enabled"`
    Settings json.RawMessage `json:"settings"`

    EventToggles          map[string]bool `json:"eventToggles"`
    IncludeHealthWarnings bool            `json:"includeHealthWarnings"`
    Tags                  []int64         `json:"tags,omitempty"`
}

// UpdateInput is used when updating an existing notification.
type UpdateInput struct {
    Name     *string          `json:"name,omitempty"`
    Type     *NotifierType    `json:"type,omitempty"`
    Enabled  *bool            `json:"enabled,omitempty"`
    Settings *json.RawMessage `json:"settings,omitempty"`

    EventToggles          map[string]bool `json:"eventToggles,omitempty"`
    IncludeHealthWarnings *bool           `json:"includeHealthWarnings,omitempty"`
    Tags                  *[]int64        `json:"tags,omitempty"`
}
```

Note on `UpdateInput.EventToggles`: This is `map[string]bool` (not a pointer). On update, the map is **merged** — keys present in the input override existing keys; keys absent in the input retain their existing value. This matches the PATCH-style semantics the service already uses (see `mergeFlag` pattern). An empty map `{}` means "no changes to toggles"; to explicitly disable an event, send `{"eventId": false}`.

**Verify:**
- `go build ./internal/notification/...` — should fail at this point because `service.go` still references the old fields. This is expected and fixed in Task 7.6.

---

### Task 7.6: Update Notification Service Dispatch ✅ ✅

**Modify** `internal/notification/service.go`

Update `rowToConfig` to parse `event_toggles` JSON from the database row:

```go
func (s *Service) rowToConfig(row *sqlc.Notification) Config {
    var tags []int64
    if err := json.Unmarshal([]byte(row.Tags), &tags); err != nil {
        s.logger.Warn().Err(err).Int64("notificationID", row.ID).Msg("Failed to unmarshal notification tags")
    }

    eventToggles := make(map[string]bool)
    if err := json.Unmarshal([]byte(row.EventToggles), &eventToggles); err != nil {
        s.logger.Warn().Err(err).Int64("notificationID", row.ID).Msg("Failed to unmarshal event toggles")
    }

    return Config{
        ID:                    row.ID,
        Name:                  row.Name,
        Type:                  NotifierType(row.Type),
        Enabled:               row.Enabled,
        Settings:              json.RawMessage(row.Settings),
        EventToggles:          eventToggles,
        IncludeHealthWarnings: row.IncludeHealthWarnings,
        Tags:                  tags,
        CreatedAt:             row.CreatedAt,
        UpdatedAt:             row.UpdatedAt,
    }
}
```

Update `configSubscribesToEvent` to use the JSON map:

```go
func (s *Service) configSubscribesToEvent(cfg *Config, eventType EventType) bool {
    return cfg.EventToggles[eventType]
}
```

Update `Create` to serialize `EventToggles` to JSON for the DB:

```go
func (s *Service) Create(ctx context.Context, input *CreateInput) (*Config, error) {
    if _, ok := GetSchema(input.Type); !ok {
        return nil, errors.New("unsupported notification type")
    }

    settings := input.Settings
    if settings == nil {
        settings = json.RawMessage("{}")
    }

    tags, _ := json.Marshal(input.Tags)
    eventToggles, _ := json.Marshal(input.EventToggles)
    if input.EventToggles == nil {
        eventToggles = []byte("{}")
    }

    row, err := s.queries.CreateNotification(ctx, sqlc.CreateNotificationParams{
        Name:                  input.Name,
        Type:                  string(input.Type),
        Enabled:               input.Enabled,
        Settings:              string(settings),
        EventToggles:          string(eventToggles),
        IncludeHealthWarnings: input.IncludeHealthWarnings,
        Tags:                  string(tags),
    })
    if err != nil {
        return nil, err
    }

    cfg := s.rowToConfig(row)
    return &cfg, nil
}
```

Update `buildUpdateParams` — replace the 10 `mergeFlag` calls with a map merge:

```go
func (s *Service) buildUpdateParams(existing *Config, input *UpdateInput, id int64) sqlc.UpdateNotificationParams {
    name := existing.Name
    if input.Name != nil {
        name = *input.Name
    }

    notifType := existing.Type
    if input.Type != nil {
        notifType = *input.Type
    }

    enabled := existing.Enabled
    if input.Enabled != nil {
        enabled = *input.Enabled
    }

    settings := existing.Settings
    if input.Settings != nil {
        settings = *input.Settings
    }

    // Merge event toggles: input overrides existing keys, preserves unmentioned keys
    mergedToggles := make(map[string]bool, len(existing.EventToggles))
    for k, v := range existing.EventToggles {
        mergedToggles[k] = v
    }
    for k, v := range input.EventToggles {
        mergedToggles[k] = v
    }

    ihw := existing.IncludeHealthWarnings
    if input.IncludeHealthWarnings != nil {
        ihw = *input.IncludeHealthWarnings
    }

    tags := existing.Tags
    if input.Tags != nil {
        tags = *input.Tags
    }
    tagsJSON, _ := json.Marshal(tags)
    togglesJSON, _ := json.Marshal(mergedToggles)

    return sqlc.UpdateNotificationParams{
        Name:                  name,
        Type:                  string(notifType),
        Enabled:               enabled,
        Settings:              string(settings),
        EventToggles:          string(togglesJSON),
        IncludeHealthWarnings: ihw,
        Tags:                  string(tagsJSON),
        ID:                    id,
    }
}
```

Remove the `mergeFlag` helper method — it's no longer used.

The `dispatchToNotifier` and `dispatchRemainingEvents` methods remain unchanged — they still route by EventType constant to the typed Notifier methods. The only difference is the constant values have changed (e.g., `EventMovieAdded` is now `"movie:added"` instead of `"movie_added"`), but since the routing uses constants, not hardcoded strings, this is transparent.

**Verify:**
- `go build ./internal/notification/...` compiles
- `go vet ./internal/notification/...` passes

---

### Task 7.7: Update Notification Bridges ✅ ✅

**Modify** `internal/api/notification_bridge.go`

The bridges adapt movie/TV services to the notification service. The only change: ensure the event type constants used in `Dispatch` calls match the new values. Since bridges call convenience methods (`DispatchMovieAdded`, `DispatchSeriesAdded`, etc.) that internally use the updated constants from Task 7.5, **no code changes are needed in the bridge file itself**.

However, verify the bridge compiles by checking that:
- `notification.EventGrab` (used in `grabNotificationAdapter.OnGrab`) resolves to `"grab"`
- The typed convenience methods (`DispatchMovieAdded`, etc.) still exist on the service

If any bridge file directly references `notification.EventMovieAdded` or similar (rather than using convenience methods), update those references. Based on the current code, only `grabNotificationAdapter.OnGrab` calls `Dispatch` directly (using `notification.EventGrab`), and the rest use convenience methods — so no changes expected.

**Verify:**
- `go build ./internal/api/...` compiles

---

### Task 7.8: Add Notification Events Catalog API Endpoint ✅ ✅

**Modify** `internal/notification/handlers.go`

Add a new handler method for the events catalog. The handler needs access to the module registry. Add the registry as a field on the `Handlers` struct:

```go
// Handlers provides HTTP handlers for notification management
type Handlers struct {
    service  *Service
    registry *module.Registry // add this field
}
```

Update the constructor to accept `*module.Registry`:

```go
func NewHandlers(service *Service, registry *module.Registry) *Handlers {
    return &Handlers{service: service, registry: registry}
}
```

**Add the endpoint handler:**

```go
func (h *Handlers) GetEventCatalog(c echo.Context) error {
    groups := h.registry.CollectNotificationEvents()
    return c.JSON(http.StatusOK, groups)
}
```

The response shape is `[]module.NotificationEventGroup`, which serializes as:

```json
[
  {
    "id": "framework",
    "label": "General",
    "events": [
      {"id": "grab", "label": "On Grab", "description": "When a release is grabbed from an indexer"},
      {"id": "import", "label": "On Import", "description": "When a file is imported to the library"},
      ...
    ]
  },
  {
    "id": "movie",
    "label": "Movies",
    "events": [
      {"id": "movie:added", "label": "Movie Added", "description": "When a movie is added to the library"},
      {"id": "movie:deleted", "label": "Movie Deleted", "description": "When a movie is removed"}
    ]
  },
  {
    "id": "tv",
    "label": "TV Shows",
    "events": [
      {"id": "tv:added", "label": "Series Added", "description": "When a series is added to the library"},
      {"id": "tv:deleted", "label": "Series Deleted", "description": "When a series is removed"}
    ]
  }
]
```

**Modify** `internal/notification/handlers.go` — add the route in `RegisterRoutes`:

The new `/events` route must be registered before the `/:id` routes to avoid conflicts. Add it as the first route in `RegisterRoutes`:

```go
func (h *Handlers) RegisterRoutes(g *echo.Group) {
    g.GET("/events", h.GetEventCatalog) // add this — before :id routes
    g.GET("", h.List)
    g.POST("", h.Create)
    g.GET("/:id", h.Get)
    g.PUT("/:id", h.Update)
    g.DELETE("/:id", h.Delete)
    g.POST("/:id/test", h.Test)
}
```

**Modify** `internal/api/routes.go` — update the `NewHandlers` call to pass the registry:

`Handlers` is constructed inline in `setupNotificationRoutes`, not via Wire. The `Server` struct gains access to the module `Registry` in Phase 0. Update the construction call:

```go
func (s *Server) setupNotificationRoutes(api, protected *echo.Group) {
    notificationHandlers := notification.NewHandlers(s.notification.Service, s.registry)
    // ... rest unchanged
}
```

(No `wire.go` or `make wire` changes needed — the notification handler is not wired through DI.)

**Verify:**
- `go build ./internal/notification/...` compiles
- `go build ./internal/api/...` compiles
- Manual test: `curl localhost:8080/api/v1/notifications/events` returns the grouped event catalog

---

### Task 7.9: Add BroadcastEntity to WebSocket Hub ✅

**Modify** `internal/domain/contracts/contracts.go`

Add `BroadcastEntity` to the `Broadcaster` interface:

```go
type Broadcaster interface {
    Broadcast(msgType string, payload any)
    BroadcastEntity(moduleType, entityType string, entityID int64, action string, payload any)
}
```

**Modify** `internal/websocket/hub.go`

Add entity fields to the `Message` struct:

```go
type Message struct {
    Type       string      `json:"type"`
    Payload    interface{} `json:"payload"`
    Timestamp  string      `json:"timestamp"`
    Module     string      `json:"module,omitempty"`
    EntityType string      `json:"entityType,omitempty"`
    EntityID   int64       `json:"entityId,omitempty"`
    Action     string      `json:"action,omitempty"`
}
```

Implement `BroadcastEntity` on `Hub`:

```go
// BroadcastEntity sends an entity lifecycle event to all connected clients.
// The type field is derived from entityType:action for backward compatibility.
func (h *Hub) BroadcastEntity(moduleType, entityType string, entityID int64, action string, payload interface{}) {
    msg := Message{
        Type:       entityType + ":" + action,
        Module:     moduleType,
        EntityType: entityType,
        EntityID:   entityID,
        Action:     action,
        Payload:    payload,
        Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
    }
    data, err := json.Marshal(msg)
    if err != nil {
        h.logger.Error().
            Err(err).
            Str("module", moduleType).
            Str("entityType", entityType).
            Str("action", action).
            Msg("Failed to marshal entity WebSocket message")
        return
    }
    h.broadcast <- data
}
```

**Fix compile errors in other Broadcaster implementations:**

Search for all types that implement `contracts.Broadcaster` (compile-time assertions: `var _ contracts.Broadcaster = ...`). The Hub is the only real implementation. However, check for mock broadcasters in test files:

- `internal/api/` test helpers — if there's a `mockBroadcaster`, add a no-op `BroadcastEntity` method
- Any other test doubles

For each mock, add:

```go
func (m *mockBroadcaster) BroadcastEntity(moduleType, entityType string, entityID int64, action string, payload any) {
    // Delegate to Broadcast for test compatibility
    m.Broadcast(entityType+":"+action, payload)
}
```

**Verify:**
- `go build ./...` compiles (this catches all implementors of `Broadcaster`)
- Existing tests pass: `make test`

---

### Task 7.10: Update Entity Broadcast Call Sites — Backend ✅

Migrate all entity lifecycle broadcasts to use `BroadcastEntity`. Each call site replaces `s.broadcaster.Broadcast("type:action", payload)` with `s.broadcaster.BroadcastEntity(moduleType, entityType, entityID, action, payload)`.

**Modify** `internal/library/movies/service.go`

All movie entity broadcasts. Find by searching for `Broadcast("movie:`:

| Current Call | New Call |
|---|---|
| `s.hub.Broadcast("movie:added", movie)` | `s.hub.BroadcastEntity("movie", "movie", movie.ID, "added", movie)` |
| `s.hub.Broadcast("movie:updated", movie)` | `s.hub.BroadcastEntity("movie", "movie", movie.ID, "updated", movie)` |
| `s.hub.Broadcast("movie:updated", map[string]any{"movieId": movieID})` | `s.hub.BroadcastEntity("movie", "movie", movieID, "updated", nil)` |
| `s.hub.Broadcast("movie:deleted", map[string]int64{"id": id})` | `s.hub.BroadcastEntity("movie", "movie", id, "deleted", nil)` |

Note: For `library:updated` broadcasts in this file — leave as `Broadcast("library:updated", nil)`. This is a framework event, not an entity event.

**Modify** `internal/library/tv/service.go`

All TV entity broadcasts in service.go. Find by searching for `Broadcast("series:`:

| Current Call | New Call |
|---|---|
| `s.hub.Broadcast("series:added", series)` | `s.hub.BroadcastEntity("tv", "series", series.ID, "added", series)` |
| `s.hub.Broadcast("series:updated", series)` | `s.hub.BroadcastEntity("tv", "series", series.ID, "updated", series)` |
| `s.hub.Broadcast("series:deleted", map[string]int64{"id": id})` | `s.hub.BroadcastEntity("tv", "series", id, "deleted", nil)` |

**Modify** `internal/library/tv/episodes.go`

This file broadcasts `series:updated` when episodes are deleted or have status changes:

| Current Call | New Call |
|---|---|
| `s.hub.Broadcast("series:updated", map[string]int64{"id": seriesID})` (line 375, episode deletion) | `s.hub.BroadcastEntity("tv", "series", seriesID, "updated", nil)` |
| `s.hub.Broadcast("series:updated", map[string]any{"id": episode.SeriesID})` (line 399, episode status) | `s.hub.BroadcastEntity("tv", "series", episode.SeriesID, "updated", nil)` |

**Modify** `internal/library/tv/seasons.go`

This file broadcasts `series:updated` after season monitoring changes:

| Current Call | New Call |
|---|---|
| `s.hub.Broadcast("series:updated", map[string]int64{"id": seriesID})` (line 95) | `s.hub.BroadcastEntity("tv", "series", seriesID, "updated", nil)` |

**Modify** `internal/import/service.go` and `internal/import/pipeline.go`

Find broadcasts of `movie:updated` and `series:updated` in the import pipeline. These fire after successful import to notify the UI that the entity has changed. Update to `BroadcastEntity` with the correct module type.

The import pipeline determines media type from the download mapping. It already has `mediaType` available (e.g., `"movie"` or `"episode"`). Map these to module types:
- `"movie"` → module `"movie"`, entity type `"movie"`
- `"episode"` → module `"tv"`, entity type `"series"` (broadcasts at series level to trigger series detail refresh)

**Modify** `internal/downloader/completion.go`

This file broadcasts `movie:updated` and `series:updated` when downloads complete. Same pattern as import pipeline.

**Modify** `internal/autosearch/service.go`

Find broadcasts of `movie:updated` and `series:updated` after successful grab. Same pattern.

**Modify** `internal/indexer/grab/service.go`

Find broadcasts of `movie:updated` and `series:updated` after grab succeeds. Same pattern.

**General approach for subagent:** Grep for `Broadcast("movie:` and `Broadcast("series:` across the entire `internal/` directory. For each hit:
1. Determine the module type (`"movie"` or `"tv"`)
2. Determine the entity type (`"movie"` or `"series"`)
3. Extract the entity ID from context
4. Replace with `BroadcastEntity`

Leave non-entity broadcasts unchanged: `queue:*`, `progress:*`, `health:*`, `autosearch:*`, `import:*`, `download:*`, `history:*`, `artwork:*`, `scheduler:*`, `rss-sync:*`, `devmode:*`, `logs:*`, `request:*`, `portal:*`, `search:*`, `grab:*`, `library:updated`, `rename:completed`, `update:status`.

**Verify:**
- `go build ./...` compiles
- `make test` passes
- Grep confirms no remaining `Broadcast("movie:` or `Broadcast("series:` calls (all migrated to `BroadcastEntity`)

---

### Task 7.11: Frontend — Entity Event Registry and Generic Handler ✅

**Modify** `web/src/stores/ws-types.ts`

Add entity event fields to the message types. Update `LibraryMessage` to include the new structured fields:

```typescript
// Entity events carry structured module/entity fields alongside the legacy type
type EntityEventFields = {
  module: string
  entityType: string
  entityId: number
  action: string
}

type LibraryMessage = {
  type: 'movie:added' | 'movie:updated' | 'movie:deleted' | 'series:added' | 'series:updated' | 'series:deleted'
  payload: unknown
  timestamp: string
} & EntityEventFields
```

The `WSMessage` union remains the same — `LibraryMessage` just gains the extra fields. Other message types are unchanged.

**Create** `web/src/stores/ws-entity-registry.ts`

Define the entity invalidation registry:

```typescript
import { movieKeys } from '@/hooks/use-movies'
import { missingKeys } from '@/hooks/use-missing'
import { seriesKeys } from '@/hooks/use-series'

type InvalidationRule = {
  queryKeys: readonly (readonly string[])[]
}

// Maps module type → query keys to invalidate on any entity event from that module.
// New modules register here — no changes needed in the handler code.
const entityInvalidationRegistry: Record<string, InvalidationRule> = {
  movie: {
    queryKeys: [movieKeys.all, missingKeys.counts()],
  },
  tv: {
    queryKeys: [seriesKeys.all, missingKeys.counts()],
  },
}

export function getEntityInvalidationKeys(moduleType: string): readonly (readonly string[])[] | undefined {
  return entityInvalidationRegistry[moduleType]?.queryKeys
}
```

**Modify** `web/src/stores/ws-message-handlers.ts`

Replace `libraryHandler` and `handleLibraryEvent` with a generic entity handler:

```typescript
import { getEntityInvalidationKeys } from './ws-entity-registry'

function handleEntityEvent(
  queryClient: QueryClient,
  message: WSMessage,
): void {
  // Entity events carry a `module` field set by BroadcastEntity
  const moduleType = (message as { module?: string }).module
  if (!moduleType) return

  const keys = getEntityInvalidationKeys(moduleType)
  if (!keys) return

  for (const queryKey of keys) {
    void queryClient.invalidateQueries({ queryKey: [...queryKey] })
  }
}

// Replace the old libraryHandler
const entityHandler: MessageHandler = (message, ctx) =>
  handleEntityEvent(ctx.queryClient, message)
```

Update the `handlerMap` — replace `libraryHandler` references with `entityHandler`:

```typescript
const handlerMap: Partial<Record<WSMessageType, MessageHandler>> = {
  'movie:added': entityHandler,      // was libraryHandler
  'movie:updated': entityHandler,    // was libraryHandler
  'movie:deleted': entityHandler,    // was libraryHandler
  'series:added': entityHandler,     // was libraryHandler
  'series:updated': entityHandler,   // was libraryHandler
  'series:deleted': entityHandler,   // was libraryHandler
  // ... all other entries unchanged
}
```

Remove the `handleLibraryEvent` function and the `libraryHandler` const — they're replaced by `handleEntityEvent` and `entityHandler`.

**Verify:**
- `cd web && bun run lint` passes
- `cd web && bun run build` succeeds (type checking)
- Manual test: add/update/delete a movie or series → UI updates in real time

---

### Task 7.12: Frontend — Notification Types and Settings UI ✅

This is the largest frontend task. It updates the notification TypeScript types, API client, and settings UI to use dynamic event toggles from the catalog API.

**Modify** `web/src/types/notification.ts`

Replace the 10 individual boolean fields with `eventToggles`:

```typescript
export type Notification = {
  id: number
  name: string
  type: NotifierType
  enabled: boolean
  settings: Record<string, unknown>
  eventToggles: Record<string, boolean>
  includeHealthWarnings: boolean
  tags: number[]
  createdAt?: string
  updatedAt?: string
}

export type CreateNotificationInput = {
  name: string
  type: NotifierType
  enabled?: boolean
  settings: Record<string, unknown>
  eventToggles?: Record<string, boolean>
  includeHealthWarnings?: boolean
  tags?: number[]
}

export type UpdateNotificationInput = {
  name?: string
  type?: NotifierType
  enabled?: boolean
  settings?: Record<string, unknown>
  eventToggles?: Record<string, boolean>
  includeHealthWarnings?: boolean
  tags?: number[]
}
```

**Add** event catalog types (in the same file or a new `web/src/types/notification-events.ts`):

```typescript
export type NotificationEventDef = {
  id: string
  label: string
  description: string
}

export type NotificationEventGroup = {
  id: string
  label: string
  events: NotificationEventDef[]
}
```

**Modify** `web/src/hooks/use-notifications.ts`

Add a query hook for the event catalog:

```typescript
export const notificationKeys = createQueryKeys('notifications')
// Add a dedicated key for the events catalog
export const notificationEventKeys = {
  catalog: () => ['notifications', 'events'] as const,
}

export function useNotificationEventCatalog() {
  return useQuery({
    queryKey: notificationEventKeys.catalog(),
    queryFn: () => api.get<NotificationEventGroup[]>('/notifications/events').then(r => r.data),
    staleTime: Infinity, // catalog doesn't change at runtime
  })
}
```

**Modify** `web/src/components/notifications/notification-dialog-types.ts`

Remove `adminEventTriggers` array and `defaultFormData`'s 10 event booleans. Replace with:

```typescript
export const defaultFormData: CreateNotificationInput = {
  name: '',
  type: 'discord',
  enabled: true,
  settings: {},
  eventToggles: {
    grab: true,
    import: true,
    upgrade: true,
    health_issue: true,
    health_restored: true,
  },
  includeHealthWarnings: true,
  tags: [],
}
```

The `adminEventTriggers` export is removed — the triggers now come from the API. Remove the `EventTrigger` type from this file too (it's replaced by `NotificationEventDef`).

**Modify** `web/src/components/notifications/event-triggers.tsx`

Update to accept grouped events from the catalog and render them with group headings:

```tsx
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import type { NotificationEventGroup } from '@/types/notification'

type EventTriggersProps = {
  groups: NotificationEventGroup[]
  toggles: Record<string, boolean>
  onToggleChange: (eventId: string, enabled: boolean) => void
}

export function EventTriggers({ groups, toggles, onToggleChange }: EventTriggersProps) {
  if (groups.length === 0) return null

  return (
    <div className="space-y-4">
      <Label>Event Triggers</Label>
      {groups.map((group) => (
        <div key={group.id} className="space-y-2 rounded-lg border p-3">
          <p className="text-sm font-medium text-muted-foreground">{group.label}</p>
          {group.events.map((event) => (
            <div key={event.id} className="flex items-center justify-between">
              <Label htmlFor={event.id} className="cursor-pointer font-normal">
                {event.label}
              </Label>
              <Switch
                id={event.id}
                checked={toggles[event.id] ?? false}
                onCheckedChange={(checked) => onToggleChange(event.id, checked)}
              />
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}
```

**Modify** `web/src/components/notifications/notification-form-body.tsx`

Update to fetch the event catalog and pass it to `EventTriggers`. The form data now uses `eventToggles` instead of individual boolean fields:

```tsx
// In the form body component, replace:
//   <EventTriggers triggers={adminEventTriggers} formValues={formData} onTriggerChange={handleChange} />
// With:
const { data: eventCatalog } = useNotificationEventCatalog()
// ...
<EventTriggers
  groups={eventCatalog ?? []}
  toggles={formData.eventToggles ?? {}}
  onToggleChange={(eventId, enabled) => {
    setFormData(prev => ({
      ...prev,
      eventToggles: { ...prev.eventToggles, [eventId]: enabled },
    }))
  }}
/>
```

**Modify** `web/src/components/notifications/notification-dialog.tsx`

Update form initialization: when editing an existing notification, populate `eventToggles` from the notification data. When creating, use `defaultFormData.eventToggles`.

**Modify** `web/src/routes/settings/general/notifications.tsx`

The notification list page shows active events in the card description (currently `getActiveEventsText(notification)`). Update this function to use the event catalog for labels:

```typescript
function getActiveEventsText(notification: Notification, catalog: NotificationEventGroup[]): string {
  const allEvents = catalog.flatMap(g => g.events)
  const activeLabels = allEvents
    .filter(e => notification.eventToggles[e.id])
    .map(e => e.label)
  return activeLabels.length > 0 ? activeLabels.join(', ') : 'No events'
}
```

**Portal notification dialog** — check `web/src/components/portal/` or similar for portal user notification dialogs. These use `on_available`, `on_approved`, `on_denied` — these are NOT affected by Phase 7 (they're portal-specific, not module events). Verify they still work after the type changes. If the portal notification dialog reuses `NotificationDialog` with custom `eventTriggers` prop, ensure the prop interface still works (it may need updating to use the new `NotificationEventGroup` shape).

**Verify:**
- `cd web && bun run lint` passes
- `cd web && bun run build` succeeds
- Manual test: open Settings → Notifications → create/edit a notification agent → event toggles render grouped by "General" / "Movies" / "TV Shows"
- Manual test: save a notification with specific event toggles → reload → toggles persist correctly

---

### Task 7.13: Phase 7 Validation ✅

**Run all of these after all Phase 7 tasks complete:**

1. `go build ./internal/module/...` — module framework compiles with notification event types
2. `go build ./internal/modules/movie/...` — movie module compiles with `NotificationEvents`
3. `go build ./internal/modules/tv/...` — TV module compiles with `NotificationEvents`
4. `go build ./internal/notification/...` — notification service compiles with JSON event toggles
5. `go build ./internal/api/...` — API layer compiles with events catalog endpoint
6. `go build ./internal/websocket/...` — WebSocket hub compiles with `BroadcastEntity`
7. `go build ./...` — full project builds
8. `make test` — all existing tests pass
9. `make lint` — no new lint issues
10. `cd web && bun run lint` — frontend lint passes
11. `cd web && bun run build` — frontend builds successfully

**Compile-time interface assertions that must exist:**

```
internal/modules/movie/module.go:  var _ module.NotificationEvents = (*Module)(nil)
internal/modules/tv/module.go:    var _ module.NotificationEvents = (*Module)(nil)
internal/websocket/hub.go:        var _ contracts.Broadcaster = (*Hub)(nil)
```

**Runtime verification checklist:**

- [ ] `GET /api/v1/notifications/events` returns 3 groups (framework, movie, tv) with correct event IDs
- [ ] `POST /api/v1/notifications` with `{"eventToggles": {"grab": true, "movie:added": true}}` creates correctly
- [ ] `GET /api/v1/notifications/:id` returns `eventToggles` as a JSON object
- [ ] `PUT /api/v1/notifications/:id` with partial `eventToggles` merges correctly (preserves unmentioned keys)
- [ ] Notification dispatch works: add a movie → "movie:added" event fires to subscribed agents
- [ ] WebSocket entity events include `module`, `entityType`, `entityId`, `action` fields
- [ ] Frontend receives entity events and invalidates correct query keys
- [ ] Non-entity WebSocket events (progress, queue, health, etc.) still work unchanged

**What exists after Phase 7:**

- `internal/module/notification_events.go` — `NotificationEvent`, `NotificationEventGroup`, `NotificationEvents` interface
- `internal/module/framework_events.go` — framework event ID constants and `FrameworkNotificationEvents()`
- `internal/module/registry.go` — gains `CollectNotificationEvents()` method
- `internal/modules/movie/module.go` — implements `NotificationEvents` declaring `movie:added`, `movie:deleted`
- `internal/modules/tv/module.go` — implements `NotificationEvents` declaring `tv:added`, `tv:deleted`
- `internal/database/migrations/0XX_notification_event_toggles.sql` — replaces 10 columns with JSON
- `internal/database/queries/notifications.sql` — uses `event_toggles` column
- `internal/notification/events.go` — event constants use new IDs (`"movie:added"` etc.)
- `internal/notification/types.go` — `Config.EventToggles map[string]bool` replaces 10 bool fields
- `internal/notification/service.go` — dispatch checks JSON map; CRUD serializes/deserializes JSON
- `internal/notification/types/types.go` — duplicate `EventType` definition removed
- `internal/notification/handlers.go` — gains `GetEventCatalog` endpoint
- `internal/domain/contracts/contracts.go` — `Broadcaster` gains `BroadcastEntity` method
- `internal/websocket/hub.go` — `Message` gains entity fields; `BroadcastEntity` implemented
- Entity broadcast call sites (movies, tv service/episodes/seasons, import, autosearch, grab, downloader) use `BroadcastEntity`
- `sqlc.yaml` — 10 column overrides for dropped event columns removed
- `web/src/types/notification.ts` — `eventToggles: Record<string, boolean>` replaces 10 booleans
- `web/src/stores/ws-entity-registry.ts` — module → query key invalidation registry
- `web/src/stores/ws-message-handlers.ts` — generic `entityHandler` replaces `libraryHandler`
- `web/src/stores/ws-types.ts` — `LibraryMessage` gains entity fields
- `web/src/components/notifications/event-triggers.tsx` — renders grouped events from catalog
- Zero changes to: Notifier interface, notifier implementations, portal user notifications, non-entity WS events

**Files touched summary:**

**New files:**
```
internal/module/notification_events.go
internal/module/framework_events.go
internal/database/migrations/0XX_notification_event_toggles.sql
web/src/stores/ws-entity-registry.ts
```

**Modified files:**
```
internal/module/registry.go (CollectNotificationEvents method)
internal/module/interfaces.go (NotificationEvents in interface list)
internal/modules/movie/module.go (NotificationEvents impl + compile-time assertion)
internal/modules/tv/module.go (NotificationEvents impl + compile-time assertion)
internal/database/queries/notifications.sql (event_toggles column)
internal/database/sqlc/* (regenerated from sqlc generate)
internal/notification/events.go (new event ID values)
internal/notification/types.go (EventToggles map replaces 10 bools on Config/CreateInput/UpdateInput)
internal/notification/types/types.go (remove duplicate EventType definition)
internal/notification/service.go (JSON parse/serialize, configSubscribesToEvent, Create, buildUpdateParams)
internal/notification/handlers.go (GetEventCatalog endpoint, registry field, /events route in RegisterRoutes)
internal/domain/contracts/contracts.go (BroadcastEntity on Broadcaster interface)
internal/websocket/hub.go (Message entity fields, BroadcastEntity impl)
internal/library/movies/service.go (BroadcastEntity calls)
internal/library/tv/service.go (BroadcastEntity calls)
internal/library/tv/episodes.go (BroadcastEntity calls)
internal/library/tv/seasons.go (BroadcastEntity calls)
internal/import/service.go (BroadcastEntity calls)
internal/import/pipeline.go (BroadcastEntity calls)
internal/downloader/completion.go (BroadcastEntity calls)
internal/autosearch/service.go (BroadcastEntity calls)
internal/indexer/grab/service.go (BroadcastEntity calls)
internal/api/routes.go (pass registry to NewHandlers call)
sqlc.yaml (remove 10 column overrides for dropped event columns)
web/src/types/notification.ts (eventToggles, event catalog types)
web/src/hooks/use-notifications.ts (useNotificationEventCatalog hook)
web/src/stores/ws-types.ts (EntityEventFields on LibraryMessage)
web/src/stores/ws-message-handlers.ts (entityHandler replaces libraryHandler)
web/src/components/notifications/event-triggers.tsx (grouped rendering from catalog)
web/src/components/notifications/notification-dialog-types.ts (remove hardcoded triggers, update defaults)
web/src/components/notifications/notification-form-body.tsx (fetch catalog, pass to EventTriggers)
web/src/components/notifications/notification-dialog.tsx (eventToggles form handling)
web/src/routes/settings/general/notifications.tsx (getActiveEventsText uses catalog)
```

---

