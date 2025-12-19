# Frontend Development Plan

## Overview

This document outlines the implementation plan for the SlipStream frontend, a React-based web application for media library management (similar to Radarr/Sonarr).

### Current State

**Already Set Up:**
- Vite + React 19 + TypeScript
- Tailwind CSS v4
- shadcn/ui components (Base UI)
- TanStack Query (react-query) for data fetching
- TanStack Router for routing
- Zustand for state management
- React Hook Form + Zod for form handling
- Lucide React for icons

**Existing Components:**
- Basic UI primitives: Button, Card, Input, Label, Badge, Select, Dropdown, Alert Dialog, etc.
- Simple placeholder App.tsx with header and feature cards

### Backend API Endpoints Available

| Group | Endpoints |
|-------|-----------|
| **System** | `GET /health`, `GET /api/v1/status` |
| **Auth** | `POST /login`, `POST /logout`, `GET /auth/status` |
| **Movies** | Full CRUD at `/api/v1/movies` |
| **Series** | Full CRUD at `/api/v1/series` (with seasons/episodes) |
| **Quality Profiles** | Full CRUD at `/api/v1/qualityprofiles` |
| **Root Folders** | Full CRUD at `/api/v1/rootfolders` |
| **Metadata** | Search/fetch at `/api/v1/metadata` |
| **Indexers** | CRUD + test at `/api/v1/indexers` (placeholder) |
| **Download Clients** | CRUD + test at `/api/v1/downloadclients` (placeholder) |
| **Queue** | `GET /api/v1/queue`, `DELETE /api/v1/queue/:id` |
| **History** | `GET /api/v1/history` |
| **Search** | `GET /api/v1/search` |

---

## Target Features (from TODO.md)

- [ ] Dashboard/home page (system status, recent activity)
- [ ] Movies list and detail views
- [ ] Series list and detail views (with season/episode hierarchy)
- [ ] Add movie/series workflow (search + add)
- [ ] Quality profiles management UI
- [ ] Root folders management UI
- [ ] Settings pages
- [ ] Indexer configuration UI
- [ ] Download client configuration UI
- [ ] Queue/activity view
- [ ] History view
- [ ] Real-time updates via WebSocket

---

## 1. Routing & Navigation Architecture

### Route Structure

Using TanStack Router with file-based routing convention:

```
/                       → Dashboard (home page)
/movies                 → Movies list
/movies/:id             → Movie detail
/movies/add             → Add movie wizard (search → select → configure)
/series                 → Series list
/series/:id             → Series detail (with seasons/episodes)
/series/add             → Add series wizard
/activity               → Activity view (queue + recent)
/activity/queue         → Download queue
/activity/history       → History log
/settings               → Settings overview
/settings/profiles      → Quality profiles
/settings/rootfolders   → Root folders
/settings/indexers      → Indexers
/settings/downloadclients → Download clients
/settings/general       → General settings
```

### Layout Components

```
<RootLayout>                    # App shell with sidebar/header
  ├── <Sidebar />               # Main navigation
  ├── <Header />                # Top bar with search, notifications
  └── <Outlet />                # Page content
      └── <PageLayout>          # Per-page wrapper (title, actions)
```

### Router Setup

**File: `src/router.tsx`**
- Define route tree with TanStack Router
- Set up route guards (auth check if needed)
- Configure 404 handling

**File: `src/routes/__root.tsx`**
- Root layout with sidebar and header
- WebSocket connection provider
- Global error boundary

### Navigation Components

| Component | Purpose |
|-----------|---------|
| `Sidebar` | Main nav links, collapsible, active state |
| `Header` | Search bar, notifications, user menu |
| `Breadcrumbs` | Context navigation for nested routes |
| `PageHeader` | Page title + action buttons |

---

## 2. State Management Strategy

### Zustand Stores

Use Zustand for client-side UI state (not for server data - that's TanStack Query).

**File: `src/stores/ui.ts`**
```typescript
// UI state: sidebar collapsed, theme, notifications
interface UIStore {
  sidebarCollapsed: boolean
  toggleSidebar: () => void
  notifications: Notification[]
  addNotification: (n: Notification) => void
  dismissNotification: (id: string) => void
}
```

**File: `src/stores/websocket.ts`**
```typescript
// WebSocket connection state
interface WebSocketStore {
  connected: boolean
  lastMessage: WSMessage | null
  connect: () => void
  disconnect: () => void
}
```

### TanStack Query for Server State

All API data fetched via TanStack Query with proper caching:

| Query Key | Data |
|-----------|------|
| `['movies']` | Movie list |
| `['movies', id]` | Single movie |
| `['series']` | Series list |
| `['series', id]` | Single series with seasons/episodes |
| `['qualityProfiles']` | Quality profiles list |
| `['rootFolders']` | Root folders list |
| `['indexers']` | Indexers list |
| `['downloadClients']` | Download clients list |
| `['queue']` | Download queue |
| `['history']` | History entries |
| `['status']` | System status |
| `['metadata', 'movie', query]` | Movie search results |
| `['metadata', 'series', query]` | Series search results |

### Query Invalidation Strategy

WebSocket events should trigger query invalidation:

```typescript
// When WS receives 'movie:updated' event
queryClient.invalidateQueries({ queryKey: ['movies'] })

// When WS receives 'queue:updated' event
queryClient.invalidateQueries({ queryKey: ['queue'] })
```

---

## 3. API Client Layer

### HTTP Client Setup

**File: `src/lib/api.ts`**
```typescript
// Base fetch wrapper with error handling
const API_BASE = '/api/v1'

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })
  if (!res.ok) throw new ApiError(res.status, await res.json())
  return res.json()
}
```

### API Modules

Organize API calls by domain:

**File: `src/api/movies.ts`**
```typescript
export const moviesApi = {
  list: () => apiFetch<Movie[]>('/movies'),
  get: (id: number) => apiFetch<Movie>(`/movies/${id}`),
  create: (data: CreateMovie) => apiFetch<Movie>('/movies', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: number, data: UpdateMovie) => apiFetch<Movie>(`/movies/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: number) => apiFetch<void>(`/movies/${id}`, { method: 'DELETE' }),
  scan: (id: number) => apiFetch<void>(`/movies/${id}/scan`, { method: 'POST' }),
  search: (id: number) => apiFetch<void>(`/movies/${id}/search`, { method: 'POST' }),
}
```

**Similar modules for:**
- `src/api/series.ts`
- `src/api/qualityProfiles.ts`
- `src/api/rootFolders.ts`
- `src/api/indexers.ts`
- `src/api/downloadClients.ts`
- `src/api/queue.ts`
- `src/api/history.ts`
- `src/api/metadata.ts`
- `src/api/system.ts`

### Query Hooks

**File: `src/hooks/useMovies.ts`**
```typescript
export function useMovies() {
  return useQuery({
    queryKey: ['movies'],
    queryFn: moviesApi.list,
  })
}

export function useMovie(id: number) {
  return useQuery({
    queryKey: ['movies', id],
    queryFn: () => moviesApi.get(id),
    enabled: !!id,
  })
}

export function useCreateMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: moviesApi.create,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['movies'] }),
  })
}
```

### Type Definitions

**File: `src/types/movie.ts`**
```typescript
export interface Movie {
  id: number
  title: string
  year: number
  tmdbId: number
  imdbId?: string
  overview?: string
  posterPath?: string
  backdropPath?: string
  status: 'missing' | 'downloading' | 'available'
  qualityProfileId: number
  rootFolderPath: string
  path?: string
  monitored: boolean
  createdAt: string
  updatedAt: string
}
```

**Similar types for:**
- Series, Season, Episode
- QualityProfile
- RootFolder
- Indexer
- DownloadClient
- QueueItem
- HistoryEntry

---

## 4. Page Implementation Plan

### 4.1 Dashboard (Home Page)

**Route:** `/`
**File:** `src/routes/index.tsx`

**Features:**
- System status card (version, uptime, counts)
- Library summary (movies count, series count, episodes)
- Recent activity feed (last 10 history items)
- Download queue preview (active downloads)
- Quick action buttons (Add Movie, Add Series)

**Components needed:**
- `StatusCard` - System health/version display
- `LibrarySummary` - Movie/series counts with links
- `ActivityFeed` - Recent history items
- `QueuePreview` - Active downloads mini-list

**API calls:**
- `GET /api/v1/status`
- `GET /api/v1/queue`
- `GET /api/v1/history?limit=10`

---

### 4.2 Movies List

**Route:** `/movies`
**File:** `src/routes/movies/index.tsx`

**Features:**
- Grid/list view toggle (poster grid vs table)
- Filter by status (all, monitored, missing, available)
- Sort by (title, year, date added, status)
- Search/filter input
- Bulk actions (delete selected, refresh all)
- "Add Movie" button → navigates to `/movies/add`

**Components needed:**
- `MovieGrid` - Poster-based grid display
- `MovieTable` - Table with columns
- `MovieCard` - Individual movie poster card
- `MovieFilters` - Filter/sort controls
- `ViewToggle` - Grid/List switch

**API calls:**
- `GET /api/v1/movies`

---

### 4.3 Movie Detail

**Route:** `/movies/:id`
**File:** `src/routes/movies/$id.tsx`

**Features:**
- Hero section with backdrop image
- Movie metadata (title, year, runtime, genres, overview)
- Poster image
- File info (if available) - path, size, quality
- Action buttons:
  - Search (trigger indexer search)
  - Refresh (refresh metadata)
  - Edit (open edit modal)
  - Delete (with confirmation)
- Quality profile badge
- Monitored toggle

**Components needed:**
- `MovieHero` - Backdrop + overlay info
- `MovieInfo` - Metadata display
- `MovieFiles` - File list/info
- `MovieActions` - Action button group
- `EditMovieModal` - Edit form dialog

**API calls:**
- `GET /api/v1/movies/:id`
- `POST /api/v1/movies/:id/search`
- `PUT /api/v1/movies/:id`
- `DELETE /api/v1/movies/:id`

---

### 4.4 Add Movie Wizard

**Route:** `/movies/add`
**File:** `src/routes/movies/add.tsx`

**Features:**
- Step 1: Search input (searches TMDB via metadata API)
- Step 2: Select from search results
- Step 3: Configure options:
  - Root folder (select from available)
  - Quality profile (select from available)
  - Monitored (toggle)
  - Start search immediately (toggle)
- Step 4: Confirm and add

**Components needed:**
- `MetadataSearch` - Search input with debounce
- `SearchResults` - Grid of search result cards
- `SearchResultCard` - Poster + title + year
- `AddMovieForm` - Configuration form

**API calls:**
- `GET /api/v1/metadata/movie/search?query=...`
- `GET /api/v1/rootfolders`
- `GET /api/v1/qualityprofiles`
- `POST /api/v1/movies`

---

### 4.5 Series List

**Route:** `/series`
**File:** `src/routes/series/index.tsx`

**Features:**
- Same as Movies List but for TV series
- Grid/list view toggle
- Filter by status
- Episode count badges
- Network/year info

**Components needed:**
- `SeriesGrid` / `SeriesTable`
- `SeriesCard`
- `SeriesFilters`

**API calls:**
- `GET /api/v1/series`

---

### 4.6 Series Detail

**Route:** `/series/:id`
**File:** `src/routes/series/$id.tsx`

**Features:**
- Hero section with backdrop
- Series metadata (title, year, network, status, overview)
- Season accordion/tabs:
  - Each season expandable
  - Episode list within season
  - Episode status (missing/available)
  - Episode file info
- Action buttons (Search All, Refresh, Edit, Delete)
- Season monitoring toggles
- Episode-level actions (search individual episode)

**Components needed:**
- `SeriesHero`
- `SeriesInfo`
- `SeasonList` - Collapsible season sections
- `SeasonCard` - Season header with stats
- `EpisodeTable` - Episodes within season
- `EpisodeRow` - Single episode with status/actions
- `EditSeriesModal`

**API calls:**
- `GET /api/v1/series/:id` (includes seasons/episodes)
- `POST /api/v1/series/:id/search`
- `PUT /api/v1/series/:id`
- `DELETE /api/v1/series/:id`

---

### 4.7 Add Series Wizard

**Route:** `/series/add`
**File:** `src/routes/series/add.tsx`

**Features:**
- Same flow as Add Movie
- Search TVDB/TMDB for series
- Select series from results
- Configure root folder, quality profile, monitoring
- Season selection (which seasons to monitor)

**Components needed:**
- Reuse `MetadataSearch`, `SearchResults`
- `SeriesSearchResultCard`
- `AddSeriesForm`
- `SeasonSelector` - Multi-select for seasons

**API calls:**
- `GET /api/v1/metadata/series/search?query=...`
- `POST /api/v1/series`

---

### 4.8 Activity / Queue View

**Route:** `/activity` or `/activity/queue`
**File:** `src/routes/activity/index.tsx`

**Features:**
- Active downloads table:
  - Title, progress, ETA, speed, status
  - Cancel/remove action per item
- Tabs: Queue | History
- Real-time updates via WebSocket

**Components needed:**
- `QueueTable` - Download queue list
- `QueueItem` - Single queue row with progress bar
- `ProgressBar` - Visual progress indicator

**API calls:**
- `GET /api/v1/queue`
- `DELETE /api/v1/queue/:id`
- WebSocket subscription for updates

---

### 4.9 History View

**Route:** `/activity/history`
**File:** `src/routes/activity/history.tsx`

**Features:**
- Paginated history table
- Filter by event type (grabbed, imported, deleted, failed)
- Filter by date range
- Search by title
- Event details on click/expand

**Components needed:**
- `HistoryTable` - Paginated table
- `HistoryRow` - Single history entry
- `HistoryFilters` - Event type, date filters
- `Pagination` - Page controls

**API calls:**
- `GET /api/v1/history?page=1&limit=50&eventType=...`

---

### 4.10 Settings - Quality Profiles

**Route:** `/settings/profiles`
**File:** `src/routes/settings/profiles.tsx`

**Features:**
- List existing profiles
- Create new profile button
- Edit profile modal:
  - Name
  - Quality tiers (drag to reorder)
  - Cutoff quality
  - Upgrade allowed toggle
- Delete profile (with usage check)

**Components needed:**
- `ProfileList` - Cards for each profile
- `ProfileCard` - Profile summary
- `ProfileEditor` - Modal with quality config
- `QualitySelector` - Multi-select quality tiers
- `DraggableList` - Reorder quality priority

**API calls:**
- `GET /api/v1/qualityprofiles`
- `POST /api/v1/qualityprofiles`
- `PUT /api/v1/qualityprofiles/:id`
- `DELETE /api/v1/qualityprofiles/:id`

---

### 4.11 Settings - Root Folders

**Route:** `/settings/rootfolders`
**File:** `src/routes/settings/rootfolders.tsx`

**Features:**
- List root folders with free space
- Add new root folder (path input + browse?)
- Delete root folder (with usage warning)

**Components needed:**
- `RootFolderList`
- `RootFolderCard` - Path + free space
- `AddRootFolderModal`

**API calls:**
- `GET /api/v1/rootfolders`
- `POST /api/v1/rootfolders`
- `DELETE /api/v1/rootfolders/:id`

---

### 4.12 Settings - Indexers

**Route:** `/settings/indexers`
**File:** `src/routes/settings/indexers.tsx`

**Features:**
- List configured indexers
- Add indexer (Torznab/Newznab form)
- Edit indexer
- Test connection button
- Enable/disable toggle
- Delete indexer

**Components needed:**
- `IndexerList`
- `IndexerCard`
- `IndexerForm` - Add/edit form with URL, API key, categories
- `TestConnectionButton`

**API calls:**
- `GET /api/v1/indexers`
- `POST /api/v1/indexers`
- `PUT /api/v1/indexers/:id`
- `DELETE /api/v1/indexers/:id`
- `POST /api/v1/indexers/:id/test`

---

### 4.13 Settings - Download Clients

**Route:** `/settings/downloadclients`
**File:** `src/routes/settings/downloadclients.tsx`

**Features:**
- List configured download clients
- Add client (qBittorrent, Transmission, SABnzbd)
- Dynamic form based on client type
- Test connection
- Priority ordering

**Components needed:**
- `DownloadClientList`
- `DownloadClientCard`
- `DownloadClientForm` - Type-specific fields
- `ClientTypeSelector`

**API calls:**
- `GET /api/v1/downloadclients`
- `POST /api/v1/downloadclients`
- `PUT /api/v1/downloadclients/:id`
- `DELETE /api/v1/downloadclients/:id`
- `POST /api/v1/downloadclients/:id/test`

---

### 4.14 Settings - General

**Route:** `/settings/general`
**File:** `src/routes/settings/general.tsx`

**Features:**
- Application settings form:
  - Port number
  - Authentication (enable/disable, password)
  - API key display/regenerate
  - Log level
  - Update notifications

**Components needed:**
- `GeneralSettingsForm`

**API calls:**
- `GET /api/v1/settings`
- `PUT /api/v1/settings`

---

## 5. Shared Components

### Layout Components

| Component | File | Purpose |
|-----------|------|---------|
| `RootLayout` | `src/components/layout/RootLayout.tsx` | Main app shell |
| `Sidebar` | `src/components/layout/Sidebar.tsx` | Navigation sidebar |
| `Header` | `src/components/layout/Header.tsx` | Top bar |
| `PageHeader` | `src/components/layout/PageHeader.tsx` | Page title + actions |

### Data Display

| Component | File | Purpose |
|-----------|------|---------|
| `DataTable` | `src/components/data/DataTable.tsx` | Generic table with sorting/pagination |
| `EmptyState` | `src/components/data/EmptyState.tsx` | "No data" placeholder |
| `LoadingState` | `src/components/data/LoadingState.tsx` | Loading skeleton |
| `ErrorState` | `src/components/data/ErrorState.tsx` | Error display with retry |
| `Pagination` | `src/components/data/Pagination.tsx` | Page navigation |

### Media Components

| Component | File | Purpose |
|-----------|------|---------|
| `PosterImage` | `src/components/media/PosterImage.tsx` | Poster with fallback |
| `BackdropImage` | `src/components/media/BackdropImage.tsx` | Backdrop with gradient overlay |
| `StatusBadge` | `src/components/media/StatusBadge.tsx` | Missing/Available/Downloading |
| `QualityBadge` | `src/components/media/QualityBadge.tsx` | Quality tier display |
| `ProgressBar` | `src/components/media/ProgressBar.tsx` | Download progress |

### Form Components

| Component | File | Purpose |
|-----------|------|---------|
| `SearchInput` | `src/components/forms/SearchInput.tsx` | Debounced search field |
| `ConfirmDialog` | `src/components/forms/ConfirmDialog.tsx` | Confirmation modal |
| `FormField` | `src/components/forms/FormField.tsx` | Label + input + error |

### Existing shadcn/ui Components

Already available in `src/components/ui/`:
- `Button`, `Card`, `Input`, `Label`, `Badge`
- `Select`, `Dropdown`, `Combobox`
- `AlertDialog`, `Separator`, `Textarea`

### Additional shadcn Components to Add

```bash
# Run these to add more shadcn components as needed
npx shadcn@latest add dialog
npx shadcn@latest add tabs
npx shadcn@latest add table
npx shadcn@latest add skeleton
npx shadcn@latest add toast
npx shadcn@latest add switch
npx shadcn@latest add checkbox
npx shadcn@latest add slider
npx shadcn@latest add progress
npx shadcn@latest add collapsible
npx shadcn@latest add accordion
```

---

## 6. File Structure

```
web/src/
├── main.tsx                          # Entry point
├── App.tsx                           # (Remove - use router)
├── index.css                         # Global styles
├── router.tsx                        # TanStack Router config
├── vite-env.d.ts
│
├── api/                              # API client modules
│   ├── client.ts                     # Base fetch wrapper
│   ├── movies.ts
│   ├── series.ts
│   ├── qualityProfiles.ts
│   ├── rootFolders.ts
│   ├── indexers.ts
│   ├── downloadClients.ts
│   ├── queue.ts
│   ├── history.ts
│   ├── metadata.ts
│   └── system.ts
│
├── hooks/                            # React Query hooks
│   ├── useMovies.ts
│   ├── useSeries.ts
│   ├── useQualityProfiles.ts
│   ├── useRootFolders.ts
│   ├── useIndexers.ts
│   ├── useDownloadClients.ts
│   ├── useQueue.ts
│   ├── useHistory.ts
│   ├── useMetadata.ts
│   ├── useStatus.ts
│   └── useWebSocket.ts
│
├── stores/                           # Zustand stores
│   ├── ui.ts
│   └── websocket.ts
│
├── types/                            # TypeScript types
│   ├── movie.ts
│   ├── series.ts
│   ├── qualityProfile.ts
│   ├── rootFolder.ts
│   ├── indexer.ts
│   ├── downloadClient.ts
│   ├── queue.ts
│   ├── history.ts
│   └── api.ts                        # Common API types
│
├── lib/                              # Utilities
│   ├── utils.ts                      # (existing)
│   ├── constants.ts                  # App constants
│   └── formatters.ts                 # Date, size formatters
│
├── components/
│   ├── ui/                           # shadcn primitives (existing)
│   │
│   ├── layout/                       # Layout components
│   │   ├── RootLayout.tsx
│   │   ├── Sidebar.tsx
│   │   ├── Header.tsx
│   │   └── PageHeader.tsx
│   │
│   ├── data/                         # Data display
│   │   ├── DataTable.tsx
│   │   ├── EmptyState.tsx
│   │   ├── LoadingState.tsx
│   │   ├── ErrorState.tsx
│   │   └── Pagination.tsx
│   │
│   ├── media/                        # Media-specific
│   │   ├── PosterImage.tsx
│   │   ├── BackdropImage.tsx
│   │   ├── StatusBadge.tsx
│   │   ├── QualityBadge.tsx
│   │   └── ProgressBar.tsx
│   │
│   ├── forms/                        # Form helpers
│   │   ├── SearchInput.tsx
│   │   ├── ConfirmDialog.tsx
│   │   └── FormField.tsx
│   │
│   ├── movies/                       # Movie-specific
│   │   ├── MovieGrid.tsx
│   │   ├── MovieTable.tsx
│   │   ├── MovieCard.tsx
│   │   ├── MovieHero.tsx
│   │   ├── MovieInfo.tsx
│   │   ├── MovieFilters.tsx
│   │   └── EditMovieModal.tsx
│   │
│   ├── series/                       # Series-specific
│   │   ├── SeriesGrid.tsx
│   │   ├── SeriesCard.tsx
│   │   ├── SeriesHero.tsx
│   │   ├── SeasonList.tsx
│   │   ├── SeasonCard.tsx
│   │   ├── EpisodeTable.tsx
│   │   └── EditSeriesModal.tsx
│   │
│   ├── activity/                     # Activity views
│   │   ├── QueueTable.tsx
│   │   ├── QueueItem.tsx
│   │   ├── HistoryTable.tsx
│   │   └── HistoryFilters.tsx
│   │
│   ├── settings/                     # Settings forms
│   │   ├── ProfileEditor.tsx
│   │   ├── IndexerForm.tsx
│   │   ├── DownloadClientForm.tsx
│   │   └── GeneralSettingsForm.tsx
│   │
│   └── dashboard/                    # Dashboard widgets
│       ├── StatusCard.tsx
│       ├── LibrarySummary.tsx
│       ├── ActivityFeed.tsx
│       └── QueuePreview.tsx
│
└── routes/                           # TanStack Router pages
    ├── __root.tsx                    # Root layout
    ├── index.tsx                     # Dashboard /
    │
    ├── movies/
    │   ├── index.tsx                 # /movies
    │   ├── $id.tsx                   # /movies/:id
    │   └── add.tsx                   # /movies/add
    │
    ├── series/
    │   ├── index.tsx                 # /series
    │   ├── $id.tsx                   # /series/:id
    │   └── add.tsx                   # /series/add
    │
    ├── activity/
    │   ├── index.tsx                 # /activity (queue)
    │   └── history.tsx               # /activity/history
    │
    └── settings/
        ├── index.tsx                 # /settings overview
        ├── profiles.tsx              # /settings/profiles
        ├── rootfolders.tsx           # /settings/rootfolders
        ├── indexers.tsx              # /settings/indexers
        ├── downloadclients.tsx       # /settings/downloadclients
        └── general.tsx               # /settings/general
```

---

## 7. Implementation Order

Recommended order for building out the frontend:

### Phase 1: Foundation (Core Infrastructure)

1. **Router Setup**
   - Configure TanStack Router
   - Create route tree structure
   - Set up root layout

2. **Layout Components**
   - `RootLayout` with sidebar + content area
   - `Sidebar` with navigation links
   - `Header` component
   - `PageHeader` component

3. **API Client Layer**
   - Base fetch wrapper (`api/client.ts`)
   - Type definitions for all entities
   - API modules for each domain

4. **Query Hooks**
   - TanStack Query provider setup
   - Basic hooks for movies, series, status

### Phase 2: Core Pages (Library Views)

5. **Dashboard**
   - Status display
   - Library counts
   - Quick actions

6. **Movies List**
   - Movie grid/table
   - Filtering and sorting
   - Empty/loading states

7. **Movie Detail**
   - Hero section
   - Movie info display
   - Action buttons

8. **Add Movie**
   - Search functionality
   - Result display
   - Add form

9. **Series List + Detail + Add**
   - Mirror movie implementation
   - Season/episode display

### Phase 3: Activity & Settings

10. **Queue View**
    - Queue table
    - Progress display
    - WebSocket integration

11. **History View**
    - History table
    - Filtering

12. **Settings Pages**
    - Quality profiles
    - Root folders
    - Indexers
    - Download clients
    - General settings

### Phase 4: Polish

13. **WebSocket Integration**
    - Real-time updates
    - Notifications

14. **Error Handling**
    - Error boundaries
    - Toast notifications
    - Retry logic

15. **Responsive Design**
    - Mobile sidebar
    - Responsive grids
    - Touch interactions

---

## 8. Design Guidelines

### Visual Style

- **Dark theme by default** (media management apps work well dark)
- Use TMDB/TVDB poster images prominently
- Status indicators with color coding:
  - Green: Available/Complete
  - Yellow: Downloading/In Progress
  - Red: Missing/Error
  - Gray: Unmonitored

### UX Patterns

- **Optimistic updates** for toggle actions (monitoring)
- **Skeleton loaders** during data fetch
- **Toast notifications** for actions (added, deleted, error)
- **Confirmation dialogs** for destructive actions
- **Keyboard shortcuts** for power users (future)

### Accessibility

- Proper ARIA labels on interactive elements
- Focus management in modals
- Color contrast compliance
- Screen reader friendly

---

## Summary

This plan outlines a comprehensive frontend implementation with:
- **14 main pages/views**
- **~50+ components** (shared + page-specific)
- **10 API modules** with corresponding query hooks
- **Clear file organization** for maintainability

The implementation is phased to deliver core functionality first (library views) before moving to settings and polish.

