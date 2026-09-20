# 16: Missing and History pages

Spec: `docs/admin-native-overhaul-spec.md` (stories 77–78)

**What to build:** The Missing page uses a segmented control across enabled modules and a secondary Missing or Upgradable toggle, listing items as rows with a trailing Search action that triggers a search in one tap. History renders as rows with an event tile, title, detail and relative time, grouped by day, matching the Dashboard's Recent section. Both pages adopt `Screen` and open from More on phones and the sidebar on wide screens.

**Blocked by:** 07 (Library tab: module segmented control, chip filters and PosterCell)

**Status:** done

- [x] Both projects: Missing shows one segment per enabled module and a Missing/Upgradable toggle; switching either changes the visible rows
- [x] Both projects: activating a row's Search action starts a search against the developer-mode indexer and the row reflects the searching state; activating the row itself opens the detail
- [x] Both projects: History groups rows by day with an event tile, title, detail and relative time; a filter change updates the list
- [x] Phone project: the back label from either page reads "More"

## Comments

**Missing.** `/missing` is now `isScreenFillPath` and renders a `Screen` with `back={usePushBack()}`.
Two stacked `Segmented` controls sit under the large title: one option per `getEnabledModules()`
module (hidden when only one module is enabled), then Missing / Upgradable. The selection drives one
`Group` whose header counts what is listed ("182 EPISODES"), with `MissingRow`s underneath — poster,
title, and a subtitle that is the year and profile for movies, `N missing episodes` /
`N upgradable episodes` for series. Series are listed per series rather than per episode: the row's
search is then the series auto search, and the row opens the series detail where per-season and
per-episode controls already live.

The trailing control is ticket 09's `MediaSearchMonitorControls` with `variant="row"`, unmodified:
its Auto Search button starts a real search against the developer-mode indexer in one tap and the
whole control swaps to its "Searching..." state in place. That is the reuse ticket 09 asked for, at
the cost of the row carrying manual search and monitor next to it rather than a single Search pill.
Because a `Row` with `href` would nest those buttons inside a link, the row's own surface is an
absolutely positioned `Link` behind a `relative` trailing slot — the same trick `SettingsItemRow`
uses. The trailing slot is a `role="group"` named `Actions for <title>`, which is also what the e2e
test scopes to.

The "Search All" header button survives as a plain `Screen` trailing action for the selected module
and view; its glow classes are gone. The tab/accordion lists (`MediaTabs`, `ViewToggle`, the four
`*-tab-content` files, `SearchButton`, `LoadingSkeleton`, and the seven list/accordion components
under `src/components/missing/`) were deleted.

**History.** `/history` is `isScreenFillPath` too. Entries are grouped into one `Group` per day —
"Today", "Yesterday", then `EEEE, d MMMM yyyy` — by `groupByDay`, which relies on the API returning
newest first. Each `Row` is the Dashboard's Recent tile: `IconTile` in the event colour, the title
with its year (movies) or `mediaQualifier` (`S02`, `S02E03`) beside it, `label · detail` as the
subtitle and the relative time trailing. The event-look mapping moved out of
`components/dashboard/recent-group.tsx` into `eventLook` in `src/lib/history-utils.ts`, so both
screens read one table. The media type filter is now a `Segmented` (All / Movies / Series); the
event-type `FilterDropdown` and date preset `Select` stay as web/CLAUDE.md prescribes. The table,
its expandable detail rows and the `Pagination` block are gone — paging is a "Load more" button that
grows the page size, and the detail-row helpers (`getDetailRows`, `getPaginationPages`) were dropped
from `history-utils.ts`.

**Tests.** `web/e2e/missing-history.spec.ts` adds four tests (three in both projects, one phone
only; 7 runs). The series segment is the deterministic one: the developer library has no missing
movies but every series has missing episodes. One unrelated fix travelled with it: `expectTitleDetail`
in `dashboard.spec.ts` now takes `.first()`, because a series detail matches its title on the
backdrop, the poster and the heading at once, which turned into a strict-mode violation once
season grabs started carrying the series title.

**Flakes not from this ticket.** Across three full suite runs the new spec passed every time, as did
`shell.spec.ts`, but three other tests failed intermittently and in different combinations:
`detail.spec.ts` "monitored pill toggles" (strict-mode violation between the Library `ChipRow`'s
"Monitored" chip, still mounted behind the pushed detail, and the detail's Monitored pill),
`dashboard.spec.ts` "activating a downloading row opens that title" and `calendar-import.spec.ts`
(two different tests). They are data races on the shared developer-mode backend between specs that
all reach for whatever is downloading, and they belong to the library, detail and import tickets, so
they were left alone.
