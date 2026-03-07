# Phase 10: Frontend Module Framework — Implementation Plan

## Overview

Build the generic frontend module registration system and deduplicate movie/TV frontend code. By this phase, all backend APIs are stable — module-scoped quality profiles, discriminator-based shared tables, module-provided routes, and the module registry are all in place.

**Spec sections covered:** §13.1, §13.2, §13.3, §13.4, §13.5, §19.4, Appendix C (frontend rows)

---

## Context Management Instructions

This phase is frontend-heavy and touches many files. The executing agent MUST manage context aggressively:

### Delegation Strategy

- **Split every task group below into a subagent** using `subagent_type: "general-purpose"` in backgrounded mode. Each task group is designed to be independently executable.
- The main agent's role is **orchestration only**: launch subagents, validate their output via lint + spot-checks, and sequence dependent groups.
- Subagents MUST be instructed that they are **forbidden to use `git stash` or similar commands that affect the entire worktree**.
- When launching subagents for targeted code changes, prefer using **Sonnet**.

### Validation Protocol

After each task group completes:
1. Run `cd web && bun run lint` — fix any lint errors introduced.
2. Run `make build` — confirm the app compiles.
3. Spot-check 1-2 modified files by reading ~20 lines around the key change. Do NOT read entire files back into main context.
4. If a subagent's changes break lint or build, resume that subagent with the error output and let it fix.

### Context Hygiene

- Do NOT read large files into the main agent context. Use subagents or the Explore agent for research.
- When writing instructions for subagents, include the specific file paths and line ranges they need to modify. Front-load the context so subagents don't waste tokens re-exploring.
- Each subagent prompt should be self-contained: include the relevant type definitions, the target file paths, and the expected API shape.

---

## Prerequisites

Before starting Phase 10, confirm:
- [ ] Backend module registry exists at `internal/modules/registry.go` (or similar) with `ListEnabledModules()` API
- [ ] Backend exposes `GET /api/v1/modules` returning enabled modules with their metadata (id, name, pluralName, icon, themeColor, nodeSchema)
- [ ] Quality profiles are module-scoped (`module_type` column on `quality_profiles`)
- [ ] Root folders have `module_type` column
- [ ] Notification events are module-declared (JSON event toggles, not hard-coded columns)
- [ ] WebSocket library events use the generic shape: `{ module, entityType, entityId, action }`
- [ ] Calendar API returns `moduleType` field on events (not just `mediaType: 'movie' | 'episode'`)
- [ ] Missing/wanted API accepts `moduleType` filter parameter
- [ ] Download queue uses `module_type` discriminator
- [ ] History API uses `module_type` discriminator

If any prerequisite is missing, the corresponding task group below notes what to stub or defer.

---

## Deferred Items from Earlier Phases

Earlier phases deferred specific items to Phase 10. This section tracks them and assigns each to the appropriate task group or to Phase 11.

### Covered by Phase 10 Task Groups

| Deferred Item | Source | Assigned To |
|---|---|---|
| Frontend JSON shape changes — update API response types to use `moduleType`/`entityType` instead of `movieId`/`seriesId`/`episodeId` where applicable | Phase 1 (line 806) | TG15 |
| `PREDEFINED_QUALITIES` — frontend dynamically loads quality items from `/qualityprofiles/qualities` instead of hard-coding the list | Phase 2 (AD #6) | TG13 |

### Reassigned to Phase 11 (Backend Cleanup)

These are backend items that the Phase 10 frontend work does not depend on. They belong in Phase 11 ("Contributor Tooling & Final Cleanup") alongside the Appendix C backend deduplication work.

| Deferred Item | Source |
|---|---|
| Remove legacy `calendar.Service` helper functions (`getMovieEvents`, `getEpisodeEvents`, `resolveMovieStatus`, `movieRowToEvents`, `createEpisodeEvent`, `createSeasonReleaseEvent`) — legacy path no longer needed once module framework is validated | Phase 4 (line 5475) |
| Consolidate `streamingServicesWithEarlyRelease` map duplicated between `internal/calendar/service.go` and `internal/modules/tv/calendar.go` | Phase 4 (line 5618) |
| Align `UpdateUnreleasedMoviesToMissing` SQL to use earliest-of digital/physical instead of priority chain | Phase 4 (line 5619) |
| Generic framework cascade (§6.1) — parent `monitored` propagation to descendants via node schema introspection | Phase 4 (line 5620) |
| Add TV `existing` monitoring preset — monitor only episodes that currently have files (spec §6.1 lists it, deferred because it requires a new SQL query) | Phase 4 (decision 9) |
| Eliminate `decisioning.SearchableItem` struct — migrate autosearch pipeline to use `module.SearchableItem` interface throughout | Phase 5 (lines 6585, 6874–6875) |
| Refactor `SelectBestRelease` to use `SearchStrategy.FilterRelease` instead of hard-coded `shouldSkipTVRelease` | Phase 5 (line 6878) |
| Full scanner refactoring — `FileParser` replaces scanner internals instead of wrapping | Phase 6 (line 6978) |
| Extract shared naming template helpers (`qualityVariables()`, `mediaInfoVariables()`, `metadataVariables()`) to a shared location | Phase 6 (lines 7781, 7944) |
| Remove unused naming columns from `import_settings` table (columns became unused when `module_naming_settings` was introduced) | Phase 6 (line 8034) |
| Remove `populateLegacyFields` bridge in renamer — no longer needed when renamer is fully generic | Phase 6 (line 8221) |
| Remove legacy `availability.Service`, `calendar.Service`, `missing.Service` dispatcher fallback paths once module framework is fully validated | Phase 4 (line 4324) |

### Deferred to Phase 10+ (When 3rd Module Added)

These items provide no value for just movie+TV and are explicitly deferred until a 3rd module makes the current approach insufficient.

| Deferred Item | Source |
|---|---|
| Wire scoring pipeline through module `QualityDefinition` — `matchExact`/`bestQualityByField` in `internal/indexer/scoring/` use `quality.PredefinedQualities` directly instead of module-registered items via `GetQualitiesForModule` | Phase 5 (deferred item 2, line 41) |
| Generic portal duplicate detection via module external IDs (replacing per-media-type request queries) | Phase 8 (line 10532) |
| Generic hierarchy walk for request completion inference (replacing module-dispatched `CheckAvailability`) | Phase 8 (line 10607) |

---

## Task Group 1: Module Registry Types & Runtime Config ✅ COMPLETE

**Goal:** Define the `ModuleConfig` TypeScript interface and the module registry that the rest of the frontend consumes.

**Depends on:** Nothing (foundational)

### Files to create

#### `web/src/modules/types.ts`

Define the core module config type. This is the frontend equivalent of spec §13.4:

```typescript
import type { ComponentType, ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'

// Matches the backend ModuleDescriptor + frontend-specific fields
export type ModuleConfig = {
  // Identity (from backend)
  id: string                          // e.g. "movie", "tv", "music"
  name: string                        // e.g. "Movies", "Series", "Music"
  singularName: string                // e.g. "Movie", "Series", "Artist"
  pluralName: string                  // e.g. "Movies", "Series", "Artists"
  icon: LucideIcon                    // Lucide icon component
  themeColor: string                  // CSS theme key: "movie", "tv", "music"

  // Routing
  basePath: string                    // e.g. "/movies", "/series", "/music"
  routes: ModuleRouteConfig[]

  // Data layer
  queryKeys: ModuleQueryKeys
  wsInvalidationRules: WSInvalidationRule[]

  // List page
  filterOptions: ModuleFilterOption[]
  sortOptions: ModuleSortOption[]
  tableColumns: ModuleTableColumns

  // Components (lazy-loaded)
  cardComponent: ComponentType<ModuleCardProps>
  detailComponent: ComponentType<ModuleDetailProps>
  addConfigFields?: ComponentType<ModuleAddConfigProps>

  // API
  api: ModuleApi
}

export type ModuleRouteConfig = {
  path: string                        // Relative to basePath, e.g. "/" for list, "/$id" for detail, "/add"
  id: string                          // Route ID for TanStack Router
}

export type ModuleQueryKeys = {
  all: readonly string[]
  list: () => readonly unknown[]
  detail: (id: number) => readonly unknown[]
  // Modules with hierarchy can add extra keys
  [key: string]: unknown
}

export type WSInvalidationRule = {
  // Pattern to match against `${module}:${action}` WS event types
  pattern: string
  // Query keys to invalidate when matched
  queryKeys: readonly unknown[][]
  // Additional side effects (e.g. invalidate missing counts)
  alsoInvalidate?: readonly unknown[][]
}

export type ModuleFilterOption = {
  value: string
  label: string
  icon: LucideIcon
}

export type ModuleSortOption = {
  value: string
  label: string
}

export type ModuleTableColumns = {
  static: unknown[]     // ColumnDef[] — typed per module
  defaults: string[]    // Default visible column IDs
}

// Props passed to module-provided components
export type ModuleCardProps = {
  item: ModuleEntity
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}

export type ModuleDetailProps = {
  id: number
}

export type ModuleAddConfigProps = {
  form: unknown         // react-hook-form instance — typed per module
}

// Minimal entity shape that the framework requires
export type ModuleEntity = {
  id: number
  title: string
  sortTitle: string
  status: string
  monitored: boolean
  qualityProfileId: number
  rootFolderId: number
  path: string
  sizeOnDisk?: number
  addedAt: string
}

// Generic CRUD API shape every module provides
export type ModuleApi = {
  list: (options?: Record<string, unknown>) => Promise<ModuleEntity[]>
  get: (id: number) => Promise<ModuleEntity>
  update: (id: number, data: Record<string, unknown>) => Promise<ModuleEntity>
  delete: (id: number, deleteFiles?: boolean) => Promise<void>
  bulkDelete: (ids: number[], deleteFiles?: boolean) => Promise<void>
  bulkUpdate: (ids: number[], data: Record<string, unknown>) => Promise<void>
  bulkMonitor: (ids: number[], monitored: boolean) => Promise<unknown>
  search: (id: number) => Promise<void>
  refresh: (id: number) => Promise<ModuleEntity>
  refreshAll: () => Promise<unknown>
}
```

#### `web/src/modules/registry.ts`

Module registration and lookup:

```typescript
import type { ModuleConfig } from './types'

const modules = new Map<string, ModuleConfig>()

export function registerModule(config: ModuleConfig): void {
  if (modules.has(config.id)) {
    throw new Error(`Module "${config.id}" is already registered`)
  }
  modules.set(config.id, config)
}

export function getModule(id: string): ModuleConfig | undefined {
  return modules.get(id)
}

export function getModuleOrThrow(id: string): ModuleConfig {
  const mod = modules.get(id)
  if (!mod) throw new Error(`Module "${id}" not found`)
  return mod
}

export function getAllModules(): ModuleConfig[] {
  return Array.from(modules.values())
}

export function getEnabledModules(): ModuleConfig[] {
  // Initially all registered modules are enabled.
  // When backend provides enabled/disabled state, filter here.
  return getAllModules()
}
```

#### `web/src/modules/index.ts`

Barrel export:

```typescript
export type { ModuleConfig, ModuleEntity, ModuleApi, /* ... */ } from './types'
export { registerModule, getModule, getModuleOrThrow, getAllModules, getEnabledModules } from './registry'
```

### Files to create — Module registrations

#### `web/src/modules/movie/index.ts`

Register the movie module config. This file imports existing movie components/API and packages them into a `ModuleConfig`. Implementation:

- `id: "movie"`, `name: "Movies"`, `singularName: "Movie"`, `pluralName: "Movies"`
- `icon: Film` (from lucide-react — matches current `sidebar-nav-config.ts:21`)
- `themeColor: "movie"` (matches current CSS variables `movie-*` in `web/CLAUDE.md`)
- `basePath: "/movies"`
- `api`: Wrap existing `moviesApi` from `web/src/api/movies.ts`
- `queryKeys`: Wrap existing `movieKeys` from `web/src/hooks/use-movies.ts`
- `cardComponent`: Existing `MovieCard` from `web/src/components/movies/movie-card.tsx`
- `filterOptions`: The array currently defined in `web/src/routes/movies/movie-list-layout.tsx:18-26`
- `sortOptions`: The array currently defined in `web/src/routes/movies/movie-list-layout.tsx:28-36`
- `wsInvalidationRules`: Derived from current `handlerMap` entries for `movie:added/updated/deleted` in `web/src/stores/ws-message-handlers.ts:180-183`

#### `web/src/modules/tv/index.ts`

Register the TV module config. Same pattern:

- `id: "tv"`, `name: "Series"`, `singularName: "Series"`, `pluralName: "Series"`
- `icon: Tv` (from lucide-react — imported at `sidebar-nav-config.ts:13`, used at line 22)
- `themeColor: "tv"`
- `basePath: "/series"`
- `api`: Wrap existing `seriesApi` from `web/src/api/series.ts`
- `queryKeys`: Wrap existing `seriesKeys` from `web/src/hooks/use-series.ts`
- `cardComponent`: Existing `SeriesCard` from `web/src/components/series/series-card.tsx`
- `filterOptions`: The array currently defined in `web/src/routes/series/series-list-layout.tsx:20-30` (note: includes TV-specific `continuing`/`ended` filters)
- `sortOptions`: The array currently defined in `web/src/routes/series/series-list-layout.tsx:32-40`
- `wsInvalidationRules`: Derived from current `handlerMap` entries for `series:added/updated/deleted` in `web/src/stores/ws-message-handlers.ts:183-185`

#### `web/src/modules/setup.ts`

Called at app startup (from `main.tsx` or `App.tsx`). Imports and registers all compiled-in modules:

```typescript
import { registerModule } from './registry'
import { movieModuleConfig } from './movie'
import { tvModuleConfig } from './tv'

export function setupModules(): void {
  registerModule(movieModuleConfig)
  registerModule(tvModuleConfig)
}
```

### Integration point

In app entry (`web/src/main.tsx` or wherever `ReactDOM.createRoot` is called), call `setupModules()` before rendering. This ensures the registry is populated before any component reads from it.

---

## Task Group 2: Sidebar Navigation — Module-Driven ✅ COMPLETE

**Goal:** Replace the hard-coded `libraryNavItems` array in the sidebar with module-driven entries. (Spec §13.2)

**Depends on:** Task Group 1

### Current state

- `web/src/components/layout/sidebar-nav-config.ts:20-23` hard-codes:
  ```typescript
  export const libraryNavItems: NavItem[] = [
    { title: 'Movies', href: '/movies', icon: Film, theme: 'movie' },
    { title: 'Series', href: '/series', icon: Tv, theme: 'tv' },
  ]
  ```
- `web/src/components/layout/sidebar-types.ts:5` hard-codes `theme?: 'movie' | 'tv'`

### Changes

#### `web/src/components/layout/sidebar-types.ts`

Change `theme` from `'movie' | 'tv'` to `string | undefined`:

```typescript
export type NavItem = {
  title: string
  href: string
  icon: React.ElementType
  theme?: string           // was: 'movie' | 'tv'
  activePrefix?: string
}
```

#### `web/src/components/layout/sidebar-nav-config.ts`

Replace the hard-coded `libraryNavItems` with a function that reads from the module registry:

```typescript
import { getEnabledModules } from '@/modules'
import type { NavItem } from './sidebar-types'

export function getLibraryNavItems(): NavItem[] {
  return getEnabledModules().map((mod) => ({
    title: mod.name,
    href: mod.basePath,
    icon: mod.icon,
    theme: mod.themeColor,
  }))
}
```

Keep `discoverNavItems`, `settingsGroup`, `systemNavItem`, `standaloneActions` unchanged — they are framework sections, not module-provided.

#### `web/src/components/layout/sidebar.tsx`

Change `sidebar.tsx:171` from:
```typescript
<NavSection items={libraryNavItems} collapsed={sidebar.sidebarCollapsed} />
```
to:
```typescript
<NavSection items={getLibraryNavItems()} collapsed={sidebar.sidebarCollapsed} />
```

Import `getLibraryNavItems` instead of `libraryNavItems` from `sidebar-nav-config`.

#### `web/src/components/layout/sidebar-nav-link.tsx`

Audit this file: the `NavLink` component applies theme-colored active states. Currently it likely checks `theme === 'movie'` or `theme === 'tv'` to pick CSS classes. Change to use the theme string dynamically:

- For active state styling, use `text-${theme}-500` via a utility or a lookup map. Since Tailwind needs to see full class names at build time, use a small map:

```typescript
const THEME_ACTIVE_CLASSES: Record<string, string> = {
  movie: 'text-movie-500 bg-movie-500/10',
  tv: 'text-tv-500 bg-tv-500/10',
}
// Fallback for new modules:
function getThemeClasses(theme?: string): string {
  if (!theme) return 'text-foreground'
  return THEME_ACTIVE_CLASSES[theme] ?? 'text-foreground'
}
```

**Important**: For new modules added later, their theme colors need to be added to both `web/src/index.css` (CSS variables) and this map. Document this in the shared UI component catalog (Task Group 10).

#### `web/src/components/layout/sidebar.tsx` — MissingBadge

The `MissingBadge` at line 20-36 currently hard-codes movie/episode counts with `text-movie-500`/`text-tv-500`. After this phase, the missing counts API should return per-module counts. For now, keep the current implementation since the backend API hasn't changed yet. Add a `// TODO: Module system — derive from per-module missing counts` comment.

---

## Task Group 3: Generic WebSocket Event Handling ✅ COMPLETE

**Goal:** Replace the hard-coded library event handler with a module-driven dispatch. (Spec §10.1, Appendix C row "WebSocket event handlers")

**Depends on:** Task Group 1

### Current state

`web/src/stores/ws-message-handlers.ts:30-37` hard-codes:
```typescript
function handleLibraryEvent(queryClient, type) {
  const isMovie = type.startsWith('movie:')
  const keys = isMovie ? movieKeys.all : seriesKeys.all
  void queryClient.invalidateQueries({ queryKey: keys })
  void queryClient.invalidateQueries({ queryKey: missingKeys.counts() })
}
```

And `ws-types.ts:9-13` hard-codes movie/series library message types.

### Changes

#### `web/src/stores/ws-types.ts`

Replace the hard-coded `LibraryMessage` type. The backend will send generic entity events (spec §10.1):

```typescript
type LibraryMessage = {
  type: `${string}:${'added' | 'updated' | 'deleted'}`
  payload: unknown
  timestamp: string
}
```

This already works with the existing union — just ensure the type is permissive enough for new modules. The discriminated union approach (`WSMessage`) should accept `${moduleId}:${action}` patterns. Since TypeScript template literal unions can't be dynamically extended, use a generic library event type:

```typescript
type LibraryEntityMessage = {
  type: string  // "${moduleType}:added" | "${moduleType}:updated" | "${moduleType}:deleted"
  payload: { module?: string; entityType?: string; entityId?: number }
  timestamp: string
}
```

Add `LibraryEntityMessage` to the `WSMessage` union. Keep existing specific types (`QueueStateMessage`, `ProgressMessage`, etc.) unchanged — they are framework events, not module events.

#### `web/src/stores/ws-message-handlers.ts`

Replace `handleLibraryEvent` and the hard-coded `handlerMap` entries:

```typescript
import { getModule, getEnabledModules } from '@/modules'

function handleLibraryEvent(queryClient: QueryClient, type: string): void {
  // Extract module ID from event type: "movie:updated" → "movie"
  const moduleId = type.split(':')[0]
  const mod = getModule(moduleId)
  if (!mod) return

  // Invalidate the module's own query keys
  void queryClient.invalidateQueries({ queryKey: mod.queryKeys.all })

  // Apply module-declared additional invalidations
  for (const rule of mod.wsInvalidationRules) {
    if (type.match(rule.pattern)) {
      for (const keys of rule.alsoInvalidate ?? []) {
        void queryClient.invalidateQueries({ queryKey: keys })
      }
    }
  }

  // Always invalidate missing counts on any library event
  void queryClient.invalidateQueries({ queryKey: missingKeys.counts() })
}
```

Update `dispatchWSMessage` to try the module-based handler for any unrecognized event type:

```typescript
export function dispatchWSMessage(message: WSMessage, ctx: DispatchContext): void {
  const handler = handlerMap[message.type as WSMessageType]
  if (handler) {
    handler(message, ctx)
    return
  }
  // Fall through to generic library event handler for module events
  const action = message.type.split(':')[1]
  if (action === 'added' || action === 'updated' || action === 'deleted') {
    handleLibraryEvent(ctx.queryClient, message.type)
  }
}
```

Remove the hard-coded `'movie:added'`, `'movie:updated'`, `'movie:deleted'`, `'series:added'`, `'series:updated'`, `'series:deleted'` entries from `handlerMap`. The generic fallback handles them.

---

## Task Group 4: Generic Module API Client Factory ✅ COMPLETE

**Goal:** Create a factory that generates a typed API client from a module's base path, eliminating the duplication between `web/src/api/movies.ts` and `web/src/api/series.ts`. (Appendix C row "Frontend API clients")

**Depends on:** Task Group 1

### New file: `web/src/api/module-api.ts`

```typescript
import type { ModuleEntity } from '@/modules/types'
import { apiFetch, buildQueryString } from './client'

export function createModuleApi<T extends ModuleEntity>(basePath: string) {
  return {
    list: (options?: Record<string, unknown>) =>
      apiFetch<T[]>(`${basePath}${buildQueryString(options ?? {})}`),

    get: (id: number) => apiFetch<T>(`${basePath}/${id}`),

    update: (id: number, data: Record<string, unknown>) =>
      apiFetch<T>(`${basePath}/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

    delete: (id: number, deleteFiles?: boolean) =>
      apiFetch<undefined>(`${basePath}/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, { method: 'DELETE' }),

    bulkDelete: (ids: number[], deleteFiles?: boolean) =>
      Promise.all(ids.map((id) =>
        apiFetch<undefined>(`${basePath}/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, { method: 'DELETE' }),
      )),

    bulkUpdate: (ids: number[], data: Record<string, unknown>) =>
      Promise.all(ids.map((id) =>
        apiFetch<T>(`${basePath}/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
      )),

    bulkMonitor: (ids: number[], monitored: boolean) =>
      apiFetch<{ status: string }>(`${basePath}/monitor`, {
        method: 'PUT', body: JSON.stringify({ ids, monitored }),
      }),

    search: (id: number) => apiFetch<undefined>(`${basePath}/${id}/search`, { method: 'POST' }),

    refresh: (id: number) => apiFetch<T>(`${basePath}/${id}/refresh`, { method: 'POST' }),

    refreshAll: () => apiFetch<{ message: string }>(`${basePath}/refresh`, { method: 'POST' }),
  }
}
```

### Migration strategy

Do NOT delete `web/src/api/movies.ts` or `web/src/api/series.ts` yet. They contain module-specific methods (e.g., `seriesApi.getSeasons`, `seriesApi.getEpisodes`, `seriesApi.bulkMonitorEpisodes`). Instead:

1. The movie module config's `api` field uses `createModuleApi<Movie>('/movies')` for the shared CRUD operations.
2. Module-specific methods (seasons, episodes) remain on `seriesApi` and are accessed directly by TV-specific components.
3. In a later cleanup pass, `movies.ts` and `series.ts` can be trimmed to only export module-specific extensions, re-exporting the generic CRUD from the factory.

---

## Task Group 5: Generic Module Hook Factory ✅ COMPLETE

**Goal:** Create a factory that generates TanStack Query hooks from a module config, eliminating the duplication between `web/src/hooks/use-movies.ts` and `web/src/hooks/use-series.ts`. (Appendix C row "TanStack Query hooks")

**Depends on:** Task Group 1, Task Group 4

### New file: `web/src/hooks/use-module.ts`

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ModuleConfig, ModuleEntity } from '@/modules/types'
import { calendarKeys } from './use-calendar'
import { missingKeys } from './use-missing'

export function createModuleHooks<T extends ModuleEntity>(mod: ModuleConfig) {
  const keys = mod.queryKeys

  function useList(options?: Record<string, unknown>) {
    return useQuery({
      queryKey: [...keys.list(), options ?? {}],
      queryFn: () => mod.api.list(options),
    })
  }

  function useDetail(id: number) {
    return useQuery({
      queryKey: keys.detail(id),
      queryFn: () => mod.api.get(id),
      enabled: !!id,
    })
  }

  function useUpdate() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ id, data }: { id: number; data: Record<string, unknown> }) =>
        mod.api.update(id, data),
      onSuccess: (entity: unknown) => {
        const typed = entity as T
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: missingKeys.all })
        void queryClient.setQueryData(keys.detail(typed.id), typed)
      },
    })
  }

  function useDelete() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ id, deleteFiles }: { id: number; deleteFiles?: boolean }) =>
        mod.api.delete(id, deleteFiles),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
      },
    })
  }

  function useBulkDelete() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ ids, deleteFiles }: { ids: number[]; deleteFiles?: boolean }) =>
        mod.api.bulkDelete(ids, deleteFiles),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
      },
    })
  }

  function useBulkMonitor() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ ids, monitored }: { ids: number[]; monitored: boolean }) =>
        mod.api.bulkMonitor(ids, monitored),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: missingKeys.all })
      },
    })
  }

  function useBulkUpdate() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ ids, data }: { ids: number[]; data: Record<string, unknown> }) =>
        mod.api.bulkUpdate(ids, data),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: missingKeys.all })
      },
    })
  }

  function useRefresh() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: (id: number) => mod.api.refresh(id),
      onSuccess: (entity: unknown) => {
        const typed = entity as T
        void queryClient.setQueryData(keys.detail(typed.id), typed)
      },
    })
  }

  function useRefreshAll() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: () => mod.api.refreshAll(),
      onSettled: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
      },
    })
  }

  function useSearch() {
    return useMutation({
      mutationFn: (id: number) => mod.api.search(id),
    })
  }

  return {
    useList, useDetail, useUpdate, useDelete,
    useBulkDelete, useBulkMonitor, useBulkUpdate,
    useRefresh, useRefreshAll, useSearch,
  }
}
```

### Migration strategy

Existing `use-movies.ts` and `use-series.ts` remain for now — they are consumed by many files. The module configs wire up the generic hooks internally. Gradual migration: as components are refactored to use the generic list page (Task Group 6), they switch from specific hooks to the module-provided generic hooks. Module-specific hooks (e.g., `useEpisodes`, `useUpdateSeasonMonitored`) remain on `use-series.ts`.

---

## Task Group 6: Generic List Page ✅ COMPLETE

**Goal:** Eliminate the duplication between `movie-list-layout.tsx` / `use-movie-list.ts` and `series-list-layout.tsx` / `use-series-list.ts` by creating a single generic list page driven by module config. (Spec §13.1, Appendix C row "List page components")

**Depends on:** Task Groups 1, 4, 5

### Current duplication

| Movie | TV | Duplicated Logic |
|---|---|---|
| `routes/movies/index.tsx` | `routes/series/index.tsx` | Page wrapper |
| `routes/movies/movie-list-layout.tsx` | `routes/series/series-list-layout.tsx` | Filter/sort config → `MediaListLayout` |
| `routes/movies/use-movie-list.ts` | `routes/series/use-series-list.ts` | Filtering, sorting, grouping, bulk ops, UI state |
| `routes/movies/movie-list-utils.ts` | `routes/series/series-list-utils.ts` | Filter/sort comparator functions |

The shared `MediaListLayout` component (`web/src/components/media/media-list-layout.tsx`) already exists and is generic. The duplication is in the **hook** and **config wiring** above it.

### New file: `web/src/hooks/use-module-list.ts`

Generic list hook that replaces both `use-movie-list.ts` and `use-series-list.ts`. Parameterized by `ModuleConfig`:

- Reads filter/sort options from `ModuleConfig.filterOptions` / `ModuleConfig.sortOptions`
- Uses `ModuleConfig.api.list()` for data fetching (via the generic hooks)
- Reads persisted view preference from `useUIStore` keyed by `${moduleId}View` (currently `moviesView` / `seriesView`)
- Reads persisted table columns from `useUIStore` keyed by `${moduleId}TableColumns`
- Provides the same `MovieListState` / `SeriesListState` shape but generalized

The filtering and sorting logic currently in `movie-list-utils.ts` / `series-list-utils.ts` uses module-specific field access (e.g., `movie.physicalReleaseDate` vs `series.nextAirDate`). For the generic list hook:

- **Status filtering** (`monitored`, `missing`, `available`, etc.) is identical — uses `entity.status` and `entity.monitored`. Move to a shared utility.
- **Module-specific filters** (TV's `continuing`/`ended` filter uses `series.productionStatus`) — the module config provides a `customFilter` function on its `filterOptions` entries.
- **Sorting** — common fields (`title`, `sortTitle`, `monitored`, `qualityProfileId`, `addedAt`, `sizeOnDisk`, `rootFolderId`) use the standard `ModuleEntity` shape. Module-specific sort fields (movie `releaseDate` vs TV `nextAirDate`) are handled by a `getSortValue(item: ModuleEntity, field: string): unknown` function on the module config.

### New file: `web/src/components/media/module-list-page.tsx`

A generic page component:

```typescript
export function ModuleListPage({ moduleId }: { moduleId: string }) {
  const mod = getModuleOrThrow(moduleId)
  const state = useModuleList(mod)
  return (
    <MediaListLayout
      theme={mod.themeColor}
      title={mod.name}
      addLabel={`Add ${mod.singularName}`}
      mediaLabel={mod.singularName}
      pluralMediaLabel={mod.pluralName}
      filterOptions={mod.filterOptions}
      sortOptions={mod.sortOptions}
      // ... pass state props ...
      emptyIcon={<mod.icon className={`text-${mod.themeColor}-500 size-8`} />}
      emptyTitle={`No ${mod.pluralName.toLowerCase()} found`}
      renderCard={(item, opts) => <mod.cardComponent item={item} {...opts} />}
      // ... rest of callbacks from state ...
    />
  )
}
```

### Migration

1. Create the generic hook and page component.
2. Change `web/src/routes/movies/index.tsx` to render `<ModuleListPage moduleId="movie" />`.
3. Change `web/src/routes/series/index.tsx` to render `<ModuleListPage moduleId="tv" />`.
4. Verify both pages work identically to before.
5. Delete `movie-list-layout.tsx`, `series-list-layout.tsx`, and the bulk of `use-movie-list.ts` / `use-series-list.ts`. Keep module-specific utils that are still referenced.

### `MediaListLayout` props change

The `theme` prop type in `web/src/components/media/media-list-layout.tsx:21` is currently `'movie' | 'tv'`. Change to `string`:

```typescript
theme: string  // was: 'movie' | 'tv'
```

This also affects `MediaListFilters`, `MediaListToolbar`, `MediaListContent`, `MediaPageActions` — all take a `theme` prop. Update all to accept `string`.

### UIStore changes

`web/src/stores/ui.ts` (or wherever `useUIStore` is defined) currently has `moviesView`, `seriesView`, `movieTableColumns`, `seriesTableColumns` as separate fields. Generalize to a keyed map:

```typescript
// Add to UIStore state:
moduleViewPrefs: Record<string, 'grid' | 'table'>     // keyed by moduleId
moduleTableColumns: Record<string, string[]>            // keyed by moduleId

// Getters:
getModuleView: (moduleId: string) => 'grid' | 'table'
setModuleView: (moduleId: string, view: 'grid' | 'table') => void
getModuleTableColumns: (moduleId: string) => string[]
setModuleTableColumns: (moduleId: string, cols: string[]) => void
```

Migrate existing persisted values: on first access, if `moduleViewPrefs['movie']` is undefined but `moviesView` exists, use `moviesView` as the initial value. This preserves user preferences across the migration.

---

## Task Group 7: Unified Wanted/Missing Page ✅ COMPLETE

**Goal:** Make the Missing/Wanted page show data from all enabled modules with module-based grouping/filtering. (Spec §13.3)

**Depends on:** Task Group 1

### Current state

- `web/src/routes/missing/index.tsx` shows `MissingTabContent` and `UpgradableTabContent`
- `web/src/routes/missing/missing-tab-content.tsx` renders `missingMovies` and `missingSeries` side by side
- `web/src/routes/missing/upgradable-tab-content.tsx` renders `upgradableMovies` and `upgradableSeries`
- `web/src/hooks/use-missing.ts` has `useMissingMovies()`, `useMissingSeries()`, `useUpgradableMovies()`, `useUpgradableSeries()`, `useMissingCounts()`
- `web/src/routes/missing/media-tabs.tsx` has tabs for Missing/Upgradable with counts split as `movieCount` / `episodeCount`

### Changes

#### Backend prerequisite

The missing/wanted API should accept a `moduleType` query parameter. If not yet available, the current approach (separate endpoints for movies vs series) works — we just iterate over enabled modules.

#### `web/src/routes/missing/media-tabs.tsx`

Replace the hard-coded movie/episode count display with per-module counts. The tab badges should show counts per enabled module, using each module's theme color:

```typescript
// Instead of fixed "movieCount | episodeCount":
{getEnabledModules().map(mod => (
  counts[mod.id] > 0 && (
    <span key={mod.id} className={`text-${mod.themeColor}-500`}>
      {counts[mod.id]}
    </span>
  )
))}
```

#### `web/src/routes/missing/missing-tab-content.tsx`

Currently renders two hard-coded sections (movies, series). Change to iterate over enabled modules:

```typescript
{getEnabledModules().map(mod => (
  <MissingModuleSection key={mod.id} module={mod} items={missingByModule[mod.id]} />
))}
```

Each `MissingModuleSection` renders using the module's card component and theme color. The existing `MissingMovieRow` / `MissingSeriesRow` components become module-specific detail renderers — keep them for now, but wrap in a module-dispatch pattern.

#### `web/src/routes/missing/upgradable-tab-content.tsx`

Same pattern — iterate over enabled modules instead of hard-coded movie/series sections.

#### `web/src/hooks/use-missing.ts`

The `missingKeys` and individual hooks remain (they're backed by specific API endpoints). Add a `useModuleMissingCounts()` hook that aggregates counts across all enabled modules. For now this wraps the existing `useMissingCounts()`. When the backend provides per-module counts, this hook consumes that API directly.

---

## Task Group 8: Unified Calendar ✅ COMPLETE

**Goal:** Calendar shows items from all enabled modules with module-colored items. (Spec §13.3)

**Depends on:** Task Group 1

### Current state

- `web/src/types/calendar.ts:4` hard-codes `mediaType: 'movie' | 'episode'`
- `web/src/components/calendar/` renders events with movie/episode colors
- The calendar API returns events from both movies and TV already

### Changes

#### `web/src/types/calendar.ts`

Change `mediaType` to accept any string (module-provided):

```typescript
export type CalendarEvent = {
  id: number
  title: string
  mediaType: string          // was: 'movie' | 'episode'
  moduleType?: string        // new: "movie", "tv", "music", etc.
  // ... rest unchanged
}
```

#### `web/src/components/calendar/*.tsx`

Calendar components currently use `mediaType` to pick colors. Change to use `moduleType` (falling back to `mediaType` for backward compatibility) and look up the module's theme color:

```typescript
function getEventThemeColor(event: CalendarEvent): string {
  // Prefer moduleType if available (new backend), fall back to mediaType mapping
  if (event.moduleType) {
    return event.moduleType  // "movie" → movie-*, "tv" → tv-*
  }
  // Legacy fallback
  return event.mediaType === 'movie' ? 'movie' : 'tv'
}
```

Then use `text-${themeColor}-500`, `bg-${themeColor}-500/10`, etc.

**Tailwind safelist note:** Since Tailwind purges unused classes, dynamic class names like `text-${var}-500` won't work unless safelisted. Add a safelist in `tailwind.config.ts` for module theme classes, or use the existing approach of mapping to full class strings via a lookup object. The current codebase uses `text-movie-500` and `text-tv-500` directly, so a lookup map is the right pattern:

```typescript
const THEME_TEXT_CLASSES: Record<string, string> = {
  movie: 'text-movie-500',
  tv: 'text-tv-500',
}
```

New modules add their entries here. This is documented in the shared component catalog (Task Group 10).

---

## Task Group 9: Notification Settings — Module-Grouped Events ✅ COMPLETE

**Goal:** Notification event toggles are driven by module-declared event catalogs, grouped by module in the UI. (Spec §9.1 frontend portion)

**Depends on:** Task Group 1, backend Phase 7 complete (notification events as JSON)

**Status:** Already fully implemented. The "current state" described below was the state at plan creation time; all changes have since been completed (likely during backend Phase 7).

### What was implemented

- **Types** (`web/src/types/notification.ts`): `Notification` uses `eventToggles: Record<string, boolean>` (no individual boolean fields). `NotificationEventDef` and `NotificationEventGroup` types are defined.
- **Backend API** (`GET /api/v1/notifications/events`): Returns `NotificationEventGroup[]` built dynamically by `module.Registry.CollectNotificationEvents()`, which collects framework events ("General" group) plus each registered module's `DeclareEvents()` output (e.g., "Movies", "Series").
- **Event catalog hook** (`web/src/hooks/use-notifications.ts`): `useNotificationEventCatalog()` fetches groups from the API with `staleTime: Infinity`.
- **Event triggers component** (`web/src/components/notifications/event-triggers.tsx`): `EventTriggers` renders groups dynamically — iterates over `groups` array, showing each group label and its event toggles.
- **Active events text** (`web/src/routes/settings/general/notifications.tsx`): `getActiveEventsText()` uses the fetched catalog to resolve event labels dynamically — no hard-coded label map.
- **Dialog** (`web/src/components/notifications/notification-dialog.tsx`): Delegates to `NotificationFormBody` which uses `EventTriggersSection`, accepting optional `eventGroups` override or falling back to the API catalog.

No hard-coded `EVENT_FLAGS`, no individual `onGrab`/`onMovieAdded` boolean fields, and no hard-coded movie/TV event references exist in the notification frontend code.

---

## Task Group 10: File Naming Settings — Module Tabs ✅ COMPLETE

**Goal:** File naming settings use module-provided token contexts and generate tabs per module. (Spec §13.5)

**Depends on:** Task Group 1, backend Phase 6 complete (NamingProvider)

### Current state

- `web/src/components/settings/sections/file-naming-section.tsx` has hard-coded tabs: "Validation", "Matching", "TV Naming", "Movie Naming", "Tokens"
- `web/src/components/settings/sections/naming-movie-tab.tsx` and `naming-tv-tab.tsx` are separate components
- Token contexts are defined in `web/src/components/settings/sections/file-naming-constants.ts`
- The `PatternEditor` component at `web/src/components/settings/sections/naming-pattern-editor.tsx` is already generic — it takes a `tokenContext` prop

### Changes

#### `web/src/components/settings/sections/file-naming-section.tsx`

Replace hard-coded tabs with module-driven tabs:

```typescript
const FRAMEWORK_TABS = ['validation', 'matching', 'tokens']

// Dynamic module naming tabs
{getEnabledModules().map(mod => (
  <TabsTrigger key={mod.id} value={`${mod.id}-naming`}>
    {mod.singularName} Naming
  </TabsTrigger>
))}

// Dynamic tab content
{getEnabledModules().map(mod => (
  <TabsContent key={mod.id} value={`${mod.id}-naming`}>
    <ModuleNamingTab moduleId={mod.id} form={form} updateField={updateField} />
  </TabsContent>
))}
```

#### New: `web/src/components/settings/sections/module-naming-tab.tsx`

Generic naming tab that reads token contexts from the module's backend-provided naming config. Uses the existing `PatternEditor` component. This replaces `naming-movie-tab.tsx` and `naming-tv-tab.tsx`.

The token contexts come from the backend `NamingProvider.TokenContexts()` — fetch via `GET /api/v1/modules/{id}/naming-config`. Each context provides:
- Context name (e.g., "movie-folder", "episode-file")
- Available tokens with descriptions
- Current template value
- Preview result

#### `web/src/components/settings/sections/naming-pattern-editor.tsx`

The `PatternEditor` component is already generic. Its `mediaType` prop (currently `'episode' | 'movie' | 'folder'`) should accept `string` to support new modules. The `tokenContext` prop is already externally provided. No changes needed beyond widening the type.

---

## Task Group 11: Search Page — Module-Driven ✅ COMPLETE

**Goal:** Global search renders results for all enabled modules. (Spec §13.3)

**Depends on:** Task Group 1

### Current state

`web/src/routes/search/index.tsx` hard-codes:
- `useLibrarySearch` fetches from `useMovies` and `useSeries` separately
- `useExternalSearch` fetches from `useMovieSearch` and `useSeriesSearch` separately
- Renders `MovieCard` and `SeriesCard` explicitly
- Builds `libraryMovieTmdbIds` / `librarySeriesTmdbIds` separately

### Changes

#### `web/src/routes/search/index.tsx`

Replace the hard-coded movie/series fetching with a loop over enabled modules:

```typescript
function useLibrarySearch(query: string) {
  const modules = getEnabledModules()
  // Each module provides its own list hook with search filter
  const results = modules.map(mod => ({
    moduleId: mod.id,
    module: mod,
    query: mod.api.list(query ? { search: query } : undefined),
    // ... track loading state
  }))
  // ...
}
```

For external search (metadata search), the search API is module-specific (movie search uses TMDB, TV uses TVDB). The external search hooks (`useMovieSearch`, `useSeriesSearch`) remain module-specific. The search page iterates over modules and renders their results using the module's card component:

```typescript
{getEnabledModules().map(mod => (
  <ExpandableMediaGrid
    key={mod.id}
    items={libraryResults[mod.id]}
    getKey={(item) => item.id}
    label={mod.name}
    icon={mod.themeColor}
    renderItem={(item) => <mod.cardComponent item={item} />}
  />
))}
```

**Note:** The external metadata search section currently renders `ExternalMediaCard` with an "Add..." action that navigates to `/movies/add` or `/series/add`. These add routes are module-specific. The module config's `basePath + '/add'` provides the correct URL.

---

## Task Group 12: Route Registration ✅ COMPLETE

**Goal:** Module routes are registered from module configs instead of hard-coded in `routes-config.tsx`. (Spec §13.4)

**Depends on:** Task Groups 1, 6

### Current state

`web/src/routes-config.tsx` hard-codes every route:
```typescript
export const moviesRoute = lazyRoute('/movies', ...)
export const movieDetailRoute = createRoute(...)
export const addMovieRoute = lazyRoute('/movies/add', ...)
export const seriesRoute = lazyRoute('/series', ...)
export const seriesDetailRoute = createRoute(...)
export const addSeriesRoute = lazyRoute('/series/add', ...)
```

### Approach

TanStack Router requires routes to be statically analyzable for type safety. We **cannot** dynamically generate routes at runtime with full type safety. Instead:

1. **Keep the route definitions in `routes-config.tsx`** — they are compile-time constants.
2. **Replace the hard-coded page component imports** with the generic `ModuleListPage`:

```typescript
export const moviesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/movies',
  component: lazyRouteComponent(
    () => import('@/components/media/module-list-page'),
    'MovieListPageWrapper'
  ),
})
```

Where `MovieListPageWrapper` is:
```typescript
export function MovieListPageWrapper() {
  return <ModuleListPage moduleId="movie" />
}
```

3. **Detail and Add routes remain module-specific** — `MovieDetailPage` and `SeriesDetailPage` have fundamentally different content (movie info vs seasons/episodes). The spec says detail pages have a "shared shell with module-provided detail content components." The shared shell (hero section, action bar, edit dialog) already exists. The module-specific parts (`MovieDetailContent`, `SeasonList`) stay.

   **Add pages** also stay module-specific for now. The spec §13.1 says "Add pages use the shared `AddMediaConfigure` flow with module-specific configuration fields." The `AddMediaConfigure` component already exists as a shared shell — each module's add page wraps it with module-specific configuration (metadata search provider, default options). The `ModuleConfig.addConfigFields` component slot enables future full generalization if needed, but since add pages are thin wrappers around the shared component, the ROI of a generic `ModuleAddPage` is low.

4. **Route loaders remain module-specific** — `movieDetailRoute` prefetches movie data and extended metadata, `seriesDetailRoute` prefetches series data. These loaders use module-specific query options and are not easily generalized without losing type safety.

### Migration

- List routes: switch to `ModuleListPage` wrapper
- Detail routes: keep as-is (module-specific detail components)
- Add routes: keep as-is (module-specific add configuration)
- When a new module is added in the future, it adds its routes to `routes-config.tsx` and provides its detail/add components. The list page is free.

---

## Task Group 13: Quality Profile Settings — Module-Scoped ✅ COMPLETE

**Goal:** Quality profile settings UI shows profiles grouped by module. (Spec §4.1 frontend)

**Depends on:** Task Group 1, backend Phase 2 complete (module-scoped quality profiles)

### Current state

- `web/src/components/settings/sections/quality-profiles-section.tsx` shows all profiles in a flat list
- `web/src/hooks/use-quality-profiles.ts` fetches from `/api/v1/quality-profiles`
- `web/src/types/quality-profile.ts` — profile type has no `moduleType` field

### Backend prerequisite

Quality profiles have a `module_type` column. The API returns `moduleType` on each profile.

### Changes

#### `web/src/types/quality-profile.ts`

Add `moduleType` field:
```typescript
export type QualityProfile = {
  // ... existing fields
  moduleType: string  // "movie", "tv", etc.
}
```

#### `web/src/components/settings/sections/quality-profiles-section.tsx`

Group profiles by module:

```typescript
const profilesByModule = useMemo(() => {
  const grouped = new Map<string, QualityProfile[]>()
  for (const profile of profiles) {
    const list = grouped.get(profile.moduleType) ?? []
    list.push(profile)
    grouped.set(profile.moduleType, list)
  }
  return grouped
}, [profiles])

// Render grouped
{getEnabledModules().map(mod => {
  const moduleProfiles = profilesByModule.get(mod.id) ?? []
  return (
    <div key={mod.id}>
      <h3>{mod.name} Profiles</h3>
      {moduleProfiles.map(profile => <ProfileCard ... />)}
    </div>
  )
})}
```

#### Dynamic quality items loading

The `PREDEFINED_QUALITIES` constant in the frontend currently hard-codes the list of quality items. Phase 2 made the `/qualityprofiles/qualities` endpoint module-aware but deferred dynamic frontend loading. Replace the hard-coded `PREDEFINED_QUALITIES` with a fetch from the backend endpoint, keyed by module type. Quality items may differ per module in the future (e.g., audio quality items for a music module).

#### Profile creation

When creating a new profile, the user must select which module it's for. Add a module selector to the create profile dialog.

#### `web/src/hooks/use-root-folders.ts`

The `useRootFoldersByType` hook already exists. Ensure it filters by `module_type`. Used by add pages to show only root folders for the correct module.

---

## Task Group 14: Shared UI Component Catalog Documentation ✅ COMPLETE

**Goal:** Document the reusable frontend components available to module authors. (Spec §19.4)

**Depends on:** All previous task groups

### Deliverable

Create `web/src/components/COMPONENTS.md` documenting:

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `MediaListLayout` | `components/media/media-list-layout.tsx` | Generic list/grid page with filtering, sorting, bulk actions | `theme`, `filterOptions`, `sortOptions`, `renderCard` |
| `MediaEditDialog` | `components/media/media-edit-dialog.tsx` | Generic edit dialog for any entity | `item`, `updateMutation`, `mediaLabel` |
| `AddMediaConfigure` | `components/media/add-media-configure.tsx` | Shared add-media form | `preview`, `rootFolders`, `qualityProfiles` |
| `MediaPreview` | `components/media/add-media-configure.tsx` | Poster + metadata preview | `title`, `year`, `overview`, `posterUrl`, `type` |
| `MediaDeleteDialog` | `components/media/media-delete-dialog.tsx` | Bulk delete confirmation | `selectedCount`, `deleteFiles` |
| `MediaStatusBadge` | `components/media/media-status-badge.tsx` | Status pill rendering | `status` |
| `QualityBadge` | `components/media/quality-badge.tsx` | Quality label with color | `quality` |
| `PosterImage` | `components/media/poster-image.tsx` | Poster artwork with cache-busting | `src`, `alt` |
| `BackdropImage` | `components/media/backdrop-image.tsx` | Backdrop artwork with cache-busting | `src`, `alt` |
| `PatternEditor` | `components/settings/sections/naming-pattern-editor.tsx` | Token-aware template editor | `value`, `onChange`, `tokenContext` |
| `ModuleListPage` | `components/media/module-list-page.tsx` | Generic module list page | `moduleId` |

Also document:
- How to add a new module's theme colors to `web/src/index.css`
- How to add theme class mappings to the Tailwind safelist/lookup objects
- The `ModuleConfig` interface and all its fields
- How to register a new module in `web/src/modules/setup.ts`

---

## Task Group 15: Activity/Downloads & History — Module Awareness ✅ COMPLETE

**Goal:** Activity and History pages handle module-typed items generically. (Spec §13.3)

**Depends on:** Task Group 1

### Current state

- `web/src/routes/downloads/` — renders queue items that have `mediaType` field
- `web/src/routes/history/` — renders history entries with `mediaType` field
- `web/src/types/queue.ts` — likely has `mediaType: 'movie' | 'episode'` or similar
- `web/src/types/history.ts` — likely has `mediaType` field

### Changes

These pages already display items generically (they show title, status, poster). The main changes:

1. **Type widening**: Change `mediaType: 'movie' | 'episode'` to `mediaType: string` in queue and history types. Add optional `moduleType: string` field.
2. **Color/icon lookup**: Where these pages render colored badges or icons based on media type, use the module registry to look up theme color and icon instead of hard-coded conditionals.
3. **Download row poster**: `web/src/routes/downloads/download-row-poster.tsx` likely picks poster URL format based on media type. Generalize the lookup.

#### Frontend JSON shape changes (deferred from Phase 1)

Phase 1 preserved legacy JSON shapes (`movieId`, `seriesId`, `episodeId` on queue items and download mappings) to avoid frontend breakage. Now that the frontend is module-aware, update the API response types to use the generic discriminator fields:

- `web/src/types/queue.ts`: Add `moduleType`, `entityType`, `entityId` fields. Existing `movieId`/`seriesId`/`episodeId` fields can be kept as optional for backward compatibility during the transition, or removed if the backend has dropped them.
- `web/src/types/history.ts`: Same changes.
- Update any queue/history rendering logic that reads `movieId`/`seriesId`/`episodeId` to use the generic fields instead.

These are small, surgical changes — not full rewrites.

---

## Execution Order

```
Task Group 1: Module Registry Types & Runtime Config
    ├── Task Group 2: Sidebar Navigation (depends on 1)
    ├── Task Group 3: WebSocket Event Handling (depends on 1)
    ├── Task Group 4: API Client Factory (depends on 1)
    └── Task Group 8: Calendar Module Awareness (depends on 1)

Task Group 5: Hook Factory (depends on 1, 4)

Task Group 6: Generic List Page (depends on 1, 4, 5)
    └── Task Group 12: Route Registration (depends on 1, 6)

Task Group 7: Unified Wanted/Missing (depends on 1)
Task Group 9: Notification Events (depends on 1, backend Phase 7)
Task Group 10: File Naming Settings (depends on 1, backend Phase 6)
Task Group 11: Search Page (depends on 1)
Task Group 13: Quality Profile Settings (depends on 1, backend Phase 2)
Task Group 15: Activity/History (depends on 1)

Task Group 14: Documentation (depends on all above)
```

**Parallelizable groups** (after TG1 completes): TG2, TG3, TG4, TG7, TG8, TG11, TG15 can all run in parallel since they touch non-overlapping files.

**Sequential dependencies**: TG4 → TG5 → TG6 → TG12 (the data layer chain).

**Backend-gated groups**: TG9 (needs Phase 7), TG10 (needs Phase 6), TG13 (needs Phase 2). If backend phases are incomplete, defer these and note the dependency.

---

## Definition of Done

- [x] All enabled modules appear in sidebar dynamically — no hard-coded movie/TV entries
- [x] Movie and TV list pages use the single generic `ModuleListPage` component
- [x] WebSocket library events dispatch via module registry — no hard-coded movie/series handlers
- [~] `movie-list-layout.tsx` and `series-list-layout.tsx` are deleted (replaced by generic page) — **Deviation:** layouts refactored to use module config for theme/labels/icons but retained as thin shells wrapping `MediaListLayout`, since each module has unique hook state shapes; `module-list-page.tsx` dispatch wrapper exists
- [x] Calendar events use module theme colors via registry lookup
- [x] Missing/Wanted page iterates over enabled modules
- [x] Search page iterates over enabled modules for library results
- [x] File naming settings generate tabs per enabled module (if backend ready)
- [x] Notification event toggles are module-grouped (if backend ready) — already complete from Phase 7
- [x] Quality profile settings are module-grouped (if backend ready)
- [x] `theme` prop on `MediaListLayout` and sub-components accepts `string` (not union)
- [x] `cd web && bun run lint` passes with 0 new errors
- [x] `make build` succeeds
- [x] No hard-coded `'movie' | 'tv'` type unions remain in shared/framework components (module-specific components may still use concrete types)
- [ ] Quality items loaded dynamically from backend instead of hard-coded `PREDEFINED_QUALITIES` (if backend ready) — deferred, backend API not yet available
- [x] Queue/history types use generic `moduleType`/`entityType`/`entityId` fields instead of `movieId`/`seriesId`/`episodeId`
- [x] Backend items deferred from earlier phases are tracked in the Phase 11 plan (not silently dropped) — all 12 items verified in `module-system-plan-phase11.md`
- [x] Shared UI component catalog documented in `web/src/components/COMPONENTS.md`
