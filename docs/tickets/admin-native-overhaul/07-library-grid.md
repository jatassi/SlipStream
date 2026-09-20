# 07: Library tab: module segmented control, chip filters and PosterCell

Spec: `docs/admin-native-overhaul-spec.md` (stories 29–35, 38, 86–87)

**What to build:** The Library tab shows a segmented control of enabled modules (Movies, Series, and any future module) and remembers which one the admin last viewed. On a phone the library is a three-column poster grid with the title, year, quality and a status dot under each poster, no table view, and filters (Monitored, Missing, Upgradable, status, and the module's own options) in a horizontally scrolling chip row. On a wide screen the grid grows to as many columns as fit at the chosen poster size and the table view remains available with the existing column preferences. Tapping a poster cell gives immediate press feedback and opens the detail screen.

This ticket introduces `PosterCell` (poster via the existing poster image component, title, year, quality, status dot) and the `ChipRow`. `PosterCell` replaces the per-module card components: the movie and series cards are deleted, `cardComponent` is removed from the module config type, and the search page's use of module cards is switched to `PosterCell` so nothing depends on the removed field. The module list route is the same route the phone Library tab targets; the last-used module is the UI-store preference introduced with the shell.

**Blocked by:** 06 (Activity queue rows, segmented filter and the action presenter)

**Status:** done

- [x] Both projects: the Library shows a segmented control with one option per enabled module; choosing Series shows the series grid; reloading returns to Series
- [x] Phone project: exactly three poster columns; no table view control is offered; each cell shows title, year, quality and a status dot
- [x] Wide project: the grid column count changes with the poster-size preference; the table view is selectable and shows the previously configured columns
- [x] Both projects: the chip row scrolls horizontally on phone; activating Missing changes the visible set to items without files and activating it again restores the full set
- [x] Both projects: activating a poster cell opens that title's detail
- [x] `cardComponent` no longer exists on the module config type; the movie and series card components are deleted; the search page still renders library results
- [x] Press feedback on cells is a scale on pointer-down; row-style elements tint instead; hover styles apply only on fine pointers

## Comments

The screen keeps the module's own name as its title (`Movies`, `Series`) rather than "Library": the
route is per-module, the wide sidebar marks the module, and every existing spec names the module.
The segmented control carries the switching; it sits under the large title, as in the prototype.

`PosterCell` is presentational and takes a `PosterCellItem`. The module supplies it through
`movieToCell` / `seriesToCell` in `components/media/library-cells.ts`, which is what the spec means
by "the cell reads title, year, quality and status from the module's existing item type" now that
`cardComponent` is gone; the search page builds its cells from the same two functions. Quality is
the movie's file quality when it has one and otherwise the quality profile's name (a series always
shows the profile name, since a series has no single file). Series status is aggregated from
`statusCounts` by `aggregateMediaStatus` in `media-status.ts` (downloading > failed > missing >
upgradable > available > unreleased), matching the aggregate order used elsewhere.

`PosterGrid` renders `role="list"` with `PosterGridItem` wrappers. That is both better for screen
readers than a bare `div` of links and the seam the tests need: on a wide screen the `Screen` puts
the trailing actions inside the scroll region, so "every link in the region" is not "every cell".

Chip semantics changed with the chip row: no chip selected now means the whole library, where the
old `FilterDropdown` started with every status ticked and had a Reset. Selecting chips ORs them, so
`filterMovies`/`filterSeries` are unchanged apart from the renamed `unfiltered` argument.
`FilterDropdown` itself stays — History and Logs still use it.

The ticket's "activating Missing" check runs against the developer library's own truth: that
library currently holds no movie in `missing` (every released title has files and the rest are
unreleased), so the test asserts the Missing chip selects exactly the set the API reports as
missing, and then proves narrowing and restoring with the `Unreleased` chip, whose members never
change status mid-run. Picking a movie and deleting its files to manufacture a missing one would
have raced the dashboard spec, which auto-searches the first missing movie.

Ticket 08 owns the options presenter, edit mode and the add flow, but the wide acceptance criteria
here need poster size and the view switch to be reachable, so this ticket lands a working version of
both: one trailing "Options" action opening `ActionPresenter` menus (Sort by, View, Poster size,
Columns, Select, Refresh all), with view, size and columns offered on wide only. Edit mode still
works — cells grow a selection mark and the existing bulk toolbar renders above the grid; ticket 08
moves that toolbar to the bottom. The trailing Add control is a plus linking to Search, which is
where the add flow starts today.

The button is labelled "Options", not "Library options": Playwright's accessible-name matching is a
substring match, so "Library options" collided with the detail screen's "Library" back button.

Legacy removed: `MovieCard`, `SeriesCard`, `MediaListLayout`, `MediaListFilters`,
`MediaListContent`, `MediaPageActions`, `GroupedMediaGrid`, `ModuleListPage` (unreferenced), the
`cardComponent` field, the dead view/poster-size/reset handlers in both list hooks, and the module
cards' hover-glow borders. `MediaGrid` stays — the requests portal uses it. Group headings are now
plain inset headings inside the poster grid instead of the old sticky bar.

`tab-panes.tsx` and `push-routes.ts` list the movies and series panes as screen-filling, the same
change tickets 05 and 06 made for their panes, now that the list owns its own scroll container
through `Screen`. `smoke.spec.ts`'s status assertion moved from pill text to the status dot's
accessible name, which is where status lives in a grid of cells.
