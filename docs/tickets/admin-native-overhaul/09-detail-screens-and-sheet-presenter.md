# 09: Movie and series detail: hero, pill actions, grouped metadata, sheet presenter

Spec: `docs/admin-native-overhaul-spec.md` (stories 40–48)

**What to build:** A movie or series detail screen shows a backdrop hero with the poster overlapping it, the title, year, runtime or season count, genres, rating and a status pill. Under the hero sit three 44 px pill actions (Search, Auto Search, Monitored) that take the media colour when active; Monitored toggles immediately. The existing search and monitor control states (default, searching manual or auto, progress with pause, completed flash, error with dismiss) render inside the pill row. The overview is clamped to three lines and expands on tap. File, Details and, for series, Seasons are grouped lists with trailing values; version slots and quality profile appear in the File group; seasons are collapsible groups with episode rows showing air date, status dot and a per-episode search action. Edit and Delete live in a trailing menu in the compact bar and present as a bottom sheet on phones and a dialog on wide screens.

This ticket introduces the `PillAction` (wrapping the existing search/monitor control states; the `xs`/`sm`/`lg`/`responsive` size matrix collapses to one pill size plus a compact row variant for lists) and the sheet presenter: a draggable bottom sheet for forms on phones built on vaul (1:1 drag, rubber-band, velocity dismiss, catchable mid-animation), rendering the same content inside the existing Dialog on wide screens through one component. The media edit form and the delete confirmation are the first two consumers; the settings and requests forms adopt it in later tickets. Artwork uses the existing poster, backdrop and title-treatment components. Loaders and prefetches are unchanged.

**Blocked by:** 04 (Push navigation), 06 (Activity queue rows, segmented filter and the action presenter)

**Status:** done

- [x] Both projects: a movie detail shows backdrop, overlapping poster, title, year, runtime, genres, rating and a status pill; a series detail shows season count in place of runtime
- [x] Both projects: activating Monitored flips the pill to the media colour and the state survives a reload; activating again reverts it
- [x] Both projects: activating Search against the developer-mode indexer walks the control through searching and completed states inside the pill row; the error state offers dismiss
- [x] Both projects: a long overview shows three lines with an expand affordance; activating it reveals the full text
- [x] Both projects: File shows version slot and quality profile rows; Details shows metadata rows with trailing values
- [x] Both projects: on a series, a season group collapses and expands; an episode row shows air date and status dot and offers a search action
- [x] Phone project: the compact bar's trailing menu opens Edit as a bottom sheet that can be dragged to dismiss; Delete confirms in an action sheet and, on confirm, returns to the library without the item
- [x] Wide project: Edit opens a dialog with the same form; Delete confirms in a dialog
- [x] The sheet presenter and `PillAction` are shared components documented for reuse; the old size matrix on the search/monitor controls is gone

## Comments

The hero renders the title as a plain `h1` rather than `TitleTreatment`. The Native hero pairs a
text title with the poster, and a logo image would leave the screen without a heading; the
treatment component stays in the tree for other surfaces.

`MediaSearchMonitorControls` keeps its `theme` prop for the tint and gains `variant` (`pill` |
`row`, default `row`). Every list call site simply dropped its `size`; the `xs` ghost styling is
gone — lists now use what `sm` was. The `/dev/controls` showcase drives its live components through
the two variants, but its static mockups keep their own local `lg`/`sm`/`xs` types: they are a
dev-only gallery of states and the spec puts that page out of scope.

The old movie Files table, the SlotStatusCard and the episode table are replaced by Group/Row
lists. Two things went with them: the version-slot summary badges, and the per-episode file→slot
`Select`. Episode slot monitoring and slot search survive behind a chevron on the episode row (only
when multi-version is on), and the movie File group keeps the slot assignment `Select` per file.
Movie credits still render under the groups, unchanged.

`Row` gained no new props; the season header is its own button so it can carry `aria-expanded`.
`mediaStatusFromCounts` in `media-status.ts` now derives one status from `StatusCounts` (the spec's
aggregate priority) for the series hero and the season pills.

vaul does not export its stylesheet through its package exports (`Missing "./style.css" specifier`),
so `sheet-presenter.css` carries the drawer geometry, the handle, the `::after` rubber-band filler
and the animations on the app's tokens — the sheet slides over `--dur-sheet` instead of vaul's
0.5s, and reduced motion swaps the slide for a fade. The progress bar's `transition-all` became
`transition-[width]`/`transition-[left]` at 700 ms linear, the one width exception the motion
policy allows.

E2E: `detail.spec.ts` runs in both projects (the drag check is phone-only, the dialog check
wide-only) and is picked up by the reduced-motion project as well. Three things are served from
page routes so a state holds still: version slots (`/slots*`), an auto-search that finds nothing
(the error state), and a finished download (the queue stub, which now carries `movieId`). The
monitored toggle uses a movie of the project's own because both projects share one backend, and the
delete check creates a scratch movie of its own instead of removing library data other specs read.
vaul ignores a drag that begins within 500 ms of the sheet opening, so the drag gesture waits for
the entrance to settle.

Full-suite runs on a loaded machine flake in other specs (blank pages and portless 502s, and the
known settings/activity contention); `detail.spec.ts` passes on its own and in clean full runs.

The per-episode file→slot `Select` is back: `EpisodeSlotAssign`
(`web/src/components/series/episode-slot-assign.tsx`) renders it inside the episode row's slot panel
on `useAssignEpisodeFile`, leaving the row's own 44 px geometry untouched. It has no e2e cover
because the developer-mode database seeds no episode files and multi-version can only be switched on
through the global slot settings the other specs share.
