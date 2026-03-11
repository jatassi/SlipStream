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
