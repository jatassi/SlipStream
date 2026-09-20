# Frontend Agent Documentation

### Making Changes
When making changes to this file, ALWAYS make the same update to CLAUDE.md in the same directory.

## Design System — Media Type Theming

Movies use orange (`movie-*`), TV shows use blue (`tv-*`). CSS variables defined in `src/index.css` with shades 50-950 (OKLCH color space).

**Tailwind usage:** `text-movie-500`, `bg-tv-400`, `border-movie-600`, etc.

**Conventions:**
- Movie content -> `movie-*` classes; TV content -> `tv-*` classes
- Mixed content -> `media-gradient` utilities (`bg-media-gradient`, `text-media-gradient`)
- Backgrounds: use `/10` or `/15` opacity (e.g., `bg-movie-500/10`)
- Text on dark backgrounds: 400 shades; borders/accents: 500 shades
- Glow effects for interactivity: `glow-movie`, `glow-tv`, `hover:glow-movie`, `glow-media`

## Base UI (NOT Radix)

This project uses **Base UI** (`@base-ui/react`) for shadcn/ui components.

**Trigger composition — use `render` prop, NOT Radix-style `asChild`:**
```tsx
// WRONG
<TooltipTrigger asChild><Button>Click</Button></TooltipTrigger>

// CORRECT
<TooltipTrigger render={<Button />}>Click</TooltipTrigger>
```
Applies to: `TooltipTrigger`, `DialogTrigger`, `PopoverTrigger`, `DropdownMenuTrigger`, etc.

**SelectValue gotcha:** `SelectValue` renders the raw `value`, not the display label. When value differs from label, render the label manually in the trigger:
```tsx
<SelectTrigger>
  {OPTIONS.find((o) => o.value === selected)?.label}
</SelectTrigger>
```

## Module System

Frontend media modules live in `src/modules/`. Each module exports a `ModuleConfig` (defined in `src/modules/types.ts`) containing identity, routes, query keys, WS invalidation rules, filter/sort options, table columns, lazy-loaded components, and API bindings.

**Key files:**
- `src/modules/types.ts` — `ModuleConfig` type definition
- `src/modules/registry.ts` — `registerModule()`, `getModule()`, `getEnabledModules()`, `setModuleEnabledState()`
- `src/modules/setup.ts` — Calls `registerModule()` for each module at app init
- `src/modules/<id>/index.ts` — Module-specific config (one per media type)

**Enabled state:** On app init, `GET /api/v1/system` returns enabled modules. `setModuleEnabledState()` filters the registry. Use `getEnabledModules()` (not `getAllModules()`) for user-facing lists like nav, missing tabs, calendar.

**Adding a frontend module:** Create `src/modules/<id>/index.ts` exporting a `ModuleConfig`, register it in `src/modules/setup.ts`, add theme color CSS variables to `src/index.css`.

## Routes & Code-Splitting

Routes are lazy-loaded via `lazyRouteComponent` in `src/routes-config.tsx`. Never eagerly import page components — use the `lazyRoute()` / `lazyPortalRoute()` helpers. Routes needing `validateSearch` use `createRoute` with `component: lazyRouteComponent(importer, 'ExportName')` directly.

Barrel files (`hooks/index.ts`, `api/index.ts`) use named re-exports — no `export *`.

## React Patterns

### State Synchronization
Do NOT use `useEffect` to sync state from props (triggers `react-hooks/set-state-in-effect` lint error). Use render-time state adjustment:
```tsx
const [formData, setFormData] = useState(null)
const [prevData, setPrevData] = useState(data)

if (data !== prevData) {
  setPrevData(data)
  if (data) setFormData(data)
}
```

### Hook Extraction
Separate logic from presentation. Extract state, queries, mutations, handlers into `use-<feature>.ts` in the same directory. Component should be a thin JSX shell (<50 lines).

### Error Handling
- Mutations: `onError` callback for user-facing toast
- Fire-and-forget: `void` prefix (e.g., `void queryClient.invalidateQueries(...)`)
- Never use `.catch(() => {})` — it swallows errors
- Never make `onSuccess` async

### Conditional Rendering
Priority: early returns > component map > single ternary. **Never nest ternaries.**
Use `&&` for conditional rendering without an else branch: `{condition && <X />}`. Never use `condition ? <X /> : null`.

### Null Handling
Use `??` by default. Use `||` only when falsy coalescing is intentional (0, `""`, NaN should fallback) — add a comment explaining why.

## Grouped lists

`Group`, `Row`, `IconTile` and `ProgressLine` live in `src/components/grouped-list/`. Use them for inset grouped lists (Dashboard, More, Settings, and later History).

- `Group`: inset (`px-screen`) card stack with optional `header` (uppercase footnote), `header` `action` and `footer` (footnote text under the card, for the explanatory copy a settings group needs). Pass `inset={false}` when the parent already provides the 16 px gutter (e.g. a two-column dashboard grid).
- `Row`: 44 px minimum (`min-h-tap`) row with `leading` (tile or thumbnail), `title`, `subtitle`, `trailing`, optional `chevron`, `tone` (`default` | `warning` | `destructive`). Renders a link when `href` is set, a button when `onClick` is set, otherwise a static row. Press feedback is the `press-row` tint.
- `IconTile`: 28 px rounded square for a leading glyph (`[&_svg]:size-4`). Pass the fill with `className` (`bg-amber-500`, `bg-tv-600`, …).
- `ProgressLine`: media-coloured bar (`kind` `movie` | `series`) with `role="progressbar"`. Width eases 700 ms linear. `muted` paints with `--muted-foreground`.
- `RowSkeleton`: loading stand-in with the same padding and 44 px minimum as `Row`. `leading="poster"` and `progress` match download rows.

## Settings control rows

Long-form settings screens (Import & Naming, Auto Search, RSS Sync, Server, Authentication) are
grouped lists of controls: a `Group` per topic, its explanatory text in the group `footer`, never a
paragraph between controls. The rows live in `src/components/settings/control-row.tsx`.

- `ControlRow`: 44 px row, label leading, control trailing. `StackedRow` is the full-width variant
  (label above, control below) for sliders, textareas and anything wider than a trailing slot.
- `SwitchRow`, `SelectRow`, `InputRow` (`stacked` for full width), `TextareaRow`, `SliderRow`. Every
  field is 16 px and at least 44 px tall; each one carries an `aria-label` (or a `<label htmlFor>`)
  so it is reachable by its visible label name.
- `SectionLoading` / `SectionError` (`src/components/settings/section-state.tsx`) wrap the shared
  loading and error states in the screen gutter, since `Group` supplies its own.

## Segmented control and the action presenter

`Segmented` (`src/components/ui/segmented.tsx`) is the sliding-thumb switch for two to five options. It replaces the Tabs primitive wherever tabs were only switching a filter — the Activity All/Movies/Series filter today, Library, Missing, Calendar and the Requests queue later.

- Props: `label` (the group's accessible name), `value`, `options` (`{ value, label }`), `onChange`, `className`.
- Renders a `radiogroup` of `radio` buttons. Arrow keys wrap the selection, Home and End jump to the ends, and only the selected option stays in the tab order.
- The thumb is a single absolutely positioned span translated by `index * 100%`; only `transform` animates (200 ms), and reduced motion drops the transition.

`ActionPresenter` (`src/components/presenter/`) takes one action list and picks the surface the shell calls for: a bottom-anchored `ActionSheet` on phones, a dropdown menu positioned against `anchor` on wide screens.

- `ActionItem` is `{ label, onClick?, destructive?, disabled?, keepOpen?, confirm? }`. Destructive actions render last, in the destructive colour, in their own stack (sheet) or below a separator (menu). `keepOpen` leaves the surface up after the action runs, for actions whose own progress is shown in it.
- `confirm: { title, description, actions }` routes the action through a second step instead of running it — another sheet on phones, an `AlertDialog` on wide. Use it for anything that deletes.
- `wide="dialog"` swaps the wide menu for an `AlertDialog` so the presenter *is* the confirmation on both shells (action sheet on phones, dialog on wide). `description` is the sheet's and the dialog's supporting line, and `locked` disables Cancel and refuses dismissal while an action is in flight.
- Describe the actions once and pass them in; never branch on `useViewport()` at the call site, and never hand-roll a second sheet.

## Library grid: PosterCell and ChipRow

The Library tab (`src/components/media/library-screen.tsx`, rendered by
`src/routes/movies/index.tsx` and `src/routes/series/index.tsx`) is one `Screen` per module with a
`Segmented` of `getEnabledModules()` on top; switching navigates to the other module's `basePath`
and the shell records it as the last-used Library module.

- `PosterCell` (`src/components/media/poster-cell.tsx`) is the only media cell: `PosterImage` in a
  2:3 frame, then the title, then a caption line of `StatusDot`, year and quality. It takes a
  `PosterCellItem` (`id`, `title`, `href`, `status`, `year`, `quality`, poster ids) — build one with
  `movieToCell` / `seriesToCell` from `library-cells.ts`. It renders a `Link` normally and a
  selectable button in edit mode. It replaced the per-module card components; `ModuleConfig` has no
  `cardComponent`.
- `PosterGrid` (`poster-grid.tsx`) lays cells out as an accessible list (`role="list"` plus
  `PosterGridItem`): exactly three columns on phone, `auto-fill` at the poster-size preference on
  wide. The table view is wide-only — the phone shell forces `view` to `grid`.
- `ChipRow` (`src/components/ui/chip-row.tsx`) is the primary filter set: a horizontally scrolling
  `role="group"` of `aria-pressed` chips fed by the module's `filterOptions`. No chip selected means
  the whole library; selecting chips ORs them. It replaced the `FilterDropdown` here (that
  component still serves History and Logs).
- Secondary options (sort, view, poster size, columns, select, refresh) live behind the single
  "Options" action in the top bar (`library-options.tsx`), which drives `ActionPresenter` menus.

## Pill actions and the sheet presenter

`PillAction` (`src/components/media/pill-action.tsx`) is the 44 px rounded action used under a
detail hero: an icon, a label and the media tint (`movie` or `tv`) when it is active. `PillRow`
is the flex row that holds three of them.

- `MediaSearchMonitorControls` renders its states through it. The old `xs`/`sm`/`lg`/`responsive`
  size matrix is gone: `variant="pill"` is the detail-screen row of three 44 px pills,
  `variant="row"` (the default) is the compact set of icon buttons lists use. Searching, progress,
  completed and error render inside whichever shape the variant picked.

`SheetPresenter` (`src/components/presenter/sheet-presenter.tsx`) is the form counterpart of
`ActionPresenter`: a draggable vaul bottom sheet on phones, the existing `Dialog` on wide screens,
from one set of props — `title`, `description`, `open`, `onOpenChange`, `children` and an optional
`footer`. The media edit form is its first consumer; settings and requests forms adopt it next.

- Never branch on `useViewport()` at the call site and never hand-roll a second sheet.
- vaul ships no stylesheet through its package exports, so `sheet-presenter.css` carries the
  geometry and the motion: the sheet slides on `--dur-sheet` and reduced motion swaps the slide for
  a fade. Drag is vaul's — 1:1, rubber-banded, velocity dismiss, catchable mid-animation — and it
  ignores a gesture that starts within 500 ms of the sheet opening.

## Settings screens

Every settings route renders a `Screen` with `back={usePushBack()}`; `backLabelForPathname` in
`src/components/layout/push-routes.ts` derives the label from `SETTINGS_SECTIONS` (back from a leaf
reads its section name, back from a section reads "More"). Settings paths are `isScreenFillPath`, so
both shells let the `Screen` own its own back control and scrolling.

- `/settings/media`, `/settings/download-pipeline` and `/settings/general` are section index pages
  built from `src/routes/settings/settings-section-screen.tsx`: one `Group` of `Row`s with an icon
  tile, a trailing count and a warning summary from `use-health-warnings.ts`.
- List pages (Root Folders, Quality Profiles, Version Slots, Indexers, Download Clients,
  Notifications) use `SettingsList` (loading / error / empty states around a `Group`) and
  `SettingsItemRow` (icon tile, primary line, optional subtitle, trailing detail, `Switch` and row
  actions that present through `ActionPresenter`). Add is `AddAction` — a plus passed as the
  `Screen` `trailing` slot.

## Missing and History

`/missing` and `/history` are `isScreenFillPath` and each renders a `Screen` with
`back={usePushBack()}` (label "More").

- Missing (`src/routes/missing/`) stacks two `Segmented` controls: one per `getEnabledModules()`
  module, then Missing / Upgradable. Items are one `Group` of `MissingRow`s
  (`src/components/missing/`): poster, title, count or quality subtitle, and
  `MediaSearchMonitorControls` with `variant="row"` in the trailing slot — Auto Search starts the
  search in one tap and the row carries its searching state. The row's own surface is an absolutely
  positioned `Link` behind the controls (the `SettingsItemRow` trick), so the controls stay tappable
  inside a row that opens the detail. The tab and accordion lists it replaced are gone.
- History (`src/routes/history/`) groups entries into one `Group` per day ("Today", "Yesterday",
  then the date) via `groupByDay` in `history-utils.ts`, and each `Row` matches the Dashboard's
  Recent tiles: `IconTile`, title with its year or `S02E03` qualifier, `label · detail` subtitle and
  relative time. `eventLook` in `src/lib/history-utils.ts` is the one source of the tile colour,
  glyph and label for both screens. A media `Segmented` sits above the `FilterDropdown` of event
  types and the date preset; paging is a "Load more" button, not the `Pagination` component.

## System screens and session actions

`/system/*` paths are `isScreenFillPath` like settings, and each route renders a `Screen` with
`back={usePushBack()}`. `/system/health` is the System index (title "System", back "More"): a `Group`
of rows for Scheduled Tasks, Logs and Update, then one `HealthGroup` per health category — group
header is the category name, its `action` is "Test All", and each item is a `Row` with a status
`IconTile`, the message and age as subtitle, and a per-item test button. The other three paths are
sub-screens whose back label reads "System" (`backLabelForPathname` in `push-routes.ts`).

Restart and Log out live in `use-session-actions.ts` and present through `SessionActionPresenters`
(`src/components/layout/session-action-presenters.tsx`) — one `ActionPresenter` each with
`wide="dialog"`, so both shells show the same confirmation. The restart action is `keepOpen`, so the
surface stays up and its label counts the restart down before the page reloads; `locked` holds it
there. More (phone) and the sidebar (wide) both render that component; there are no bespoke session
dialogs.

Developer Tools are `DevModeControls` (`src/components/layout/dev-mode-controls.tsx`): a
"Developer Tools" `Group` of `<label>` rows carrying the `Developer mode` and `Force Loading`
switches, shown on More. The wide shell keeps the header hammer popover instead.

## Playwright E2E

Browser tests live in `e2e/` and run against the real app in developer mode (`phone` 390×844 touch, `wide` 1440×900 mouse). `bun run test:e2e` starts the Go backend (`--dev-mode`) and Vite via Playwright `webServer`, then runs both projects. Locally both servers start through [portless](https://portless.sh) under the names `slipstream-e2e` and `slipstream-e2e-api` (prefixed with the branch inside a git worktree), so several suites can run at once in different checkouts without sharing a port or a database; `e2e/helpers/paths.ts` resolves the origins with `portless get`. CI (`CI=1`) has no proxy and uses fixed ports 3000 and 8080. Playwright always starts its own servers and refuses to reuse a leftover one; `portless prune` clears servers orphaned by a crashed run. `bun run test:e2e:headed` is the debugging variant. `bun run test:e2e:reduced-motion` runs the shell tests with `prefers-reduced-motion`.

Screenshot comparison is opt-in: `PLAYWRIGHT_SCREENSHOTS=1 bun run test:e2e:screenshots` (or the same env with `bun run test:e2e`). Baselines are stored at `e2e/snapshots/{project}/{spec}-{name}.png`. Update them with `PLAYWRIGHT_SCREENSHOTS=1 bunx playwright test --project=phone --project=wide e2e/screenshots.spec.ts --update-snapshots`. Do not treat pixel diffs as a required gate; they exist to catch layout regressions.

