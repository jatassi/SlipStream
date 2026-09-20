# 08: Library actions: sort and view presenter, edit mode toolbar, add flow presentation

Spec: `docs/admin-native-overhaul-spec.md` (stories 36–37, 39)

**What to build:** The Library's secondary options (sort, sort direction, view mode on wide, poster size, table columns on wide) sit behind a single trailing action in the top bar that opens an action sheet on phones and a menu on wide screens. Edit mode keeps working for bulk monitor, quality profile and delete: checkboxes overlay poster cells and a bottom toolbar sits above the tab bar on phones (and at the bottom of the content column on wide screens). The Add action opens the existing add flow (search, configure, add) as a pushed full screen on phones and a dialog on wide screens.

**Blocked by:** 07 (Library tab: module segmented control, chip filters and PosterCell)

**Status:** done

- [x] Both projects: the top bar shows one trailing options action; on `phone` it opens an action sheet, on `wide` a menu; choosing a sort reorders the grid and the choice persists across reload
- [x] Wide project: view mode, poster size and column options are present in the menu; phone project: view mode and column options are absent
- [x] Both projects: entering edit mode overlays checkboxes on cells; selecting two items enables the toolbar; a bulk monitor change updates both items' status; leaving edit mode hides the toolbar
- [x] Phone project: the edit toolbar is fully visible above the tab bar and not obscured by safe-area insets
- [x] Both projects: Add opens the add flow; searching the developer-mode metadata provider, configuring and adding a title results in it appearing in the grid; on `phone` the flow is a pushed screen with a back button, on `wide` a dialog

## Comments

The options presenter landed in ticket 07 and this ticket finished it. Two changes: "Reverse order"
is now its own action driving a real `onToggleSortDirection` (it used to re-send the current sort
field and rely on `handleColumnSort` toggling the direction as a side effect), and `Select movies`
only enters edit mode — leaving it is the "Done" control the top bar shows while editing, which is
the iOS pattern and gives the tests a visible exit. Sort field and direction already persisted in
the UI store, so reloading keeps them.

`Screen` grew a `bottomBar` slot rather than the edit bar positioning itself. The slot is absolute
inside the `Screen`'s own container, which is the tab pane on phones (so it takes the
`--safe-bottom` + `--spacing-tab-bar` clearance) and the content column on wide (so it stops at the
sidebar and the column's bottom); it also adds its height to the scroll region's bottom inset so the
last row of posters is never behind it. `LibraryEditBar` is a `role="toolbar"`: Select All, the
selected count, then monitor, unmonitor, quality profile and delete as 44 px icon actions, which is
what fits a 390 px phone. The quality profile list presents through `ActionPresenter`, so it is a
sheet on phones and a menu on wide; delete still opens `MediaDeleteDialog` with its delete-files
checkbox. `MediaListToolbar` and its theme-tinted bar are gone.

The add flow is `AddMediaPresenter`: the one place that branches on the shell, like the other
presenters. On phones it is a `Screen` with a "Library" back control and the Add button in the
`bottomBar`; on wide it is `SheetPresenter` with `open`, which is the existing `Dialog`. `/movies/add`
and `/series/add` became `isScreenFillPath` so neither shell adds a second back control or a second
scroll container. On wide the route renders only the dialog: the content column behind it is empty
under the dialog's scrim, which is the price of keeping the add flow a real route rather than state
owned by the Search screen (the Search tab only links to it, per the brief). Closing the dialog
navigates back to the module list, which is what Cancel and the phone back control do too.

`PageHeader`, the `Card` chrome around the add form and `FormActions` are gone from the add flow;
the form is a preview block plus the existing fields, and `AddMediaActions` is the Cancel + Add pair
the presenter places. The four add-flow selects carry an `aria-label` — their `<Label htmlFor>` never
matched an id, so those comboboxes had no accessible name at all.

Testing notes (`e2e/library-actions.spec.ts`, 6 tests × 2 projects, 2 phone-only):

- The developer library cannot prove an arbitrary sort: its movies carry no release date, no
  recorded size on disk, one shared "date added" and one quality profile, so every sort field except
  the title collapses to the API's own order. The sort test therefore proves the reorder with
  "Reverse order" (two titles swap places and stay swapped across a reload) and proves choosing a
  sort field with Date Added, whose grouping heading appears and survives a reload along with the
  menu's "Sort by Date Added" line.
- Bulk monitor is checked on two library titles per project (phone: Fight Club and Pulp Fiction,
  wide: Inception and Dune: Part Two — never the two `detail.spec` toggles), and the outcome is read
  on each title's own detail screen as an "Unmonitored" pill before the state is restored. Creating
  throwaway movies for it was tried first and dropped: it made the library's size change under the
  other specs running in parallel.
- The add test adds "Ford v Ferrari" (phone) and "Everything Everywhere All at Once" (wide), deleting
  the title first if an earlier run left it, and leaves it in the library as the ticket allows.
- `library.spec.ts`'s Missing check no longer compares the visible count against the API's count:
  with titles being added and removed in parallel that count can never settle. It now asserts every
  visible title is one the API reports as missing, and reads the narrowing and the restore through
  "The Matrix", which keeps its files.
- `dashboard.spec.ts`'s `expectTitleDetail` gained a `.first()`: a series detail carries a backdrop,
  a poster and a heading with the same name, so the locator was a strict-mode violation waiting for
  a series to show up in Recent.
- Left alone: `detail.spec.ts`'s monitored-pill assertion matches `Monitored` as a substring, so it
  can also match the "Unmonitored" pill and the Library chip on the pane behind the pushed screen.
  It is flaky under load; scoping it to the detail region belongs with that screen's ticket.
