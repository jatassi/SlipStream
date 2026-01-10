# Automatic Release Searching

## Feature Overview

Automatic release searching and fetching allows SlipStream to autonomously download movies and series files. This is useful when users add movies or TV series to the library that are not yet released, or when existing files need quality upgrades.

## Existing Functionality

Most building blocks for automatic search exist already:
- Manual search and release processing
- Item release dates and availability status (to avoid searching for unavailable items)
- Missing items (to know what to search for)
- Release desirability scoring (to know which release to download)
- Task scheduling system (add tasks to periodically search for missing items)
- Monitored status (only search for monitored items) - needs improvement for granularity and bulk operations

## Entry Points

Automatic search is triggered in three ways:

### 1. Search Upon Add
Triggered when adding new movies or series to the library.

#### Movies
- Existing toggle control for "search upon add"
- Wire to automatic search feature
- Remembers last user selection

#### Series
New dropdown for search-upon-add with options:
- **No** - Do not auto-search upon add
- **First Episode Only** - Search only S01E01
- **First Season Only** - Search all available episodes in season 1
- **Latest Season Only** - Search all available episodes in the most recent season
- **All** - Search all available episodes across all seasons

**Behavior:**
- Only searches items that are currently available (past release date)
- Unavailable items get monitored for scheduled task to pick up later
- Same UX as per-item button (background search + toast notification)
- Remembers last user selection

#### Series Monitoring Upon Add
New dropdown replacing the existing monitor toggle:
- **None** - Do not monitor anything (also unmonitors the series itself)
- **First Season Only** - Monitor all episodes in season 1
- **Latest Season Only** - Monitor all episodes in the most recent season
- **Future Seasons Only** - Monitor unreleased seasons and any seasons added later
- **All** - Monitor all seasons and episodes

**Behavior:**
- One-time initial setup only - sets which episodes are monitored when series is added
- Series-level monitoring controls future behavior (if series is monitored, new seasons/episodes are auto-monitored)
- Single-season series with "First Season Only" = that season gets monitored
- Remembers last user selection

#### Specials (Season 0)
- Separate toggle for "Include Specials"
- Controls whether specials are affected by monitoring/search dropdowns
- Remembers last user selection (default off)

#### Dropdown Independence
- Monitoring and search dropdowns are independent - any combination is valid
- Selecting a search option with "None" monitoring is allowed (search happens, items remain unmonitored)

#### Default Values
- Persist last user selection for all options (movies and series)
- Initial defaults (no prior selection): Monitor "Future Seasons Only", Search "No"

### 2. Per-Item Automatic Search Button
Located on:
- Movie detail page
- Series detail page (searches all missing in that series)
- Seasons within series detail page (searches all missing in that season)
- Episodes within series detail page
- Missing items page (same granularity for TV as detail page)

**Behavior:**
- Runs in background immediately when clicked
- Button icon replaced with loading spinner during search
- Toast notification shows result (found/downloaded, not found, error)
- Button is disabled if item is already in download queue

### 3. Scheduled Task for All Missing Items
Runs at a configurable interval to search all available, missing, monitored items.

**Missing Page Triggers:**
- "Search All" button - searches all missing items
- "Search All Movies" button - searches all missing movies
- "Search All Series" button - searches all missing TV content

---

## Core Behavior

### Release Selection
- Always download the release with the highest desirability score
- No minimum threshold required - best available release is selected

### Quality Upgrades
- Automatic search includes items below their quality profile cutoff
- Existing files are automatically replaced silently when upgrades are found

### Availability Criteria
Note: availability logic already implemented, use/extend it
- Items must be past their release date to be eligible for automatic search
- Availability is immediate on release date (no delay buffer)

### Monitored Status
- Only monitored items are searched
- Parent monitored status overrides children (unmonitored series = all seasons/episodes unmonitored)
- **Improvements needed:**
  - Granular monitoring at series/season/episode levels
  - Bulk monitor/unmonitor operations

---

## Scheduled Task Behavior

### Processing Order
1. Items ordered by release date (newest first)
2. For TV series: prioritize boxset downloads (entire series, full seasons) over individual episodes
3. Fall back to individual episode downloads only if the entire season is not yet available

### Rate Limiting
- Items processed sequentially (one at a time)
- Adaptive delay between searches based on indexer response times and rate limit headers

### Schedule Configuration
- Default interval: 1 hour
- Allowed range: 1-24 hours
- Master enable/disable toggle in settings
- If a search task is still running when the next scheduled run triggers, skip that run

### Backoff Mechanism
- After N consecutive failures to find a release for an item, search less frequently
- Default threshold: 12 failures
- Configurable in settings
- Failure count resets when item metadata changes (quality profile, monitored status, etc.)

---

## Conflict Resolution

### Manual vs Scheduled Search Conflict
If user triggers manual automatic search while scheduled task is searching the same item:
- Cancel the scheduled search for that item
- Run the manual search instead

### Manual Add During Automatic Search
If user manually adds a release (via manual search) while automatic search is running for that item:
- Cancel the automatic search
- User's manual selection takes precedence

---

## Download Client Selection

Extends the defaults system used for root folders:
1. Use default download client for the media type (movies or series)
2. If no per-type default, use global default
3. If no global default, fall back to priority attribute

---

## Indexer Configuration

- Per-indexer "Enable for automatic search" toggle
- Only indexers with this flag enabled are queried during automatic search
- Manual search continues to use all enabled indexers

---

## Failure Handling

When automatic search fails (indexer down, no results found, download client error):
- Log the failure
- Retry on next scheduled run
- **TODO:** Integrate with future system health feature to notify users of persistent configuration issues

When download client rejects a release (full disk, connection error, etc.):
- Same behavior as general search failure
- Log and retry on next run
- **TODO:** Integrate with system health feature

---

## History & Logging

Log significant events to the history system:
- Successful downloads
- Quality upgrades
- Failures

Do not log "not found" events to avoid history clutter.

---

## UI Components

### Status Indicator
- Use existing task indicator mechanism (same as library scanning)
- Shows when scheduled automatic search task is running

### Settings Page

#### "Release Searching" Section
- Master enable/disable toggle
- Search interval (1-24 hours)
- Backoff threshold (number of failures before reduced frequency)

#### "Adding Content" Section
Defaults for the add flow (all options remember last user selection):
- **Movies:**
  - Default root folder
  - Default quality profile
  - Default search upon add (toggle)
- **Series:**
  - Default root folder
  - Default quality profile
  - Default monitoring selection (dropdown)
  - Default search upon add selection (dropdown)
  - Default include specials (toggle)

### Missing Page
- "Search All" button
- "Search All Movies" button
- "Search All Series" button

### Per-Item Buttons
Automatic search button on:
- Movie detail page
- Series detail page
- Season detail page
- Episode detail page
- Missing items page (per-item)

Button states:
- Normal: Clickable search icon
- Searching: Loading spinner
- In queue: Disabled (item already has pending download)

---

## Architectural Notes

- Centralize search code to be consumed by both automatic and manual search functions
- Major refactoring of manual search is acceptable to produce a clean, modular design
- Backward compatibility is not important - both manual and automatic search must work when complete
- **This is a core feature of SlipStream, so the code MUST be simple, maintainable, extensible, error-free, and self-documenting.**

---

## Rollout

- Feature is fully available immediately once implemented
- No feature flag or developer mode restriction
- No delay profiles - releases are grabbed as soon as they're found

---

## Future Considerations

- System health integration for persistent failure notifications
- Additional settings in "Release Searching" section as needed
