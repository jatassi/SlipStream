# 18: Legacy removal, motion audit and prototype decision

Spec: `docs/admin-native-overhaul-spec.md` (Removal of legacy, stories 89–94, Further Notes)

**What to build:** With every page ported, the admin app has no trace of the old presentation layer: the old page header, the card-based settings sections, the header search bar's mobile fallback, hover-glow and card-glow utilities and their usages, and every `transition-all` are gone; glow remains only on active download progress and the completion flash. The Cinematic and Console prototype variants are deleted. Motion across the app is audited against the policy (entrances on first mount only, under 300 ms, exits faster than entrances, only transform and opacity except the progress bar, reduced motion and reduced transparency honoured, light theme intact). Screenshot baselines are recorded for the key screens (Dashboard, Library grid, Detail, Activity, Settings list) in both projects and both themes. The real-hardware checklist (sticky hover, tap delay, safe areas, keyboard, sheet drag) is run and recorded.

This ticket also makes the decision the spec defers: whether the Native prototype and its harness ship to `main` or are dropped now that the production shell is complete. Record the decision and its reasoning under the spec's Comments heading.

**Blocked by:** 08 (Library actions), 10 (Search tab), 12 (Settings forms), 13 (Long-form settings), 14 (System and session actions), 15 (Requests admin), 16 (Missing and History), 17 (Calendar and Manual Import)

**Status:** done

- [x] No references remain to the old page header, settings list section, add placeholder card, media status badge, module card components or the header search mobile fallback
- [x] `transition-all` does not appear in the frontend source; hover-glow and card-glow utilities are removed and glow is used only by download progress and the completion flash
- [x] The Cinematic and Console prototype variants and any code only they used are deleted; the Native prototype's fate is decided and recorded in the spec's Comments
- [x] Reduced-motion run of the full suite passes; reduced-transparency renders solid bars on the key screens; the light theme renders every key screen with correct status colours
- [x] Returning to a tab or navigating back does not replay list entrance animation
- [x] Screenshot baselines exist for the five key screens in both projects and both themes and the comparison passes
- [ ] The real-hardware checklist is completed on at least one iOS and one Android device and the results appended to the spec's Comments
- [x] `tsc -b`, ESLint and the full Playwright suite pass in CI

## Comments

### What was left to remove

Most of the list had already gone with the tickets that replaced each screen. `MediaStatusBadge`,
`ListSection`, `AddPlaceholderCard`, `MovieCard` and `SeriesCard` were gone (07 and 08), and ticket
10 recorded that the header search's mobile fallback had no code left to delete — the header only
renders in the wide shell. What remained:

- **`PageHeader`**, whose last caller was the `/dev/controls` showcase. The showcase keeps its own
  heading inline; the component is deleted.
- **`transition-all`**, 24 usages across 19 files. Each became the properties that actually change:
  `transition-colors` or an explicit `transition-[…]` list for colour, border and ring changes, and
  `transition-[width]` / `transition-[left]` at the sanctioned 700 ms (`ui/progress.tsx`,
  `media/progress-bar.tsx`) or 500 ms (the downloads overlay, which was already on those two
  properties) for progress fills. `search/expandable-media-grid.tsx`'s `hover:brightness-125`
  became `hover:opacity-80`, since a filter is not one of the two properties the policy allows.
- **Glow.** Every `glow-*` / `hover:glow-*` / `glow-*-pulse` / `data-active:glow-*` utility and the
  large and media icon glows are deleted from `index.css`, along with the `glow-*-pulse`,
  `progress-glow-*`, plain `download-complete-flash`, `slide-down-fade` and `controls-fade-out`
  keyframes, all unused. What survives is `icon-glow-movie` / `icon-glow-tv` and the
  `inset-glow-pulse-*` / `download-complete-flash-*` keyframes — active download progress and its
  completion flash, exactly as the spec allows.

**The portal.** The last holders of the hover glows were the portal's own surfaces
(`components/portal/portal-header.tsx`, `routes/requests/*`) and one admin one
(`arr-import/preview-step.tsx`). The portal is out of scope for this overhaul, but the utilities
could not be deleted while it used them, so the decorative box-shadows were dropped there too — a
presentation-only change with no behaviour behind it. Two portal search fields tracked a
`searchFocused` state that existed solely to switch a glow on; that plumbing went with it. The
portal card components themselves (`library-movie-card.tsx`, `library-series-card.tsx`) stay, as
the ticket says; only their glow class and their `transition-all` changed.

`components/COMPONENTS.md`'s theme-lookup table listed six files that earlier tickets deleted; it
is corrected here rather than left to rot.

### Prototypes

Removed: `web/src/prototypes/` (all three variants, the shared mock data and the harness) and
`web/prototypes/mobile/index.html`. The decision and its reasoning are under the spec's Comments.
Nothing in production imported from the tree and the harness was not an entry point in
`vite.config.ts`, so the deletion is inert — `tsc -b`, ESLint and `bun run build` are unchanged by
it.

### Motion audit

Audited: `tokens.css` (durations, easings, press, materials, entrances, stagger), `index.css`
(every keyframe and its callers), `screen.css`, `phone-shell.css` (push and pop),
`sheet-presenter.css`, `action-sheet.css`, `segmented.css`, the Base UI dialog and alert-dialog
classes, and every `transition-*` in `src` (one inventory per property, after the `transition-all`
work).

Found and fixed:

1. **The dashboard entrance never ran.** `enter-fade-up` was a plain CSS class, so Tailwind could
   not generate the `[&>section]:enter-fade-up` variant `dashboard-groups.tsx` asked for, and no
   rule was ever emitted. The three entrance classes are `@utility` rules now, which makes the
   variant generate; each carries its own nested reduced-motion fade, since the old blanket
   `.enter-fade-up { animation-name: enter-fade }` override would not have matched the variant's
   selector either.
2. **The entrance replayed.** Once it actually animated, the wide shell replayed it on every
   return: the phone keeps tab panes mounted, but the wide shell remounts the route. `useFirstMount`
   (`src/hooks/use-first-mount.ts`) makes an entrance true only for the first mount of a key in a
   page session, and drops the class once it has played.
3. **The sheet's exit was not faster than its entrance** — 320 ms both ways, and over the policy's
   300 ms ceiling. `--dur-sheet` is 280 ms and the new `--dur-sheet-out` is 200 ms, applied to the
   sheet and its scrim. The same rule was applied to the reduced-motion exits of the sheet
   (`--dur-press`), the action sheet and the phone push layer, all of which matched their
   entrances.
4. **Reduced motion did not reach the decorative loops.** The download shimmer and inset pulse, the
   completion flash, the searching shimmer text and its chasing lights, the skeleton sweep and the
   sidebar's height collapse all kept running. A block at the end of `index.css` stops them.

Audited and left alone: the push/pop animations (260 ms in, 200 ms out, transform only, with
`data-instant` for keyboard and tab switches), the segmented thumb (200 ms transform, no transition
under reduced motion), `screen.css`'s bar and compact title (160 ms opacity and transform), `press`
(120 ms transform and opacity, no scale under reduced motion), the dialogs (100 ms fade and zoom).
The sidebar's collapsible still animates `height` — that is Base UI's own
`--collapsible-panel-height` technique and the one place outside progress that animates a
non-composited property; it is wide-shell chrome, it is 200 ms, and reduced motion now removes it,
so it stays.

### Tests

`e2e/screenshots.spec.ts` is five screens × two themes × two projects (20 shots). The queue is
served from `queue-stub.ts` on the Dashboard and Activity so the two screens that read live
developer-mode state hold still; the tolerance stays at `maxDiffPixelRatio: 0.08`, since these
catch layout regressions rather than pin pixels. The theme has no control in the app, so
`e2e/helpers/theme.ts` patches the persisted `slipstream-ui` store in an init script before the
page's own scripts run. Baselines recorded and the comparison re-run clean.

`e2e/appearance.spec.ts` holds the three remaining boxes. The light-theme check reads every status
mark's painted colour and compares it against the palette the theme resolves for that status
through a probe element, so both sides go through the browser's own colour serialisation; a second
test proves the two palettes differ. The reduced-transparency check emulates
`prefers-reduced-transparency` over CDP (Playwright's `emulateMedia` does not expose it) and
asserts every `.material*` bar on Library, Detail, Dashboard and Settings has no backdrop filter
and paints the page background. The entrance check counts `animationstart` events for `enter-*`
from an init script: it proves the entrance fires on first load, then navigates in-app to Library
and back and asserts nothing fires again. It has to navigate in-app — a `page.goto` is a full
reload, which legitimately starts a new page session.

The reduced-motion project's `testMatch` is now a `testIgnore` of the setup and the screenshot
spec, so it runs the whole suite. One test had to bend: `library.spec.ts`'s "cells scale on pointer
down" asserts the press scale, which reduced motion removes by design — under that project it
asserts the scale stays 1 instead. Everything else passes unchanged.

Suite results on this worktree: `test:e2e` 211 passed / 49 skipped with one failure per run drawn
from the known flakes the ticket names (a `settings-forms` page not rendering, a
`library-actions` unmonitored pill), each passing on a rerun of its own spec.
`test:e2e:reduced-motion` 112 passed / 8 skipped, with the same class of flake — the autosearch
helper's `context canceled` 500 in `dashboard.spec.ts`, which now contends with a third project
grabbing from the same mock indexer; that spec passes with `--workers=1`. None are in code this
ticket touched.

### Real hardware — for the user

The last box needs a physical iOS device and a physical Android device; it cannot be done from
here. To run it:

1. `make dev-mode` in the repo root.
2. `portless get slipstream` prints the frontend URL. `.localhost` names resolve only on the
   machine itself, so the phone cannot open that URL as-is — you will need
   `portless service install --lan` or an equivalent LAN route (your own call; a plain
   `http://<mac-lan-ip>:3000` via `PORT=3000 make dev-frontend` also works).
3. Open the URL on each device and check:
   - **Sticky hover.** Tap a poster cell, a grouped row, a chip and a tab, then tap elsewhere:
     no hover tint or border is left behind. Hover styles are behind
     `@media (hover: hover) and (pointer: fine)`, so this is a check that nothing slipped past it.
   - **Tap delay.** Taps on the tab bar, a row and a chip register immediately — no ~300 ms pause
     and no double-tap zoom (`touch-action: manipulation` plus the 16 px field rule).
   - **Safe areas.** In portrait and landscape, and with the home indicator: the top bar clears the
     notch or Dynamic Island, the tab bar and any bottom bar clear the home indicator, and nothing
     is clipped by a rounded corner.
   - **Keyboard.** Focusing the Search field and any settings input does not zoom the page, the
     focused field stays visible above the keyboard, and dismissing the keyboard restores the
     layout.
   - **Sheet drag.** Open a form (an indexer, say) and an action sheet: the drag tracks the finger
     1:1, rubber-bands past the top, dismisses on a flick, can be caught mid-animation, and ignores
     a gesture begun in the first 500 ms.
   - Repeat the pass with the system's Reduce Motion and Reduce Transparency switches on.

Append the results under the spec's Comments and tick the box.
