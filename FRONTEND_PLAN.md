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

**shadcn/ui Components:**
- `Card` - Container for each dashboard section
- `Badge` - Status indicators (healthy, warning)
- `Button` - Quick action buttons
- `Skeleton` - Loading states for async data
- `@magic-ui/bento-grid` - Dashboard layout
- `@magic-ui/animated-number` - Animated statistics counters
- `@origin-ui/stat-card` - Enhanced stat displays
- `@origin-ui/metric-card` - KPI metrics

**Custom Components:**
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

**shadcn/ui Components:**
- `Data Table` - Advanced table with sorting, filtering, pagination
- `Card` - Movie poster cards in grid view
- `Input` - Search input field
- `Select` - Sort and filter dropdowns
- `Toggle Group` - Grid/List view switcher
- `Button` - Action buttons (Add Movie, Bulk Actions)
- `Checkbox` - Multi-select for bulk actions
- `Badge` - Status indicators (Missing, Available, Downloading)
- `Dropdown Menu` - Bulk actions menu
- `Skeleton` - Loading placeholders
- `Pagination` - Page navigation

**Custom Components:**
- `MovieGrid` - Poster-based grid display
- `MovieCard` - Individual movie poster card (using `Card`)
- `MovieFilters` - Filter/sort controls

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

**shadcn/ui Components:**
- `Card` - Content sections
- `Badge` - Genre tags, quality profile, status
- `Button` - Action buttons
- `Dialog` - Edit movie modal
- `Alert Dialog` - Delete confirmation
- `Switch` - Monitored toggle
- `Separator` - Visual dividers
- `Tabs` - Organize file info, metadata sections
- `Table` - File information display
- `@aceternity/background-gradient` - Hero section background
- `@aceternity/3d-card` - Poster display with 3D effect

**Custom Components:**
- `MovieHero` - Backdrop + overlay info
- `MovieInfo` - Metadata display
- `MovieFiles` - File list/info
- `MovieActions` - Action button group
- `EditMovieModal` - Edit form dialog (wraps `Dialog`)

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

**shadcn/ui Components:**
- `Input` - Search field with debounce
- `Card` - Search result cards
- `Dialog` or `Sheet` - Wizard container
- `Form` - Configuration form with validation
- `Select` - Root folder and quality profile dropdowns
- `Switch` - Toggle options (monitored, auto-search)
- `Button` - Navigation buttons (Next, Back, Add)
- `Badge` - Year, status badges
- `Skeleton` - Loading states for search results
- `Command` - Alternative search interface with keyboard nav

**Custom Components:**
- `MetadataSearch` - Search input with debounce
- `SearchResults` - Grid of search result cards
- `SearchResultCard` - Poster + title + year
- `AddMovieForm` - Configuration form (wraps `Form`)

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

**shadcn/ui Components:**
- `Accordion` - Collapsible season sections (perfect use case!)
- `Tabs` - Alternative layout for seasons
- `Card` - Season cards and content sections
- `Table` or `Data Table` - Episode list with columns
- `Badge` - Episode status, network, quality
- `Button` - Action buttons
- `Dialog` - Edit series modal
- `Alert Dialog` - Delete confirmation
- `Switch` - Season monitoring toggles
- `Separator` - Visual dividers
- `Scroll Area` - Long episode lists
- `Collapsible` - Individual episode details
- `@aceternity/hero-parallax` - Hero section with parallax effect

**Custom Components:**
- `SeriesHero`
- `SeriesInfo`
- `SeasonList` - Wraps `Accordion`
- `SeasonCard` - Season header with stats
- `EpisodeTable` - Episodes within season (wraps `Table`)
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

**shadcn/ui Components:**
- `Tabs` - Switch between Queue and History
- `Data Table` or `Table` - Queue items list
- `Progress` - Download progress bars
- `Badge` - Status indicators (downloading, paused, completed)
- `Button` - Cancel/remove actions
- `Alert Dialog` - Confirm removal
- `Card` - Container for queue section
- `Skeleton` - Loading states

**Custom Components:**
- `QueueTable` - Download queue list (wraps `Data Table`)
- `QueueItem` - Single queue row with progress bar
- `ProgressBar` - Custom progress indicator (or use `Progress`)

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

**shadcn/ui Components:**
- `Data Table` - Paginated history with sorting and filtering
- `Select` - Event type filter
- `Date Picker` - Date range filter
- `Input` - Search input
- `Pagination` - Page navigation
- `Badge` - Event type badges
- `Collapsible` - Expandable event details
- `Card` - Container
- `@origin-ui/timeline` - Alternative timeline view for history

**Custom Components:**
- `HistoryTable` - Paginated table (wraps `Data Table`)
- `HistoryRow` - Single history entry
- `HistoryFilters` - Event type, date filters

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

**shadcn/ui Components:**
- `Card` - Profile cards
- `Button` - Add/Edit/Delete buttons
- `Dialog` - Edit profile modal
- `Alert Dialog` - Delete confirmation
- `Form` - Profile configuration form
- `Input` - Profile name input
- `Select` - Cutoff quality dropdown
- `Switch` - Upgrade allowed toggle
- `Checkbox` - Quality tier multi-select
- `Badge` - Quality tier badges
- `Separator` - Visual dividers

**Custom Components:**
- `ProfileList` - Cards for each profile
- `ProfileCard` - Profile summary
- `ProfileEditor` - Modal with quality config (wraps `Dialog`)
- `QualitySelector` - Multi-select quality tiers
- `DraggableList` - Reorder quality priority (implement with dnd-kit or similar)

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

**shadcn/ui Components:**
- `Card` - Indexer cards
- `Button` - Add/Edit/Delete/Test buttons
- `Dialog` - Add/edit indexer modal
- `Alert Dialog` - Delete confirmation
- `Form` - Indexer configuration form
- `Input` - URL, API key inputs
- `Select` - Indexer type dropdown
- `Switch` - Enable/disable toggle
- `Badge` - Status indicator (connected, error)
- `Alert` - Test results display
- `Spinner` - Testing state

**Custom Components:**
- `IndexerList`
- `IndexerCard`
- `IndexerForm` - Add/edit form with URL, API key, categories (wraps `Form`)
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

| Component | File | Purpose | shadcn Components Used |
|-----------|------|---------|------------------------|
| `RootLayout` | `src/components/layout/RootLayout.tsx` | Main app shell | `Sidebar` (v4), `Separator` |
| `Sidebar` | `src/components/layout/Sidebar.tsx` | Navigation sidebar | `Sidebar` component from shadcn/ui |
| `Header` | `src/components/layout/Header.tsx` | Top bar | `Input`, `Command`, `Dropdown Menu`, `Avatar` |
| `PageHeader` | `src/components/layout/PageHeader.tsx` | Page title + actions | `Breadcrumb`, `Button` |

### Data Display

| Component | File | Purpose | shadcn Components Used |
|-----------|------|---------|------------------------|
| `DataTable` | `src/components/data/DataTable.tsx` | Generic table with sorting/pagination | `Data Table`, `Table`, `Checkbox` |
| `EmptyState` | `src/components/data/EmptyState.tsx` | "No data" placeholder | `Card`, custom illustration |
| `LoadingState` | `src/components/data/LoadingState.tsx` | Loading skeleton | `Skeleton`, `Card` |
| `ErrorState` | `src/components/data/ErrorState.tsx` | Error display with retry | `Alert`, `Button` |
| `Pagination` | `src/components/data/Pagination.tsx` | Page navigation | `Pagination` component |

### Media Components

| Component | File | Purpose | shadcn Components Used |
|-----------|------|---------|------------------------|
| `PosterImage` | `src/components/media/PosterImage.tsx` | Poster with fallback | `@aceternity/3d-card`, `Card` |
| `BackdropImage` | `src/components/media/BackdropImage.tsx` | Backdrop with gradient overlay | `@aceternity/background-gradient` |
| `StatusBadge` | `src/components/media/StatusBadge.tsx` | Missing/Available/Downloading | `Badge` |
| `QualityBadge` | `src/components/media/QualityBadge.tsx` | Quality tier display | `Badge` |
| `ProgressBar` | `src/components/media/ProgressBar.tsx` | Download progress | `Progress` |

### Form Components

| Component | File | Purpose | shadcn Components Used |
|-----------|------|---------|------------------------|
| `SearchInput` | `src/components/forms/SearchInput.tsx` | Debounced search field | `Input`, `Command` |
| `ConfirmDialog` | `src/components/forms/ConfirmDialog.tsx` | Confirmation modal | `Alert Dialog` |
| `FormField` | `src/components/forms/FormField.tsx` | Label + input + error | `Form`, `Label`, `Input` |

### Existing shadcn/ui Components

Already available in `src/components/ui/`:
- `Button`, `Card`, `Input`, `Label`, `Badge`
- `Select`, `Dropdown`, `Combobox`
- `AlertDialog`, `Separator`, `Textarea`

### shadcn/ui Components to Add

Based on the requirements for a media management application, install these components from the official [shadcn/ui registry](https://ui.shadcn.com/docs/components):

#### Essential Data Display Components
```bash
# Data tables with sorting, filtering, and pagination
npx shadcn@latest add data-table

# Basic table for simpler displays
npx shadcn@latest add table

# Skeleton loaders for async content
npx shadcn@latest add skeleton

# Avatar for user profiles
npx shadcn@latest add avatar

# Charts for dashboard analytics
npx shadcn@latest add chart
```

#### Navigation & Layout Components
```bash
# Tabs for organizing content (series seasons, activity views)
npx shadcn@latest add tabs

# Accordion for collapsible sections (episode lists)
npx shadcn@latest add accordion

# Collapsible for expandable content
npx shadcn@latest add collapsible

# Breadcrumb for navigation context
npx shadcn@latest add breadcrumb

# Sidebar for main navigation (v4 component)
npx shadcn@latest add sidebar

# Scroll area for long content lists
npx shadcn@latest add scroll-area

# Resizable panels for advanced layouts
npx shadcn@latest add resizable
```

#### Modal & Overlay Components
```bash
# Dialog for modals (edit forms, confirmations)
npx shadcn@latest add dialog

# Drawer for side panels
npx shadcn@latest add drawer

# Sheet for overlays
npx shadcn@latest add sheet

# Popover for contextual information
npx shadcn@latest add popover

# Hover card for preview on hover
npx shadcn@latest add hover-card

# Context menu for right-click actions
npx shadcn@latest add context-menu

# Dropdown menu for action menus
npx shadcn@latest add dropdown-menu
```

#### Form Components
```bash
# Form primitives with validation
npx shadcn@latest add form

# Switch for toggle options (monitoring)
npx shadcn@latest add switch

# Checkbox for multi-select
npx shadcn@latest add checkbox

# Radio group for single selection
npx shadcn@latest add radio-group

# Slider for quality/priority settings
npx shadcn@latest add slider

# Calendar and date picker for scheduling
npx shadcn@latest add calendar
npx shadcn@latest add date-picker

# Input OTP for auth (if needed)
npx shadcn@latest add input-otp
```

#### Feedback & Status Components
```bash
# Toast notifications for actions
npx shadcn@latest add toast
npx shadcn@latest add sonner  # Alternative toast library

# Progress bars for downloads
npx shadcn@latest add progress

# Spinner for loading states
npx shadcn@latest add spinner

# Alert for important messages
npx shadcn@latest add alert
```

#### Advanced Components
```bash
# Carousel for poster galleries
npx shadcn@latest add carousel

# Pagination for long lists
npx shadcn@latest add pagination

# Command palette for quick actions
npx shadcn@latest add command

# Toggle group for view modes
npx shadcn@latest add toggle-group
npx shadcn@latest add toggle
```

### Community Registry Components

Enhance the UI with components from popular shadcn/ui community registries:

#### Origin UI (@origin-ui) - 400+ Advanced Components
[Origin UI](https://originui.com/) provides enterprise-grade components across 25+ categories.

**Recommended components for SlipStream:**
```bash
# Timeline for activity feed/history view
npx shadcn@latest add @origin-ui/timeline

# Advanced data display components
npx shadcn@latest add @origin-ui/stat-card
npx shadcn@latest add @origin-ui/metric-card

# Enhanced dialogs with better UX
npx shadcn@latest add @origin-ui/modal

# File upload for manual imports
npx shadcn@latest add @origin-ui/file-upload
```

#### Magic UI (@magic-ui) - 50+ Animated Components
[Magic UI](https://magicui.design/) specializes in beautiful animations using Framer Motion.

**Recommended for visual polish:**
```bash
# Animated number counters for dashboard stats
npx shadcn@latest add @magic-ui/animated-number

# Animated background effects for hero sections
npx shadcn@latest add @magic-ui/particles

# Marquee for scrolling announcements
npx shadcn@latest add @magic-ui/marquee

# Bento grid for dashboard layout
npx shadcn@latest add @magic-ui/bento-grid

# Animated beam for connecting elements
npx shadcn@latest add @magic-ui/animated-beam
```

#### Aceternity UI (@aceternity) - Modern Animated Components
[Aceternity UI](https://ui.aceternity.com/) offers highly interactive, modern components.

**Recommended for media-rich pages:**
```bash
# 3D card effect for movie/series posters
npx shadcn@latest add @aceternity/3d-card

# Hero sections with parallax effects
npx shadcn@latest add @aceternity/hero-parallax

# Background gradients for visual depth
npx shadcn@latest add @aceternity/background-gradient

# Animated spotlight for featured content
npx shadcn@latest add @aceternity/spotlight
```

### Additional Registries to Explore

- **[registry.directory](https://registry.directory/)** - Curated collection of 88+ shadcn/ui registries
- **[Tremor](https://tremor.so/)** - Specialized dashboard and analytics components (charts, KPIs)
- **[Lukacho UI](https://ui.lukacho.com/)** - Modern components with unique designs
- **[Shadcn Studio](https://shadcnstudio.com/)** - Component variants and customizations

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

## 9. Quick Start: Component Installation

### Install All Essential shadcn/ui Components

Run this command to install all core components at once:

```bash
cd web && npx shadcn@latest add \
  data-table table skeleton avatar chart \
  tabs accordion collapsible breadcrumb sidebar scroll-area resizable \
  dialog drawer sheet popover hover-card context-menu dropdown-menu \
  form switch checkbox radio-group slider calendar date-picker \
  toast sonner progress spinner alert \
  carousel pagination command toggle-group toggle
```

### Install Community Registry Components

After setting up the base components, enhance with community components:

```bash
# Origin UI components (verify namespace with Origin UI docs)
npx shadcn@latest add @origin-ui/timeline @origin-ui/stat-card @origin-ui/metric-card

# Magic UI components (verify namespace with Magic UI docs)
npx shadcn@latest add @magic-ui/animated-number @magic-ui/bento-grid

# Aceternity UI components (verify namespace with Aceternity docs)
npx shadcn@latest add @aceternity/3d-card @aceternity/background-gradient
```

**Note:** Community registry namespaces may vary. Check the official documentation for each registry for exact installation commands.

---

## 10. Implementation Tips

### Component Usage Patterns

1. **Data Tables** - Use `Data Table` for complex lists with sorting/filtering, and plain `Table` for simple displays
2. **Modals** - Use `Dialog` for forms/content, `Sheet` for side panels, `Alert Dialog` for confirmations
3. **Loading States** - Always provide `Skeleton` loaders during data fetching for better UX
4. **Form Validation** - Leverage `Form` component with React Hook Form + Zod integration
5. **Notifications** - Use `Sonner` (modern toast library) instead of basic `Toast` for better animations

### Accessibility Checklist

- ✅ All interactive elements have proper ARIA labels
- ✅ Keyboard navigation works throughout (Tab, Enter, Escape)
- ✅ Focus management in modals and dialogs
- ✅ Color contrast meets WCAG 2.1 AA standards
- ✅ Screen reader friendly component structure

### Performance Optimization

- Use `React.lazy()` for code-splitting large pages
- Implement virtual scrolling for very long lists (react-window or @tanstack/react-virtual)
- Optimize images with proper sizing and lazy loading
- Use TanStack Query's built-in caching to minimize API calls
- Debounce search inputs (300ms recommended)

### Design System Consistency

**Color Coding for Status:**
- 🟢 Green: Available, Complete, Success
- 🟡 Yellow: Downloading, In Progress, Warning
- 🔴 Red: Missing, Error, Failed
- ⚪ Gray: Unmonitored, Disabled, Neutral

**Component Sizes:**
- Buttons: Use `size="default"` for primary actions, `size="sm"` for compact areas
- Inputs: Match button sizes for visual alignment
- Cards: Use consistent padding (p-6 for content, p-4 for compact)

### Community Registry Best Practices

**When to use community components:**
- ✅ Use for unique features not in core shadcn/ui
- ✅ Use for enhanced visual polish (animations, gradients)
- ✅ Use for specialized layouts (bento grids, timelines)
- ❌ Avoid overusing animations (can hurt performance)
- ❌ Don't mix too many different style systems

**Recommended approach:**
1. Build core functionality with official shadcn/ui components
2. Add community components selectively for key visual areas (dashboard, detail pages)
3. Test performance impact before committing to animated components
4. Keep community components in separate directories for easy replacement

---

## Summary

This plan outlines a comprehensive frontend implementation with:
- **14 main pages/views**
- **~50+ shadcn/ui components** from official and community registries
- **~50+ custom components** (shared + page-specific)
- **10 API modules** with corresponding query hooks
- **Clear file organization** for maintainability
- **Specific component recommendations** for each page

### Component Breakdown:
- **Official shadcn/ui**: 40+ core components
- **Origin UI**: Timeline, stat cards, metric cards
- **Magic UI**: Animated numbers, bento grid
- **Aceternity UI**: 3D cards, background gradients, hero parallax

The implementation is phased to deliver core functionality first (library views) before moving to settings and polish.

### Key References:
- [shadcn/ui Components](https://ui.shadcn.com/docs/components) - Official component library
- [Data Table Documentation](https://ui.shadcn.com/docs/components/data-table) - TanStack Table integration
- [registry.directory](https://registry.directory/) - Community registry collection
- [Origin UI](https://originui.com/) - 400+ enterprise components
- [Magic UI](https://magicui.design/) - Animated components
- [Aceternity UI](https://ui.aceternity.com/) - Modern interactive components

