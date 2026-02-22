# Portal In-Library Search Result Enhancement — Implementation Plan

**Spec:** `docs/portal-in-library-search-result-enhancement-spec.md`

---

## Agent Execution Strategy

### Workflow

1. **Write all new SQL queries first** (Phases 2.3, 3.3, 4.1), then `sqlc generate` once
2. **Implement all backend Go changes** (Phases 1, 2, 3, 4, 5) — they share files and compile together
3. **`go build ./...`** after backend Go changes to catch type errors
4. **Lint** after all backend phases — fix only issues you introduced
5. **Implement frontend** (Phases 6–11) sequentially — each depends on the previous
6. **Lint frontend** after all frontend phases
7. **Run backend tests** you wrote: `go test -v ./internal/portal/requests/...`

### Context Management

- Read files just-in-time per phase using the **Pre-read** lists below — don't read everything upfront
- After editing a file, you already have it in context — don't re-read it in later phases
- The plan provides enough detail to implement without broad codebase exploration

### Subagent Usage

- **Phase 12** (Admin Context): independent and low priority — run as a background Sonnet subagent after completing Phase 11
- **Linting**: run lint commands in background shells after completing each group (backend/frontend)
- All other phases: execute sequentially in the main context

### Feature-Specific Pitfalls

These are specific to this feature and NOT covered in CLAUDE.md or MEMORY.md:

- **`requested_seasons`** is stored as JSON TEXT — use the existing `seasonsFromJSON()` helper (`service.go:599`) to parse it, not manual JSON unmarshaling
- **`season_number`** on requests is `sql.NullInt64` — only set for `mediaType = 'season'`, must check `.Valid` before using
- **`UpdateFutureSeasonsMonitored`** has a `SeriesID_2` param — the series_id appears twice in the SQL subquery, both must be set to the same value
- **`monitored` columns** are `INTEGER` (0/1) in SQLite — sqlc maps to `int64`, use `1` for true, `0` for false
- **Table is `request_watchers`** (not `request_watches`) — `IsWatchingRequest` query already exists in `request_watchers.sql`
- **`canRequest`** already exists on both `AvailabilityResult` (Go) and `AvailabilityInfo` (TS) — do NOT add it again

---

## Phase 1: Backend — Movie Availability Fix

**Goal:** Movies in library with no files should be requestable.

> **Pre-read:** `internal/portal/requests/library_check.go` (full file — understand `CheckMovieAvailability`, `checkMovieInLibrary`, `getMovieSlots`, `canRequestWithSlots` flow)
>
> **Pitfall:** Do NOT clear `result.MediaID` or `result.AddedAt` when setting `InLibrary = false` — they're needed by `checkExistingMovieRequest()` which runs next

### 1.1 Update `CheckMovieAvailability()` — `library_check.go`

The current `checkMovieInLibrary()` (line 65) sets `InLibrary = true` unconditionally when the movie exists. It then checks slots for `canRequest` via the quality-profile-aware `canRequestWithSlots()`.

**Change:** After computing slots, add an additional `InLibrary` override:
- If the movie exists in DB but **zero** slots have `HasFile = true`, set `InLibrary = false` and `CanRequest = true`.
- This makes the movie appear in the "Requestable" section instead of "In Library".
- Keep `MediaID` and `AddedAt` populated regardless (they're needed if an existing request exists).

**Implementation:**
```go
// In checkMovieInLibrary(), after getting slots:
hasAnyFile := false
for _, slot := range slots {
    if slot.HasFile {
        hasAnyFile = true
        break
    }
}
if !hasAnyFile {
    result.InLibrary = false
    result.CanRequest = true
}
```

This is the minimal change. The `checkExistingMovieRequest()` call runs after and will set `CanRequest = false` if an active request exists — that's correct behavior per the spec.

### 1.2 No SQL changes needed for movies

The existing `ListMovieSlotAssignments` query already returns all slot data including `file_id`.

---

## Phase 2: Backend — Series Per-Season Availability

**Goal:** Compute per-season availability data during search and return it in `AvailabilityResult`.

> **Pre-read:** `internal/portal/requests/library_check.go` (lines 119–157 for `CheckSeriesAvailability`), `internal/database/queries/series.sql` (scan for existing query patterns/naming)
>
> **Pitfall:** The `LEFT JOIN` in the SQL correctly returns `COUNT(e.id) = 0` for seasons with zero episodes (it counts non-NULL values). `COALESCE(SUM(CASE...))` returns 0 when all episode rows are NULL.

### 2.1 Add `SeasonAvailability` struct — `library_check.go`

```go
type SeasonAvailability struct {
    SeasonNumber         int   `json:"seasonNumber"`
    Available            bool  `json:"available"`
    HasAnyFiles          bool  `json:"hasAnyFiles"`
    AiredEpisodesWithFiles int `json:"airedEpisodesWithFiles"`
    TotalAiredEpisodes   int   `json:"totalAiredEpisodes"`
    TotalEpisodes        int   `json:"totalEpisodes"`
    Monitored            bool  `json:"monitored"`
}
```

### 2.2 Add `SeasonAvailability` to `AvailabilityResult`

`CanRequest` already exists on `AvailabilityResult` — do NOT add it again. Only add the new field:

```go
type AvailabilityResult struct {
    // ... existing fields (InLibrary, ExistingSlots, CanRequest, etc.) ...
    SeasonAvailability []SeasonAvailability `json:"seasonAvailability,omitempty"`
}
```

### 2.3 New SQL query: `GetSeriesSeasonAvailabilitySummary`

**File:** `internal/database/queries/series.sql`

A single efficient query that computes per-season stats for an in-library series. Joins `seasons`, `episodes`, `episode_slot_assignments` to produce:
- season_number, monitored (season)
- total episodes, aired episodes, aired episodes with at least one file

```sql
-- name: GetSeriesSeasonAvailabilitySummary :many
SELECT
    sea.season_number,
    sea.monitored,
    COUNT(e.id) as total_episodes,
    COALESCE(SUM(CASE WHEN e.air_date IS NOT NULL AND substr(e.air_date, 1, 10) <= date('now') THEN 1 ELSE 0 END), 0) as aired_episodes,
    COALESCE(SUM(CASE WHEN e.air_date IS NOT NULL AND substr(e.air_date, 1, 10) <= date('now')
        AND EXISTS (SELECT 1 FROM episode_slot_assignments esa WHERE esa.episode_id = e.id AND esa.file_id IS NOT NULL)
        THEN 1 ELSE 0 END), 0) as aired_with_files,
    COALESCE(SUM(CASE WHEN (e.air_date IS NULL OR substr(e.air_date, 1, 10) > date('now'))
        AND e.monitored = 1 THEN 1 ELSE 0 END), 0) as unaired_monitored
FROM seasons sea
LEFT JOIN episodes e ON e.series_id = sea.series_id AND e.season_number = sea.season_number
WHERE sea.series_id = ? AND sea.season_number > 0
GROUP BY sea.season_number, sea.monitored
ORDER BY sea.season_number;
```

**Season "available" definition** (from spec):
- Every aired episode has files in at least one slot (`aired_with_files == aired_episodes`)
- AND either: no unaired episodes, OR all unaired episodes are monitored

Compute in Go after fetching query results:
```go
unairedCount := row.TotalEpisodes - row.AiredEpisodes
available := row.AiredWithFiles == row.AiredEpisodes &&
    (unairedCount == 0 || row.UnairedMonitored == unairedCount)
```

**Do not run `sqlc generate` yet** — write Phase 3.3 and 4.1 SQL queries first, then generate all at once.

### 2.4 Update `CheckSeriesAvailability()` — `library_check.go`

Currently (line 119), this method:
1. Checks if series exists by TVDB ID → sets `InLibrary = true`
2. Calls `canRequestWithSlots()` with empty slots (always returns `true`, but then gets overridden by existing request check)
3. Checks for existing request → sets `CanRequest = false` if found

**New logic:**
1. If series exists in DB, fetch `GetSeriesSeasonAvailabilitySummary(ctx, series.ID)`
2. Compute per-season `SeasonAvailability` structs (use the availability formula above)
3. Determine `InLibrary`:
   - `true` only if at least one season has `hasAnyFiles = true`
   - `false` if zero files across all seasons (treat as not in library)
4. Determine `CanRequest`:
   - `true` if any library season is not "available" (i.e., requestable)
   - `false` only if ALL seasons are fully available
5. Attach `SeasonAvailability` array to result (initialize as `[]SeasonAvailability{}` if empty)
6. Existing request check stays as-is (but see Phase 4 for cross-type changes)

> **Note on metadata seasons not in library:** The SQL query only returns rows for seasons in the `seasons` table. If metadata reports more seasons than exist in the library (e.g., a newly announced season), those will surface when the user opens the request modal (Phase 3's enriched endpoint handles the merge, defaulting non-library seasons to requestable). At the search card level, `CanRequest` is computed solely from library data — this is acceptable because new seasons are typically added by admins via metadata refresh, which creates the season rows in the DB.

### 2.5 No changes to `CheckSeasonAvailability()`

The individual season check is used for single-season request duplicate detection and doesn't need the per-season enrichment. That function stays as-is.

---

## Phase 3: Backend — Enhanced Seasons Endpoint

**Goal:** Merge library availability data into the seasons response.

> **Pre-read:** `internal/portal/search/handlers.go` (lines 163–192 for `GetSeriesSeasons` handler), `internal/metadata/provider.go` (for `SeasonResult` and `EpisodeResult` struct definitions — `Episodes []EpisodeResult` is already on `SeasonResult`)
>
> **Pitfall:** `EnrichedSeasonResult` embeds `metadata.SeasonResult` which has `Episodes []EpisodeResult`. You must override this field with `Episodes []EnrichedEpisodeResult` using the same JSON tag `"episodes"` to shadow the embedded field.

### 3.1 New response types — `internal/portal/search/handlers.go`

```go
type EnrichedEpisodeResult struct {
    metadata.EpisodeResult
    HasFile   bool `json:"hasFile"`
    Monitored bool `json:"monitored"`
    Aired     bool `json:"aired"`
}

type EnrichedSeasonResult struct {
    metadata.SeasonResult
    Episodes              []EnrichedEpisodeResult `json:"episodes,omitempty"` // shadows embedded Episodes
    InLibrary             bool                    `json:"inLibrary"`
    Available             bool                    `json:"available"`
    Monitored             bool                    `json:"monitored"`
    AiredEpisodesWithFiles int                    `json:"airedEpisodesWithFiles"`
    TotalAiredEpisodes    int                     `json:"totalAiredEpisodes"`
    EpisodeCount          int                     `json:"episodeCount"`
    ExistingRequestID     *int64                  `json:"existingRequestId,omitempty"`
    ExistingRequestUserID *int64                  `json:"existingRequestUserId,omitempty"`
    ExistingRequestStatus *string                 `json:"existingRequestStatus,omitempty"`
    ExistingRequestIsWatching *bool               `json:"existingRequestIsWatching,omitempty"`
}
```

### 3.2 Update `GetSeriesSeasons` handler

Currently (line 164) this handler returns raw metadata `[]SeasonResult`.

**Change:** After fetching metadata, look up the series in the library by TVDB ID. If found, fetch the season availability summary and merge the library data into each season result.

The handler already has access to `libraryChecker`. Add a new method to `LibraryChecker`:

```go
func (c *LibraryChecker) GetSeasonAvailabilityMap(ctx context.Context, tvdbID int64) (map[int]SeasonAvailability, error)
```

This returns a map keyed by season number. The handler merges by matching `SeasonResult.SeasonNumber` to the map.

**Default state for non-library seasons:** Metadata seasons that don't appear in the availability map (i.e., they don't exist in the library at all) must default to:
```go
InLibrary: false, Available: false, Monitored: false,
AiredEpisodesWithFiles: 0, TotalAiredEpisodes: 0, EpisodeCount: len(seasonResult.Episodes)
```
These seasons are inherently requestable — the frontend will show them as selectable checkboxes in the modal.

### 3.3 Episode-level enrichment

The existing `metadata.SeasonResult` already contains `Episodes []EpisodeResult`. When the series exists in the library, enrich each episode with library data.

**New SQL query:** `GetEpisodeAvailabilityForSeason`

**File:** `internal/database/queries/episodes.sql`

```sql
-- name: GetEpisodeAvailabilityForSeason :many
SELECT
    e.episode_number,
    e.monitored,
    CASE WHEN e.air_date IS NOT NULL AND substr(e.air_date, 1, 10) <= date('now') THEN 1 ELSE 0 END as aired,
    CASE WHEN EXISTS (SELECT 1 FROM episode_slot_assignments esa WHERE esa.episode_id = e.id AND esa.file_id IS NOT NULL) THEN 1 ELSE 0 END as has_file
FROM episodes e
WHERE e.series_id = ? AND e.season_number = ?
ORDER BY e.episode_number;
```

**Write this SQL now** (alongside Phase 2.3 SQL), then run `sqlc generate` for all new queries together.

**Merge logic in handler:** After building the `EnrichedSeasonResult`, iterate over each season's `Episodes` and replace with `[]EnrichedEpisodeResult` by matching episode numbers to the query results. Episodes not found in the library default to `HasFile: false, Monitored: false, Aired: computed from EpisodeResult.AirDate`.

### 3.4 Add `tvdbId` param to seasons endpoint

Currently the endpoint accepts `tmdbId` and `tvdbId` but only uses them for metadata lookup. No change needed to the route — `tvdbId` is already available for the library lookup.

---

## Phase 4: Backend — Cross-Type Duplicate Detection

**Goal:** Detect overlapping requests across `series` and `season` media types.

> **Pre-read:** `internal/database/queries/requests.sql` (existing query patterns, especially `GetActiveRequestByTvdbID` and `GetActiveRequestByTvdbIDAndSeason`), `internal/portal/requests/service.go` (lines ~100–180 for `Create` and `checkExistingRequest`)

### 4.1 New SQL queries — `internal/database/queries/requests.sql`

**Query 1: Find active requests covering specific seasons**

```sql
-- name: FindRequestsCoveringSeasons :many
-- Returns all active requests (series and season types) for a TVDB ID
SELECT * FROM requests
WHERE tvdb_id = ?
  AND media_type IN ('series', 'season')
  AND status NOT IN ('denied', 'available')
ORDER BY created_at DESC;
```

**Now run `sqlc generate`** — all SQL from Phases 2.3, 3.3, and 4.1 is written.

This returns all active requests. The Go code then:
- For `mediaType = 'series'`: call `seasonsFromJSON(row.RequestedSeasons)` to get covered season numbers
- For `mediaType = 'season'`: use `row.SeasonNumber` field

This is simpler and more maintainable than doing JSON parsing in SQL.

### 4.2 New method: `GetCoveredSeasons()` — `library_check.go`

```go
type CoveredSeason struct {
    RequestID  int64
    UserID     int64
    Status     string
    IsWatching bool
}

func (c *LibraryChecker) GetCoveredSeasons(ctx context.Context, tvdbID int64, currentUserID int64) (map[int]CoveredSeason, error)
```

Returns a map of season numbers → covering request info.

Logic:
1. Call `FindRequestsCoveringSeasons(tvdbID)`
2. For each result:
   - If `media_type = 'series'`: call `seasonsFromJSON(row.RequestedSeasons)`, each season in the list is "covered"
   - If `media_type = 'season'`: `season_number.Int64` is "covered" (it's `sql.NullInt64`)
3. For each covering request, call `IsWatchingRequest(ctx, requestID, currentUserID)` to populate `IsWatching`
4. Return map

> **Note:** `IsWatchingRequest` already exists in `internal/database/queries/request_watchers.sql`. The table is `request_watchers` (not `request_watches`).

### 4.3 Expose covered seasons in seasons endpoint response

The `EnrichedSeasonResult` struct (Phase 3.1) already includes the request fields. The `GetSeriesSeasons` handler calls `GetCoveredSeasons()` and merges the request info into each season.

### 4.4 Update `Service.Create()` duplicate detection — `service.go`

The current `checkExistingRequest()` only checks within the same media type.

**For `series` requests with `requestedSeasons`:**
Add a check: call `FindRequestsCoveringSeasons(tvdbID)`, then check if ALL requested seasons are already covered. If so, return `ErrAlreadyRequested`.

**For `season` requests:**
Add a check: call `FindRequestsCoveringSeasons(tvdbID)`, check if the specific season is covered by any existing request.

**Important:** Partial overlap should NOT block creation. The frontend will have already disabled the overlapping seasons in the modal. The backend just needs to prevent the exact duplicate case. The frontend ensures the user only submits non-overlapping seasons.

---

## Phase 5: Backend — Approval Flow Fix for Existing Series

**Goal:** When approving a request for a series that already exists in the library, apply monitoring changes.

> **Pre-read:** `internal/api/adapters.go` (full `EnsureSeriesInLibrary` at line 98, and `applyRequestedSeasonsMonitoring` at line 152), `internal/portal/requests/searcher.go` (lines 25–33 for `MediaProvisionInput`, lines 247–293 for `ensureMediaInLibrary`)
>
> **Critical pitfall:** `applyRequestedSeasonsMonitoring()` currently iterates ALL seasons and unmonitors any not in the requested set. For an **existing** series, this would unmonitor seasons that already have files! Phase 5.4 introduces a separate additive method to avoid this.

### 5.1 Update `EnsureSeriesInLibrary()` — `internal/api/adapters.go`

Currently (line 98), when a series already exists, it returns the ID immediately at line 103 without applying any monitoring.

**Change:** Before the early return for existing series, also apply monitoring:

```go
if err == nil && existing != nil {
    a.logger.Debug().Int64("tvdbID", input.TvdbID).Int64("seriesID", existing.ID).Msg("found existing series in library")

    // Apply requested seasons monitoring for existing series
    if len(input.RequestedSeasons) > 0 {
        if err := a.applyRequestedSeasonsMonitoringAdditive(ctx, existing.ID, input.RequestedSeasons); err != nil {
            a.logger.Warn().Err(err).Int64("seriesID", existing.ID).Msg("failed to apply requested seasons monitoring to existing series")
        }
    }

    // Apply monitor future if requested
    if input.MonitorFuture {
        a.applyMonitorFuture(ctx, existing.ID)
    }

    return existing.ID, nil
}
```

### 5.2 Add `MonitorFuture` to `MediaProvisionInput`

**File:** `internal/portal/requests/searcher.go` — the `MediaProvisionInput` struct (line 25).

Add `MonitorFuture bool` field. Set it in `ensureMediaInLibrary()` (line 247) based on the request's `MonitorType`:

```go
input.MonitorFuture = request.MonitorType != nil && *request.MonitorType == "future"
```

> **Agent note:** Check the actual `MonitorType` values used in the codebase. Search for `MonitorType` assignments in `service.go` and the frontend request creation to confirm the exact string value.

### 5.3 Add `applyMonitorFuture()` helper — `internal/api/adapters.go`

```go
func (a *portalMediaProvisionerAdapter) applyMonitorFuture(ctx context.Context, seriesID int64) {
    if err := a.queries.UpdateFutureEpisodesMonitored(ctx, sqlc.UpdateFutureEpisodesMonitoredParams{
        Monitored: 1,
        SeriesID:  seriesID,
    }); err != nil {
        a.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future episodes")
    }
    if err := a.queries.UpdateFutureSeasonsMonitored(ctx, sqlc.UpdateFutureSeasonsMonitoredParams{
        Monitored:  1,
        SeriesID:   seriesID,
        SeriesID_2: seriesID, // same series ID appears twice in the SQL subquery
    }); err != nil {
        a.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future seasons")
    }
}
```

> **Agent note:** Verify the adapter struct has access to `a.queries` (type `*sqlc.Queries`). If not, access it through the tv service or add it. Check existing adapter methods for the pattern.

### 5.4 Add `applyRequestedSeasonsMonitoringAdditive()` for existing series

The existing `applyRequestedSeasonsMonitoring()` unmonitors everything EXCEPT the requested seasons — **this is correct for newly-created series** but destructive for existing ones.

**Add a new method** `applyRequestedSeasonsMonitoringAdditive()` that only monitors the requested seasons without touching others:

```go
func (a *portalMediaProvisionerAdapter) applyRequestedSeasonsMonitoringAdditive(ctx context.Context, seriesID int64, requestedSeasons []int64) error {
    for _, sn := range requestedSeasons {
        if _, err := a.tvService.UpdateSeasonMonitored(ctx, seriesID, int(sn), true); err != nil {
            a.logger.Warn().Err(err).Int64("seriesID", seriesID).Int64("seasonNumber", sn).Msg("failed to monitor requested season")
        }
    }
    return nil
}
```

The original `applyRequestedSeasonsMonitoring()` stays unchanged — it's still used for newly-created series (line 146).

---

## Phase 6: Frontend — Types & API

> **Pre-read:** `web/src/types/portal.ts` (full file — note current `AvailabilityInfo`, `SeasonInfo`, `SlotInfo`, `RequestStatus` types)

### 6.1 Update TypeScript types — `web/src/types/portal.ts`

Add new type and extend existing:

```typescript
export type SeasonAvailabilityInfo = {
    seasonNumber: number
    available: boolean
    hasAnyFiles: boolean
    airedEpisodesWithFiles: number
    totalAiredEpisodes: number
    totalEpisodes: number
    monitored: boolean
}

// Add to existing AvailabilityInfo (canRequest already exists — only add seasonAvailability):
export type AvailabilityInfo = {
    // ... all existing fields stay unchanged ...
    seasonAvailability?: SeasonAvailabilityInfo[]
}
```

### 6.2 Update enriched season and episode types — `web/src/types/portal.ts`

```typescript
export type EnrichedEpisode = {
    episodeNumber: number
    seasonNumber: number
    title: string
    overview?: string
    airDate?: string
    runtime?: number
    imdbRating?: number
    hasFile: boolean
    monitored: boolean
    aired: boolean
}

export type EnrichedSeason = {
    seasonNumber: number
    name: string
    overview?: string
    posterUrl?: string
    airDate?: string
    episodes?: EnrichedEpisode[]
    inLibrary: boolean
    available: boolean
    monitored: boolean
    airedEpisodesWithFiles: number
    totalAiredEpisodes: number
    episodeCount: number
    existingRequestId?: number
    existingRequestUserId?: number
    existingRequestStatus?: RequestStatus
    existingRequestIsWatching?: boolean
}
```

> **Agent note:** Write out the full type definitions with all fields rather than using `// ... existing fields ...` comments. This prevents field mismatches with the backend. Match the exact JSON field names from the Go structs.

### 6.3 No API hook changes needed

The existing `useSeriesSeasons()` hook already fetches from the seasons endpoint. The response shape expands but the hook doesn't need changes since TypeScript handles the additional fields transparently.

---

## Phase 7: Frontend — Search Result Splitting

> **Pre-read:** `web/src/routes/requests/use-request-search.ts` (full file — understand `useSearchResults`, `useSeriesDialog`, `buildSeriesRequestPayload`), `web/src/routes/requests/search-results-content.tsx`

### 7.1 Update splitting logic — `use-request-search.ts`

Current logic (line 23-37) splits into `libraryX` and `requestableX` based on `inLibrary`.

**New series splitting:**

```typescript
const fullyAvailableSeries = series.filter(
    (s) => s.availability?.inLibrary && !s.availability?.canRequest
)
const partialSeries = series.filter(
    (s) => s.availability?.inLibrary && s.availability?.canRequest
)
const requestableSeries = series.filter(
    (s) => !s.availability?.inLibrary
)
```

- `fullyAvailableSeries` → In Library section, disabled button (unchanged behavior)
- `partialSeries` → In Library section, but with Request button + availability badge
- `requestableSeries` → Requestable section (unchanged behavior)

Update `librarySeriesItems` to include both `fullyAvailableSeries` and `partialSeries` (sorted by addedAt). Pass `partialSeries` separately so the UI can distinguish them.

**Movie splitting:** No frontend change needed. The backend now returns `inLibrary = false` for movies with no files, so the existing filter works correctly.

### 7.2 Update `search-results-content.tsx`

Pass `partialSeries` to the In Library section grid. These cards need a Request button instead of the disabled "In Library" button.

---

## Phase 8: Frontend — Season Availability Badge

> **Pre-read:** `web/src/components/search/card-poster.tsx` or `web/src/components/search/status-badge.tsx` (whichever handles the "In Library" badge currently)

### 8.1 New component: `SeasonAvailabilityBadge`

**File:** `web/src/components/search/season-availability-badge.tsx`

Props: `seasonAvailability: SeasonAvailabilityInfo[]`

**Logic:**
1. Filter to available seasons (`available === true`)
2. Compute total seasons (non-special, i.e., `seasonNumber > 0`)
3. Format available season numbers into compact text:
   - Collapse contiguous ranges: `[1,2,3]` → `S1-3`
   - Non-contiguous: `[1,3]` → `S1, S3`
   - Mixed: `[1,2,3,5]` → `S1-3, S5`
4. If formatted text exceeds 10 characters, use fallback: `{available}/{total} seasons`
5. Render: `<BookIcon /> {text}`

### 8.2 Format helper: `formatSeasonRange()`

Pure utility function, testable independently. Takes array of available season numbers, returns formatted string.

### 8.3 Integrate badge into `CardPoster` or `StatusBadge`

For partially-available series in the In Library section, show the `SeasonAvailabilityBadge` instead of (or alongside) the "In Library" `StatusBadge`.

---

## Phase 9: Frontend — Card Action Button Changes

> **Pre-read:** `web/src/components/search/card-action-button.tsx`, `web/src/hooks/use-external-media-card.ts`

### 9.1 Update `CardActionButton` — `card-action-button.tsx`

Add new priority case for `isInLibrary && canRequest`:

```
if (hasActiveDownload) → DownloadProgressBar
if (isInLibrary && !canRequest) → "In Library" disabled
if (isInLibrary && canRequest) → Request button (opens series request modal)
if (hasExistingRequest) → ExistingRequestButton
→ Request button
```

### 9.2 Update `useExternalMediaCard` hook

Add `canRequest` derivation from `availability?.canRequest`. Pass it through to `CardActionButton`.

---

## Phase 10: Frontend — Request Modal Enhancements

> **Pre-read:** `web/src/routes/requests/series-request-dialog.tsx` (full file), `web/src/routes/requests/use-request-search.ts` (for `buildSeriesRequestPayload` and `selectAllSeasons`)
>
> **Pitfall:** The dialog currently receives `SeasonInfo[]` from the `useSeriesSeasons` hook. After Phase 3, the endpoint returns `EnrichedSeason[]`. The hook's return type will automatically pick up the new shape, but the dialog's `seasons` prop type needs updating from `SeasonInfo[]` to `EnrichedSeason[]`.

### 10.1 Update `SeriesRequestDialog` — `series-request-dialog.tsx`

Currently all seasons are checkboxes. Now they need three visual states.

**Per season:**
1. **Available (disabled, green):** Season `available === true` from the enriched season data
2. **Requested (disabled, yellow):** Season has `existingRequestId` from the enriched season data
3. **Selectable (enabled, checkbox):** Neither of the above

### 10.2 Filter Season 0

Already filtered (spec: line 288 says seasons are filtered for S0). Verify the existing code handles this — the current `SeriesRequestDialog` already filters season 0.

### 10.3 Season state rendering

Replace simple checkbox with conditional rendering:
- **Available:** Green check icon + season name, disabled
- **Requested:** Yellow clock icon + "Requested" label + season name, disabled + Watch button
- **Selectable:** Checkbox + season name

### 10.4 Submission payload adjustment

When building the request payload, only include seasons the user actually checked (exclude disabled-available and disabled-requested seasons). This is already the implicit behavior since disabled checkboxes won't be in the selected state.

**mediaType switching based on selected count:**
The current `buildSeriesRequestPayload()` always uses `mediaType: 'series'`. Per the spec, this must change:
- **Single season selected** → `mediaType: "season"` with `seasonNumber` set (no `requestedSeasons`)
- **Multiple seasons selected** → `mediaType: "series"` with `requestedSeasons` array
- **Zero seasons selected** (monitor future only) → `mediaType: "series"` with no `requestedSeasons`

```typescript
const seasonsArray = [...selectedSeasons].toSorted((a, b) => a - b)
if (seasonsArray.length === 1) {
    return { mediaType: 'season', seasonNumber: seasonsArray[0], /* ...common fields */ }
} else {
    return { mediaType: 'series', requestedSeasons: seasonsArray.length > 0 ? seasonsArray : undefined, /* ...common fields */ }
}
```

This is critical because Phase 4's cross-type duplicate detection relies on requests using the correct `mediaType`.

### 10.5 Helper text updates

- No seasons selected + monitor future on: "No seasons selected. Series will be added to library and only future episodes will be monitored."
- All selectable seasons are already disabled (nothing to check): Only show monitor future toggle.

### 10.6 Select All / Deselect All must exclude disabled seasons

The current `selectAllSeasons` selects every season unconditionally. With three visual states, Select All must only toggle **selectable** seasons (not available or already-requested):

```typescript
const selectableSeasons = enrichedSeasons.filter(
    (s) => !s.available && !s.existingRequestId && s.seasonNumber > 0
)
selectAllSeasons: () => setSelectedSeasons(new Set(selectableSeasons.map((s) => s.seasonNumber)))
```

`deselectAllSeasons` can remain as-is (clearing disabled seasons from the set is harmless since they're never in it).

---

## Phase 11: Frontend — Watch Button in Search Results

> **Pre-read:** Check existing watch mutation hook (search for `useWatchRequest` or similar in `web/src/hooks/`), `web/src/routes/requests/series-request-dialog.tsx` (for where to add Watch buttons)

### 11.1 `ExistingRequestButton` enhancement

When showing an existing request status on a card, add a Watch option. The watch API already exists (`POST /api/v1/portal/requests/{id}/watch`).

For movies with no files that have an existing request: show the request status + a small Watch button.

**"Already watching" check:** The `AvailabilityResult` from the backend includes `ExistingRequestID`. To check watch status for search result cards, extend the backend `checkExistingMovieRequest()` and `checkExistingSeasonRequest()` methods to also call `IsWatchingRequest` (query already exists in `request_watchers.sql`) and populate a new `ExistingRequestIsWatching *bool` field on `AvailabilityResult`.

### 11.2 Watch button in request modal

For seasons marked as "Requested" in the modal, show a small Watch button next to the request status. Clicking it calls the watch mutation with the covering request's ID.

The `CoveredSeason` struct (Phase 4.2) already includes `IsWatching bool`, populated by checking `IsWatchingRequest` during `GetCoveredSeasons()`.

---

## Phase 12: Backend — Admin Context for Requests

> **This phase is independent and low priority. Run as a background Sonnet subagent after completing Phase 11.**

### 12.1 Add library indicator to admin request detail

When the admin views a request for a series that already exists in the library, include:
- `inLibrary: true` flag on the request response
- `librarySeriesId` or `libraryUrl` for linking to the series detail page

**Implementation:** In the admin request detail handler/endpoint, look up whether `mediaId` is set on the request. If so, it's already linked. If not, check by TVDB/TMDB ID.

---

## Test Specifications

Write tests **after completing all backend phases** (1–5) but **before starting frontend phases**. Tests validate the backend contract that the frontend will depend on.

**File:** `internal/portal/requests/library_check_test.go` (new file)

Use the same test infrastructure as `status_tracker_test.go`:
- `testutil.NewTestDB(t)` + `defer tdb.Close()`
- `sqlc.New(tdb.Conn)` for queries
- Create test data via sqlc queries directly

### Test helper: `createTestSeriesWithEpisodes`

```go
// createTestSeriesWithEpisodes creates a series with seasons and episodes for testing.
// Returns the series ID. Each season gets the specified number of episodes.
// airedEpisodes controls how many episodes have an air_date in the past.
// fileEpisodes controls how many aired episodes get a file via episode_slot_assignments.
func createTestSeriesWithEpisodes(t *testing.T, q *sqlc.Queries, tvdbID int, seasons []testSeason) int64

type testSeason struct {
    Number        int
    Monitored     bool
    Episodes      int  // total episode count
    AiredEpisodes int  // how many have air_date <= now
    WithFiles     int  // how many aired episodes have files (must be <= AiredEpisodes)
    UnairedMonitored int // how many unaired episodes are monitored
}
```

### Test 1: `TestCheckMovieAvailability_NoFiles`

```
Setup:  Create movie with quality profile, create version slot, create slot assignment with file_id = NULL
Assert: InLibrary = false, CanRequest = true, MediaID is populated
```

### Test 2: `TestCheckMovieAvailability_HasFiles`

```
Setup:  Create movie with quality profile, create version slot, create slot assignment with file_id set
Assert: InLibrary = true, CanRequest depends on slot/profile logic
```

### Test 3: `TestCheckSeriesAvailability_ZeroFiles`

```
Setup:  Create series (tvdbID=1000), 2 seasons with episodes, zero episode_slot_assignments with files
Assert: InLibrary = false, CanRequest = true, SeasonAvailability populated but all hasAnyFiles = false
```

### Test 4: `TestCheckSeriesAvailability_PartiallyAvailable`

```
Setup:  Series with 3 seasons:
  - S1: 10 aired episodes, all with files (available)
  - S2: 10 aired episodes, 5 with files (NOT available)
  - S3: 5 aired + 3 unaired monitored, all aired with files (available)
Assert: InLibrary = true, CanRequest = true
        SeasonAvailability[0]: Available=true, HasAnyFiles=true, AiredEpisodesWithFiles=10
        SeasonAvailability[1]: Available=false, HasAnyFiles=true, AiredEpisodesWithFiles=5
        SeasonAvailability[2]: Available=true, HasAnyFiles=true, AiredEpisodesWithFiles=5
```

### Test 5: `TestCheckSeriesAvailability_FullyAvailable`

```
Setup:  Series with 2 seasons, all aired episodes have files, no unaired
Assert: InLibrary = true, CanRequest = false
```

### Test 6: `TestSeasonAvailable_UnairedNotMonitored`

```
Setup:  Season with 5 aired (all with files) + 3 unaired (0 monitored)
Assert: Available = false (unaired episodes not monitored)
```

### Test 7: `TestGetCoveredSeasons_CrossType`

```
Setup:  Create two requests for same tvdbID:
  - "series" request with requested_seasons = [1, 2, 3]
  - "season" request for season_number = 5
Assert: GetCoveredSeasons returns map with keys 1, 2, 3, 5
        Each entry has correct RequestID, UserID, Status
```

### Test 8: `TestCreateRequest_CrossTypeDuplicate`

```
Setup:  Active "series" request for tvdbID=1000 with seasons [1, 2, 3]
Action: Create "season" request for tvdbID=1000, season 2
Assert: Returns ErrAlreadyRequested
Action: Create "season" request for tvdbID=1000, season 4
Assert: Succeeds (no overlap)
```

### Test 9: `TestCreateRequest_PartialOverlap`

```
Setup:  Active "season" request for tvdbID=1000, season 2
Action: Create "series" request for tvdbID=1000 with seasons [2, 3, 4]
Assert: Succeeds (partial overlap — only season 2 overlaps, seasons 3-4 are new)
        The frontend ensures the user deselected season 2; backend allows partial overlap
```

### Frontend test: `formatSeasonRange`

**File:** `web/src/components/search/season-availability-badge.test.ts` (or colocated)

```typescript
test('contiguous range', () => expect(formatSeasonRange([1, 2, 3])).toBe('S1-3'))
test('single season', () => expect(formatSeasonRange([1])).toBe('S1'))
test('non-contiguous', () => expect(formatSeasonRange([1, 3])).toBe('S1, S3'))
test('mixed', () => expect(formatSeasonRange([1, 2, 3, 5])).toBe('S1-3, S5'))
test('exceeds 10 chars with fallback', () => expect(formatSeasonRange([1, 2, 3, 5, 7, 8, 9, 11], 15)).toBe('8/15 seasons'))
test('empty', () => expect(formatSeasonRange([])).toBe(''))
```

---

## Implementation Order & Dependencies

```
Phase 1 (Movie fix)          — DONE
Phase 2 (Series availability) — DONE
Phase 3 (Seasons endpoint)   — DONE
Phase 4 (Cross-type dupes)   — DONE
Phase 5 (Approval flow fix)  — DONE
Phase 6 (Frontend types)     — DONE
Phase 7 (Search splitting)   — DONE
Phase 8 (Availability badge) — DONE
Phase 9 (Card button)        — DONE
Phase 10 (Request modal)     — DONE
Phase 11 (Watch button)      — DONE
Phase 12 (Admin context)     — DONE
```

### Agent execution sequence

```
1. Write all new SQL queries (2.3, 3.3, 4.1) → sqlc generate — DONE
2. Implement Go changes: Phase 1 → 2 → 3 → 4 → 5 → go build ./... — DONE
3. Write backend tests (see Test Specifications) → run them — DONE (9/9 passing)
4. Lint backend — fix only issues you introduced — DONE (0 issues)
5. Implement frontend: Phase 6 → 7 → 8 → 9 → 10 → 11 — DONE
6. Lint frontend — fix only issues you introduced — DONE (0 errors)
7. Launch Phase 12 as background Sonnet subagent — DONE
8. Update this plan: mark phases as complete — DONE
```

---

## Files Modified Summary

### Backend
| File | Changes |
|---|---|
| `internal/portal/requests/library_check.go` | `SeasonAvailability` struct, `CoveredSeason` struct, movie file check, series per-season computation, `GetSeasonAvailabilityMap()`, `GetCoveredSeasons()` |
| `internal/portal/requests/library_check_test.go` | **New file** — tests for availability logic |
| `internal/database/queries/series.sql` | `GetSeriesSeasonAvailabilitySummary` query |
| `internal/database/queries/episodes.sql` | `GetEpisodeAvailabilityForSeason` query |
| `internal/database/queries/requests.sql` | `FindRequestsCoveringSeasons` query |
| `internal/portal/search/handlers.go` | `EnrichedSeasonResult` + `EnrichedEpisodeResult` types, merge library data in `GetSeriesSeasons`, episode-level enrichment |
| `internal/portal/requests/service.go` | Cross-type duplicate check in `checkExistingRequest()` |
| `internal/api/adapters.go` | `EnsureSeriesInLibrary()` monitoring for existing series, `applyMonitorFuture()`, `applyRequestedSeasonsMonitoringAdditive()` |
| `internal/portal/requests/searcher.go` | `MonitorFuture` on `MediaProvisionInput`, set in `ensureMediaInLibrary()` |
| `internal/database/sqlc/*` | Regenerated (sqlc generate) |

### Frontend
| File | Changes |
|---|---|
| `web/src/types/portal.ts` | `SeasonAvailabilityInfo`, `EnrichedEpisode`, `EnrichedSeason` types, add `seasonAvailability` to `AvailabilityInfo` |
| `web/src/routes/requests/use-request-search.ts` | Three-way series splitting, `buildSeriesRequestPayload` mediaType switching, `selectAllSeasons` exclusion of disabled seasons |
| `web/src/routes/requests/search-results-content.tsx` | Render partial series in library section with Request button |
| `web/src/components/search/season-availability-badge.tsx` | **New file** — badge component + `formatSeasonRange()` helper |
| `web/src/components/search/season-availability-badge.test.ts` | **New file** — `formatSeasonRange` unit tests |
| `web/src/components/search/card-action-button.tsx` | `inLibrary && canRequest` case |
| `web/src/hooks/use-external-media-card.ts` | Derive `canRequest` from availability |
| `web/src/components/search/card-poster.tsx` | Show availability badge for partial series |
| `web/src/routes/requests/series-request-dialog.tsx` | Three visual states per season, Watch button, `EnrichedSeason` prop type |

---

## Edge Cases to Test

1. Movie in library, no files, no request → Requestable section, normal button
2. Movie in library, no files, active request → Requestable section, request status + Watch
3. Movie in library, has files → In Library, disabled
4. Series with zero files → Requestable section
5. Series partially available → In Library section with Request button + badge
6. Series fully available → In Library, disabled
7. Currently airing season, all aired eps with files, unaired monitored → Available (disabled)
8. Currently airing season, all aired eps with files, unaired NOT monitored → Requestable (enabled)
9. Series request [1,2,3] exists, user searches → S1-3 disabled as "Requested"
10. All modal seasons disabled, user toggles monitor future → Allowed (submits)
11. Existing series request approved → Monitoring applied to new seasons, no duplicate series
12. Series in library with 5 seasons, metadata reports 7 → seasons 6-7 selectable in modal (default requestable)
13. Single season selected in modal → request created with `mediaType: "season"`, not `"series"`
14. Select All in modal with S1 available, S2 requested → only S3+ selected
15. User already watching an existing request → Watch button hidden or shows "Watching"
