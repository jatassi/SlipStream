# Shared UI Component Catalog

Reusable components available to module authors. All paths are relative to `web/src/`.

---

## 1. Component Catalog

### Media List & Layout

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `ModuleListPage` | `components/media/module-list-page.tsx` | Top-level lazy-loading wrapper for a module's list page. Resolves module by ID and renders with Suspense. | `moduleId` |
| `MediaListLayout` | `components/media/media-list-layout.tsx` | Full list/grid page shell with header, filters, sorting, bulk edit toolbar, and delete dialog. Generic over item type `T`, filter keys `F`, and sort keys `S`. | `theme`, `title`, `addLabel`, `mediaLabel`, `pluralMediaLabel`, `filterOptions`, `sortOptions`, `renderCard`, `allTableColumns`, `items`, `groups`, and many callback props (see type `MediaListLayoutProps`) |
| `MediaListFilters` | `components/media/media-list-filters.tsx` | Filter dropdown, sort selector, view toggle (grid/table), poster size slider, column config. | `filterOptions`, `sortOptions`, `statusFilters`, `sortField`, `view`, `posterSize`, `theme`, `onToggleFilter`, `onSortFieldChange`, `onViewChange` |
| `MediaListToolbar` | `components/media/media-list-toolbar.tsx` | Bulk-edit toolbar shown in edit mode. Select all, monitor/unmonitor, change quality profile, delete. | `selectedCount`, `totalCount`, `qualityProfiles`, `isBulkUpdating`, `theme`, `onSelectAll`, `onMonitor`, `onDelete` |
| `MediaListContent` | `components/media/media-list-content.tsx` | Switches between loading, empty, grid, grouped grid, and table views based on state. | `isLoading`, `view`, `items`, `groups`, `renderCard`, `theme`, `emptyIcon`, `emptyTitle` |
| `MediaPageActions` | `components/media/media-page-actions.tsx` | Header action buttons: Refresh, Edit mode toggle, Add button (themed). | `isLoading`, `editMode`, `isRefreshing`, `theme`, `addLabel`, `onRefreshAll`, `onEnterEdit`, `onExitEdit` |
| `MediaGrid` | `components/media/media-grid.tsx` | Responsive CSS grid that renders cards via `renderCard`. | `items`, `renderCard`, `posterSize`, `editMode`, `selectedIds`, `onToggleSelect` |
| `GroupedMediaGrid` | `components/media/grouped-media-grid.tsx` | Renders items in labeled groups with sticky headers. | `groups: MediaGroup<T>[]`, `renderGrid` |
| `MediaTable` | `components/media/media-table.tsx` | Sortable table view with optional edit-mode checkboxes. | `items`, `columns`, `visibleColumnIds`, `renderContext`, `sortField`, `sortDirection`, `editMode`, `selectedIds`, `theme` |

### Dialogs

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `MediaDeleteDialog` | `components/media/media-delete-dialog.tsx` | Confirmation dialog for bulk deletion with "also delete files" checkbox. | `open`, `onOpenChange`, `selectedCount`, `deleteFiles`, `onDeleteFilesChange`, `onConfirm`, `isPending`, `mediaLabel`, `pluralMediaLabel` |
| `MediaEditDialog` | `components/media/media-edit-dialog.tsx` | Edit dialog for a single media item (quality profile, monitored toggle). Generic over item type. | `open`, `onOpenChange`, `item`, `updateMutation`, `mediaLabel`, `moduleType`, `monitoredDescription` |

### Add Media

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `AddMediaConfigure` | `components/media/add-media-configure.tsx` | Configuration step when adding media: preview card, root folder select, quality profile select, and module-specific `children` slot. | `preview`, `rootFolders`, `qualityProfiles`, `rootFolderId`, `qualityProfileId`, `onFolderChange`, `onProfileChange`, `isPending`, `onBack`, `onAdd`, `addLabel`, `children` |
| `MediaPreview` | `components/media/add-media-configure.tsx` | Poster + title + overview preview card for the add flow. | `title`, `year`, `overview`, `posterUrl`, `type`, `subtitle` |
| `FolderSelect` | `components/media/media-configure-fields.tsx` | Root folder dropdown. | `rootFolderId`, `rootFolders`, `onChange` |
| `ProfileSelect` | `components/media/media-configure-fields.tsx` | Quality profile dropdown. | `qualityProfileId`, `qualityProfiles`, `onChange` |
| `ToggleField` | `components/media/media-configure-fields.tsx` | Label + description + switch toggle row. | `label`, `description`, `checked`, `onChange` |
| `FormActions` | `components/media/media-configure-fields.tsx` | Back + Add action buttons with validation. | `rootFolderId`, `qualityProfileId`, `isPending`, `onBack`, `onAdd`, `addLabel` |
| `MonitorSelect` | `components/media/media-configure-fields.tsx` | Monitor strategy dropdown (TV-specific: all, future, etc.). | `value`, `onChange` |
| `SearchOnAddSelect` | `components/media/media-configure-fields.tsx` | Search-on-add strategy dropdown. | `value`, `onChange` |

### Images & Artwork

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `PosterImage` | `components/media/poster-image.tsx` | Poster image with local artwork cache, TMDB fallback chain, loading skeleton, and fallback icon. | `path`, `url`, `tmdbId`, `tvdbId`, `alt`, `size`, `type` (`'movie'`/`'series'`), `version`, `className`, `onAllFailed` |
| `BackdropImage` | `components/media/backdrop-image.tsx` | Backdrop/fanart image with gradient overlay, artwork cache, and error fallback. | `path`, `tmdbId`, `tvdbId`, `type`, `alt`, `size`, `version`, `className`, `overlay` |
| `TitleTreatment` | `components/media/title-treatment.tsx` | Logo/title treatment image with fallback to a custom `ReactNode`. | `tmdbId`, `tvdbId`, `type`, `alt`, `version`, `fallback`, `className` |
| `StudioLogo` | `components/media/studio-logo.tsx` | Production studio logo with auto-sizing. | `tmdbId`, `type`, `alt`, `version`, `fallback`, `className` |
| `NetworkLogo` | `components/media/network-logo.tsx` | TV network logo badge (inverted white on dark). | `logoUrl`, `network`, `className` |

### Badges & Status

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `MediaStatusBadge` | `components/media/media-status-badge.tsx` | Color-coded badge for media status (unreleased, missing, downloading, failed, upgradable, available). Supports icon-only mode. | `status: MediaStatus`, `iconOnly`, `className` |
| `QualityBadge` | `components/media/quality-badge.tsx` | Monospace badge showing quality/resolution label. Variant adjusts by resolution tier. | `quality`, `resolution`, `className` |
| `ProductionStatusBadge` | `components/media/production-status-badge.tsx` | Badge for series production status (continuing, ended, upcoming). | `status: ProductionStatus`, `className` |
| `ProgressBar` | `components/media/progress-bar.tsx` | Themed progress bar (sm/md/lg). Variant controls color (`default`, `movie`, `tv`). | `value`, `max`, `showLabel`, `size`, `variant`, `className` |

### Rating Icons

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `RTFreshIcon` | `components/media/rating-icons.tsx` | Rotten Tomatoes "Fresh" icon | `className` |
| `RTRottenIcon` | `components/media/rating-icons.tsx` | Rotten Tomatoes "Rotten" icon | `className` |
| `IMDbIcon` | `components/media/rating-icons.tsx` | IMDb logo icon | `className` |
| `MetacriticIcon` | `components/media/rating-icons.tsx` | Metacritic logo icon | `className` |

### Settings

| Component | Location | Purpose | Key Props |
|---|---|---|---|
| `PatternEditor` | `components/settings/sections/naming-pattern-editor.tsx` | File naming pattern editor with token builder dialog, live preview, and token breakdown. | `label`, `value`, `onChange`, `description`, `mediaType`, `tokenContext` |
| `VersionSlotsSection` | `components/settings/sections/version-slots-section.tsx` | Quality version slots management UI with dry-run modals and debug panel. | (internal hook-driven; used in settings pages) |
| `QualityProfilesSection` | `components/settings/sections/quality-profiles-section.tsx` | Quality profile CRUD list with inline allowed-quality badges. | (internal hook-driven; used in settings pages) |

---

## 2. Adding a New Module

### Step 1: Create module config

Create `web/src/modules/<module-id>/index.ts` exporting a `ModuleConfig` object:

```ts
import { MyIcon } from 'lucide-react'
import { myApi } from '@/api/my-module'
import { MyCard } from '@/components/my-module/my-card'
import { myKeys } from '@/hooks/use-my-module'
import type { ModuleConfig } from '../types'

export const myModuleConfig: ModuleConfig = {
  id: 'mymodule',
  name: 'My Module',
  singularName: 'Item',
  pluralName: 'Items',
  icon: MyIcon,
  themeColor: 'mymodule',
  basePath: '/my-module',
  routes: [
    { path: '/', id: 'myModuleList' },
    { path: '/$id', id: 'myModuleDetail' },
    { path: '/add', id: 'addMyModule' },
  ],
  queryKeys: myKeys,
  wsInvalidationRules: [
    { pattern: 'mymodule:(added|updated|deleted)', queryKeys: [myKeys.all] },
  ],
  filterOptions: [ /* ... */ ],
  sortOptions: [ /* ... */ ],
  tableColumns: { static: [], defaults: [] },
  cardComponent: MyCard,
  detailComponent: () => null,
  api: { /* implement ModuleApi */ },
}
```

### Step 2: Register in setup

Add to `web/src/modules/setup.ts`:

```ts
import { myModuleConfig } from './my-module'

export function setupModules(): void {
  // ... existing registrations
  registerModule(myModuleConfig)
}
```

### Step 3: Add theme colors to CSS

See Section 3 below.

### Step 4: Add theme class mappings

Add your module's theme key to every `Record<string, string>` lookup map that maps theme names to Tailwind classes. See Section 3 for the full list of files.

### Step 5: Add routes

Register lazy-loaded routes in `web/src/routes-config.tsx` and update `ModuleListPage` in `components/media/module-list-page.tsx` to handle the new module ID.

---

## 3. Theme System

### Color Variables

Theme colors are defined in `web/src/index.css` using OKLCH color space with shades 50 through 950. Each module needs two blocks:

1. **CSS variable definitions** (inside `:root` / `.dark`):
```css
/* MyModule (green) palette */
--mymodule-50: oklch(97% 0.01 145);
--mymodule-100: oklch(94% 0.03 145);
/* ... shades 200-800 ... */
--mymodule-900: oklch(35% 0.08 145);
--mymodule-950: oklch(25% 0.06 145);
```

2. **Tailwind color scale mappings** (inside `@theme inline`):
```css
--color-mymodule-50: var(--mymodule-50);
--color-mymodule-100: var(--mymodule-100);
/* ... all shades ... */
--color-mymodule-950: var(--mymodule-950);
```

This enables Tailwind classes like `text-mymodule-500`, `bg-mymodule-500/10`, `border-mymodule-600`, etc.

### Theme Class Conventions

- Text on dark backgrounds: **400 shades** (`text-{theme}-400`)
- Borders and accents: **500 shades** (`text-{theme}-500`, `border-{theme}-500`)
- Subtle backgrounds: **500 shade with opacity** (`bg-{theme}-500/10`, `bg-{theme}-500/20`)
- Buttons: `bg-{theme}-500 hover:bg-{theme}-400 border-{theme}-500`
- Glow effects: `glow-{theme}`, `hover:glow-{theme}`
- Mixed/gradient: `bg-media-gradient`, `text-media-gradient`

### Files with Theme Lookup Maps

The following files contain `Record<string, string>` maps that must be updated when adding a new module theme. Each maps a theme key (e.g., `'movie'`, `'tv'`) to Tailwind class strings:

| File | Map Name | Purpose |
|---|---|---|
| `components/media/progress-bar.tsx` | `indicatorClasses` | Progress bar fill color |
| `components/media/media-list-filters.tsx` | `accentMap` | Sort indicator accent |
| `components/media/media-table.tsx` | `checkboxClassMap` | Edit-mode checkbox color |
| `components/media/media-page-actions.tsx` | (inline object) | Add button styling |
| `components/media/media-list-toolbar.tsx` | (inline object) | Bulk toolbar border/bg |
| `components/tables/column-config-popover.tsx` | `accentMap` | Column config accent |
| `components/ui/filter-dropdown.tsx` | `THEME_ACTIVE_CLASS` | Active filter text color |
| `routes/downloads/download-row.tsx` | `THEME_HOVER_TEXT`, `THEME_HOVER_BG` | Download row hover styles |
| `routes/downloads/download-row-poster.tsx` | `THEME_POSTER_CLASSES` | Download row poster fallback |
| `routes/downloads/download-row-progress.tsx` | `MEDIA_TYPE_VARIANT` | Download progress variant |
| `routes/missing/media-tabs.tsx` | `THEME_GLOW_CLASSES` | Tab glow effect |
| `routes/missing/missing-tab-content.tsx` | `THEME_TEXT_CLASSES` | Missing tab text color |
| `routes/missing/upgradable-tab-content.tsx` | `THEME_TEXT_CLASSES` | Upgradable tab text color |
| `routes/history/history-components.tsx` | `MEDIA_HOVER_CLASSES`, `MEDIA_ROW_HOVER`, `MEDIA_ICON_CLASSES` | History row styling |

---

## 4. ModuleConfig Interface Reference

Defined in `web/src/modules/types.ts`.

| Field | Type | Description |
|---|---|---|
| `id` | `string` | Unique module identifier (e.g., `'movie'`, `'tv'`). Used as lookup key everywhere. |
| `name` | `string` | Display name for the module (e.g., `'Movies'`, `'Series'`). Used in nav and page headers. |
| `singularName` | `string` | Singular noun for one item (e.g., `'Movie'`, `'Series'`). Used in dialogs and labels. |
| `pluralName` | `string` | Plural noun (e.g., `'Movies'`, `'Series'`). Used in counts and bulk action labels. |
| `icon` | `LucideIcon` | Lucide icon component shown in navigation and headers. |
| `themeColor` | `string` | Theme key matching CSS color variables (e.g., `'movie'`, `'tv'`). Drives all themed styling. |
| `basePath` | `string` | URL base path for this module's routes (e.g., `'/movies'`, `'/series'`). |
| `routes` | `ModuleRouteConfig[]` | Array of `{ path, id }` defining the module's sub-routes (list, detail, add). |
| `queryKeys` | `ModuleQueryKeys` | TanStack Query key factory with `all`, `list(...)`, `detail(id)`, and optional custom keys. |
| `wsInvalidationRules` | `WSInvalidationRule[]` | WebSocket event patterns that trigger query invalidation. Each has `pattern` (regex), `queryKeys`, and optional `alsoInvalidate`. |
| `filterOptions` | `ModuleFilterOption[]` | Status filter options shown in the filter dropdown. Each has `value`, `label`, `icon`. |
| `sortOptions` | `ModuleSortOption[]` | Sort field options. Each has `value` and `label`. |
| `tableColumns` | `ModuleTableColumns` | Table column definitions: `static` (always visible) and `defaults` (default visible column IDs). |
| `cardComponent` | `ComponentType<any>` | Card component rendered in grid view for each item. |
| `detailComponent` | `ComponentType<any>` | Detail page component rendered for a single item. |
| `addConfigFields` | `ComponentType<any>` (optional) | Extra configuration fields rendered as `children` inside `AddMediaConfigure`. |
| `api` | `ModuleApi` | API adapter implementing `list`, `get`, `update`, `delete`, `bulkDelete`, `bulkUpdate`, `bulkMonitor`, `search`, `refresh`, `refreshAll`. |

### Supporting Types

**`ModuleRouteConfig`**: `{ path: string; id: string }`

**`ModuleQueryKeys`**: `{ all: readonly string[]; list: (...args) => readonly unknown[]; detail: (id: number) => readonly unknown[]; [key: string]: unknown }`

**`WSInvalidationRule`**: `{ pattern: string; queryKeys: readonly unknown[][]; alsoInvalidate?: readonly unknown[][] }`

**`ModuleFilterOption`**: `{ value: string; label: string; icon: LucideIcon }`

**`ModuleSortOption`**: `{ value: string; label: string }`

**`ModuleTableColumns`**: `{ static: unknown[]; defaults: string[] }`

**`ModuleEntity`**: Base entity shape that module items should conform to: `{ id, title, sortTitle, status, monitored, qualityProfileId, rootFolderId, path, sizeOnDisk?, addedAt }`

**`ModuleApi`**: `{ list, get, update, delete, bulkDelete, bulkUpdate, bulkMonitor, search, refresh, refreshAll }`
