# 09: Movie and series detail: hero, pill actions, grouped metadata, sheet presenter

Spec: `docs/admin-native-overhaul-spec.md` (stories 40–48)

**What to build:** A movie or series detail screen shows a backdrop hero with the poster overlapping it, the title, year, runtime or season count, genres, rating and a status pill. Under the hero sit three 44 px pill actions (Search, Auto Search, Monitored) that take the media colour when active; Monitored toggles immediately. The existing search and monitor control states (default, searching manual or auto, progress with pause, completed flash, error with dismiss) render inside the pill row. The overview is clamped to three lines and expands on tap. File, Details and, for series, Seasons are grouped lists with trailing values; version slots and quality profile appear in the File group; seasons are collapsible groups with episode rows showing air date, status dot and a per-episode search action. Edit and Delete live in a trailing menu in the compact bar and present as a bottom sheet on phones and a dialog on wide screens.

This ticket introduces the `PillAction` (wrapping the existing search/monitor control states; the `xs`/`sm`/`lg`/`responsive` size matrix collapses to one pill size plus a compact row variant for lists) and the sheet presenter: a draggable bottom sheet for forms on phones built on vaul (1:1 drag, rubber-band, velocity dismiss, catchable mid-animation), rendering the same content inside the existing Dialog on wide screens through one component. The media edit form and the delete confirmation are the first two consumers; the settings and requests forms adopt it in later tickets. Artwork uses the existing poster, backdrop and title-treatment components. Loaders and prefetches are unchanged.

**Blocked by:** 04 (Push navigation), 06 (Activity queue rows, segmented filter and the action presenter)

**Status:** ready-for-agent

- [ ] Both projects: a movie detail shows backdrop, overlapping poster, title, year, runtime, genres, rating and a status pill; a series detail shows season count in place of runtime
- [ ] Both projects: activating Monitored flips the pill to the media colour and the state survives a reload; activating again reverts it
- [ ] Both projects: activating Search against the developer-mode indexer walks the control through searching and completed states inside the pill row; the error state offers dismiss
- [ ] Both projects: a long overview shows three lines with an expand affordance; activating it reveals the full text
- [ ] Both projects: File shows version slot and quality profile rows; Details shows metadata rows with trailing values
- [ ] Both projects: on a series, a season group collapses and expands; an episode row shows air date and status dot and offers a search action
- [ ] Phone project: the compact bar's trailing menu opens Edit as a bottom sheet that can be dragged to dismiss; Delete confirms in an action sheet and, on confirm, returns to the library without the item
- [ ] Wide project: Edit opens a dialog with the same form; Delete confirms in a dialog
- [ ] The sheet presenter and `PillAction` are shared components documented for reuse; the old size matrix on the search/monitor controls is gone
