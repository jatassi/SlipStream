# SlipStream v1 user-facing feature inventory

Source: `~/Git/SlipStream-v1` (paths below are relative to that root). Scope: admin web UI, requests portal, external API surface. iOS repo ignored.
Format: one line per feature, user-observable, with v1 paths. `[HALF]` marks half-built or suspicious behavior. `[DEV]` marks dev-build-only.

## 0. Route map (TanStack Router, `web/src/routes-config.tsx`, `web/src/router.tsx`)

| Path | Page | Notes |
|---|---|---|
| `/auth/setup` | First-run admin setup | public |
| `/` | Dashboard | |
| `/search?q=` | Global search results | |
| `/movies`, `/movies/$id`, `/movies/add` | Movie library, detail, add | module-driven |
| `/series`, `/series/$id`, `/series/add` | Series library, detail, add | module-driven |
| `/calendar` | Calendar | |
| `/missing` | Missing / upgradable (wanted) | |
| `/downloads` | Activity (download queue) | |
| `/history` | History | |
| `/import?path=` | Manual import | |
| `/more` | Phone "More" tab hub | phone shell |
| `/settings` -> `/settings/media` | redirect | |
| `/settings/media{,/root-folders,/quality-profiles,/version-slots,/file-naming,/arr-import}` | Media settings | |
| `/settings/download-pipeline{,/indexers,/clients,/auto-search,/rss-sync}` | Download pipeline settings | |
| `/settings/general{,/server,/authentication,/notifications}` | General settings | |
| `/requests-admin` -> `/requests-admin/queue`; `/requests-admin/users`; `/requests-admin/settings` | Admin side of requests portal | |
| `/system` -> `/system/health`; `/system/tasks`; `/system/logs`; `/system/update` | System | |
| `/dev/controls` | Control showcase | [DEV] |
| `/requests/auth/login`, `/requests/auth/signup?token=` | Portal (and admin) login, invite signup | public |
| `/requests`, `/requests/search?q=`, `/requests/library`, `/requests/$id`, `/requests/settings` | Portal pages | behind `PortalAuthGuard` |

## 1. App shell and global chrome

- Two shells chosen by viewport: wide (sidebar + header) and phone (bottom tab bar, push-navigation stack, "More" tab). `web/src/components/layout/shell-layout.tsx`, `wide-shell.tsx`, `phone-shell.tsx`, `tab-bar.tsx`, `use-tab-nav.ts`, `push-routes.ts`, `use-push-*.ts`, `web/src/hooks/use-viewport.ts`.
- Sidebar nav: Library entries generated from enabled modules (Movies, Series, each with theme color), Discover (Calendar, Requests, Missing, Manual Import, History), collapsible Settings group (Media, Download Pipeline, General), System, plus Log out and Restart actions. `web/src/components/layout/sidebar-nav-config.ts`, `sidebar.tsx`, `sidebar-collapsible-nav.tsx`.
- Sidebar Activity (downloads) nav link shows live download progress overlay/count. `web/src/components/layout/downloads-nav-link.tsx`, `downloads-progress-overlay.tsx`, `use-downloads-nav.ts`.
- Phone tab bar: Dashboard, Library (remembers last library module), Activity (badge = downloading count), Search, More. `web/src/components/layout/tab-bar.tsx`, `use-tab-nav.ts`, `web/src/stores/ui.ts`.
- More page (phone): Discover links (Missing row shows movie/episode missing counts), Settings links, System, Developer Tools group [DEV], Log out, Restart, version footer. `web/src/routes/more/index.tsx`.
- Header global search box: debounced (500 ms) or Enter navigates to `/search?q=`. `web/src/components/search/search-bar.tsx`.
- Header running-tasks pill: spinner + task name or "N tasks", tooltip lists running scheduled tasks. `web/src/components/layout/header-running-tasks.tsx`.
- Header activity indicator: hover card of live progress activities (in-progress/pending/recent), each dismissible. `web/src/components/layout/header-activity-indicator.tsx`, `web/src/components/progress/`, `web/src/stores/progress.ts`.
- Header notifications bell: badge count, dropdown of last 5 in-app notifications, click dismisses. `web/src/components/layout/header-notifications.tsx`, `web/src/stores/ui.ts`. [HALF] notifications are a client-side store only; no persistence or history.
- Developer mode toggle [DEV]: header hammer popover (wide) / More page group (phone); only rendered when `status.isDevBuild`. Toggling sends `devmode:set` over WebSocket; server switches to a dev DB with mock data (see section 10). "Force Loading" switch forces every screen into its loading skeleton. `web/src/components/layout/header-dev-mode*.tsx`, `dev-mode-controls.tsx`, `web/src/stores/devmode.ts`, `internal/api/devmode.go`, `docs/developer-mode-testing.md`. [HALF] the wide header popover shows only Force Loading; the Developer-mode switch itself is in `HeaderDevMode`/More page.
- Restart action: confirm dialog, calls restart API, 5 s countdown then page reload. Log out: confirm, clears session, goes to login. `web/src/components/layout/use-session-actions.ts`, `session-action-presenters.tsx`.
- Theme: light / dark / system, stored in UI store, applied to `<html>`. `web/src/components/layout/use-layout-effects.ts` (`useThemeEffect`), `web/src/stores/ui.ts`. [HALF] no theme picker exists anywhere in the UI (`setTheme` has no caller outside the store), so users always get the default.
- Live updates: admin WebSocket connected only when authenticated admin; reconnects on tab visibility/pageshow; drives queue, progress, task, health, library refresh. `web/src/stores/websocket.ts`, `ws-connection.ts`, `ws-message-handlers.ts`, `internal/websocket/`.
- Toasts: bottom-right (wide), bottom-center above tab bar (phone). `web/src/components/layout/shell-layout.tsx`.
- Document title tracks route. `web/src/hooks/use-document-title.ts`.
- Auth gating: any non-portal route redirects to `/auth/setup` if no admin exists, else to `/requests/auth/login` (admin and portal share one login page); remembers redirect URL; global `auth:unauthorized` event logs out. `web/src/components/layout/use-layout-effects.ts`.
- Error boundary per route. `web/src/components/error-boundary.tsx`.

## 2. Admin pages

### 2.1 First-run setup (`/auth/setup`)
- Create the single "Administrator" account with a 4-digit masked PIN; username fixed. `web/src/routes/auth/setup.tsx`, `internal/auth/service.go`, `internal/api/auth_handlers.go`.

### 2.2 Dashboard (`/`)
- Four groups: Downloading (live queue rows with thumbnail + speed/ETA stats, "See all" -> `/downloads`, empty state), Health (current issues or "All systems healthy"), Recent (last 5 history events), Storage (per-disk/root-folder usage). `web/src/routes/index.tsx`, `web/src/components/dashboard/*.tsx`.

### 2.3 Global search (`/search?q=`)
- Empty query shows hint "Search N titles across movies and series". `web/src/routes/search/index.tsx`.
- Instant client-side title match across library movies and series, sorted A-Z, poster rows labelled Movie/Series, tap opens detail. `web/src/routes/search/use-search-page.ts`, `search-groups.tsx`.
- "Add New" section (query >= 2 chars): online TMDB movie + series search; auto-opens when no library hit, otherwise a "Search online for ..." row reveals it; each result flagged in-library or opens `/movies/add?tmdbId=` / `/series/add?tmdbId=`. `web/src/routes/search/search-groups.tsx`, `web/src/hooks/use-metadata.ts`, `internal/metadata/`.

### 2.4 Library lists (`/movies`, `/series`)
- Grid (poster) or table view; poster size Small/Medium/Large; table column chooser (movies: Title, Year, Studio, Status, Quality Profile, Root Folder, Release Date, Added, Size, Runtime, Path; series: Title, Network, Seasons, Episodes, Production, Quality Profile, Root Folder, Next Airing, Added, Size, Path). View prefs persisted in UI store. `web/src/components/media/library-options.tsx`, `library-screen.tsx`, `web/src/lib/movie-columns.tsx`, `series-columns.tsx`, `web/src/stores/ui.ts`.
- Sort by Title, Monitored, Quality Profile, Release Date (movies) / Next Air Date (series), Date Added, Root Folder, Size; reverse order; clickable table headers; sort persisted. `web/src/routes/movies/use-movie-list.ts`, `web/src/routes/series/use-series-list.ts`.
- Status filters (multi-select): movies Monitored, Unreleased, Missing, Downloading, Failed, Upgradable, Available; series adds Continuing, Ended. `web/src/routes/movies/movie-list-utils.ts`, `web/src/routes/series/series-list-utils.ts`.
- Module segmented control (Movies / Series) at top of Library when 2+ modules enabled. `web/src/components/media/library-screen.tsx`.
- List is sectioned by the active sort field (none for Title): monitored, quality profile, root folder, release date, next air date (TBA / Aired / Today / Tomorrow / weekday / Next Week / Later This Month / Next Month), date added, size. `web/src/lib/grouping.ts`.
- Per-row actions in table: Search (auto search), Delete. `web/src/lib/movie-columns.tsx`, `series-columns.tsx`.
- "Refresh all" metadata for the module. `use-movie-list.ts` / `use-series-list.ts` (`useRefreshAll*`).
- Edit/select mode: select/deselect all, bulk Monitor, Unmonitor, Set quality profile, Delete (with "also delete files" checkbox). `web/src/components/media/library-edit-bar.tsx`, `media-delete-dialog.tsx`.
- Poster status pill/dot (downloading progress, missing, available, etc). `web/src/components/media/status-pill.tsx`, `status-dot.tsx`, `media-status.ts`, `poster-cell.tsx`.
- Add button navigates to search to find new titles. `web/src/components/media/library-screen.tsx`.

### 2.5 Add movie / add series (`/movies/add?tmdbId=`, `/series/add?tmdbId=`)
- Reached from search results only (no tmdbId -> redirect back). Shows preview (poster, title, year, overview, network). `web/src/routes/movies/add.tsx`, `web/src/routes/series/add.tsx`, `web/src/components/media/add-media-configure.tsx`.
- Movie options: Root Folder*, Quality Profile*, Monitored, Search on Add. Series options: Root Folder*, Quality Profile*, Monitor (All Episodes / Future Only / First Season / Latest Season / None), Search on Add (Don't / First Episode / First Season / Latest Season / All Monitored), Season Folder, Include Specials. `web/src/components/media/media-configure-fields.tsx`, `series-form-constants.ts`.
- Remembers last-used add-flow choices server-side (`GET/PUT /preferences/addflow`) and default root folder. `web/src/api/preferences.ts`, `internal/preferences/`, `web/src/routes/movies/use-add-movie.ts`.
- After add: toast, navigate to the new detail page. `use-add-movie.ts`, `use-add-series.ts`.

### 2.6 Movie detail (`/movies/$id`)
- Hero: backdrop, title treatment/logo, status, year, runtime, genres, IMDb rating (extended TMDB/OMDb metadata). `web/src/routes/movies/$id.tsx`, `web/src/components/media/media-detail-hero.tsx`, `title-treatment.tsx`, `backdrop-image.tsx`.
- Pill action row: Monitor toggle, Auto Search (one-click, shows Searching / download progress / Completed / Error states inline), Manual Search (opens release search sheet). `web/src/components/search/media-search-monitor-controls.tsx`, `use-media-search-monitor.ts`, `searching-state.tsx`, `progress-state.tsx`, `completed-state.tsx`, `error-state.tsx`.
- Overflow menu: Edit (quality profile, monitored), Refresh Metadata, Delete (confirm). `web/src/components/media/media-detail-menu.tsx`, `media-edit-dialog.tsx`.
- Overview text; File group (quality profile, size on disk, file list with file name and media info detail). `web/src/routes/movies/movie-detail-groups.tsx`.
- Multi-version slots (when enabled): per-slot row with current quality or Empty, per-slot monitor toggle, per-slot search controls, reassign a file to a different slot. `web/src/routes/movies/movie-detail-groups.tsx`, `movie-detail-handlers.ts`, `web/src/components/slots/`.
- Details group: Year, Runtime, Studio, Director, Content Rating, Genres, Added, Added By (portal user who requested), Path. Credits strip (cast with photos). `movie-detail-groups.tsx`, `movie-detail-credits.tsx`.

### 2.7 Series detail (`/series/$id`)
- Same hero, pill row (monitor/auto/manual search for whole series), overflow menu (Edit incl. quality profile + monitored, Refresh Metadata, Delete) as movies. `web/src/routes/series/$id.tsx`.
- File group: quality profile, episode status summary counts, size on disk, Season Folders yes/no, enabled Version Slots. Details: Status (production), Year, Runtime, Network, Creator, Content Rating, Genres, First Aired, Next Airing, Added, Added By, Path. `web/src/routes/series/series-detail-groups.tsx`.
- Season list: expandable seasons; per-season monitor toggle and season auto/manual search (season pack); per-episode row with number/title/air date/status, per-episode monitor toggle, per-episode auto/manual search. `web/src/components/series/season-list.tsx`, `web/src/routes/series/use-series-detail.ts`.
- Per-episode version-slot breakdown (status Failed/Downloading/Missing/Upgrade/OK/Unreleased per slot, per-slot monitor, per-slot search, assign file to slot). `web/src/components/series/episode-slot-row.tsx`, `episode-slot-status-content.tsx`, `episode-slot-assign.tsx`.

### 2.8 Manual (interactive) release search sheet
- Opens from any movie/series/season/episode/slot search control; title reflects scope (e.g. "Show S01E02"). `web/src/components/search/search-modal.tsx`, `use-search-modal.ts`, `search-modal-utils.ts`.
- Editable query bar and re-search. `search-input-bar.tsx`.
- Results table sortable by Title, Score (default desc), Quality, Slot (when multi-version), Indexer, Size, Age, Peers (torrents only); protocol badge, Freeleech badge, release info link, Download (grab) button per row with spinner. `search-results-table.tsx`, `search-release-row.tsx`, `release-slot-cell.tsx`.
- Footer: total results, indexers searched, per-indexer error list; destructive alert when any indexer errored. `search-footer.tsx`.
- Grab result toasts ("Grabbed ...", or server error text). `use-search-modal.ts`.

### 2.9 Calendar (`/calendar`)
- Month view (prev/next month, Today button, tap a day to see its events) and List view (next 30 days, grouped by day). `web/src/routes/calendar/index.tsx`, `use-calendar-page.ts`, `web/src/components/calendar/calendar-month-view.tsx`, `calendar-list-view.tsx`, `day-grouping.ts`.
- Events: movie Digital release, movie Bluray (physical) release, episode air date; episodes from one season on one day collapse to "Season N · X episodes"; status missing/available/downloading, monitored flag, module color; tap goes to movie/series detail. `web/src/components/calendar/event-presentation.ts`, `internal/calendar/service.go`.
- [HALF] backend emits a `theatrical` event type that the frontend type and labels do not model. `internal/calendar/service.go:18` vs `web/src/types/calendar.ts`.
- No iCal/ICS feed exists in v1 (searched `internal/calendar`, `internal/api/routes.go`).

### 2.10 Missing / Wanted (`/missing`)
- Module switch (Movies / Series) and view switch (Missing / Upgradable). `web/src/routes/missing/index.tsx`.
- Missing movies list (poster, year, profile) and missing series list (series with N missing episodes); upgradable lists (below profile cutoff). Each row links to detail and has inline monitor/auto/manual search controls. `web/src/components/missing/*.tsx`, `internal/missing/`.
- "Search All" button runs a bulk auto search for the current module+view; result toasts (downloaded N / found but none downloaded / failed N / no releases); warns if a search task is already running. `web/src/routes/missing/use-missing-page.ts`, `web/src/stores/autosearch.ts`, `internal/autosearch/`.
- Missing counts (movies | episodes) surfaced on More page. `web/src/routes/more/index.tsx`.

### 2.11 Activity / Downloads queue (`/downloads`)
- Live queue across all download clients (WebSocket + polling), filter All / Movies / Series, sorted; empty state. `web/src/routes/downloads/index.tsx`, `use-activity-page.ts`, `web/src/hooks/use-queue.ts`.
- Per item: poster thumbnail, title, release name, progress bar, speed/ETA/size, status (queued, downloading, paused, completed, failed, warning), quality/source/codec/HDR, target slot. `web/src/routes/downloads/queue-row.tsx`, `web/src/types/queue.ts`.
- Actions: Pause / Resume, Remove from queue, Remove and delete files (confirm), Fast forward [DEV, mock client only]. Tap opens the media detail. `web/src/routes/downloads/use-queue-row.ts`.
- Per-client error banner "Unable to reach <client> - showing last known data". `web/src/routes/downloads/index.tsx`.

### 2.12 History (`/history`)
- Day-grouped event list with colored tile per event (Imported, Grabbed, Upgraded, Failed...) and details text; Load more pagination. `web/src/routes/history/history.tsx`, `history-groups.tsx`, `web/src/lib/history-utils.ts`.
- Filters: media (All / Movies / Series), event types multi-select (Auto Download, Auto Failed, Imported, Import Failed, File Renamed, Slot Assigned/Reassigned/Unassigned, Status Changed), date preset (All Time, Today, 7/30/90 days). `web/src/routes/history/history-filters.tsx`, `history-utils.ts`.
- Clear history (confirm). `web/src/routes/history/use-history-page.ts`, `internal/history/`.
- Retention setting lives under General > Server (see 2.17). `web/src/routes/settings/general/use-history-retention.ts`.
- [HALF] event types `grabbed`, `deleted`, `failed` exist in `web/src/types/history.ts` but are not offered as filters.

### 2.13 Manual import (`/import?path=`)
- Server filesystem browser starting at drives/roots (Windows drive letters handled), folders then video files, back navigation, scroll position kept per folder. `web/src/routes/import/index.tsx`, `use-import-browser.ts`, `browser-groups.tsx`, `use-list-scroll.ts`, `internal/filesystem/`.
- Each video file auto-scanned/parsed and matched to a library item; row shows match label; one-tap Import; "Import all" for every matched file in the folder. `web/src/routes/import/match-label.ts`, `use-import-actions.ts`, `internal/import/`.
- Edit match dialog: parsed info panel (title, year, season, episode/range, quality, source, codec, audio), pick movie or series + episode, optional target version slot ("Auto-assign (recommended)"), Import. `web/src/routes/import/edit-match-dialog.tsx`, `match-to-library-form.tsx`, `parsed-info-panel.tsx`.
- Pending Imports group at root: queued/in-progress/failed automatic imports with error text, tap failed to Retry. `web/src/routes/import/pending-imports-group.tsx`.

### 2.14 Settings > Media (`/settings/media`)
Hub list: Root Folders, Quality Profiles, Version Slots, Import & Naming, Migrate from *arr; settings hub rows can show health warnings. `web/src/routes/settings/media/index.tsx`, `web/src/routes/settings/settings-section-screen.tsx`, `web/src/routes/settings/use-health-warnings.ts`.

Root Folders (`/settings/media/root-folders`)
- List folders with name, path, free space, "Default" badge; add (path with server folder browser, optional name, media type Movies / TV Shows); delete (confirm); set / clear default per media type. `web/src/routes/settings/media/root-folders.tsx`, `root-folder-dialog.tsx`, `use-root-folders-page.ts`, `internal/library/rootfolder/` (backend).

Quality Profiles (`/settings/media/quality-profiles`)
- List by module (Movie/TV) with summary; add, edit, delete (confirm). `web/src/routes/settings/media/quality-profiles.tsx`.
- Profile editor: Module Type, Name, Allowed Qualities checklist (resolution x source tiers) with "Cinema Source" (CAM/TS) opt-in, Upgrades on/off, Upgrade Strategy (Balanced / Aggressive / Resolution Only), Cutoff quality, "Cutoff Overrides Strategy", "Allow Auto-Approve" (portal requests using this profile may auto-approve), attribute filters for HDR Format, Video Codec, Audio Codec, Audio Channels each with mode Required / Preferred / Acceptable / Not Allowed, plus a live upgrade-strategy preview listing allowed/blocked upgrade scenarios. `web/src/components/qualityprofiles/*.tsx`, `constants.ts`, `upgrade-scenarios.ts`, `upgrade-strategy-preview.tsx`.

Version Slots / multi-version (`/settings/media/version-slots`)
- Explainer card for multi-version mode (keep e.g. 4K HDR + 1080p SDR side by side), master enable toggle gated behind a "Dry Run". `web/src/components/settings/sections/master-toggle-card.tsx`, `docs/multiple-quality-versions-spec.md`.
- Up to 3 global slots; each slot: enabled toggle, name, assigned quality profile. `web/src/routes/settings/media/version-slots.tsx`, `slot-edit-dialog.tsx`, `web/src/components/settings/sections/slot-card.tsx`, `use-version-slots-section.ts`.
- Validation alerts: slot profiles must be mutually exclusive (quality tiers, HDR, codec, audio); "Resolve Profile Conflicts" modal edits conflicting profiles with highlights; naming formats must include differentiator tokens, "Resolve Naming Format Issues" modal edits movie/episode formats with a token builder. `web/src/components/settings/sections/config-validation-alert.tsx`, `naming-validation-alert.tsx`, `web/src/components/slots/resolve-config-modal.tsx`, `resolve-naming-modal.tsx`, `token-builder-dialog.tsx`, `token-reference-list.tsx`.
- Migration dry run preview: per movie/series list of existing files and which slot each would be assigned to (Total / Will Be Assigned / Ignored), filters, manual "Assign to Slot", then "Enable Multi-Version Mode" confirm. `web/src/components/slots/DryRunModal/`, `web/src/components/slots/shared/`, `web/src/routes/settings/media/slot-migration-modals.tsx`.
- Debug Tools panel [developer mode only]: Parse Release Title tester (title, year, resolution, source, codec, HDR, audio, channels) and Profile Matching tester. `web/src/components/slots/slot-debug-panel.tsx`, `parse-release-tester.tsx`, `profile-match-tester.tsx`.

Import & Naming (`/settings/media/file-naming`)
- Tabs: Validation, Matching, one "<Module> Naming" tab per module, Token Reference; import tabs auto-save with save-status indicator. `web/src/components/settings/sections/file-naming-section.tsx`, `use-file-naming-section.ts`.
- Validation: Validation Level (Basic: exists + size>0 / Standard: min size + extension / Full: + MediaInfo probe), Minimum File Size, Allowed Video Extensions add/remove, MediaInfo availability status. `naming-validation-tab.tsx`, `extension-manager.tsx`, `media-info-status.tsx`, `internal/mediainfo/`.
- Matching: Match Conflict Behavior (Trust Queue / Trust Parse / Fail with Warning), Unknown Media Handling (Ignore / Auto Add). `naming-matching-tab.tsx`, `file-naming-constants.ts`. [HALF] verify "Auto Add" is wired in `internal/import/`.
- Movie naming: Rename Movies toggle, Movie Folder Format, Movie File Format, live preview + token breakdown, filename parse tester. `naming-movie-tab.tsx`, `naming-pattern-editor.tsx`, `filename-tester.tsx`.
- TV naming: Rename Episodes toggle, Standard / Daily / Anime Episode Formats, Series / Season / Specials Folder Formats, Colon Replacement (Delete, Replace with Dash, Space Dash, Space Dash Space, Smart Replace, Custom + custom char), Multi-Episode Style (Extend, Duplicate, Repeat, Scene, Range, Prefixed Range), parse tester. `naming-tv-tab.tsx`.
- Token Reference: token list, separator modifiers, truncation modifiers. `naming-token-reference-tab.tsx`.
- Token builder dialog: click-to-insert tokens, recommended tokens highlighted. `web/src/components/settings/sections/token-builder-dialog.tsx`.

Migrate from *arr (`/settings/media/arr-import`)
- 4-step wizard: Connect (source Radarr or Sonarr; SQLite DB path with auto-detect + folder browser, or URL + API key), Map Folders & Profiles (source root folder -> SlipStream root folder, source quality profile -> SlipStream profile), Select Items (preview with Ready to Migrate / Already in Library / Unknown filters and selection), Import (live progress, created/skipped/errored counts for movies and series, error list). `web/src/components/arr-import/*.tsx`, `web/src/api/arr-import.ts`, `internal/arrimport/`.
- [HALF] Config import from *arr (download clients, indexers, notifications, quality profiles, naming) exists backend-only: `GET /arrimport/config/preview`, `POST /arrimport/config/import` with no UI. `internal/arrimport/handlers.go:27-28`, `docs/config-import-plan.md`.

### 2.15 Settings > Download Pipeline (`/settings/download-pipeline`)
Hub list: Indexers, Download Clients, Auto Search, RSS Sync. `web/src/routes/settings/download-pipeline/index.tsx`.

Indexers (`/settings/download-pipeline/indexers`)
- Indexer Mode switch: "SlipStream Indexers" (built-in Cardigann definitions) vs "Prowlarr" (external); note shown when developer mode may override mode. `web/src/components/indexers/indexer-mode-toggle.tsx`, `internal/indexer/`, `internal/prowlarr/`.
- Built-in mode: list indexers (enable toggle per row, Test, Delete, row shows supported Movies/TV). Add dialog step 1: searchable Cardigann definition table with protocol / privacy (public, semi-private, private) filters and description; step 2: Name, dynamic per-definition settings form (text, password, checkbox, select), Movies / TV Shows toggles, Priority, Enabled, Enable for Automatic Search, Enable for RSS Sync, Test connection. `web/src/routes/settings/download-pipeline/indexers.tsx`, `use-indexers-page.ts`, `web/src/components/indexers/indexer-dialog.tsx`, `definition-search-table.tsx`, `configure-step.tsx`, `dynamic-settings-form.tsx`, `use-indexer-dialog.ts`.
- Prowlarr mode: config form (Host, Port, Use SSL, API Key, Timeout seconds, Movie Categories, TV Categories), Test, Save, Refresh; Prowlarr indexer list (privacy, protocol, status healthy/failing/disabled, last failure reason); per-indexer settings dialog: Priority 1-50, Content Type (Both / Movies Only / Series Only), statistics (successful/failed counts) with Reset. `web/src/components/indexers/prowlarr-config-form.tsx`, `use-prowlarr-config-form.ts`, `prowlarr-indexer-list.tsx`, `indexer-settings-dialog.tsx`, `docs/Prowlarr-integration-spec.md`.
- Indexer health/failure tracking feeds the System Health page. `internal/health/`.

Download Clients (`/settings/download-pipeline/clients`)
- List clients (enable toggle, Test, Delete). Dialog: Client Type, Name, Host, Port (defaulted per type), Use SSL, URL Base, Username/Password or API key/secret token depending on type, Priority (1-100, lower = higher), Enabled, Category; Test connection; "Add mock download for testing" debug action. `web/src/routes/settings/download-pipeline/clients.tsx`, `use-download-clients-page.ts`, `web/src/components/downloadclients/download-client-dialog.tsx`, `use-download-client-dialog.ts`.
- Supported types (UI): Transmission, qBittorrent, Deluge, rTorrent, Vuze, Flood, Aria2, uTorrent, Hadouken, Synology Download Station, Freebox Download, rqbit, Tribler, Mock [DEV]. `internal/downloader/*/`, `docs/torrent-client-parity-plan.md`.
- [HALF] SABnzbd (usenet) client exists in backend factory but is not selectable in the UI; usenet is effectively unsupported. `internal/downloader/sabnzbd/client.go`, `internal/downloader/factory.go:41`.

Auto Search (`/settings/download-pipeline/auto-search`)
- Enable Automatic Search, Search Interval (1-24 h), Backoff Threshold (consecutive failures before an item is searched less often; default 12). `web/src/components/settings/sections/auto-search-section.tsx`, `internal/autosearch/`, `docs/automatic-search-spec.md`.

RSS Sync (`/settings/download-pipeline/rss-sync`)
- Enable RSS Sync, Sync Interval (10-120 min), Run Now; Last Sync status (last run, error, releases, matched, grabbed, elapsed). `web/src/components/settings/sections/rss-sync-section.tsx`, `internal/rsssync/`, `docs/RSS-Sync-Design.md`.

### 2.16 Settings > General (`/settings/general`)
Hub list: Server, Authentication, Notifications. `web/src/routes/settings/general/index.tsx`.

Server (`/settings/general/server`)
- Network: Port. External Access toggle (localhost-only vs LAN), with warning confirm dialog; Firewall Status panel (firewall name, enabled, whether port allowed, Check button, rule hint). `web/src/components/settings/sections/server-section.tsx`, `firewall-status-panel.tsx`, `use-server-section.ts`, `web/src/routes/settings/general/use-server-page.ts`, `internal/firewall/`.
- Logging: Log Level (Trace / Debug / Info / Warn / Error), Copy Log Path to clipboard. Log Rotation: Max File Size (MB), Max Backup Files, Max Age (days), Compress Old Logs. `server-section.tsx`, `internal/logger/`.
- History Retention: auto-cleanup toggle, retention period (days); cleanup runs daily at 2 AM. `web/src/routes/settings/general/server.tsx`, `use-history-retention.ts`.
- Port footer says restart required; if configured port was busy, shows the actual fallback port in use. `server-section.tsx:50`.

Authentication (`/settings/general/authentication`)
- Change admin PIN. `web/src/components/settings/sections/authentication-section.tsx`, `web/src/components/portal/change-pin-dialog.tsx`.
- Passkey Relying Party config: Default RP ID (bare host), Allowed Origins (one per line; RP ID derived per request), Display Name; "Restart required" banner after save. `web/src/components/settings/sections/webauthn-rp-config.tsx`, `internal/api/webauthn_settings.go`.
- Admin passkey manager (register, rename, delete passkeys) when browser supports WebAuthn. `web/src/components/portal/passkey-manager.tsx`.

Notifications (`/settings/general/notifications`)
- List channels (enable toggle, Test, Delete); add/edit dialog: Type, Name, Enabled, provider-specific schema-driven fields (incl. "Advanced" fields), Event Triggers grouped toggles. `web/src/routes/settings/general/notifications.tsx`, `use-notifications-page.ts`, `web/src/components/notifications/*.tsx`, `internal/notification/schema.go`.
- Working providers: Discord (webhook, username, avatar, include posters), Telegram (bot token, chat id, topic id, silent, include links), Webhook (URL, POST/PUT, basic auth, custom headers JSON), Email (SMTP server/port/TLS/auth, from, to, cc, bcc), Slack (webhook, username, icon emoji, channel), Pushover (user key, API token, devices, priority Lowest..Emergency, sound), Plex (OAuth "Connect to Plex" PIN flow, server pick, library section pick, path mappings, Update Library, Use Partial Refresh), Mock [DEV]. `internal/notification/{discord,telegram,webhook,email,slack,pushover,plex,mock}/`, `web/src/components/notifications/plex-oauth-button.tsx`, `plex-sections.tsx`, `use-plex-state.ts`.
- [HALF] Gotify, ntfy, Pushbullet, Join, Apprise, Custom Script appear in the schema registry (so likely selectable) but the factory returns "unsupported notifier type". `internal/notification/schema.go:137-215`, `internal/notification/factory.go:58-77`.
- Events: framework On Grab, On Import, On Upgrade, On Health Issue, On Health Restored, On App Update; module events movie:added, movie:deleted, tv:added, tv:deleted. `internal/module/framework_events.go`, `internal/modules/movie/module.go:36`, `internal/modules/tv/module.go:37`.

### 2.17 System (`/system/*`)
Health (`/system/health`, also the System hub)
- Links to Scheduled Tasks, Logs, Update; Health groups per category: Download Clients, Indexers, Prowlarr, Root Folders, Metadata (TMDB/OMDb/TVDB), Storage, Import. Each item shows ok/warning/error, message, since-timestamp; per-item Test button and per-category "test all" with summarized toast; empty category links to its settings page. `web/src/routes/system/health.tsx`, `health-group.tsx`, `health-row.tsx`, `health-utils.ts`, `use-health-page.ts`, `web/src/types/health.ts`, `internal/health/`, `docs/system-health-spec.md`.
- Health warnings also surface on settings hub rows and dashboard. `web/src/routes/settings/use-health-warnings.ts`, `web/src/components/dashboard/health-group.tsx`.

Scheduled Tasks (`/system/tasks`)
- List tasks with last run / next run (relative), Running indicator or Failed state, Run now button. Tasks: Refresh Media Status, Automatic Search, Download Client Health Check, Import Scan, Indexer Health Check, Library Scan, Prowlarr Health Check, Storage Health Check, RSS Sync, History Cleanup, Update Check, Refresh Unreleased Metadata, Plex Library Refresh. `web/src/routes/system/tasks.tsx`, `task-row.tsx`, `use-tasks-page.ts`, `internal/scheduler/tasks/*.go`.

Logs (`/system/logs`)
- Live-streamed log viewer (WebSocket), level filter (Debug / Info / Warning / Error), text search, pause/resume streaming (auto-scroll off when paused), clear view, download log file. `web/src/routes/system/logs.tsx`, `web/src/stores/logs.ts`, `internal/api/logs.go`.

Update (`/system/update`)
- States: idle "Check for Updates", checking, up to date ("Check Again"), update available (version, release notes, Install), downloading (progress %, MB), installing, restarting countdown, complete, failed (error text, "Try Again"); auto-checks on open. `web/src/routes/system/update.tsx`, `update-state-display.tsx`, `use-update-page.ts`, `internal/update/`.
- [DEV] developer-mode debug switcher to preview every update state with canned data. `use-update-page.ts` (`DEBUG_STATE_CONFIG`).
- [HALF] update settings (`autoInstall`) exist in API (`GET/PUT /update/settings`) and `web/src/api/update.ts` but no hook or UI exposes them. `internal/update/handlers.go:23-24`.
- Platform self-update / installers per OS (macOS, Windows, Linux plans). `docs/macOS-Application-Plan.md`, `docs/Windows-Application-Plan.md`, `docs/Linux-Application-Plan.md`, `internal/platform/`.

### 2.18 Developer controls showcase (`/dev/controls`) [DEV]
- Gallery of the media search/monitor control in every state (default, searching, progress, completed, completed-flash, error, monitor) with live interactive demo row. `web/src/routes/dev/*.tsx`. Not a user feature; parity optional.

### 2.19 Requests admin (`/requests-admin/*`)
Queue (`/requests-admin/queue`)
- Status segments Pending / Approved / Downloading / Available / Denied with per-segment empty copy; links to Users and Request Settings; banner "The requests portal is disabled" linking to settings. `web/src/routes/requests-admin/index.tsx`, `request-status.ts`, `use-request-queue-page.ts`.
- Request row: poster, title, type (Movie / Series / Season / Episode), seasons requested ("Season N" or "N seasons"), requester, relative age. `web/src/routes/requests-admin/request-row.tsx`.
- Pending actions: Approve (approve + add to library, using the requester's per-module quality profile or default), Deny (sheet with optional reason shown to requester), more menu: Approve & Manual Search (adds to library then opens release search sheet, stepping through each requested season), Approve & Auto Search (adds then auto searches movie / each season; toasts for downloaded / found-not-grabbed / none found), Delete request (confirm). Non-pending: Delete only. `request-row.tsx`, `use-request-approve.ts`, `use-add-to-library.ts`, `request-search-modal.tsx`, `search-modal-state.ts`, `deny-sheet.tsx`, `internal/portal/admin/requests_handlers.go`.
- [HALF] backend batch approve / deny / delete (`POST /admin/requests/batch/{approve,deny,delete}`) has no UI. `internal/portal/admin/requests_handlers.go:81-83`.

Users (`/requests-admin/users`)
- Users group: each portal user with subtitle incl. per-module quota usage ("used/limit movie"), assigned profiles; actions Enable / Disable, Edit, Delete user (confirm; requests preserved). `web/src/routes/requests-admin/users.tsx`, `users-utils.ts`, `use-request-users-page.ts`.
- Edit user sheet: username (read-only), per-module quality profile ("Default (use global)"), Auto-approve requests toggle. `user-edit-sheet.tsx`, `use-user-edit-form.ts`, `profile-select.tsx`.
- Invite user sheet: Name (becomes username), per-module quality profile, Auto-approve toggle; creates invitation link. `invite-sheet.tsx`, `internal/portal/invitations/service.go`.
- Invitations group: pending invites with expiry ("expires in ..." / "Expired"); Copy link, Resend, Delete invitation. `users.tsx`, `internal/portal/admin/invitations_handlers.go`.
- [HALF] per-user quota override (`GET/PUT/DELETE /admin/requests/users/:id/quota`) has no UI even though settings copy says "Individual users can override them". `internal/portal/admin/users_handlers.go:59-61`, `web/src/routes/requests-admin/settings.tsx:57`.

Request Settings (`/requests-admin/settings`)
- Portal Access: Enable External Requests Portal (when off, portal users cannot sign in; data preserved). `web/src/routes/requests-admin/settings.tsx`, `use-settings-page.ts`, `internal/portal/admin/settings_handlers.go`.
- Default Quotas: weekly per-module limits for new users (0 = unlimited). `settings.tsx`, `internal/portal/quota/service.go`.
- Content: Default Root Folder for requested content ("No default (use first available)").
- Notifications: Notify on New Requests (to admins) + admin-side Notification Channels list (Test, Delete) used for request notifications. `settings.tsx`.
- Rate Limiting: Searches per Minute per portal user. `settings.tsx`, `internal/portal/ratelimit/search_limiter.go`.

## 3. Requests portal (friends-and-family, `/requests/*`)
Spec: `docs/external-requests-spec.md`, `docs/external-requests-implementation-plan.md`, `docs/portal-in-library-search-result-enhancement-spec.md`, `docs/passkey-implementation-plan.md`. Spec drift is flagged where the shipped behavior differs.

### 3.1 Accounts, invitation, sign-up
- Admin creates an invitation (name = username, per-module quality profile, auto-approve) and copies/resends the link; links expire. `web/src/routes/requests-admin/invite-sheet.tsx`, `users.tsx`, `internal/portal/invitations/service.go`, `internal/portal/admin/invitations_handlers.go`.
- Sign-up page `/requests/auth/signup?token=`: validates token ("Validating invitation...", "Invalid Invitation" for bad/expired, message when no token), user sets a 4-digit PIN, account created and signed in. `web/src/routes/requests/auth/signup.tsx`, `internal/auth/handlers.go` (`/signup`, `/validate-invitation`).
- `POST /requests/auth/resend` exists for resending an invitation. `internal/auth/handlers.go:93`. [HALF] no portal UI calls it; spec 3.2.4 wanted self-service resend by email.
- Spec drift: spec describes email + password + display name and self-service email/display-name editing; v1 ships username + 4-digit PIN only, no email field. `docs/external-requests-spec.md` 3.2-3.3 vs `web/src/routes/requests/auth/signup.tsx`.
- "Requests Portal Disabled" screen on login, signup and inside the guard when the admin turned the portal off. `web/src/routes/requests/auth/login.tsx`, `signup.tsx`, `web/src/components/portal/portal-auth-guard.tsx`.

### 3.2 Authentication (shared by admin and portal users)
- Single login page for admin ("Administrator") and portal users: username + masked 4-digit PIN; remembers last username with "switch user"; redirects to remembered URL after login. `web/src/routes/requests/auth/login.tsx`, `web/src/stores/portal-auth.ts`.
- JWTs with separate audiences: portal tokens 30 days, admin tokens 24 h; admin-only routes reject portal tokens; `AnyAuth` routes accept both. `internal/auth/service.go:19-20,108-181`, `internal/portal/middleware/auth.go`.
- Auth endpoints rate-limited with lockout on repeated failures. `internal/api/routes.go:134,355-361`, `internal/api/ratelimit/`.
- Passkeys (WebAuthn): "Sign in with Passkey" (discoverable credential) on login when browser supports it, with "Use PIN" fallback; register passkey (name + PIN confirmation), list, rename, delete; per-request RP ID derived from an allowed-origins list. `web/src/components/portal/passkey-*.tsx`, `use-passkey-*.ts`, `web/src/hooks/portal/use-passkey.ts`, `web/src/api/portal/passkey.ts`, `internal/auth/passkey.go`, `passkey_handlers.go`.
- Post-login "Set up a passkey?" prompt for portal users with zero passkeys (not admins), with "don't show again". `web/src/components/portal/passkey-prompt-host.tsx`, `passkey-prompt-modal.tsx`.
- Deep link `#new-passkey` opens the passkey registration form directly (used by notifications/mobile). `web/src/components/portal/passkey-deep-link.ts`, `use-passkey-registration.ts`.
- Change PIN (portal and admin) and server-side PIN verify. `web/src/components/portal/change-pin-dialog.tsx`, `use-change-pin.ts`, `internal/auth/handlers.go` (`/profile`, `/verify-pin`).
- Logout from portal Settings. `web/src/routes/requests/settings-header.tsx`.
- Admin sessions can browse and use the portal pages too (routes use `AnyAuth`). `internal/portal/requests/handlers.go:106`.

### 3.3 Portal shell
- Header: SlipStream logo -> `/requests`, search box ("Search movies and series..."), Library link, notification bell, Settings gear. `web/src/components/portal/portal-header.tsx`, `portal-layout.tsx`.
- In-app notification inbox (bell): unread count badge, list (unread bold), opening the bell marks all read, clicking an item marks it read (does not navigate). `web/src/components/portal/notification-bell*.tsx`, `use-notification-bell.ts`, `web/src/hooks/portal/use-inbox.ts`, `internal/portal/notifications/inbox_handlers.go`.
- Active downloads strip for the user's requests (progress, ETA, Paused/Queued/Importing). `web/src/components/portal/portal-downloads.tsx`, `web/src/stores/portal-downloads.ts`. [HALF] matching downloads to requests is done client-side by normalized title/tmdb id; fragile.
- Live updates via polling (requests every 5 s and downloads every 3 s while any request is active), not WebSocket. Spec 10.3 wanted WebSocket. `web/src/hooks/portal/use-requests.ts:115-142`.
- Mobile-first responsive layout (portal is "primarily mobile"); compact badges and season text rules. `docs/portal-in-library-search-result-enhancement-spec.md`.

### 3.4 My requests (`/requests`)
- Search hero when empty ("Search for Content"); request list with Mine / All tabs (All shows requester username). `web/src/routes/requests/index.tsx`, `request-list-search.tsx`.
- Request card: poster, title, year, type, season(s) ("S1, S2" or count), status badge (Pending Approval / Approved / Searching (animated) / Denied / Downloading / Failed / Available / Cancelled), inline download progress. `web/src/routes/requests/index.tsx`, `request-status-config.tsx`, `web/src/lib/request-status-config.tsx`, `request-download-progress.tsx`.
- [HALF] spec 10.2 wanted status tabs and sorting in the list; only Mine/All exists.

### 3.5 Search and request (`/requests/search?q=`)
- TMDB movie + series search with rate limiting; results split into "In Library" and "Request" sections, Movies and Series grids. `web/src/routes/requests/search.tsx`, `search-results-content.tsx`, `search-media-grids.tsx`, `use-request-search.ts`, `internal/portal/search/handlers.go`, `internal/portal/ratelimit/search_limiter.go`.
- Card states: Request button, Requested (own), existing request by someone else (view / Watch), Available / In Library, partial series with season-availability badge ("S1-3", "S1, S3", or "3/5 seasons" when over 10 chars), live download progress. `web/src/components/search/external-media-card.tsx`, `use-external-media-card.ts`, `media-action-button.tsx`, `season-availability-badge.tsx`, `format-season-range.ts`, `download-status-card.tsx`.
- Library availability check: movie in library without files is requestable; multi-version: requestable if the user's profile targets an empty slot. `internal/portal/requests/library_check.go`, `docs/external-requests-spec.md` section 8.
- "More info" modal: poster, title/year, Director / Created by / Studio, ratings (Rotten Tomatoes critics + audience, IMDb, Metacritic), awards, cast with photos, season list with episode counts; action button (Request / Add). `web/src/components/search/media-info-modal.tsx`, `media-info-header.tsx`, `ratings-display.tsx`, `cast-list.tsx`, `seasons-list.tsx`, `use-media-info-modal.ts`, `docs/more-info-modal-spec.md`, `internal/metadata/` (TMDB + OMDb).
- Movie request: one tap submit (toast "Request submitted"). Duplicate request by another user -> offer Watch ("Now watching request"). `use-request-search.ts`.
- Series request dialog: per-season rows (Available / Requested / Watching / selectable), Select all / Deselect all over selectable seasons, Specials (S0) handling, "Monitor future episodes" toggle; submit allowed with zero seasons if monitor is on; single season -> `season` request, multiple -> `series` request with `requestedSeasons`. `web/src/routes/requests/series-request-dialog.tsx`, `docs/portal-in-library-search-result-enhancement-spec.md`.
- Episode-level requests exist in the data model (`episodeNumber`) but no portal UI creates them. [HALF] `web/src/types/portal.ts`, `docs/external-requests-spec.md` 4.1.4.
- Quality is the user's assigned per-module profile (not user-selectable); root folder is the admin default. `internal/portal/requests/service.go`, `internal/portal/users/service.go`.

### 3.6 Auto-approval and quotas
- Auto-approve if EITHER the user's auto-approve flag OR the user's module quality profile has "Allow Auto-Approve" (OR logic). `internal/portal/autoapprove/service.go:73-95`.
- Auto-approved request: consumes quota, adds media to library, and immediately searches/grabs; over quota -> stays Pending for manual approval. `internal/portal/autoapprove/service.go:97-150`, `internal/portal/requests/searcher.go`.
- Quotas: weekly per-module limit (movie, tv), 0 = unlimited, global default with per-user override, lazy reset when the period rolls over. `internal/portal/quota/service.go`. Spec drift: spec 6.1 wanted separate Movies / Seasons / Episodes pools resetting Monday midnight.
- Quota usage shown to admin per user ("used/limit"). `web/src/routes/requests-admin/users-utils.ts`.

### 3.7 Request lifecycle
- Statuses: pending, approved, searching, denied (with optional reason shown to user), downloading, failed, available, cancelled. `web/src/types/portal.ts`, `web/src/lib/request-status-config.tsx`.
- Status tracker moves requests on download started / failed / entity available (incl. per-season completion for series requests). `internal/portal/requests/status_tracker.go`.
- Requests broadcast `request:created|updated|deleted` events to the admin WebSocket. `internal/portal/requests/events.go`, `web/src/stores/ws-message-handlers.ts:214`.

### 3.8 Request detail (`/requests/$id`)
- Poster, title, year, type, season/episode, Monitor Future Episodes, requested date, approved date, requester, denial reason, status, live download progress. `web/src/routes/requests/$id.tsx`, `request-detail-card.tsx`, `request-detail-header.tsx`, `use-request-detail.ts`.
- Watch / Unwatch (non-owners); owners see "Watching" implicitly. Cancel Request (own, pending only, confirm "Cancel Request" / "Keep Request"). `request-detail-header.tsx`, `request-detail-card.tsx`, `internal/portal/requests/watchers.go`.

### 3.9 Portal library (`/requests/library`)
- Movies / Series tabs of items that have files, poster size slider, More Info modal per item; partially available series show the season badge and can request missing seasons from the modal; view existing request. `web/src/routes/requests/library.tsx`, `library-movie-card.tsx`, `library-series-card.tsx`, `use-portal-library.ts`, `internal/portal/library/handlers.go`.

### 3.10 Requester notifications
- Portal Settings > Notifications: user-owned channels (same provider set as admin notifications), add/edit/test/delete, per-channel event toggles Request Approved / Request Denied / Request Available. `web/src/routes/requests/settings.tsx`, `notification-channels-card.tsx`, `notification-channel-row.tsx`, `use-request-settings.ts`, `internal/portal/notifications/service.go`, `handlers.go`.
- Requester and all watchers are notified (external channels + in-app inbox) on approved / denied / available. `internal/portal/notifications/service.go:119-180`.
- Admin notified of new requests through admin channels when "Notify on New Requests" is on. `internal/portal/notifications/service.go:306`, `web/src/routes/requests-admin/settings.tsx`.
- Spec drift: spec 9.1.3 said notify only on available; v1 adds approved/denied toggles.

### 3.11 Portal settings (`/requests/settings`)
- Tabs Security (Change PIN, Passkeys manager) and Notifications (channels). Logout button in header. `web/src/routes/requests/settings.tsx`, `settings-header.tsx`.

### 3.12 Mobile specifics
- Admin app: phone shell (bottom tabs, push stack with named back, safe-area/notch handling, translucent bars, large titles collapsing, 44 px targets, 16 px inputs to prevent zoom, action sheets instead of dialogs). `docs/admin-native-overhaul-spec.md`, `docs/mobile-design-system.md`, `web/src/components/layout/phone-shell.tsx`, `web/src/components/screen/screen.tsx`, `web/src/tokens.css`.
- [HALF] overhaul spec story 53 wants "Remove and blocklist release" in queue actions; v1 has no blocklist at all (only `internal/arrimport/config_map.go` mentions it).
- Portal: responsive, no PWA (spec 10.4.2), passkey deep link for phones. `docs/external-requests-spec.md`.

## 4. API surface consumed outside the admin UI

### 4.1 What v1 exposes
- REST under `/api/v1`, JSON, Bearer JWT. Public: `GET /health`, `GET /api/v1/status` (version, isDevBuild, developerMode), `/api/v1/auth/{status,setup}`, portal auth `/api/v1/requests/auth/*`. Everything else needs an admin token or (portal routes) any token. `internal/api/routes.go:103-136,350-440`.
- Portal REST (the contract the portal web app, and any other portal client, uses): `/requests/auth/{login,signup,resend,logout,validate-invitation,profile,verify-pin,passkey/*}`, `/requests` (list, create, get, cancel, watch, unwatch, downloads), `/requests/search/{movie,series,series/seasons}`, `/requests/library/{movies,series}`, `/requests/notifications/*` (user channels + schema + test), `/requests/inbox/*` (list, count, read, read all). `internal/auth/handlers.go:91-101`, `passkey_handlers.go:40-50`, `internal/portal/requests/handlers.go:105-115`, `internal/portal/search/handlers.go:77-79`, `internal/portal/library/handlers.go:69-70`, `internal/portal/notifications/handlers.go:41-47`, `inbox_handlers.go:23-26`.
- Admin request management REST: `/admin/requests` (list/get/approve/deny/delete/batch), `/admin/requests/users` (list/get/update/enable/disable/delete/quota), `/admin/requests/invitations` (list/create/get/delete/resend), `/admin/requests/settings`. `internal/portal/admin/*_handlers.go`.
- WebSocket `/ws` (token passed as `Sec-WebSocket-Protocol`), admin-only in practice. Server-to-client events: `queue:updated`, `queue:state`, `download:completed`, `history:added`, `import:completed`, `progress:{started,update,completed,error,cancelled}`, `artwork:ready`, `autosearch:task:{started,progress,completed}`, `scheduler:task:{started,completed}`, `health:updated`, `devmode:{changed,error}`, `request:{created,updated,deleted}`, `portal:inbox:created`, `logs:entry`, plus module entity `added/updated/deleted` events. Client-to-server: `devmode:set`. `internal/websocket/hub.go:262`, `web/src/stores/ws-message-handlers.ts`, `ws-types.ts`.

### 4.2 Integrations v1 consumes (not exposes)
- Prowlarr: SlipStream acts as a client of Prowlarr's aggregated Torznab endpoint (`/api?t=search|tvsearch|movie`, `t=get` downloads) plus Prowlarr's management API for indexer list/status; aggregated mode only, no push/"Applications" sync from Prowlarr. `internal/prowlarr/client.go`, `docs/Prowlarr-integration-spec.md` 3.1-3.2, `docs/Prowlarr-other-arr-integration.md`.
- Built-in Cardigann indexer definitions (Torznab-like scraping). `internal/indexer/`.
- Sonarr / Radarr as migration sources (SQLite DB or v3 API + key). `internal/arrimport/reader_api.go`, `reader_sqlite.go`.
- Metadata: TMDB (search, details, credits, content ratings, logos/artwork), OMDb (RT, IMDb, Metacritic, awards), TVDB ids. `internal/metadata/`. TMDB search-ordering toggle `POST /metadata/tmdb/search-ordering` (no UI found).
- Plex (OAuth PIN flow, server/section discovery, library refresh). `internal/notification/plex/`, `internal/scheduler/tasks/plexrefresh.go`.
- Download clients listed in 2.15; notification providers listed in 2.16.

### 4.3 Not present in v1 (confirmed absent)
- No Sonarr/Radarr-compatible API (`/api/v3/*`) for Overseerr/Jellyseerr/Prowlarr-app-sync style consumers (searched `internal/` for `/api/v3`; only the arr-import reader calls one). `docs/external-requests-spec.md` 12.5 explicitly deferred an external API.
- No Torznab/Newznab endpoint served by SlipStream.
- [HALF] An admin API key is generated and stored (`api_key` setting, `POST /settings/apikey` regenerate, `apiKey` in settings type) but no middleware accepts it and no UI shows it. `internal/api/handlers_system.go:138,397`, `internal/api/auth_handlers.go:88-97`, `web/src/api/system.ts:24`.

## 5. Backend-only or unwired capabilities (UI missing)
- Module enable/disable (`GET/PUT /settings/modules`, `useUpdateModuleEnabled`) with no settings screen. `internal/api/routes.go:118-119`, `web/src/hooks/use-system.ts:84`.
- Library scan per root folder and scan-all (`/rootfolders/:id/scan`, `/scans`) with no UI trigger (only the scheduled Library Scan task). `internal/api/routes.go` (setupLibraryRoutes), `web/src/api/library.ts:26-29`.
- Indexer history (`GET /history/indexer`). `internal/api/routes.go` (setupDownloadRoutes).
- Slots debug routes (`/slots/debug`) backing the dev Debug Tools panel. `internal/api/routes.go` (setupLibraryRoutes).
- Defaults API (`/defaults/{entityType}/{mediaType}`) used for default root folders etc. `web/src/api/defaults.ts`.
- Config import from *arr, batch request actions, per-user quota override, update auto-install, invitation self-resend (see sections above).

## 6. Developer mode (cross-cutting) [DEV]
- Available only on dev builds (`isDevBuild`). Toggle swaps the server to a separate dev database seeded with mock media, mock indexer, mock download client ("Fast forward" completes downloads), mock notifier; indexer mode may be overridden. `internal/api/devmode.go`, `docs/developer-mode-testing.md`, `web/src/components/indexers/indexer-mode-toggle.tsx`.
- Unlocks: Force Loading skeletons, slot Debug Tools (parse/profile testers), update-state debug switcher, mock client/notifier types, `/dev/controls` showcase. Paths in sections 1, 2.14, 2.17, 2.18.
- Spec 11.4 wanted "admin can submit requests as a regular user in dev mode"; v1 lets any admin token use portal routes regardless of dev mode. `internal/portal/requests/handlers.go:106`.

## 7. Roadmap items and open bugs recorded in v1 docs
- Not built: config backup & restore, dashboard update, transcode during import (FFmpeg + slots), TRaSH Guides import. `docs/Remaining-features.md`.
- Partly built: onboarding wizard (only the *arr migration wizard exists, inside Settings; there is no first-run onboarding beyond PIN setup), logo and color palette. `docs/Remaining-features.md`, `web/src/routes/auth/setup.tsx`.
- "Web app settings (port, auth, https, etc)" is marked done, but no HTTPS/TLS setting exists in the Server settings UI. `docs/Remaining-features.md`, `web/src/components/settings/sections/server-section.tsx`.
- Open bug: admin login redirect issue. `docs/bugs.md`.
- Feature-parity references worth reading before porting: `docs/automatic-search-spec.md`, `docs/release-importing-spec.md`, `docs/multiple-quality-versions-spec.md`, `docs/system-health-spec.md`, `docs/Library-Import-Spec.md`, `docs/search-relevance-improvements-spec.md`, `docs/RSS-Sync-Design.md`, `docs/status-consolidation.md`.
