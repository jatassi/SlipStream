# Frontend Code Review — Adversarial Audit

*Perspective: Expert React/TypeScript developer reviewing for code quality issues.*

**Codebase stats:** 59,222 lines of TypeScript across 620 files. Zero test files.

---

## Critical Issues

### No Error Boundaries or Suspense Fallbacks

Zero `<ErrorBoundary>` components in the entire app. Routes use `lazyRouteComponent()` for code-splitting but have no `<Suspense>` fallbacks. A chunk load failure (CDN blip, deploy during active session) results in a white screen with no recovery path.

**Affected:** Root layout, all 40+ lazy routes.

### Zero Test Coverage

Not a single `.test.tsx` or `.spec.ts` file. 60k lines of frontend code — search, downloads, quality profiles, JWT-authenticated portal — with zero automated verification.

---

## Architecture Issues

### Dual State Management (Zustand + TanStack Query)

Server state is fetched via TanStack Query then `useEffect`-synced into Zustand stores, creating multiple sources of truth:

```ts
// use-queue.ts
useEffect(() => {
  if (query.data !== undefined) {
    const items = query.data.items
    setQueueItems(items)      // Zustand store #1
    setPortalQueue(items)     // Zustand store #2
  }
}, [query.data, setQueueItems, setPortalQueue])
```

Queue data now lives in three places: query cache, `useDownloadingStore`, `usePortalDownloadsStore`. Same pattern in `useLogs()` syncing to `useLogsStore`.

**Fix:** Use TanStack Query as single source of truth for server state. Restrict Zustand to transient UI state (modals, form drafts, layout preferences).

### Hook Over-Proliferation

38 custom hooks, many trivially thin:

- `use-global-loading.ts` — 5 lines, wraps a single Zustand selector
- `use-storage.ts` — 14 lines, wraps one API call
- `use-preferences.ts` — 16 lines, one function

The barrel file `hooks/index.ts` is 257 lines with 80+ named exports, hurting tree-shaking and discoverability.

**Fix:** Inline trivial hooks. Import directly from modules instead of barrel.

### Zustand Store Sprawl

11 separate stores. `useUIStore` is a god object with 12+ pieces of state (sidebar, menus, theme, views, columns, notifications, global loading). Other stores (`useAutoSearchStore`, `useProgressStore`, `useDownloadingStore`) are single-purpose but scattered with no clear organizational principle.

**Fix:** Consolidate into 2-3 stores max. UI layout/theme in one, domain-specific in others.

---

## Data Layer Issues

### Silent Mutation Failures

Most mutations lack `onError` callbacks:

```ts
export function useAutoSearchMovie() {
  return useMutation({
    mutationFn: (movieId: number) => autosearchApi.searchMovie(movieId),
    onSuccess: (result) => { /* ... */ },
    // No onError — user never knows if it failed
  })
}
```

Same for `useTestDownloadClient()`, most delete mutations, and bulk operations. The `withToast` utility exists for this but is barely used.

**Fix:** Add `onError` handlers to all mutations, leveraging `withToast`.

### Bulk Operations: N+1 Frontend Edition

```ts
bulkDelete: (ids: number[]) =>
  Promise.all(ids.map((id) => apiFetch(`/movies/${id}`, { method: 'DELETE' })))
```

Deleting 50 movies fires 50 individual HTTP requests. No batch endpoint, no transaction safety. If request #47 fails, 46 are already deleted with no rollback.

**Fix:** Implement batch endpoints on backend, single request from frontend.

### No Request Cancellation

`apiFetch` has zero `AbortController` support. Navigating away from a slow search leaves the request in-flight. Starting a new search while one's pending creates a race condition.

**Fix:** Accept `AbortSignal` in `apiFetch`, wire TanStack Query's built-in cancellation.

### WebSocket Reconnection: No Backoff

```ts
function scheduleReconnect(get, set) {
  setTimeout(() => { get().connect() }, RECONNECT_DELAY) // Fixed 3000ms
}
```

Server goes down and every client hammers it every 3 seconds indefinitely. No exponential backoff, no jitter, no max retries.

**Fix:** Implement exponential backoff with jitter (3s → 6s → 12s → 60s max).

### Query Client Defaults Too Aggressive

```ts
staleTime: 1000 * 60 * 5  // 5 min for ALL queries
```

Download queue data (changes every few seconds) shares the same 5-minute stale time as static config. No `gcTime` configured.

**Fix:** Per-domain stale times. Queue/progress: 10s. Config/profiles: 5m. Static metadata: 30m.

### Inconsistent Query Keys

```ts
// use-defaults.ts — lists() and list() are identical
const defaultsKeys = {
  lists: () => [...defaultsKeys.all, 'list'] as const,
  list: () => [...defaultsKeys.all, 'list'] as const,
}

// use-import.ts — ignores createQueryKeys helper entirely
queryKey: ['importSettings']  // hardcoded string array
```

**Fix:** Standardize all query keys through `createQueryKeys`. Remove duplicates.

### Missing Optimistic Updates

Most mutations wait for server response before updating UI. Toggling "monitored" on a movie requires a full round-trip. Only portal request creation has optimistic updates.

**Fix:** Add optimistic updates for common toggle/update mutations.

---

## Form Handling

### No Form Library or Validation

No react-hook-form, no Zod schema validation. The "add movie" page drills **14 individual props** (state values + setters) through to a configure component. Validation is limited to disabling the submit button when required fields are empty — no field-level errors, no schema enforcement.

**Fix:** Adopt react-hook-form + Zod for all forms. Use form context instead of prop drilling.

### Movie/Series Code Duplication

- `movie-list-layout.tsx` (102 lines) and `series-list-layout.tsx` (107 lines): ~85% identical
- `add-movie-configure.tsx` (196 lines) and `add-series-configure.tsx` (264 lines): ~75% identical
- Both contain identical `FolderSelect`, `ProfileSelect`, `ToggleField`, `FormActions` sub-components

**Fix:** Create generic `<MediaListLayout>` and `<MediaConfigureForm>` with media-type-specific fields.

---

## Component Quality

### Oversized Components

Multiple components exceed the project's own `max-lines: 350` ESLint rule:

| File | Lines | Issue |
|------|-------|-------|
| `mapping-step.tsx` | 356 | 4 internal components |
| `indexer-dialog.tsx` | 349 | 7 internal components |
| `episode-table.tsx` | 347 | Mixed table + slot logic |
| `slot-debug-panel.tsx` | 336 | Two distinct test panels |

**Fix:** Extract internal components to separate files.

### Inconsistent Class Name Patterns

Some components use `cn()` utility (correct), others use template string interpolation:

```tsx
// Template string (inconsistent)
className={`group relative flex ... ${editingName ? 'border-primary' : 'border-transparent'}`}

// cn() utility (preferred)
className={cn('group relative flex ...', editingName ? 'border-primary' : 'border-transparent')}
```

**Fix:** Standardize on `cn()` for all conditional class names.

### Accessibility Gaps

- Expandable grids missing `aria-expanded`
- Series cards with clickable regions and no `aria-label`
- Checkboxes without associated labels
- Icon-only buttons without screen reader text

**Fix:** Audit all interactive elements for ARIA attributes.

---

## Type Safety

### Error Type Looseness

```ts
// api/client.ts
let errorData: unknown = null
try {
  errorData = (await res.json()) as unknown
} catch { }
throw new ApiError(
  res.status,
  errorData as { message?: string; error?: string } | null  // unsafe cast
)
```

**Fix:** Validate error response shape with a type guard before casting.

### `as unknown as` Assertions

4 instances in notification form handling where type gymnastics bypass safety:

```ts
;(resetData as unknown as Record<string, unknown>)[t.key] = false
```

**Fix:** Create proper type helpers for notification form data.

---

## Minor Issues

### Barrel File `export *` Violation

`api/index.ts` line 36 uses `export * as adminApi from './admin'` despite project rules prohibiting `export *`.

### Missing `gcTime` on Query Client

No explicit garbage collection time configured. Cached data persists with default behavior.

### Portal Logout Clears All Caches

```ts
onSuccess: () => {
  storeLogout()
  queryClient.clear()  // nukes ALL cached data, including admin session
}
```

### Status Config Duplication

`STATUS_CONFIG` defined separately in portal routes and admin routes with overlapping content.

---

## What's Actually Good

- **Zero `any` types** across the entire codebase
- **Zero `console.log` statements** in production code
- **Zero abandoned TODOs**
- **Only 8 ESLint disables**, all justified
- **Consistent kebab-case file naming** across 620 files
- **Clean dependency count** — 28 production deps
- **Well-designed theme system** with `movie-*` / `tv-*` color classes
- **Proper TypeScript discipline** — strict mode, no type escape hatches
- **Good hook extraction pattern** — logic separated from rendering in most components
- **Query key factory** (`createQueryKeys`) is well-designed when used

---

## Priority Fixes

### P0 — Ship Blockers
1. Add error boundaries at route/layout level
2. Add Suspense fallbacks for lazy routes
3. Add `onError` handlers to all mutations

### P1 — Reliability
4. Implement WebSocket exponential backoff
5. Add AbortController support to `apiFetch`
6. Per-domain query stale times (queue: 10s, config: 5m)
7. Fix portal logout clearing all caches

### P2 — Maintainability
8. Eliminate dual Zustand/Query state syncing
9. Consolidate movie/series duplicated layouts into generic components
10. Adopt react-hook-form + Zod for form handling
11. Break up 350+ line component files
12. Inline trivial wrapper hooks, reduce barrel file size

### P3 — Polish
13. Accessibility audit — ARIA attributes on all interactive elements
14. Standardize on `cn()` for conditional class names
15. Standardize all query keys through `createQueryKeys`
16. Add frontend test infrastructure and initial coverage
