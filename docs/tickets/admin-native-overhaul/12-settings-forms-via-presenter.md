# 12: Settings forms through the sheet and dialog presenter

Spec: `docs/admin-native-overhaul-spec.md` (stories 66–67)

**What to build:** Create and edit forms for indexers (including indexer settings), download clients, quality profiles, notifications and root folders (including the folder browser), plus the version-slot dry-run and resolve flows and the naming token builder, open as a draggable bottom sheet on phones and a dialog on wide screens through the presenter introduced with the detail screen. Every field is at least 44 px tall with a 16 px input font so it is tappable and never zooms. Form state management is unchanged; only presentation moves.

**Blocked by:** 09 (Movie and series detail: hero, pill actions, grouped metadata, sheet presenter), 11 (Settings index and list pages as grouped rows)

**Status:** done

- [x] Phone project: Add on Indexers opens a bottom sheet; filling the form with the developer-mode indexer and saving closes the sheet and shows the new row; dragging the sheet down dismisses it without saving
- [x] Wide project: the same flow runs in a dialog
- [x] Both projects: the same create-and-appear check passes for download clients, quality profiles, notifications and root folders (using the folder browser inside the presenter)
- [x] Both projects: activating an existing row opens the edit form pre-filled; saving updates the row's primary line
- [x] Both projects: the version-slot dry-run and resolve flows and the naming token builder open through the presenter
- [x] Phone project: focusing any field inside a sheet leaves the viewport scale unchanged; the sheet stays above the on-screen keyboard
- [x] Every form field control measures at least 44 px tall

## Comments

Every form in the list moved to `SheetPresenter` untouched otherwise: the hooks, the mutations and
the validation are the ones that were there. What changed around them is the surface, the field
layout and the footer.

`SheetPresenter` grew two props. `wideClassName` lets a form keep the dialog width it had (the
indexer picker is still 3xl, the dry run 4xl, the conflict resolver 6xl); `nested` renders vaul's
`Drawer.NestedRoot` for a presenter opened from inside another one. `NestedRoot` reads the parent
drawer's context, so a nested presenter has to be rendered inside the parent's `children` — the
folder browser moved out of `root-folders.tsx` and into the root-folder form for that reason, and
the dry run's confirm and assign steps and the pattern editor's token builder are nested the same
way.

The 44 px / 16 px rule lives on the presenter rather than on each field: `presented-form` sets a
`min-height` and a `font-size` for every input, textarea and select trigger inside a presented form.
Enumerating the fields was not an option — indexer definitions and notification providers build
theirs from a schema at runtime, and a rule per call site would have been one more thing to
remember. Where a field layout was needed the forms reuse ticket 13's control rows inside a new
`ControlStack` (a bordered, divided stack with the explanatory copy in its footer), and `SwitchRow`
gained an optional `description` for the toggles that carried one. `FormActions` is the one footer
they all share.

The e2e suite caught a real defect. A modal surface turns pointer events off on the body, and the
Base UI Select popup portals out of the sheet, so every select inside a phone sheet was dead to
touch — including the one ticket 09 shipped in the media edit form, which no test had opened. The
select positioner turns pointer events back on.

`settings-presenter.spec.ts` runs 21 checks across the two projects (10 phone, 9 wide, 2 phone-only).
The version-slot test stubs `/slots`, `/slots/settings` and the two validation endpoints so the dry
run and both resolve flows are reachable without writing to the shared developer database; every
other test creates through the real API and deletes what it made, under names carrying the project
so the two shells never collide. `settings.spec.ts` and `settings-forms.spec.ts` needed no changes —
the headings they wait for read the same on either surface.

Two things stayed out. The reduced-motion project still matches only the shell, push and detail
specs: the presenter's reduced-motion behaviour was settled in ticket 09 and the spec calls that run
a smoke set. And select options are 27 px tall, not 44 — they are menu items, not form fields, and
resizing the menu primitive would reach well past settings.
