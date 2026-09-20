# 17: Calendar and Manual Import pages

Spec: `docs/admin-native-overhaul-spec.md` (stories 76, 79)

**What to build:** Calendar's month and list toggle is a segmented control; the list view is grouped rows by day so upcoming releases read well on a phone, and the month view fits the phone width without horizontal scroll. Manual Import's file browser renders as a pushed list of rows with folder chevrons: entering a folder pushes a new list with the parent folder as the back label, files show their detected match and status, and the import action works from the row list. Both pages adopt `Screen`.

**Blocked by:** 04 (Push navigation), 06 (Activity queue rows, segmented filter and the action presenter)

**Status:** done

- [x] Both projects: Calendar shows the segmented toggle; the list view groups developer-mode releases by day; the month view shows the same items on their dates without horizontal overflow on `phone`
- [x] Phone project: entering a folder in Manual Import pushes a new list whose back label is the parent folder's name; back returns to the parent at the same scroll position
- [x] Both projects: selecting a file and importing completes as today and the file leaves the list
- [x] Both projects: rows in the browser are at least 44 px and folders show a chevron

## Comments

**Calendar.** The toggle is two options, Month and List; the week view is gone with
`calendar-week-view.tsx`, and `CalendarView` is now `'month' | 'list'`. The list view is one
`Group` per day with the header reading Today, Tomorrow or `EEEE, MMMM d`, over a range of today
plus 30 days. The month view is the iOS Calendar shape rather than a wall planner: a seven-column
grid of day cells (44 px minimum, fraction widths, so nothing can overflow the phone), each cell
carrying up to three media-coloured dots, and the selected day's releases rendered as a `Group`
beneath the grid. Both shells get that same layout — a wall-planner month with titles in the cells
would need a second design for the phone, and the cell dots plus the day group say the same thing
in one. `Today` moved into the `Screen` trailing slot and the month steppers sit beside the month
name. `CalendarEventCard` and the agenda view are removed; event presentation (icon, tile colour,
title, detail line, status label, href) lives in `components/calendar/event-presentation.ts`.

**Manual Import.** The browsed folder now lives in the URL (`/import?path=…`), so browser back,
forward and a pasted link all behave the same as the in-app back control. The page is a `Screen`
titled with the folder name; back reads the parent folder's name (from the browse response's
`parent`) and calls `history.back()`, the same contract push navigation uses elsewhere, falling
back to `usePushBack()` ("More") at the root. Folders are `Row`s with a chevron; files are
44 px rows showing the filename in monospace over "detected match · size", with a trailing Import
action. The scanned-files mode, its checkbox selection and the path input are gone: the directory
scan now runs automatically for any folder that holds video files (`useDirectoryScan`, a query,
replaces the `useScanDirectory` mutation) and its matches are merged into the file rows. Bulk
import survives as an "Import All" action in the `Screen` chrome when two or more files match.
Tapping a file's label still opens the existing `EditMatchDialog` to set or change the match; that
dialog stays a `Dialog` on both shells because the sheet presenter for forms is ticket 09's.
Pending imports became a `Group` of rows shown on the root screen.

**Scroll position.** Entering a folder swaps the list inside one scroll container, so the offset
per path is remembered in `use-list-scroll.ts` and restored when a list comes back. `Screen` gained
an optional `scrollRef` so a page can reach its own scroller; nothing else about `Screen` changed.

**push-routes.** Only `isScreenFillPath` changed, gaining `/calendar` and `/import` so both shells
let the `Screen` own its back control and scrolling. No back-label rule was needed: the root back
label is the existing "More" default and folder levels derive their label from the browse response.

**E2E.** `e2e/calendar-import.spec.ts` covers all four checkboxes in both projects (the push and
scroll-restore test is phone-only). Two things it cannot take from developer mode:

- The calendar API answers `[]` in developer mode — the mock library carries no release or air
  dates, and the episode air dates it does store are written in Go's `time.String()` format, which
  the `BETWEEN` range query never matches. Events are served from `e2e/helpers/calendar-stub.ts`
  (two today, one tomorrow) so grouping and date placement can be asserted against known dates.
- `POST /import/manual` can never succeed: the handler builds the `LibraryMatch` by hand and never
  sets `ModuleEntity`, so `computeDestination` always fails with "no module entity for matched
  file". The browse and the scan in the spec run against the real backend over a real fixture
  directory the test writes under `e2e/.data/import-fixture` (24 folders plus a sparse
  `Oppenheimer.2023…mkv` the dev library really matches); only the import POST is stubbed, so the
  row list's import path is driven to completion. The backend bug is out of scope here.

The folder chevron is asserted through the lucide class on the row's icon, the one place the suite
touches the DOM: a decorative chevron has no accessible name to match on.

Running the whole suite with five workers on a loaded machine makes the dashboard checks flake —
both projects share one backend, and `ensureRecentHistory` competes for the same grabbable movies.
`dashboard.spec.ts` passes on its own and the suite is green at two workers.
