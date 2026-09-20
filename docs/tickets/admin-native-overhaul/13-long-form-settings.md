# 13: Long-form settings as grouped control lists

Spec: `docs/admin-native-overhaul-spec.md` (stories 69, 72)

**What to build:** File Naming (import and naming), Auto Search, RSS Sync, Server and Authentication settings are laid out as grouped lists of controls with section footers carrying the explanatory text that today sits inline. Switches are trailing in their rows, selects and inputs are 44 px rows with the label leading, and save behaviour is unchanged. The Migrate from *arr flow remains reachable as a settings sub-screen under Media and renders inside `Screen` with its steps intact.

**Blocked by:** 11 (Settings index and list pages as grouped rows)

**Status:** done

- [x] Both projects: each of the five pages renders grouped control rows with footers; no free-floating description paragraphs remain between controls
- [x] Both projects: toggling a switch on Auto Search and saving persists after reload; changing a naming option updates the preview as today
- [x] Both projects: Server and Authentication forms save with the same validation messages as before
- [x] Both projects: Migrate from *arr opens from Media settings, shows its first step, and its back label is "Media"
- [x] Phone project: every input on these pages is 16 px and at least 44 px tall

## Comments

Implemented on `worktree-agent-abf0b112f591167b9`.

**Group footer.** `Group` gained an optional `footer` (`web/src/components/grouped-list/group.tsx`),
rendered as footnote text under the card. Existing callers are untouched. Documented in both
`web/CLAUDE.md` and `web/AGENTS.md`.

**Control rows.** `web/src/components/settings/control-row.tsx` holds the shared row vocabulary:
`ControlRow` (44 px, label leading, control trailing), `StackedRow` (full-width control under its
label), and the `SwitchRow` / `SelectRow` / `InputRow` / `TextareaRow` / `SliderRow` wrappers. Every
field is 16 px (already global in `tokens.css`) and forced to 44 px tall — the Select trigger needs
`data-[size=default]:h-11` to beat its own data-attribute class. `Slider` gained a `label` prop so
the underlying range input has an accessible name; that is the only shared UI primitive touched.

**Judgement calls.**
- *Sliders stay sliders.* Search Interval, Sync Interval and Minimum File Size keep the existing
  slider rather than becoming select rows, so save behaviour and ranges are unchanged. They render
  as a full-width `StackedRow` (label + current value on one line, track below) rather than a 44 px
  label-leading row, which the ticket only requires of selects and inputs. The range input inside a
  slider is therefore excluded from the phone 44 px check — the assertion covers text, number and
  textarea fields.
- *Import & Naming keeps Tabs.* The spec reserves `Segmented` for controls that were only switching
  a filter and keeps Tabs "for content that is genuinely tabbed"; the five naming tabs are. The tab
  strip now scrolls horizontally and each panel is a stack of groups.
- *Per-field groups for WebAuthn.* The three relying-party fields each had their own paragraph of
  explanation, so each became its own `Group` with that paragraph as the footer rather than losing
  the text to one shared footer.
- *Save affordances.* Auto Search and RSS Sync keep their own Save button (now full-height, right
  aligned below the groups) instead of moving into the `Screen` trailing slot, which would have
  meant lifting form state out of the section components; the ticket asked for presentation changes
  only. Server keeps its existing `Screen` trailing Save.
- *RSS group header.* The first group is "Feed Schedule", not "RSS Sync", so the group header does
  not duplicate the screen title (two headings with the same name is ambiguous for assistive tech
  and for tests).
- *Log path.* The read-only log-path input became a "Copy Log Path" row with the path itself in the
  group footer — a full path does not fit a 44 px trailing slot on a phone.
- *Migrate from \*arr* already rendered inside `Screen` with a "Media" back label from ticket 11 and
  its wizard steps were left alone; this ticket only adds the Playwright coverage for it.

**Legacy removed.** The card-based layouts in the auto search, RSS sync, server, validation,
matching, movie-naming, TV-naming, token-reference, filename-tester and pattern-editor components
are gone; none of them import `Card` any more. `SectionLoading` / `SectionError`
(`web/src/components/settings/section-state.tsx`) replace the repeated inline loading and error
blocks now that the sections own the screen gutter.

**Tests.** `web/e2e/settings-forms.spec.ts`, 8 tests per project (the last is phone-only). The
"grouped control rows with footers" test asserts the explanatory text's top is at or below the
control's bottom — a layout outcome, not a DOM shape — which is how "no free-floating description
paragraphs between controls" is checked without asserting on structure.

**Flakiness seen.** With five other suites running on the machine, each full `bun run test:e2e` had
two to four failures, in a different set every run, always in specs this ticket does not touch
(`dashboard.spec.ts`, `push.spec.ts`, `shell.spec.ts`, `settings.spec.ts`) and always "element never
appeared" or the known autosearch `context canceled` 500. `settings-forms.spec.ts` passed in every
run after its own selectors were fixed, and `bun run test:e2e:reduced-motion` passed 12/12.
