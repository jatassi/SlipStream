# Backend Code Review — A Critical Analysis

*An honest, thorough review of the SlipStream Go backend, examining ~180k lines across 36 packages.*

---

## 1. The God Object: `Server` struct (84 fields)

`internal/api/server.go:80-174`

The `Server` struct has **84 fields**. Every single service in the application lives as a field on this one type. Movie service, TV service, quality service, scanner, downloader, indexer, grabber, prowlarr, autosearch, RSS sync, import, notifications, portal users, portal invitations, portal requests, portal quota, portal notifications, portal autoapprove, portal auth, portal passkeys, portal middleware, portal search limiter, portal request searcher, portal media provisioner, portal watchers, portal status tracker, portal library checker, portal settings handlers...

The `NewServer` constructor is **516 lines long** (`server.go:193-707`). Five hundred and sixteen lines of `s.fooService = foo.NewService(...)` followed by `s.fooService.SetBarService(s.barService)`.

---

## 2. The Adapter Graveyard (27 adapter types)

`internal/api/adapters.go` — 908 lines of glue code.

**27 adapter types**, most of which are single-method wrappers that forward one call to another service:

```go
type portalAutoApproveAdapter struct {
    svc *autosearch.Service
}
func (a *portalAutoApproveAdapter) SearchAndGrab(ctx context.Context, ...) error {
    return a.svc.SearchAndGrab(ctx, ...)
}
```

That's the whole type. A named struct with a method that calls the same method on another struct. 27 of these.

This is what happens when you over-segment interfaces. Each service defines its own bespoke 1-2 method interface (`MovieLookup`, `EpisodeLookup`, `SeriesLookup`, `QueueGetter`, `PortalStatusTracker`...) and then the wiring layer needs an adapter for each one. One kind of coupling traded for another — plus 900 lines of boilerplate.

---

## 3. Dev Mode: The Maintenance Minefield

`internal/api/devmode.go` — 969 lines.

`updateServicesDB()` manually calls `.SetDB()` on **32 individual services**:

```go
s.downloaderService.SetDB(db)
s.notificationService.SetDB(db)
s.indexerService.SetDB(db)
s.grabService.SetDB(db)
s.rateLimiter.SetDB(db)
s.rootFolderService.SetDB(db)
// ... 26 more
```

Every new service must be added here manually. Additionally, five nearly identical "copy X from prod to dev" functions (`copyJWTSecretToDevDB`, `copySettingsToDevDB`, `copyPortalUsersToDevDB`, `copyPortalUserNotificationsToDevDB`, `copyQualityProfilesToDevDB`) all follow the same pattern but share no code.

---

## 4. The WebSocket Race Condition

`internal/websocket/hub.go:135-144`

```go
h.mu.RLock()
for client := range h.clients {
    select {
    case client.send <- message:
    default:
        close(client.send)
        delete(h.clients, client)  // Modifying map under RLock
    }
}
h.mu.RUnlock()
```

Deleting from a map while holding an `RLock`. An `RLock` permits concurrent readers — it does not permit writes. If two goroutines hit the `default` branch simultaneously, you get a concurrent map write panic. Works fine in development with 1 browser tab, explodes in production.

---

## 5. Context? What Context?

Multiple download clients accept a `context.Context` parameter and then immediately throw it away.

`internal/downloader/rtorrent/client.go:206-217`:

```go
func (c *Client) Remove(_ context.Context, id string, _ bool) error {
    _, err := c.call(context.Background(), "d.erase", ...)
    return err
}

func (c *Client) Pause(_ context.Context, id string) error {
    _, err := c.call(context.Background(), "d.stop", ...)
    return err
}
```

The context is accepted, named `_` to explicitly ignore it, then replaced with `context.Background()`. The caller thinks they can cancel these operations. They can't. The context parameter is a lie.

---

## 6. Silent Auth Failures

`internal/auth/service.go:102-115`

`SetDB()` generates and stores the JWT secret. If `rand.Read()` fails, the function returns silently. No error. No log. No panic. The service continues with a nil JWT secret, and every token operation fails with an inscrutable error.

```go
if _, err := rand.Read(secret); err != nil {
    return  // JWT secret is now nil. Good luck debugging this.
}
```

`SetDB` returns `void`. There is no way for the caller to know this failed.

---

## 7. Error Response Inconsistency

The API uses two different error response formats depending on which handler is hit.

Some handlers:
```go
echo.NewHTTPError(http.StatusBadRequest, "invalid id")
// Returns: {"message": "invalid id"}
```

Others:
```go
c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
// Returns: {"error": "invalid id"}
```

API consumers must check both `.message` and `.error` depending on the endpoint.

---

## 8. Copy-Paste Handler Boilerplate

`internal/autosearch/handlers.go` — The same parameter parsing block appears in 8+ handlers:

```go
func (h *Handlers) SearchMovie(c echo.Context) error {
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
    }
    // ...

func (h *Handlers) SearchMovieSlot(c echo.Context) error {
    idStr := c.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
    }
    // ...
```

Eight times. The same five lines. A `parseIDParam(c echo.Context, name string) (int64, error)` helper would cost 6 lines and save 40.

---

## 9. Swallowed Errors

`internal/downloader/hadouken/client.go:128-131`:
```go
for _, t := range result.Torrents {
    item, err := c.parseTorrentArray(t)
    if err != nil {
        continue  // What error? Who knows. Moving on.
    }
}
```

`internal/indexer/cardigann/login.go:118`:
```go
evaluated, _ := engine.Evaluate(string(val), tmplCtx)
```

Template evaluation fails? Use the zero value. The login will fail mysteriously downstream.

`internal/downloader/hadouken/client.go:129`:
```go
body, _ := io.ReadAll(resp.Body)
```

Can't read the response body? Empty byte slice it is.

---

## 10. Goroutine Leaks

`internal/metadata/cache.go:51`:
```go
go c.cleanup()
```

This goroutine runs a `time.Ticker` loop forever. No stop channel, no context, no shutdown mechanism. When the cache is garbage collected, the goroutine keeps running, holding a reference to the cache, preventing collection. Classic leak.

`internal/progress/progress.go:156-161`:
```go
go func() {
    time.Sleep(5 * time.Second)
    m.mu.Lock()
    delete(m.activities, id)
    m.mu.Unlock()
}()
```

Every activity completion spawns a goroutine that sleeps for 5 seconds. 100 rapid completions = 100 sleeping goroutines.

---

## 11. The TODO Trail

```
internal/indexer/cardigann/client.go:530    // TODO: Add download URL info to context
internal/autosearch/handlers.go:227         // TODO: Check download queue
internal/library/tv/service.go:370          // TODO: If deleteFiles is true, delete actual files from disk
```

That last one: the delete function doesn't delete files. It's a delete function that TODOs the deletion.

---

## 12. `boolToInt` Ceremony

`internal/notification/service.go:532-537`:

```go
func boolToInt(b bool) int64 {
    if b { return 1 }
    return 0
}
```

Called 13 times in a single function to convert Go bools to SQLite integers. SQLite supports booleans. sqlc can handle the conversion. Instead, every notification creation is a wall of `boolToInt(input.OnGrab)`, `boolToInt(input.OnDownload)`, `boolToInt(input.OnUpgrade)`...

---

## 13. Import Queue Race Condition

`internal/import/service.go:365-382`:

```go
s.mu.Lock()
s.processing[job.SourcePath] = true  // Mark as processing
s.mu.Unlock()

select {
case s.importQueue <- job:  // Try to send to channel
    return nil
default:
    s.mu.Lock()
    delete(s.processing, job.SourcePath)  // Undo mark
    s.mu.Unlock()
    return errors.New("import queue is full")
}
```

The file is marked as "processing" *before* the channel send is known to succeed. Between the unlock and the channel send, another goroutine could check `s.processing[path]`, see it's true, and skip the file — even though it was never actually queued.

---

## 14. No `-race` Flag

No evidence of `-race` in CI or Makefile. Given the issues above (RLock/delete, processing map race, unprotected field access in the scheduler), the race detector likely hasn't been run recently.

---

## What's Actually Good

- **Interface boundaries are thoughtful.** Services depend on narrow interfaces, not concrete types. The over-segmentation creates adapter bloat, but the principle is sound.
- **sqlc usage is correct.** Type-safe queries, no raw SQL in service code, proper migration management with Goose. Better than most Go projects.
- **Test quality is genuinely high.** 967 test functions, spec-driven test naming (T1, T2...), table-driven tests used consistently, real database tests with `testutil.NewTestDB`. The quality profile and decisioning tests are excellent.
- **Error types in the indexer package** are well-designed with proper `Unwrap()` chains and factory functions.
- **Security middleware is solid.** CSP, HSTS, same-origin CORS, proxy probe blocking, rate limiting with exponential backoff.
- **No global state.** Everything is injected. No `init()` abuse (one minor exception in quality profiles). No package-level `var db *sql.DB` nightmares.
- **Naming is idiomatic.** `ctx`, `err`, `svc`, `cfg` — not `currentContextObject` or `theErrorResult`. The code reads like Go, not like Java written in Go syntax.

---

## Verdict

The architecture is intentional, the test coverage is real, and the code is readable. But it has the hallmarks of rapid development: a god object at the center, race conditions in the concurrent code, a growing maintenance burden in the wiring layer, and enough swallowed errors to make debugging in production a guessing game.

The real risk isn't code quality — it's that 84-field `Server` struct and the 32-service `SetDB()` chain. That's the thing that will make contributors bounce. Nobody wants to add a feature and discover they have to touch `server.go`, `adapters.go`, `devmode.go`, and `routes.go` just to wire up a new service.

Priority fixes: race conditions, adapter layer consolidation, and a service registry pattern before users start filing issues about random panics. The bones are good. The joints need work.
