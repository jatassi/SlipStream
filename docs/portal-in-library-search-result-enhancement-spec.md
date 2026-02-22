# Better Portal Handling for Library Items

## Background

The external requests portal currently treats all library items the same — if a movie or series exists in the database, it shows as "In Library" with a disabled button, regardless of whether any files actually exist. This creates dead ends for portal users who can't request content that technically exists in the library but isn't actually available.

## Goals

- Movies in library with no files should be requestable as if not in library
- Partially-available TV series should be requestable for missing seasons
- Existing requests should be surfaced to prevent duplicates and enable watching
- Admin approval flow should handle requests for existing series correctly

---

## Movie Behavior

### Current
A movie in the library always shows "In Library" (disabled) in the In Library section, regardless of file status.

### New Behavior

| Scenario | Section | Card State |
|---|---|---|
| In library, has files in any slot | In Library | "In Library" disabled button (unchanged) |
| In library, no files in any slot | Requestable | Normal Request button (as if not in library) |
| In library, no files, active request exists (any user) | Requestable | Shows existing request status + Watch button |

**Rule:** A movie is "effectively in library" only if at least one version slot has a file. If zero slots have files, treat it as not in library for portal purposes.

---

## Series Behavior

### Current
A series in the library always shows "In Library" (disabled) in the In Library section, regardless of season/episode file status.

### New Behavior

| Scenario | Section | Card State |
|---|---|---|
| Not in library | Requestable | Normal Request button (unchanged) |
| In library, zero files across all seasons | Requestable | Normal Request button (as if not in library) |
| In library, partially available (some seasons complete) | In Library | Request button + season availability badge |
| In library, all seasons fully available | In Library | "In Library" disabled button (unchanged) |

### Season Availability Definition

A season is **"available"** (fully complete) when ALL of the following are true:
- Every aired episode has files in at least one slot
- AND either: no unaired episodes exist, OR all unaired episodes are monitored

A season is **"requestable"** (enabled in modal) when ANY of the following are true:
- Some aired episodes are missing files
- No episodes have files at all
- Season doesn't exist in library at all
- Currently airing and all aired episodes have files, but unaired episodes are NOT monitored

**Note:** Monitored/unmonitored status only matters for unaired episodes. If all aired episodes have files, the season is available regardless of whether the season itself is monitored.

### Season Availability Badge

Displayed on the search result card for partially-available series (In Library section with Request button).

**Format:**
- Books icon (library indicator) + season text
- Compact season format: `S1-3`, `S1, S3`, `S1-3, S5`
- Contiguous seasons collapsed into ranges: seasons 1, 2, 3 → `S1-3`
- **10 character max** for season text (portal is primarily mobile)
- If season text exceeds 10 characters, fall back to count format: `3/5 seasons`
- The books icon visually indicates "available" — no need for the word "available" in text

**Examples:**
- 1 season available out of 5: `S1` (2 chars)
- Seasons 1-3 available out of 7: `S1-3` (4 chars)
- Seasons 1, 3 available: `S1, S3` (6 chars)
- Seasons 1-3, 5 available: `S1-3, S5` (8 chars)
- Seasons 1-5, 7-9, 11 available out of 15: exceeds 10 chars → `9/15 seasons`

---

## Request Modal (Series)

### Season States in Modal

Each season (except Season 0) displays one of three visual states:

1. **Disabled — Available** (green checkmark indicator)
   - Season meets the "available" definition above
   - Checked and non-interactive
   - Visual: green checkmark or similar "complete" indicator

2. **Disabled — Requested** (yellow clock indicator, "Requested" label)
   - Season is covered by an existing active request (own or another user's)
   - Checked and non-interactive
   - Visual: yellow clock icon, distinct from the green "available" state

3. **Enabled — Selectable** (unchecked checkbox)
   - Season does not meet either disabled condition
   - User can check to include in their request

### Season 0 (Specials)
Excluded from the request modal entirely. Specials are rarely complete and would add noise.

### "Monitor Future Episodes" Toggle
- Always shown in the modal, including for in-library series
- When the request is approved, the toggle value is applied to the series (even existing ones)
- A user can submit a request with only "Monitor future episodes" checked and no seasons selected, as long as the modal is accessible (i.e., the series is not fully available)

### Select All / Deselect All
- Select All only selects **selectable** seasons (state 3 above) — it must skip Available and Requested seasons
- Deselect All clears all selections (only affects selectable seasons since disabled ones are never selected)

### Submission Rules
- Submit button enabled if: at least one season is selected OR "Monitor future episodes" is toggled on
- If no seasons are selected and monitor is on, helper text: "No seasons selected. Series will be added to library and only future episodes will be monitored."

### Request Type Created
- **Single season selected** → `mediaType: "season"` with `seasonNumber` set
- **Multiple seasons selected** → `mediaType: "series"` with `requestedSeasons` array containing only the selected (non-disabled) seasons

---

## Duplicate Request Detection

### Current Behavior
Duplicate detection only checks within the same `mediaType`:
- `GetRequestByTvdbID` checks `media_type = 'series'`
- `GetRequestByTvdbIDAndSeason` checks `media_type = 'season'`

No cross-type overlap detection exists.

### New Behavior: Cross-Type Overlap Detection

When checking for duplicate/overlapping requests, the system must detect overlaps across request types:

1. **Creating a "series" request with requestedSeasons [1,2,3]:**
   - Block if an active "series" request exists for the same TVDB ID with any overlapping seasons
   - Block if an active "season" request exists for season 1, 2, or 3

2. **Creating a "season" request for S3:**
   - Block if an active "series" request exists whose requestedSeasons includes 3
   - Block if an active "season" request exists for S3

3. **Partial blocking in the modal:**
   - Only the overlapping seasons are disabled (shown as "Requested")
   - Non-overlapping seasons remain selectable
   - The user can still create a request for the non-overlapping seasons

**Status filter:** Only active requests block (exclude `denied` and `available` statuses).

### Watch Button for Existing Requests
When a portal user encounters an existing request (via search results or the request modal), show a Watch button so they can subscribe to status updates. The watch system already exists and should be surfaced here. If the user is already watching the request, hide the Watch button or show a "Watching" indicator instead.

---

## Backend Changes

### 1. Movie Availability Check

**File:** `internal/portal/requests/library_check.go` — `CheckMovieAvailability()`

Currently sets `inLibrary = true` if the movie exists in the DB. Must additionally check whether any slot assignment has a file:

```
If movie exists in DB:
  Get slot assignments via getMovieSlots()
  If ANY slot has HasFile = true:
    inLibrary = true
  Else:
    inLibrary = false  (treat as not in library)
```

### 2. Series Availability — Eager Per-Season Computation

**File:** `internal/portal/requests/library_check.go` — `CheckSeriesAvailability()`

Currently only checks if the series exists. Must be enhanced to compute per-season availability data during search (not lazily).

New data to return in `AvailabilityResult` for series:
- `seasonAvailability`: array of per-season status objects, each containing:
  - `seasonNumber`: int
  - `available`: bool (meets the "available" definition)
  - `hasAnyFiles`: bool (at least one episode has a file)
  - `airedEpisodesWithFiles`: int
  - `totalAiredEpisodes`: int
  - `totalEpisodes`: int
  - `monitored`: bool

**Logic for `inLibrary` on series:**
- `inLibrary = true` only if at least one season has `hasAnyFiles = true`
- If all seasons have zero files, `inLibrary = false`

**Logic for `canRequest` on series:**
- `true` if any season is not "available" (i.e., requestable)
- `false` if all seasons are fully available

### 3. Enhanced Seasons Endpoint

**Endpoint:** `GET /api/v1/requests/search/series/seasons`

Currently returns only TMDB/TVDB metadata. Enhance to merge library data inline when the series exists in the library.

Each season object in the response gets additional fields:
- `inLibrary`: bool — season exists in library DB
- `available`: bool — meets the "available" definition
- `monitored`: bool — season monitoring status
- `airedEpisodesWithFiles`: int — count of aired episodes that have files
- `totalAiredEpisodes`: int — count of aired episodes
- `episodeCount`: int — total episodes (aired + unaired)

Each episode object (if included) gets:
- `hasFile`: bool
- `monitored`: bool
- `aired`: bool

### 4. Cross-Type Duplicate Detection Queries

New SQL queries needed:

- `FindOverlappingSeriesRequest`: Given a TVDB ID and list of season numbers, find any active "series" request whose `requestedSeasons` JSON array overlaps with the given seasons
- `FindOverlappingSeasonRequests`: Given a TVDB ID and list of season numbers, find any active "season" requests matching those season numbers
- `FindRequestsCoveringSeasons`: Given a TVDB ID, return all active requests (both "series" and "season" types) with their covered seasons, for display in the modal

### 5. Approval Flow Fix — Existing Series Monitoring

**File:** `internal/api/adapters.go` — `EnsureSeriesInLibrary()`

Currently, when a series already exists, it returns the ID immediately without applying monitoring changes. Fix:

```
If series already exists:
  Return existing series ID
  BUT ALSO apply requested seasons monitoring:
    - Monitor the requested seasons
    - If monitorFuture is true, update series monitoring for future episodes
```

This ensures that when a portal user requests S4-S5 of an existing series (with S1-S3), those new seasons get monitored and searched.

### 6. Monitor Future Episodes — Apply on Approval

When a request with `monitorFuture = true` is approved for an existing series:
- Update the series' monitoring configuration to include future episodes
- This applies regardless of whether the series was newly created or already existed

---

## Frontend Changes

### 1. Search Result Splitting Logic

**File:** `web/src/routes/requests/use-request-search.ts`

Update the result splitting logic:

**Movies:**
- `requestableMovies`: movies where `!availability?.inLibrary` (unchanged logic, but backend now returns `inLibrary = false` for movies with no files)

**Series:**
- `librarySeries`: series where `availability?.inLibrary && !availability?.canRequest` (fully available)
- `partialSeries`: series where `availability?.inLibrary && availability?.canRequest` (partially available — In Library section with Request button)
- `requestableSeries`: series where `!availability?.inLibrary` (not in library at all)

### 2. Season Availability Badge Component

New component or extension of `StatusBadge` for search result cards showing partially-available series.

- Display: books icon + season text
- Positioned on the card (alongside or replacing the "In Library" badge for partial series)
- Implements the 10-character truncation rule with count fallback

### 3. Card Action Button Changes

**File:** `web/src/components/search/card-action-button.tsx`

Update priority logic:
```
if (hasActiveDownload) → DownloadProgressBar
if (isInLibrary && !canRequest) → "In Library" disabled button
if (isInLibrary && canRequest) → Request button (for partial series)
if (hasExistingRequest) → ExistingRequestButton (with Watch option)
→ Request button
```

### 4. Request Modal Enhancements

**File:** `web/src/routes/requests/series-request-dialog.tsx`

- Fetch season availability when modal opens (from enhanced seasons endpoint)
- Render three visual states per season: available (green), requested (yellow), selectable (checkbox)
- Fetch overlapping requests to determine which seasons are "Requested"
- Exclude Season 0 from the list
- Determine disabled state using availability data + existing requests

### 5. Watch Button Integration

When showing an existing request (on card or in modal), include a Watch button if the user is not already watching.

### 6. Types Update

**File:** `web/src/types/portal.ts`

Add to `AvailabilityInfo`:
```typescript
seasonAvailability?: SeasonAvailabilityInfo[]
```

New type:
```typescript
type SeasonAvailabilityInfo = {
  seasonNumber: number
  available: boolean
  hasAnyFiles: boolean
  airedEpisodesWithFiles: number
  totalAiredEpisodes: number
  totalEpisodes: number
  monitored: boolean
}
```

---

## Admin Changes

### Request Detail — Library Context

When viewing a request for a series that already exists in the library:
- Show an "Already in library" indicator on the request card/detail view
- Include a link to the series detail page in the admin library view
- This helps admins make informed approval decisions for partial series requests

---

## Edge Cases Summary

| Scenario | Behavior |
|---|---|
| Movie in library, no files, no request | Requestable section, normal Request button |
| Movie in library, no files, active request from another user | Requestable section, shows request status + Watch button |
| Movie in library, has files | In Library, disabled button (unchanged) |
| Series in library, zero files anywhere | Requestable section (as if not in library) |
| Series in library, some seasons complete | In Library section, Request button + badge |
| Series in library, all seasons complete | In Library, disabled button (unchanged) |
| Series not in library | Requestable section (unchanged) |
| Airing season, all aired eps have files, unaired monitored | Disabled in modal (available) |
| Airing season, all aired eps have files, unaired NOT monitored | Enabled in modal (requestable) |
| Season fully downloaded, season unmonitored | Disabled in modal (files exist, unmonitored status irrelevant) |
| User's own request covers S1-S3, searches again | S1-S3 disabled as "Requested" in modal, can request S4+ |
| Another user's request covers S1-S3 | S1-S3 disabled as "Requested" in modal, can request S4+ |
| "Series" request with seasons [1,2,3] exists | "Season" request for S2 blocked, S4 allowed |
| All seasons disabled but user toggles monitor future | Allowed (creates request with no seasons, monitor flag only) |
| Fully available series (all disabled, no Request button) | No way to create monitor-only request — monitoring is admin domain |
| Series request for existing library entry approved | Monitoring applied to requested seasons, no duplicate series created |
| Series in library with 5 seasons, metadata reports 7 seasons | Partially available — seasons 6-7 selectable in modal, `canRequest = true` |
| Single season selected in modal | Request created with `mediaType: "season"` + `seasonNumber`, NOT `mediaType: "series"` |
| Select All clicked with S1 available, S2 requested | Only S3+ get selected — disabled seasons excluded |
| User already watching an existing request | Watch button hidden or shows "Watching" indicator |
