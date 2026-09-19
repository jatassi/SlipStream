# 07: Library tab: module segmented control, chip filters and PosterCell

Spec: `docs/admin-native-overhaul-spec.md` (stories 29–35, 38, 86–87)

**What to build:** The Library tab shows a segmented control of enabled modules (Movies, Series, and any future module) and remembers which one the admin last viewed. On a phone the library is a three-column poster grid with the title, year, quality and a status dot under each poster, no table view, and filters (Monitored, Missing, Upgradable, status, and the module's own options) in a horizontally scrolling chip row. On a wide screen the grid grows to as many columns as fit at the chosen poster size and the table view remains available with the existing column preferences. Tapping a poster cell gives immediate press feedback and opens the detail screen.

This ticket introduces `PosterCell` (poster via the existing poster image component, title, year, quality, status dot) and the `ChipRow`. `PosterCell` replaces the per-module card components: the movie and series cards are deleted, `cardComponent` is removed from the module config type, and the search page's use of module cards is switched to `PosterCell` so nothing depends on the removed field. The module list route is the same route the phone Library tab targets; the last-used module is the UI-store preference introduced with the shell.

**Blocked by:** 06 (Activity queue rows, segmented filter and the action presenter)

**Status:** ready-for-agent

- [ ] Both projects: the Library shows a segmented control with one option per enabled module; choosing Series shows the series grid; reloading returns to Series
- [ ] Phone project: exactly three poster columns; no table view control is offered; each cell shows title, year, quality and a status dot
- [ ] Wide project: the grid column count changes with the poster-size preference; the table view is selectable and shows the previously configured columns
- [ ] Both projects: the chip row scrolls horizontally on phone; activating Missing changes the visible set to items without files and activating it again restores the full set
- [ ] Both projects: activating a poster cell opens that title's detail
- [ ] `cardComponent` no longer exists on the module config type; the movie and series card components are deleted; the search page still renders library results
- [ ] Press feedback on cells is a scale on pointer-down; row-style elements tint instead; hover styles apply only on fine pointers
