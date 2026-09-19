# 12: Settings forms through the sheet and dialog presenter

Spec: `docs/admin-native-overhaul-spec.md` (stories 66–67)

**What to build:** Create and edit forms for indexers (including indexer settings), download clients, quality profiles, notifications and root folders (including the folder browser), plus the version-slot dry-run and resolve flows and the naming token builder, open as a draggable bottom sheet on phones and a dialog on wide screens through the presenter introduced with the detail screen. Every field is at least 44 px tall with a 16 px input font so it is tappable and never zooms. Form state management is unchanged; only presentation moves.

**Blocked by:** 09 (Movie and series detail: hero, pill actions, grouped metadata, sheet presenter), 11 (Settings index and list pages as grouped rows)

**Status:** ready-for-agent

- [ ] Phone project: Add on Indexers opens a bottom sheet; filling the form with the developer-mode indexer and saving closes the sheet and shows the new row; dragging the sheet down dismisses it without saving
- [ ] Wide project: the same flow runs in a dialog
- [ ] Both projects: the same create-and-appear check passes for download clients, quality profiles, notifications and root folders (using the folder browser inside the presenter)
- [ ] Both projects: activating an existing row opens the edit form pre-filled; saving updates the row's primary line
- [ ] Both projects: the version-slot dry-run and resolve flows and the naming token builder open through the presenter
- [ ] Phone project: focusing any field inside a sheet leaves the viewport scale unchanged; the sheet stays above the on-screen keyboard
- [ ] Every form field control measures at least 44 px tall
