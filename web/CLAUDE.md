# Frontend Agent Documentation

### Making Changes
When making changes to this file, ALWAYS make the same update to AGENTS.md in the same directory.

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

`Group`, `Row`, `IconTile` and `ProgressLine` live in `src/components/grouped-list/`. Use them for inset grouped lists (Dashboard, More, and later History/Settings).

- `Group`: inset (`px-screen`) card stack with optional `header` (uppercase footnote) and `header` `action`. Pass `inset={false}` when the parent already provides the 16 px gutter (e.g. a two-column dashboard grid).
- `Row`: 44 px minimum (`min-h-tap`) row with `leading` (tile or thumbnail), `title`, `subtitle`, `trailing`, optional `chevron`, `tone` (`default` | `warning` | `destructive`). Renders a link when `href` is set, a button when `onClick` is set, otherwise a static row. Press feedback is the `press-row` tint.
- `IconTile`: 28 px rounded square for a leading glyph (`[&_svg]:size-4`). Pass the fill with `className` (`bg-amber-500`, `bg-tv-600`, …).
- `ProgressLine`: media-coloured bar (`kind` `movie` | `series`) with `role="progressbar"`. Width eases 700 ms linear. `muted` paints with `--muted-foreground`.
- `RowSkeleton`: loading stand-in with the same padding and 44 px minimum as `Row`. `leading="poster"` and `progress` match download rows.

## Segmented control and the action presenter

`Segmented` (`src/components/ui/segmented.tsx`) is the sliding-thumb switch for two to five options. It replaces the Tabs primitive wherever tabs were only switching a filter — the Activity All/Movies/Series filter today, Library, Missing, Calendar and the Requests queue later.

- Props: `label` (the group's accessible name), `value`, `options` (`{ value, label }`), `onChange`, `className`.
- Renders a `radiogroup` of `radio` buttons. Arrow keys wrap the selection, Home and End jump to the ends, and only the selected option stays in the tab order.
- The thumb is a single absolutely positioned span translated by `index * 100%`; only `transform` animates (200 ms), and reduced motion drops the transition.

`ActionPresenter` (`src/components/presenter/`) takes one action list and picks the surface the shell calls for: a bottom-anchored `ActionSheet` on phones, a dropdown menu positioned against `anchor` on wide screens.

- `ActionItem` is `{ label, onClick?, destructive?, confirm? }`. Destructive actions render last, in the destructive colour, in their own stack (sheet) or below a separator (menu).
- `confirm: { title, description, actions }` routes the action through a second step instead of running it — another sheet on phones, an `AlertDialog` on wide. Use it for anything that deletes.
- Describe the actions once and pass them in; never branch on `useViewport()` at the call site, and never hand-roll a second sheet.

## Playwright E2E

Browser tests live in `e2e/` and run against the real app in developer mode (`phone` 390×844 touch, `wide` 1440×900 mouse). `bun run test:e2e` starts the Go backend (`--dev-mode`) and Vite via Playwright `webServer`, then runs both projects. `bun run test:e2e:headed` is the debugging variant. `bun run test:e2e:reduced-motion` runs the shell tests with `prefers-reduced-motion`.

Screenshot comparison is opt-in: `PLAYWRIGHT_SCREENSHOTS=1 bun run test:e2e:screenshots` (or the same env with `bun run test:e2e`). Baselines are stored at `e2e/snapshots/{project}/{spec}-{name}.png`. Update them with `PLAYWRIGHT_SCREENSHOTS=1 bunx playwright test --project=phone --project=wide e2e/screenshots.spec.ts --update-snapshots`. Do not treat pixel diffs as a required gate; they exist to catch layout regressions.

