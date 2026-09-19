# 11: Settings index and list pages as grouped rows

Spec: `docs/admin-native-overhaul-spec.md` (stories 63–65, 68)

**What to build:** Media, Download Pipeline and General settings each open as a grouped list of their sub-sections with a trailing count and warning summary (for example "Indexers 4 · 1 warning"). Root Folders, Quality Profiles, Version Slots, Indexers, Download Clients and Notifications list their items as rows with an icon tile, a primary line and a trailing detail, replacing today's card grid, with the Add action as a trailing plus in the top bar. Boolean settings on these pages are switches in rows with the label leading and the switch trailing. Every settings page adopts `Screen` with the correct back label. The card-based settings list section and the add placeholder card are deleted.

The forms behind Add and row taps keep opening exactly as they do today in this ticket; moving them to the sheet/dialog presenter is the next ticket.

**Blocked by:** 05 (Dashboard as grouped sections with Group and Row)

**Status:** done

- [x] Both projects: `/settings/media`, `/settings/download-pipeline` and `/settings/general` show grouped rows per sub-section with counts drawn from developer-mode data; a sub-section with a warning shows it in the trailing detail
- [x] Both projects: each of the six list pages renders its items as rows with tile, primary line and trailing detail; no card grid remains
- [x] Both projects: the Add action is a trailing plus in the top bar on every list page and opens the existing create flow
- [x] Both projects: switch rows toggle and the value persists after reload
- [x] Phone project: back from a list page reads the settings section name; back from a section reads "More"
- [x] The card-based list section and add placeholder components no longer exist

## Comments

Section index pages replace the redirects that `/settings`, `/settings/media`,
`/settings/download-pipeline` and `/settings/general` used to perform; `/settings` still redirects,
now to `/settings/media`. `backLabelForPathname` derives the back label from a single
`SETTINGS_SECTIONS` table in `push-routes.ts`, and settings paths joined `isScreenFillPath` so that
both shells let the page's own `Screen` own the back control and the scroll container, exactly as
detail screens already do. On wide screens the Add plus therefore sits beside the large title until
the title collapses (spec story 19), so the e2e asserts its position in the top bar on phone only.

Counts come from the existing list hooks and warnings from `useSystemHealth`, mapped per sub-section
in `use-health-warnings.ts` (Root Folders reads `rootFolders` + `storage`, Indexers reads `indexers`
+ `prowlarr`, Download Clients reads `downloadClients`). Version Slots counts enabled slots only.

Row actions (Test, Set as default, Delete) are an overflow menu with an `AlertDialog` confirmation
inside `SettingsItemRow`; ticket 06 owns the action-sheet presenter and ticket 12 the form presenter,
so nothing here forks those. Version Slots keeps its master toggle card and debug panel above the
list; Indexers keeps the Prowlarr mode toggle and hides Add in Prowlarr mode.

Two pre-existing defects surfaced once e2e drove these flows:
- `/notifications/events` serialises the Go struct without JSON tags, so the event catalog arrives
  PascalCase. Every consumer read `group.events` as `undefined`, which crashed the notification row
  and the dialog's event triggers. Mapped to the domain type at the API boundary in
  `src/api/notifications.ts` rather than changing Go (this ticket ships no backend change).
- The dev-mode download client has type `mock`, which has no entry in `clientTypeConfigs`, so opening
  its edit dialog throws. Left alone (form work is ticket 12); the edit-flow e2e drives the
  notification and indexer dialogs instead.

The switch-persistence e2e toggles the notification channel rather than an indexer or download
client, because the dashboard spec drives autosearch in parallel against the same dev-mode services.
