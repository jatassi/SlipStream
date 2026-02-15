# Frontend Consolidation Plan

**Starting point:** 0 lint errors, codebase freshly decomposed from lint remediation. Many components were split during the refactor, making duplicate patterns easier to spot.

---

## Executive Summary

The lint remediation refactor successfully decomposed large files but introduced significant duplication, particularly across movie/series parallel implementations and the DryRunModal/MigrationPreviewModal slot components. This plan identifies **~2,070 lines** of duplicate or near-duplicate code that can be consolidated through shared abstractions.

The consolidation is organized into 7 waves ordered by impact, risk, and ease of implementation.

---

## Agent Execution Guide

This plan is designed for execution by an AI coding agent. This section provides meta-instructions for context management, verification, and error avoidance.

### Pre-flight Checklist

Before starting any wave:

1. Run `./scripts/consolidation/verify.sh` to establish a green baseline. If it fails, stop and fix before proceeding.
2. Read this entire Agent Execution Guide section.
3. Read the target wave's section completely before making any changes.
4. For waves that move/delete files: run `./scripts/consolidation/find-importers.sh <module>` for every file being moved BEFORE making any changes. Capture the full list of import sites.

### Helper Scripts

All scripts are in `scripts/consolidation/`. Run from the project root.

| Script | Purpose | When to Use |
|---|---|---|
| `verify.sh` | Run tsc + build + lint count | After every wave. Also run mid-wave if you've made 5+ file edits. |
| `find-importers.sh <module>` | Find all files that import a given module | Before moving/renaming any file. Captures import sites to update. |
| `snapshot-exports.sh <files>` | Snapshot exported symbols | Before and after modifying a file. Diff outputs to catch API surface changes. |
| `check-old-imports.sh <patterns>` | Verify no stale imports remain | After completing a wave. Pass all old import path patterns. |

### Context Management Strategy

Each wave should be an independent session. Never attempt multiple waves in one session.

| Wave | Strategy | Rationale |
|---|---|---|
| 1 | Single agent, sequential | File moves are interdependent — shared/ must exist before imports update. ~12 files to move, ~20 import sites to update. Fits comfortably in one session. |
| 2 | Subagent per component pair (4 parallel) | Each generic component (MediaTable, MediaGrid, GroupedMediaGrid, MediaEditDialog) is independent. Each subagent reads both files in its pair, creates the generic, updates importers, deletes originals. |
| 3 | Subagent per component (up to 5 parallel) | Toolbar, PageActions, DeleteDialog, ListContent, ListFilters are independent. Do Layout and index.tsx last since they import the others. |
| 4 | Single agent | Small scope — one component pair. |
| 5 | Single agent | Small scope — one component pair. |
| 6 | Two parallel subagents | One for `createQueryKeys`, one for `useTestableForm`. Independent concerns. |
| 7 | Subagent per item (up to 4 parallel) | PendingButton, useDialogState, withToast, FormField are independent. |

**Subagent model selection:** Use Sonnet for subagents by default. Only escalate to Opus for tasks that require complex architectural judgment (e.g., designing the `useTestableForm` abstraction from 4 differing implementations, or resolving unexpected type errors during generic extraction). File moves, import updates, and mechanical genericization are Sonnet-appropriate.

**Subagent completion requirement:** Every subagent MUST run `cd web && bunx tsc --noEmit` AND `cd web && bun run lint --quiet` on its changes before reporting completion. If either fails, the subagent must fix the issues — do not report back with broken code. A subagent that reports success with failing typecheck or new lint errors has not completed its task.

**Subagent instructions template:** When delegating to a subagent, include:
1. The full text of the relevant wave section from this plan
2. The exact file paths to read (both source files in the pair)
3. The verification commands: `cd web && bunx tsc --noEmit` AND `cd web && bun run lint --quiet`
4. Explicit instruction: "After creating the generic component, run `find-importers.sh` on the OLD file names to find all import sites. Update every import site. Then delete the old files. You MUST run tsc and lint before reporting completion — do not report back with errors."

### Verification Protocol

**After every file creation, move, or edit:**
```bash
cd web && bunx tsc --noEmit
```
If this fails, fix immediately before proceeding. Do not accumulate errors.

**After completing a wave:**
```bash
./scripts/consolidation/verify.sh
./scripts/consolidation/check-old-imports.sh <old-path-patterns...>
```

**After waves that create generic components (2, 3, 4, 5):**
Before deleting the old specific components, verify:
1. The new generic component compiles (`tsc --noEmit`)
2. All import sites have been updated (use `find-importers.sh` on old names — should return 0 results)
3. The build succeeds (`bun run build`)

### Tool Usage Recommendations

| Task | Tool | Why |
|---|---|---|
| Read a file before editing | `Read` tool | Always read before edit. Never guess file contents. |
| Find all files importing a module | `find-importers.sh` via Bash | More reliable than manual Grep — catches `import type` and re-exports. |
| Find file paths | `Glob` tool | Faster than Bash find. Use patterns like `**/movie-table.tsx`. |
| Search for patterns across codebase | `Grep` tool | Use for counting instances, finding usage patterns. |
| Multi-file exploration | `Explore` subagent | When you need to understand a pattern across 5+ files. |
| Verify after changes | `Bash` with `cd web && bunx tsc --noEmit && bun run build` | The primary regression gate. Run frequently. |
| Create new files | `Write` tool | For new generic components. |
| Update imports | `Edit` tool | Surgical edits to import lines. Prefer Edit over Write for existing files. |
| Move files (create + delete old) | `Write` new file, then `Bash rm` old file | There is no move tool. Write the new file first, update all imports, verify, then delete old. |

### Common Pitfalls

1. **Relative import paths change when files move.** When moving `DryRunModal/summary-card.tsx` to `shared/summary-card.tsx`, imports FROM DryRunModal files change from `./summary-card` to `../shared/summary-card`. Imports from MigrationPreviewModal files change from `../DryRunModal/summary-card` (if cross-importing) or `./summary-card` to `../shared/summary-card`. Always use `find-importers.sh` to find ALL import sites.

2. **Index/barrel files.** Check for `index.ts` files that re-export from moved modules. These are easy to miss and will break the build silently until a consumer triggers the import.

3. **`import type` statements.** These are a separate syntax from `import`. Both `import { X }` and `import type { X }` need updating when paths change. The `find-importers.sh` script catches both.

4. **Don't read too many files at once.** When creating a generic component from a movie/series pair, read ONLY the two files in the pair + their direct importers. Don't preload the entire route directory.

5. **Theme prop values.** The codebase uses `'movie'` and `'tv'` (NOT `'series'`) for theme props. Check existing patterns — e.g., `theme="movie"` and `theme="tv"`, color classes like `movie-500` and `tv-500`.

6. **Generic type parameter naming.** Use `T` for the media item type. Follow existing patterns: `ColumnDef<T>`, `MediaGroup<T>`.

7. **Don't merge hooks.** Waves 1 and 3 explicitly mark certain hooks as "DO NOT EXTRACT." Respect this — the hooks encode domain-specific logic that should remain separate.

8. **Series pluralization.** "Series" is both singular and plural. Don't add `s` pluralization logic that would produce "seriess." When the generic component needs a plural label, accept it as a prop rather than computing it.

### Test Strategy

**Primary regression gate (structural — run after every wave):**
```bash
cd web && bunx tsc --noEmit && bun run build
```
TypeScript compilation catches broken imports, missing exports, and type mismatches. The build catches module resolution failures and tree-shaking issues. Together, these two commands are the most reliable deterministic test for a structural refactor.

**Stale import detection (run after every wave):**
```bash
./scripts/consolidation/check-old-imports.sh <patterns...>
```
Pass the old file path patterns that should no longer appear in any import statement.

**Export surface verification (run before/after each file modification):**
```bash
# Before:
./scripts/consolidation/snapshot-exports.sh web/src/path/to/old-file.tsx > /tmp/before.txt
# After creating the new generic:
./scripts/consolidation/snapshot-exports.sh web/src/path/to/new-file.tsx > /tmp/after.txt
# Compare:
diff /tmp/before.txt /tmp/after.txt
```
The new generic should export at minimum the same symbols as the old specific components (possibly with generic type parameters added).

**Behavioral smoke test (for new generic components in Waves 2-5):**

After creating each new generic component, write a minimal render test to verify it doesn't crash with both movie and series configurations. Create test files in `web/src/__tests__/consolidation/`. Use vitest with @testing-library/react if available, otherwise use a bare `import` + type assertion test:

```tsx
// web/src/__tests__/consolidation/media-table.test.ts
// Minimal: verify the component is importable and its types work
import { MediaTable } from '@/components/media/media-table'
import type { Movie } from '@/types/movie'
import type { Series } from '@/types/series'
import type { ColumnDef } from '@tanstack/react-table'

// Type-level test: ensure generic works with both media types
const _movieCols: ColumnDef<Movie>[] = []
const _seriesCols: ColumnDef<Series>[] = []

// If these type-check, the generic is correctly parameterized
type _MovieTable = typeof MediaTable<Movie>
type _SeriesTable = typeof MediaTable<Series>
```

This is a compile-time test — if `tsc --noEmit` passes with this file present, the generic's type signature is correct for both media types. It costs zero runtime and catches the most common regression (generic type parameter too narrow/wide).

---

## Wave 1 — Slot Component Deduplication (~850 lines saved) ✅ COMPLETED

**Priority: HIGHEST** — These are 100% identical file pairs.

**Status:** Completed. All 11 files moved to `web/src/components/slots/shared/`. 22 duplicate files deleted. 16 importer files updated. Verified: tsc pass, build pass, 0 lint errors, no stale imports.

The DryRunModal and MigrationPreviewModal directories contain 7 component/utility pairs that are completely identical, copy-pasted between directories.

### Identical Component Pairs

| DryRunModal File | MigrationPreviewModal File | Lines |
|---|---|---|
| `summary-card.tsx` | `summary-card.tsx` | 33 |
| `file-item.tsx` | `file-item.tsx` | 230 |
| `movies-list.tsx` | `movies-list.tsx` | 124 |
| `series-list.tsx` | `series-list.tsx` | 221 |
| `aggregated-file-tooltip.tsx` | `aggregated-file-tooltip.tsx` | * |
| `assign-modal.tsx` | `assign-modal.tsx` | * |
| `utils.ts` | `utils.ts` | * |

### Execution Steps

**Step 1 — Discover import sites (do this FIRST, before any moves):**
```bash
for f in summary-card file-item movies-list series-list aggregated-file-tooltip assign-modal utils; do
  echo "=== $f ==="
  ./scripts/consolidation/find-importers.sh "$f"
  echo ""
done
```
Save this output — you will need it when updating imports.

**Step 2 — Create shared directory and move identical files:**
```
web/src/components/slots/shared/
```
For each of the 7 identical files:
1. Read the DryRunModal version (to have content for writing)
2. Write it to `components/slots/shared/<filename>`
3. Update all importers (from Step 1) to point to the new path
4. Delete both the DryRunModal and MigrationPreviewModal copies
5. Run `cd web && bunx tsc --noEmit` — must pass before proceeding to next file

**Step 3 — Merge `debug.ts` (100% identical, single re-export line):**
Move to `shared/debug.ts`. Update importers. Delete both copies.

**Step 4 — Merge `types.ts` (1-line difference):**
Read both versions. DryRunModal has an extra optional `onMigrationFailed?: (error: string) => void` on `DryRunModalProps`. Write the merged version to `shared/types.ts` WITH the optional prop included — MigrationPreviewModal callers won't use it, and optional props don't break consumers. Update importers. Delete both copies.

**Step 5 — Merge `filter-utils.ts` (same API, different internals):**
Both export 4 functions: `filterMovies`, `filterTvShows`, `getVisibleMovieFileIds`, `getVisibleTvFileIds`. The MigrationPreviewModal version (250 lines) is null-safe (`MigrationPreview | null`), while the DryRunModal version (160 lines) assumes non-null. Pick the MigrationPreviewModal version — null-safe is strictly more general. Write to `shared/filter-utils.ts`. Update importers. Delete both copies. **Run tsc immediately** — if DryRunModal callers pass non-nullable types to the null-safe version, it will still compile (non-null is assignable to nullable).

**Step 6 — Merge `preview-utils.ts` / `edit-utils.ts` (same purpose, different null-safety):**
Both implement `computeEditedPreview`. Pick the MigrationPreviewModal version (`edit-utils.ts`, null-safe). Write to `shared/edit-utils.ts`. Update DryRunModal's importers (they currently import from `./preview-utils`; change to `../shared/edit-utils`). Delete both copies.

**Step 7 — Final verification:**
```bash
./scripts/consolidation/verify.sh
./scripts/consolidation/check-old-imports.sh \
  "DryRunModal/summary-card" "DryRunModal/file-item" "DryRunModal/movies-list" \
  "DryRunModal/series-list" "DryRunModal/aggregated-file-tooltip" "DryRunModal/assign-modal" \
  "DryRunModal/utils" "DryRunModal/debug" "DryRunModal/types" "DryRunModal/filter-utils" \
  "DryRunModal/preview-utils" \
  "MigrationPreviewModal/summary-card" "MigrationPreviewModal/file-item" \
  "MigrationPreviewModal/movies-list" "MigrationPreviewModal/series-list" \
  "MigrationPreviewModal/aggregated-file-tooltip" "MigrationPreviewModal/assign-modal" \
  "MigrationPreviewModal/utils" "MigrationPreviewModal/debug" "MigrationPreviewModal/types" \
  "MigrationPreviewModal/filter-utils" "MigrationPreviewModal/edit-utils"
```

### Hook Consolidation — NOT RECOMMENDED

The top-level hooks (`use-dry-run-modal.ts` vs `use-migration-preview-modal.ts`) have fundamentally different architectures:
- DryRunModal uses monolithic `useModalState` + `patch` pattern with `useEffect` for reset
- MigrationPreviewModal uses split `usePreviewData` + `useFileActions` with render-time state adjustment
- Different return value shapes (`allFilesAccountedFor`/`confirmModalOpen` vs `allSelected`/`selectedCount`/`isExecuting`)

These encode genuinely different modal lifecycles. Do NOT attempt to merge them.

---

## Wave 2 — Media Table/Grid/Edit Components (~380 lines saved) ✅ COMPLETED

**Priority: HIGH** — Near-identical pairs already using generics. Easiest wins in the codebase.

**Status:** Completed. Created 4 generic components + 1 extracted hook in `web/src/components/media/`. 8 duplicate files deleted. All importers updated. Verified: tsc pass, build pass, 0 lint errors, no stale imports.

- `media-table.tsx` — Generic `MediaTable<T>` with `theme` prop for checkbox accent colors
- `media-grid.tsx` — Generic `MediaGrid<T>` with `renderCard` render prop
- `grouped-media-grid.tsx` — Generic `GroupedMediaGrid<T>` with `renderGrid` render prop
- `media-edit-dialog.tsx` — Generic dialog (37 lines) with `mediaLabel` and `monitoredDescription` props
- `use-media-edit-dialog.ts` — Extracted hook for edit dialog state (to stay under 50-line lint limit)

These pairs were introduced during the lint refactor decomposition and differ only by type parameters and theme colors.

### Component Pairs

| Movie File | Series File | Lines Each | Similarity | Key Differences |
|---|---|---|---|---|
| `movie-table.tsx` | `series-table.tsx` | 162 | ~98% | Checkbox theme class (`data-checked:bg-movie-500` vs `data-checked:bg-tv-500`), type params |
| `movie-grid.tsx` | `series-grid.tsx` | 38 | ~98% | Type params and component references only |
| `grouped-movie-grid.tsx` | `grouped-series-grid.tsx` | 42 | ~99% | Type params and component references only |
| `movie-edit-dialog.tsx` | `series-edit-dialog.tsx` | ~143 | ~97% | Hook names (`useUpdateMovie`/`useUpdateSeries`), toast messages, one description text line |

### Execution Steps (per component pair)

This wave is best executed as 4 parallel subagents, one per pair. Each subagent follows this protocol:

**Step 1 — Read both files in the pair.** Note every difference.

**Step 2 — Identify parameterization points.** For each difference, determine the prop:
- Type parameter `T` for the media item type
- `theme: 'movie' | 'tv'` for color classes
- `renderCard?: (item: T) => ReactNode` for grid components (instead of importing MovieCard/SeriesCard)
- `useUpdateMutation` callback for edit dialog
- String props for labels/messages

**Step 3 — Create the generic component.** Write to `web/src/components/media/<name>.tsx`. The generic MUST accept all props needed by both movie and series consumers. Check that the generic compiles: `cd web && bunx tsc --noEmit`.

**Step 4 — Update importers.** Run `./scripts/consolidation/find-importers.sh movie-table` (etc.) to find all import sites. Update each to import the new generic and pass the appropriate props.

**Step 5 — Delete the old specific files.** Only after all importers are updated and `tsc --noEmit` passes.

**Step 6 — Verify.** `cd web && bunx tsc --noEmit && bun run build`.

### Implementation Notes

- **MediaTable:** The table components already accept `ColumnDef<T>[]` as a prop. The only hard-coded difference is the checkbox accent color class. Add `theme: 'movie' | 'tv'` prop and template the class: `` `data-checked:bg-${theme === 'movie' ? 'movie' : 'tv'}-500` ``. NOTE: Tailwind requires full class strings for purging — do NOT use string interpolation for class names. Instead use a conditional: `theme === 'movie' ? 'data-checked:bg-movie-500' : 'data-checked:bg-tv-500'`.

- **MediaGrid / GroupedMediaGrid:** Accept `renderCard: (item: T) => ReactNode` instead of importing a specific card component. The movie/series pages pass `(movie) => <MovieCard ... />` or `(series) => <SeriesCard ... />` respectively.

- **MediaEditDialog:** Accept the update mutation hook as a prop: `useUpdate: () => UseMutationResult<...>`. The caller passes `useUpdateMovie` or `useUpdateSeries`. Accept `mediaType: 'movie' | 'series'` for the description text.

---

## Wave 3 — Media List Page UI Shells (~400 lines saved) ✅ COMPLETED

**Priority: HIGH** — Presentation components with mechanical name/color differences.

**Status:** Completed. Created 5 generic components in `web/src/components/media/`: `media-list-toolbar.tsx`, `media-page-actions.tsx`, `media-delete-dialog.tsx`, `media-list-content.tsx`, `media-list-filters.tsx`. Updated both layout files to use the generics. 10 duplicate files deleted. Filter/sort option definitions moved to the layout files. Verified: tsc pass, build pass, 0 lint errors, no stale imports.

### Near-Identical UI Components

| Movie File | Series File | Lines Each | Similarity | Key Differences |
|---|---|---|---|---|
| `movie-list-layout.tsx` | `series-list-layout.tsx` | 61 | ~95% | Component names, prop names |
| `movie-list-toolbar.tsx` | `series-list-toolbar.tsx` | 51 | ~98% | Theme color class only |
| `movie-page-actions.tsx` | `series-page-actions.tsx` | 54 | ~95% | Theme color, "Add Movie"/"Add Series" text |
| `movie-delete-dialog.tsx` | `series-delete-dialog.tsx` | ~63 | ~92% | Pluralization logic ("movie(s)" vs "series"), checkbox id |
| `movie-list-content.tsx` | `series-list-content.tsx` | ~72 | ~88% | Series empty state has extra `action` prop, Movie type has extra `releaseDate` intersection |
| `movie-list-filters.tsx` | `series-list-filters.tsx` | ~129 | ~85% | Series has 2 extra filter options (`continuing`/`ended`), different sort field (`releaseDate` vs `nextAirDate`) |
| `index.tsx` | `index.tsx` | 21 | ~95% | Component/hook names only |

### Execution Steps

Execute the 5 inner components (toolbar, page-actions, delete-dialog, list-content, list-filters) as parallel subagents. Then do layout and index.tsx sequentially afterward, since they import the inner components.

For each component, follow the same Step 1-6 protocol from Wave 2.

### Implementation Notes

- **MediaListToolbar:** The Props type is already identical between movie/series. Only the theme color className differs. Accept `theme: 'movie' | 'tv'`.

- **MediaPageActions:** Parameterize with `theme` and `addLabel: string` (e.g., "Add Movie" or "Add Series").

- **MediaDeleteDialog:** Accept `mediaLabel: string` (e.g., "movie" or "series") and `pluralMediaLabel: string` (e.g., "movies" or "series"). Do NOT compute pluralization — "series" is both singular and plural and automated pluralization will produce "seriess".

- **MediaListContent:** The series version has an `action` prop on its `EmptyState` that the movie version lacks. The generic must include this as an optional prop. The movie type uses `MediaGroup<Movie & { releaseDate?: string }>` while series uses `MediaGroup<Series>` — accept `MediaGroup<T>` and let the caller provide the intersection type.

- **MediaListFilters:** Accept `filterOptions: FilterOption[]` and `sortOptions: SortOption[]` as props instead of hardcoding `FILTER_OPTIONS` and `SORT_OPTIONS`. The movie page passes 7 filters, the series page passes 9 (adding `continuing` and `ended`). Accept `theme: 'movie' | 'tv'` for sort icon color.

- **MediaListLayout:** After the inner components are generic, this becomes trivial — it just composes them. Accept `theme` and `title: string`.

### List Hooks — DO NOT EXTRACT

The `use-movie-list.ts` (333 lines) and `use-series-list.ts` (301 lines) hooks share structural patterns but have meaningful domain differences that make a generic base hook counterproductive:

- Movie has 2 extra mutations (`searchMutation`, `deleteMutation`) with inline handlers for per-item search/delete; series does not expose these
- `FilterStatus` is a 7-member union for movies, 9-member for series (adds `continuing`/`ended`)
- `SortField` differs semantically (`releaseDate` with 3 fallback date fields vs `nextAirDate`)
- Movie does a `.map()` transformation on items before grouping; series passes data directly
- The underlying filter logic is fundamentally different — movies check a flat `status` field, series check `productionStatus` AND iterate `statusCounts`

A `useMediaList<T>` base hook would need 15+ configuration fields, conditional mutation injection, and pluggable filter/sort strategies. That's a framework, not a hook. Leave these as-is.

---

## Wave 4 — Add Media Search (~225 lines removed) ✅ COMPLETED

**Priority: HIGH** — Search components were dead code.

**Status:** Completed. Deleted `add-movie-search.tsx` (111 lines) and `add-series-search.tsx` (114 lines) as dead code. The unified search bar in the top nav handles all media searching — the per-page inline search was a legacy codepath. Add pages (`/movies/add`, `/series/add`) now redirect to `/search` when accessed without a `tmdbId` query param. The add page hooks were simplified to remove all search-related state (Step type, searchQuery, debouncedSearchQuery, searchInputRef, useMovieSearch/useSeriesSearch queries, handleSelectMovie/handleSelectSeries handlers).

---

## Wave 5 — External Search Cards (~140 lines saved) ✅ COMPLETED

**Priority: MEDIUM** — Two card components with ~88% shared logic.

**Status:** Completed. The generic `ExternalMediaCard` (created during lint remediation) already existed with decomposed subcomponents (`CardPoster`, `CardActionButton`, `StatusBadge`) and `useExternalMediaCard` hook. Migrated the admin search page (`routes/search/index.tsx`) to use it instead of the old `ExternalMovieCard`/`ExternalSeriesCard`. Added `toAvailability` helper to map admin request data to the `AvailabilityInfo` format. Extracted `ExternalResults` component for 50-line lint compliance. Deleted both old card files (251 lines removed). Verified: tsc pass, build pass, 0 lint errors, no stale imports.

| Movie File | Series File | Lines | Similarity |
|---|---|---|---|
| `external-movie-card.tsx` | `external-series-card.tsx` | 123/128 | ~88% |

Shared code that is 100% identical:
- `STATUS_BADGE_MAP` with badge configurations (~37 lines)
- `CardAction` button logic (~11 lines)
- `RequestInfo` display

Differences:
- Series shows `NetworkLogo` on poster (conditional rendering)
- Series shows network text badge when no logo URL exists
- Theme colors (`movie-500` vs `tv-500`)
- Navigation targets (`/movies/add` vs `/series/add`)

### Execution Steps

Single agent. Follow Wave 2 Step 1-6 protocol.

### Implementation

Create `web/src/components/search/external-media-card.tsx`. Accept a union type or generic for the media item, plus `mediaType: 'movie' | 'series'`. The `NetworkLogo` and network badge render conditionally when `mediaType === 'series'` and the relevant fields exist on the item.

Extract `STATUS_BADGE_MAP`, `getStatusBadge`, and `CardAction` as module-level definitions in the same file (they're already shared logic, just currently duplicated).

---

## Wave 6 — Query Key Factory (~40 lines saved) ✅ COMPLETED

**Priority: MEDIUM** — Standardized query key definitions across hook files.

**Status:** Completed. Created `web/src/lib/query-keys.ts` with `createQueryKeys(...scope)` factory supporting variadic scope segments (e.g., `createQueryKeys('admin', 'users')` for multi-segment keys). Migrated 12 of 38 key definitions — these are the files where the standard 5-key pattern (all, lists, list, details, detail) applies cleanly. The remaining 26 definitions have entirely custom key structures (e.g., `metadataKeys` with `movieSearch`/`seriesSearch`, `prowlarrKeys` with `config`/`capabilities`) or would gain dead unused keys from the factory. Verified: tsc pass, build pass, 0 lint errors.

**Files migrated (12):**
- Exact standard 5: `downloadClientKeys`, `adminUserKeys`
- Standard 5 + extras (spread + extend): `qualityProfileKeys`, `notificationKeys`, `rootFolderKeys`, `userNotificationKeys`, `indexerKeys`, `slotsKeys`
- Standard 5 with `list(filters)` override: `movieKeys`, `seriesKeys`, `adminRequestKeys`, `requestKeys`

### NOT RECOMMENDED: `createMutation` factory

~100-120 mutation hooks follow a simple CRUD pattern, but each is only 4-8 lines. A factory:
1. **Loses hook identity** — React DevTools shows the generic inner name instead of `useDeleteMovie`
2. **Can't be extended** — The moment one mutation needs `onError` or `onMutate`, you bloat the factory or eject
3. **Solves a 4-line problem** — Structural boilerplate that serves as self-documenting intent

Leave individual mutation hooks as-is.

### NOT RECOMMENDED: `useTestableForm` hook

`use-indexer-dialog.ts`, `use-download-client-dialog.ts`, `use-form-actions.ts` (notifications), and `use-prowlarr-config-form.ts` share the test-then-submit lifecycle pattern (`isTesting` state, try/catch/toast). However, after reading all 4 hooks, the implementations diverge too much for a useful generic:
1. **Indexer dialog** has a 2-step wizard (select → configure), definition/schema loading, and constructs test input from `selectedDefinition.id` + `settings`
2. **Download client dialog** has type-specific port defaults, debug torrent feature, and passes full `formData` as test input
3. **Notifications** uses callback overrides (`onCreate`/`onUpdate`/`onTest`), schema-based validation, and passes form data as handler params (not from internal state)
4. **Prowlarr config** has no create/update — only save. URL parsing/building, category management, dirty tracking, refresh action

A `useTestableForm` generic would need 10+ config fields, conditional mutation injection, pluggable validation, and custom toast messages per form. The shared boilerplate is only ~10 lines per hook (the `isTesting` + try/catch/toast pattern). Net savings after adding the generic's complexity: near zero.

---

## Wave 7 — Shared UI Primitives & Utilities (~90 lines saved) ✅ COMPLETED

**Priority: LOWER** — Smaller wins that improve consistency.

**Status:** Completed. Created 2 new shared utilities. Migrated 15 files total. Verified: tsc pass, build pass, 0 lint errors.

### `LoadingButton` Component

**File:** `web/src/components/ui/loading-button.tsx`

Handles two usage patterns: spinner-only (no `icon` prop — shows Loader2 when loading, nothing otherwise) and icon-swap (with `icon` prop — shows Loader2 when loading, the specified icon otherwise). Extends `ButtonProps` with `loading: boolean`, `icon?: LucideIcon`, and `iconClassName?: string`. Automatically disables when loading.

**Files migrated (10 files, ~18 instances):**
1. `indexer-dialog.tsx` — Test button (icon-swap), Submit button (spinner-only)
2. `prowlarr-config-form.tsx` — Deleted 3 inline components (TestButton, RefreshButton, SaveButton)
3. `download-client-dialog.tsx` — Deleted 3 inline components (TestButton, DebugButton, SubmitButton)
4. `notification-dialog.tsx` — Test button, Submit button
5. `media-edit-dialog.tsx` — Save button
6. `search-input-bar.tsx` — Search button (icon-swap)
7. `search-release-row.tsx` — Grab button (icon-swap, icon-only size)
8. `search-button.tsx` — Search All button (icon-swap)
9. `setup.tsx` — Create Administrator button
10. `login.tsx` — Passkey button, Sign In button, Debug Delete button (with custom `iconClassName` for responsive sizing)

### `withToast` Utility

**File:** `web/src/lib/with-toast.ts`

Wraps an async function with try/catch, showing `toast.error(errorMsg)` on failure. Promoted from local definition in `use-notifications-page.ts`.

**Files migrated (5 files):**
1. `use-notifications-page.ts` — Replaced local definition with import
2. `use-series-detail.ts` — 6 handlers simplified
3. `quality-profiles-section.tsx` — 1 handler
4. `root-folders-section.tsx` — 4 handlers
5. `indexer-mode-toggle.tsx` — 1 handler

### Items Assessed But NOT Done (with rationale)

- **`useDialogState<T>`**: Only 2 files are clean fits. The other 5 have external open/close management, form state, multi-step wizards, or other complexity. Not worth a 15-line hook for 2 callers.
- **FormField wrappers**: The wrapped pattern is 3 lines (`div > Label + Input`). Abstraction adds import overhead and indirection for minimal savings. `react-hook-form` is installed (v7.68) but unused — creating our own FormField may conflict with its adoption.
- **Settings CRUD Sections**: Plan says "low priority" — skipped.

---

## Summary

| Wave | Target | Lines Saved | Effort | Risk | Status |
|---|---|---|---|---|---|
| 1 | Slot component dedup | ~850 | Low | Low — pure moves + pick canonical filter-utils | ✅ Done |
| 2 | Media table/grid/edit components | ~380 | Low | Low — already use generics | ✅ Done |
| 3 | Media list page UI shells | ~400 | Medium | Low — presentation only, hooks untouched | ✅ Done |
| 4 | Add media search | ~225 | Low | Low — dead code removal | ✅ Done |
| 5 | External search cards | ~140 | Low | Low — small component | ✅ Done |
| 6 | Query key factory | ~40 | Low | Low — behind-the-scenes | ✅ Done |
| 7 | Shared UI primitives & utilities | ~90 | Medium | Low — additive abstractions | ✅ Done |
| **Total** | | **~2,125** | | | **All done** |

### Areas Assessed But NOT Recommended for Consolidation

- **Individual mutation hooks**: ~100-120 hooks follow a simple CRUD pattern, but each is only 4-8 lines. A `createMutation` factory loses hook identity in DevTools, can't be extended without reimplementing `useMutation`, and solves a documentation-as-code pattern that doesn't need DRYing.
- **`useTestableForm` hook**: The 4 dialog hooks (`use-indexer-dialog.ts`, `use-download-client-dialog.ts`, `use-form-actions.ts`, `use-prowlarr-config-form.ts`) share the test-then-submit lifecycle pattern but diverge too much in practice: 2-step wizard vs flat form, create/update vs save-only, callback overrides vs direct mutations, schema-based validation vs field checks. The shared boilerplate is ~10 lines per hook; a generic would need 10+ config fields. Net savings near zero.
- **Media list hooks** (`use-movie-list.ts` / `use-series-list.ts`): Share structural patterns but have meaningful domain differences (different filter/sort types, different mutations, different data transformations). A generic base hook would be more complex than the duplication.
- **API layer**: Already well-structured with consistent `apiFetch` pattern. No significant duplication.
- **Detail pages** (`movies/$id.tsx` vs `series/$id.tsx`): Similar structure but series has enough unique complexity (seasons, episodes) that forcing a generic would add unnecessary abstraction.
- **Missing/upgradable lists**: Movies (flat list) vs series (accordion with seasons/episodes) are structurally different enough that a shared component would be more complex than the duplication.
- **Configure pages** (`add-movie-configure.tsx` vs `add-series-configure.tsx`): Series has significantly more options (season folders, specials, monitoring granularity) making the shared portion small relative to the differences.
- **Media cards** (`movie-card.tsx` vs `series-card.tsx`): SeriesCard is nearly 2x the size with NetworkLogo, status counts, production status, next airing display. Too divergent.
- **Column definitions** (`movie-columns.tsx` vs `series-columns.tsx`): ~50% similar but column definitions are domain-specific by nature.
- **Responsive button pattern**: Only 2 files, 4 button pairs. Not enough duplication to justify an abstraction.
- **Stores**: Each store serves a distinct purpose with no meaningful overlap.
- **Types**: Type definitions are well-organized and not duplicated.

### Execution Notes

- Each wave is independent and can be done in any order
- Wave 1 is pure file moves — lowest risk, highest impact
- Wave 2 is the easiest generic extraction since the components already use generic type parameters
- Wave 3 extracts presentation-only components — hooks are intentionally left alone
- Waves 4-5 extract generic components — test by verifying the pages render correctly
- Wave 6 is a behind-the-scenes refactor — no UI impact
- Wave 7 is additive — new shared components/utilities, then migrate callers gradually
- After each wave: `./scripts/consolidation/verify.sh`
- **Tailwind class purging caveat:** Never construct Tailwind class names dynamically via string interpolation (e.g., `` `bg-${theme}-500` ``). Tailwind's JIT compiler needs to see full class strings in source code. Always use conditionals: `theme === 'movie' ? 'bg-movie-500' : 'bg-tv-500'`.
