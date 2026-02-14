# Lint Remediation Plan

**Starting point:** 1,796 errors, 34 warnings (after auto-fix + Prettier pass)

---

## Agent Execution Guide

You are executing a lint remediation plan for a React 19 + TypeScript frontend. Read this entire section before starting any work.

### Key Project Facts

- **UI library:** Base UI (`@base-ui/react`), NOT Radix. Use `render` prop, NOT `asChild`.
- **State:** TanStack Query for server state, Zustand for client state, `react-hook-form` for forms.
- **Routing:** TanStack Router with file-based route convention.
- **Styling:** Tailwind CSS v4 with OKLCH color system (movie=orange, tv=blue).
- **No semicolons.** Prettier enforces this.
- **Path alias:** `@/` maps to `src/`.

### Context Management

**Wave 2 (mechanical):** Batch by rule. Use one subagent per rule category. Each subagent opens files, makes targeted expression-level changes, moves on. No need to understand architecture.

**Wave 3 (architecture):** One subagent per file (or per pair of related files). Each subagent:
1. Reads the target file in full
2. Reads its direct imports (hooks, types) — NOT transitive deps
3. Plans decomposition
4. Executes
5. Runs type-check on the file

Do NOT load the entire plan into subagents. Pass only the relevant task and the patterns section below.

**Wave 4 (async):** Batch hooks by group (3-5 per subagent). The fix is formulaic: `void` prefix on `queryClient.invalidateQueries()` calls in mutation `onSuccess`/`onSettled` handlers.

**Wave 5 (nullish):** Single-pass, batch by file. Mechanical except for `Number.parseInt(x) ||` patterns — keep those as `||`.

**Wave 6 (renames):** Script-based. Single execution, no subagents.

### Verification Protocol

After **every file** you modify:
```bash
./scripts/lint/file.sh src/path/to/file.tsx
```

After **every completed wave**:
```bash
./scripts/lint/verify.sh     # Must pass — types + build + lint count
./scripts/lint/summary.sh    # Record new total in tracking table
```

**Never proceed to the next wave until the current wave's type-check and build pass.**

### Helper Scripts

| Script | Purpose |
|--------|---------|
| `./scripts/lint/summary.sh` | Total error count + top 25 rules |
| `./scripts/lint/file.sh <path>` | Lint a single file (path relative to `web/`) |
| `./scripts/lint/verify.sh` | Full verification: tsc + build + lint count |
| `./scripts/lint/count-rule.sh <rule>` | Count violations for one rule, grouped by file |

### Subagent Prompt Templates

**Wave 2 — Mechanical fix subagent:**
> Fix all `{RULE_NAME}` violations in the following files: {FILE_LIST}.
>
> This is a mechanical fix. Do not restructure or refactor — only change the specific expressions that violate the rule.
>
> Pattern: {BEFORE} → {AFTER}
>
> After fixing each file, verify with: `cd web && bunx eslint {FILE_PATH} 2>&1 | grep {RULE_NAME}`
> The grep should return no results.

**Wave 3 — Architectural refactoring subagent:**
> Refactor `src/{FILE_PATH}` ({LINE_COUNT} lines, {VIOLATION_COUNT} lint violations).
>
> Goal: Decompose to eliminate `max-lines-per-function`, `complexity`, `max-nested-callbacks`, and `no-nested-ternary` violations. The file must end up under 350 lines.
>
> Steps:
> 1. Read the file in full. Read its direct imports (hooks, types, components it uses).
> 2. Identify concerns: data fetching, state management, event handlers, derived values, presentation.
> 3. Extract a custom hook (`use-{feature}.ts`) for all non-rendering logic. Return a flat object of values and handlers.
> 4. Extract sub-components for distinct UI sections (each under 50 lines).
> 5. Replace nested ternaries with early returns or component lookup objects.
> 6. New files go in the same directory. Use kebab-case filenames.
>
> Constraints:
> - Do NOT change observable behavior. Props interfaces, route params, query keys, and mutation side effects must be preserved exactly.
> - Do NOT use `asChild` — this project uses Base UI with `render` prop.
> - Do NOT add `eslint-disable` comments.
> - Verify: `cd web && bunx tsc --noEmit 2>&1 | head -20` must show no errors.

**Wave 4 — Floating promises subagent:**
> Fix all `@typescript-eslint/no-floating-promises` violations in: {FILE_LIST}.
>
> Two patterns only:
> 1. `queryClient.invalidateQueries(...)` in mutation `onSuccess`/`onSettled` → prefix with `void`
> 2. Other async calls → wrap with `void` if fire-and-forget, or `await` if in an async context
>
> Do NOT add `.catch(() => {})` — that swallows errors silently.
> Do NOT convert `onSuccess` to `async onSuccess` just to use `await` — TanStack Query doesn't await these.

### Common Pitfalls

1. **Base UI, not Radix.** Never use `asChild`. Use `render={<Component />}` or just nest children.
2. **`SelectValue` doesn't show labels.** When value differs from display text, manually render the label in the trigger — don't use `<SelectValue />`. (See CLAUDE.md for details.)
3. **`verbatimModuleSyntax` is on.** Type-only imports MUST use `import type`. ESLint enforces this, but if you create new files, use `import type` from the start.
4. **No `useEffect` for state sync.** Use render-time state adjustment pattern. (See CLAUDE.md.)
5. **TanStack Query mutation `onSuccess` is not async.** `void` prefix is correct for invalidation calls. Do NOT make them async.
6. **Wave 3 files will have Wave 4/5 violations too.** That's fine — only fix the structural issues (max-lines, complexity, nesting). Waves 4/5 handle the rest.
7. **`no-unnecessary-condition` can be wrong.** If removing a check feels unsafe, the type is too loose. Tighten the type (e.g., make a field required instead of optional) rather than silencing the check.
8. **`no-invalid-void-type` in API layer.** These are `Promise<void>` return types on mutation functions. The fix is usually changing the return type to `Promise<unknown>` or properly typing the API response.

---

## Canonical Patterns

### Hook Extraction

```tsx
// BEFORE: God component (200+ lines)
export function MovieDetailPage() {
  const { id } = useParams({ from: '/movies/$id' })
  const navigate = useNavigate()
  const [editOpen, setEditOpen] = useState(false)
  const { data: movie, isLoading } = useMovie(Number.parseInt(id))
  const deleteMutation = useDeleteMovie()

  const handleDelete = () => {
    deleteMutation.mutate(movie.id, {
      onSuccess: () => {
        toast.success('Deleted')
        navigate({ to: '/movies' })
      },
    })
  }

  if (isLoading) return <Loading />
  // ... 150 more lines of JSX
}

// AFTER: Hook + thin shell
// use-movie-detail.ts
export function useMovieDetail() {
  const { id } = useParams({ from: '/movies/$id' })
  const navigate = useNavigate()
  const movieId = Number.parseInt(id)
  const [editOpen, setEditOpen] = useState(false)
  const query = useMovie(movieId)
  const deleteMutation = useDeleteMovie()

  const handleDelete = () => {
    deleteMutation.mutate(movieId, {
      onSuccess: () => {
        toast.success('Deleted')
        void navigate({ to: '/movies' })
      },
    })
  }

  return { movie: query.data, isLoading: query.isLoading, editOpen, setEditOpen, handleDelete }
}

// movie-detail-page.tsx (thin rendering shell)
export function MovieDetailPage() {
  const { movie, isLoading, ...handlers } = useMovieDetail()
  if (isLoading) return <Loading />
  return (
    <>
      <MovieHeader movie={movie} onEdit={handlers.setEditOpen} />
      <MovieContent movie={movie} />
    </>
  )
}
```

### Conditional Rendering

```tsx
// NEVER: nested ternary
{status === 'loading' ? <Spinner /> : status === 'error' ? <Error /> : status === 'empty' ? <Empty /> : <Content />}

// GOOD: early returns (in component body)
if (isLoading) return <Spinner />
if (isError) return <Error />
if (!data.length) return <Empty />
return <Content data={data} />

// GOOD: component map (when rendering inline in JSX)
const statusView = {
  loading: <Spinner />,
  error: <Error />,
  empty: <Empty />,
} as const

return statusView[status] ?? <Content data={data} />
```

### Floating Promise Fix

```tsx
// BEFORE: 27 violations in useSeries.ts — every invalidateQueries is a floating promise
onSuccess: () => {
  queryClient.invalidateQueries({ queryKey: seriesKeys.all })
  queryClient.invalidateQueries({ queryKey: calendarKeys.all })
},

// AFTER: void prefix
onSuccess: () => {
  void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
  void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
},
```

### Nullish Coalescing

```tsx
// SAFE to convert (null/undefined check):
const name = user.displayName || 'Anonymous'    // → ??
const items = response.data || []               // → ??
const message = error.message || 'Unknown'      // → ??

// KEEP as || (intentional falsy coalescing):
const port = Number.parseInt(input) || 3000     // NaN is falsy but not nullish
const timeout = Number.parseInt(value) || 30    // 0 would pass through ??
```

---

## Testing Strategy

### Verification Tiers

**Tier 1 — Always (every file, every wave):**
- `bunx tsc --noEmit` — types pass
- `bun run build` — build succeeds
- Lint error count went down, not up

**Tier 2 — Wave 3 only (architectural refactoring):**

Before refactoring each file, record its public interface:
- What props/params does it accept?
- What hooks does it call?
- What mutations/queries does it trigger?
- What routes does it navigate to?

After refactoring, verify all of the above are preserved. The easiest way: search the codebase for every callsite of the component and confirm the props still match.

```bash
# Find all places that import from the file you refactored
cd web && grep -r "from.*path/to/file" src/ --include="*.ts" --include="*.tsx"
```

**Tier 3 — Pre-Wave-3 recommended setup:**

Install Vitest + React Testing Library for smoke tests:
```bash
cd web && bun add -D vitest @testing-library/react @testing-library/jest-dom jsdom
```

Add to `vite.config.ts`:
```ts
/// <reference types="vitest/config" />
test: {
  environment: 'jsdom',
  setupFiles: ['./src/test/setup.ts'],
}
```

Create `src/test/setup.ts`:
```ts
import '@testing-library/jest-dom/vitest'
```

Create `src/test/providers.tsx`:
```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

export function TestProviders({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}
```

Per-file render smoke test (write before refactoring, must pass after):
```tsx
import { render } from '@testing-library/react'
import { TestProviders } from '@/test/providers'
import { MyComponent } from './my-component'

test('renders without errors', () => {
  expect(() =>
    render(<TestProviders><MyComponent {...mockProps} /></TestProviders>)
  ).not.toThrow()
})
```

### What Tests Catch

| Check | Catches |
|-------|---------|
| `tsc --noEmit` | Missing props, wrong types, broken imports |
| `bun run build` | Bundling errors, circular deps at build time |
| Render smoke test | Missing context providers, hook call order, runtime crashes |
| Manual page visit | Visual regressions, incorrect data flow |

### What Tests Don't Catch

- Subtle behavioral changes (click handler wired to wrong action)
- CSS/layout regressions
- Race conditions introduced by restructuring async logic

For these: test the affected pages manually in the browser after Wave 3.

---

## Wave 1 — Canonical Patterns

Establish the patterns before writing any code. Each item becomes a short section in `CLAUDE.md` under a new "Frontend Patterns" heading.

- [x] Define hook extraction pattern (naming: `use-<feature>.ts`, return shape, file location)
- [x] Define async error handling pattern (mutations: `onError` toast; fire-and-forget: `void` prefix)
- [x] Define conditional rendering pattern (early returns > component map > single ternary; never nested)
- [x] Define null handling pattern (`??` by default; `||` only for intentional falsy coalescing with comment)

---

## Wave 2 — Mechanical Sweep (~396 violations)

Expression-level changes. No structural impact. Batch by rule using subagents.

### `eqeqeq` — 53 violations ✅
- [x] All 53 violations fixed across 12 files

### `no-non-null-assertion` — 28 violations ✅
- [x] All 28 violations fixed across 18 files

### `no-unnecessary-condition` — 146 violations ✅
- [x] All 146 violations fixed across 55 files

### `no-array-index-key` — 29 violations ✅
- [x] All 29 violations fixed across 17 files

### `no-invalid-void-type` — 31 violations ✅
- [x] All 31 violations fixed across 18 API files (void → undefined)

### `no-unsafe-*` (assignment/argument/member-access/return) — 51 violations ✅
- [x] All 51 violations fixed across 20 files

### Accessibility (`jsx-a11y`) — 39 violations ✅
- [x] All 39 violations fixed (buttons, keyboard handlers, labels, autofocus)

### Unicorn misc ✅
- [x] `unicorn/no-array-sort` (23) — converted to `toSorted()` (required bumping tsconfig lib to ES2023)
- [x] `unicorn/prefer-number-properties` (11) — `Number.isNaN()`, `Number.parseInt()`, etc.
- [x] `unicorn/no-negated-condition` — already at 0
- [x] `unicorn/numeric-separators-style` — already at 0
- [x] `unicorn/consistent-function-scoping` (13) — hoisted to module scope
- [x] `react/no-unescaped-entities` (12) — escaped `'` and `"` in JSX text

---

## Wave 3 — Architectural Refactoring (~620 violations)

Decompose the 37 files over 350 lines. Use one subagent per file. Ordered by violation density.

### Tier 1 — Critical (50+ violations each)
- [ ] `slots/DryRunModal/index.tsx` (868 lines, 72 violations) — extract hook, break into sub-components
- [ ] `slots/MigrationPreviewModal/index.tsx` (774 lines, 71 violations) — near-identical to DryRunModal; refactor together, extract shared patterns
- [ ] `settings/sections/VersionSlotsSection.tsx` (821 lines, 40 violations) — extract hook, simplify 14 nested ternaries

### Tier 2 — High (25-49 violations each)
- [ ] `qualityprofiles/QualityProfileDialog.tsx` (37) — extract form logic hook, decompose dialog sections
- [ ] `routes/import/index.tsx` (36) — extract import logic hook, break render into sub-components
- [ ] `search/ExternalMediaCard.tsx` (34) — extract status logic, simplify conditional rendering
- [ ] `slots/ResolveConfigModal.tsx` (30) — extract hook, reduce nesting
- [ ] `search/MediaInfoModal.tsx` (30) — extract hook, decompose info sections
- [ ] `notifications/NotificationDialog.tsx` (29) — extract form hook, decompose form sections
- [ ] `routes/movies/$id.tsx` (606 lines, 28 violations) — extract `useMovieDetail` hook, split into detail sub-components
- [ ] `routes/movies/index.tsx` (26) — extract `useMovieList` hook, split toolbar/grid/table
- [ ] `routes/missing/index.tsx` (25) — extract hook, decompose tabs

### Tier 3 — Medium (20-24 violations each)
- [ ] `routes/system/health.tsx` (24)
- [ ] `routes/settings/requests/users.tsx` (24)
- [ ] `routes/settings/requests/index.tsx` (24)
- [ ] `routes/activity/history.tsx` (24)
- [ ] `routes/series/index.tsx` (23)
- [ ] `routes/requests/search.tsx` (23)
- [ ] `search/SearchModal.tsx` (22)
- [ ] `settings/sections/FileNamingSection.tsx` (22)
- [ ] `routes/series/$id.tsx` (22)
- [ ] `stores/websocket.ts` (22) — not a component; decompose into smaller store modules
- [ ] `routes/series/add.tsx` (20)
- [ ] `routes/dev/controls.tsx` (20) — dev-only, lowest priority in this tier

### Tier 4 — Lower (14-19 violations each)
- [ ] `routes/requests/$id.tsx` (19)
- [ ] `lib/table-columns.tsx` (18) — split into per-entity column files (`movie-columns.ts`, `series-columns.ts`)
- [ ] `slots/ResolveNamingModal.tsx` (17)
- [ ] `settings/sections/ServerSection.tsx` (17)
- [ ] `search/MediaSearchMonitorControls.tsx` (17)
- [ ] `routes/movies/add.tsx` (16)
- [ ] `routes/system/tasks.tsx` (15)
- [ ] `routes/activity/index.tsx` (14)
- [ ] `search/ExpandableMediaGrid.tsx` (14)
- [ ] `indexers/DefinitionSearchTable.tsx` (14)

### Non-component files
- [ ] `router.tsx` — split route definitions if over 350 lines
- [ ] `routes/dev/colors.tsx` — dev-only, lowest priority
- [ ] `components/ui/sidebar.tsx` — UI primitive, handle carefully (don't break shadcn patterns)
- [ ] `components/layout/Sidebar.tsx` — extract nav config, simplify render

---

## Wave 4 — Async Discipline (~220 violations)

Fix floating promises. Many will already be gone after Wave 3 hook extractions.

The fix for mutation hooks is formulaic. Every `queryClient.invalidateQueries()` and `queryClient.setQueryData()` in `onSuccess`/`onSettled` needs a `void` prefix. See the pattern in the Canonical Patterns section.

### Hooks (highest concentration)
- [ ] `hooks/useSeries.ts` (27)
- [ ] `hooks/useSlots.ts` (14)
- [ ] `hooks/useMovies.ts` (14)
- [ ] `hooks/useProwlarr.ts` (12)
- [ ] `hooks/useImport.ts` (8)
- [ ] `hooks/useAutosearch.ts` (7)
- [ ] `hooks/portal/useRequests.ts` (6)
- [ ] `hooks/admin/useAdminRequests.ts` (6)
- [ ] `hooks/useLibrary.ts` (5)
- [ ] `hooks/admin/useAdminUsers.ts` (5)

### Stores
- [ ] `stores/websocket.ts` (12) — handle reconnection/message promises

### Remaining hooks (4 each)
- [ ] `useUpdate.ts`, `useSearch.ts`, `useRootFolders.ts`, `useQueue.ts`, `useIndexers.ts`

### Route-level (stragglers after Wave 3)
- [ ] `routes/missing/index.tsx` (4)
- [ ] `routes/requests/auth/login.tsx` (3)
- [ ] `routes/movies/$id.tsx` (3)
- [ ] Remaining scattered across routes/components

---

## Wave 5 — Nullish Coalescing (~256 violations)

Convert `||` → `??`. Review each for `Number.parseInt`/falsy-value edge cases.

### High-count files
- [ ] `routes/missing/index.tsx` (14)
- [ ] `routes/settings/requests/users.tsx` (12)
- [ ] `routes/settings/requests/index.tsx` (11), `routes/series/index.tsx` (11), `routes/movies/index.tsx` (11), `lib/table-columns.tsx` (11)
- [ ] `slots/ResolveNamingModal.tsx` (10), `notifications/NotificationDialog.tsx` (10)
- [ ] `qualityprofiles/QualityProfileDialog.tsx` (9)
- [ ] `routes/requests/search.tsx` (8), `routes/import/index.tsx` (8)
- [ ] `routes/activity/history.tsx` (7), `slots/ResolveConfigModal.tsx` (7), `search/SearchModal.tsx` (7)
- [ ] `routes/system/health.tsx` (6)
- [ ] Remaining files (5 or fewer each)

### Keep as `||` — do NOT convert
- [ ] Any `Number.parseInt(x) || default` — NaN is falsy but not nullish
- [ ] Any `|| ""` where empty string should genuinely trigger the fallback
- [ ] Any `|| 0` where zero should genuinely trigger the fallback
- [ ] If keeping `||`, add inline comment: `// intentional || — falsy coalescing for NaN/0/""`

---

## Wave 6 — File Renames (161 files)

Rename all non-kebab-case files. One atomic operation.

- [ ] Write a Bun script (`scripts/lint/rename-to-kebab.ts`) that:
  1. Finds all `.ts`/`.tsx` files in `web/src/` not matching kebab-case
  2. Computes the kebab-case equivalent (`useMovies.ts` → `use-movies.ts`, `AdminAuthGuard.tsx` → `admin-auth-guard.tsx`)
  3. Runs `git mv` for each file
  4. Reads every `.ts`/`.tsx` file and updates import paths (both `@/` alias and relative)
  5. Handles barrel files (`index.ts`) that re-export renamed modules
- [ ] Run `bun run build` to verify
- [ ] Run `bun run lint 2>&1 | grep filename-case` to verify zero violations

### Files to rename (by category)
- `src/App.tsx` → `src/app.tsx`
- Hooks: all `use*.ts` files (`useMovies.ts` → `use-movies.ts`, etc.) — ~50 files
- Components: PascalCase `.tsx` files (`AdminAuthGuard.tsx` → `admin-auth-guard.tsx`, etc.) — ~90 files
- API: `downloadClients.ts`, `qualityProfiles.ts`, `rootFolders.ts`
- Types: `downloadClient.ts`, `qualityProfile.ts`, `rootFolder.ts`
- Stores: `portalAuth.ts`, `portalDownloads.ts`
- Nav: `SystemNav.tsx`, `MediaNav.tsx`, `RequestsNav.tsx`, `DownloadsNav.tsx`

---

## Tracking

After each wave, run `./scripts/lint/summary.sh` and update:

| Wave | Target | Errors Before | Errors After |
|------|--------|--------------|-------------|
| Auto-fix + format | — | 3,801 | 1,796 |
| Wave 1 | Patterns | — | — |
| Wave 2 | Mechanical | 1,796 | 1,328 |
| Wave 3 | Architecture | | |
| Wave 4 | Async | | |
| Wave 5 | Nullish | | |
| Wave 6 | Renames | | |
