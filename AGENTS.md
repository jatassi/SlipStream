# CLAUDE.md
This file provides guidance to coding agents when working with code in this repository.

### Making Changes
When making changes to this file, ALWAYS make the same update to AGENTS.md in the same directory.

## Project Overview

SlipStream is a unified media management system (similar to Sonarr/Radarr) with a Go backend and React frontend. It manages movies and TV shows, integrates with metadata providers (TMDB/TVDB), and supports torrent/usenet download clients. This project is still undergoing initial development meaning that it is not necessary to maintain backward compatibility with existing implementations. Always prioritize neat, clean code over cumbersome backward compatible workarounds.

## Additional Documentation

Various documents detailing specific aspects of the application may be available in the `docs/` directory. When creating new documents, ALWAYS put them in this directory.

## Mandatory Directives
- Avoid frivolous comments. Frivolous comments include describing well-named variables, self explanatory logic, common operations, etc. - **your code should be self-documenting and simple**
- Aside from running automated tests, do not conduct your own testing after implementation unless explicitly instructed. Suggest testing to the user if necessary.
- At the end of each of your messages, put **Restart Backend** if a backend restart is required
- If following an external plan or to-do list, always update it once you finish a task/phase/feature
- Deviation from the plan or the spec requires pausing to ask the user's approval. If approved, update the plan/spec before making code changes
- When a tool use of a frequently used operation fails, think carefully about the root cause, find the issue, and improve the common commands sections of AGENTS.md and CLAUDE.md to avoid the failure in the future
- When completing a major unit of work, check for and remove legacy codepaths that have been replaced by your changes
- **Do NOT commit, tag, or release unless explicitly instructed by the user**

## Common Commands

Important: Do not attempt to start either frontend or backend servers. Assume servers are already running on default ports. Prompt user to start or restart servers as required.

### Development

**Windows:**
```powershell
.\dev.bat             # Run both backend (:8080) and frontend (:3000)
.\dev.ps1             # PowerShell alternative

# Or manually in separate terminals:
go run ./cmd/slipstream        # Backend only
cd web && bun run dev          # Frontend only
```

**Unix/Mac (with Make):**
```bash
make dev              # Run both backend (:8080) and frontend (:3000)
make dev-backend      # Run Go backend only
make dev-frontend     # Run Vite frontend only
```

### Building
```bash
make build            # Build both backend and frontend
make build-backend    # Build Go binary to bin/slipstream
make build-frontend   # Build frontend to web/dist/
```

### Releasing

The release pipeline is triggered via GitHub Actions and produces platform-specific installers.

**Option 1: Tag-based release (recommended for production)**
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Option 2: Manual workflow dispatch (for testing)**
```bash
gh workflow run release.yml -f version=1.0.0
```

**Monitor release progress:**
```bash
gh run watch                    # Watch current run
gh run list --workflow=release.yml --limit=3  # List recent runs
```

**Release artifacts produced:**
- Windows: `.exe` installer, `.zip` portable
- macOS: `.dmg` (amd64 and arm64)
- Linux: `.AppImage` (amd64), `.deb` (amd64/arm64), `.rpm` (amd64/arm64)

**Re-releasing a version:**
```bash
gh release delete v1.0.0 --yes
git push --delete origin v1.0.0
git tag -d v1.0.0
# Then trigger release again
```

### Testing
```bash
make test             # Run all Go tests
make test-verbose     # Run tests with verbose output
make test-unit        # Run scanner, quality, organizer tests
make test-integration # Run movies, tv, api tests
make test-coverage    # Generate coverage report (coverage/coverage.html)

# Run a single test
go test -v -run TestFunctionName ./internal/package/...
```

### Dependencies
```bash
make install          # Install Go and bun dependencies
go mod download       # Go dependencies only
cd web && bun install # Frontend dependencies only
```

### Database
```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate   # Regenerate Go code from SQL queries
```

After modifying `internal/database/queries/*.sql`, run the sqlc generate command to update `internal/database/sqlc/`.

### Frontend
```bash
cd web
bun run dev           # Start dev server
bun run build         # Production build (runs tsc first)
bun run lint          # ESLint
bun run lint:fix      # ESLint with auto-fix
bun run format        # Prettier format all files
bun run format:check  # Prettier check (CI-friendly)
```

### Lint Helper Scripts
```bash
./scripts/lint/summary.sh              # Total errors + top rules
./scripts/lint/file.sh src/path.tsx     # Lint single file (path relative to web/)
./scripts/lint/verify.sh               # Full check: tsc + build + lint count
./scripts/lint/count-rule.sh <rule>    # Count violations for one rule by file
```

### Linting (Go)
```bash
make lint             # Run golangci-lint
make lint-fix         # Run with auto-fix
make lint-verbose     # Run with verbose output
make lint-new         # Lint only new/changed code vs main
```

### Go Lint Helper Scripts
```bash
./scripts/lint/go-summary.sh              # Total errors + top linters (uncapped)
./scripts/lint/go-breakdown.sh            # Full per-linter and per-file breakdown
./scripts/lint/go-breakdown.sh --linter gocritic  # Breakdown + issues for one linter
./scripts/lint/go-file.sh <path>          # Lint single file or package
./scripts/lint/go-verify.sh              # Full check: vet + build + test + lint count
./scripts/lint/go-count-linter.sh <name>  # Count violations for one linter by file
./scripts/lint/go-snapshot.sh save <name>    # Save lint counts snapshot
./scripts/lint/go-snapshot.sh compare <name> # Compare current vs saved snapshot
./scripts/lint/go-snapshot.sh list           # List saved snapshots
./scripts/lint/go-test-affected.sh           # Test only packages with modified Go files
./scripts/lint/go-check-signatures.sh save <name>    # Save exported function signatures
./scripts/lint/go-check-signatures.sh compare <name> # Compare signatures for API regression
```

## Key Patterns

### API Routes
All API endpoints are under `/api/v1`. Route groups: `/auth`, `/movies`, `/series`, `/qualityprofiles`, `/rootfolders`, `/metadata`, `/indexers`, `/downloadclients`, `/queue`, `/history`, `/search`, `/portal`, `/admin`.

### Database
- SQLite with WAL mode
- Migrations via Goose (embedded in binary)
- Queries via sqlc (type-safe generated Go)

### Configuration
Priority: environment variables > `.env` file > config.yaml > defaults
- Config file: `--config` flag or `configs/config.yaml`
- Env file: `configs/.env` or project root `.env`
- Env vars: `SERVER_PORT`, `METADATA_TMDB_API_KEY`, etc.

### Logging
When running via `go run` (development), the log level is automatically set to `debug` regardless of configuration. This is detected by checking if the executable path contains "go-build". Production builds use the configured log level (default: `info`).

### Frontend-Backend Communication
- HTTP API: Vite dev server proxies `/api` to backend
- WebSocket: `/ws` endpoint for real-time library/progress updates
- State: TanStack Query for data fetching, Zustand for local state

### Design System - Media Type Theming

SlipStream uses distinct color palettes to visually differentiate movies (orange) from TV shows (blue). All colors use OKLCH color space for perceptual consistency.

#### Color Palettes

CSS variables are defined in `web/src/index.css` with 11 shades each (50-950):

| Type | Color | Primary Shade | Light Accent |
|------|-------|---------------|--------------|
| Movies | Orange | `--movie-500` | `--movie-400` |
| TV Shows | Blue | `--tv-500` | `--tv-400` |

Use via Tailwind: `text-movie-500`, `bg-tv-400`, `border-movie-600`, etc.

#### When to Use Each Color

**Movie-specific elements:**
```tsx
// Cards, buttons, badges for movie content
className="border-movie-500 text-movie-400 bg-movie-500/10 hover:glow-movie"
```

**TV-specific elements:**
```tsx
// Cards, buttons, badges for TV/series content
className="border-tv-500 text-tv-400 bg-tv-500/10 hover:glow-tv"
```

**Mixed/generic media content:**
```tsx
// Homepage, search results with both types, generic actions
className="bg-media-gradient text-media-gradient glow-media"
```

#### Glow Effects

Glow utilities create visual emphasis on interactive elements:

| Class | Description |
|-------|-------------|
| `glow-movie` / `glow-tv` | Standard 15px blur glow |
| `glow-movie-sm` / `glow-tv-sm` | Subtle 8px blur |
| `glow-movie-lg` / `glow-tv-lg` | Intense layered glow |
| `glow-movie-border` / `glow-tv-border` | Border + glow combo |
| `glow-movie-pulse` / `glow-tv-pulse` | Animated pulsing glow |
| `hover:glow-movie` / `hover:glow-tv` | Glow on hover |
| `glow-media` | Dual-color gradient glow |
| `icon-glow-movie` / `icon-glow-tv` | Drop shadow for SVG icons |

#### Gradient Utilities

For sections featuring both media types:

| Class | Description |
|-------|-------------|
| `bg-media-gradient` | Orange-to-blue background gradient |
| `bg-media-gradient-vibrant` | Brighter variant (400 shades) |
| `bg-media-gradient-muted` | Subdued variant (700 shades) |
| `text-media-gradient` | Gradient text effect |

#### Component Patterns

**Cards (MovieCard, SeriesCard):**
- Default: `border-border` (neutral)
- Hover: `hover:border-movie-500/50 hover:glow-movie-sm` or `hover:border-tv-500/50 hover:glow-tv-sm`
- Selected (edit mode): `border-movie-500 glow-movie` or `border-tv-500 glow-tv`

**Navigation items with theme:**
```tsx
const navItem = { title: 'Movies', href: '/movies', icon: Film, theme: 'movie' }
// Then conditionally apply:
item.theme === 'movie' && 'text-movie-500 hover:bg-movie-500/10'
item.theme === 'tv' && 'text-tv-500 hover:bg-tv-500/10'
```

**Progress bars:**
```tsx
<ProgressBar variant="movie" value={progress} />  // Orange
<ProgressBar variant="tv" value={progress} />     // Blue
<ProgressBar value={progress} />                   // Default (primary)
```

#### Best Practices

1. **Consistency**: Always use movie colors for movie content, TV colors for TV content
2. **Opacity for backgrounds**: Use `/10` or `/15` opacity for subtle backgrounds (e.g., `bg-movie-500/10`)
3. **400 vs 500**: Use 400 shades for text on dark backgrounds, 500 for borders/accents
4. **Gradients for mixed**: Use `media-gradient` utilities when content includes both types
5. **Glows for interaction**: Add glow effects to indicate interactivity and selection states

#### Reference

A live color palette preview is available at `/dev/colors` when developer mode is enabled.

### UI Components (Base UI, NOT Radix)
This project uses **Base UI** (`@base-ui/react`) for shadcn/ui components, NOT Radix UI. This affects how you customize trigger elements:

**WRONG - Radix-style `asChild` does not work:**
```tsx
<TooltipTrigger asChild>
  <Button>Click me</Button>
</TooltipTrigger>
```

**CORRECT - Use `render` prop instead:**
```tsx
<TooltipTrigger render={<Button />}>
  Button content here
</TooltipTrigger>
```

This applies to: `TooltipTrigger`, `DialogTrigger`, `PopoverTrigger`, `DropdownMenuTrigger`, and similar components.

For simple cases where you just need the trigger to be a div or the default element, you can skip the render prop entirely:
```tsx
<TooltipTrigger>
  <div>Content</div>
</TooltipTrigger>
```

### Base UI Select - Displaying Labels Instead of Values

**CRITICAL**: Base UI's `SelectValue` component displays the raw `value` attribute, NOT the display label. When your Select options have different values vs labels (e.g., `value="trust_queue"` but label="Trust Queue"), you MUST manually render the label in the trigger.

**WRONG - Shows raw value like "trust_queue":**
```tsx
const OPTIONS = [
  { value: 'trust_queue', label: 'Trust Queue' },
  { value: 'trust_parse', label: 'Trust Parse' },
]

<Select value={selected} onValueChange={setSelected}>
  <SelectTrigger>
    <SelectValue />  {/* Will show "trust_queue" NOT "Trust Queue" */}
  </SelectTrigger>
  <SelectContent>
    {OPTIONS.map((opt) => (
      <SelectItem key={opt.value} value={opt.value}>
        {opt.label}
      </SelectItem>
    ))}
  </SelectContent>
</Select>
```

**CORRECT - Manually look up and display the label:**
```tsx
<Select value={selected} onValueChange={setSelected}>
  <SelectTrigger>
    {OPTIONS.find((o) => o.value === selected)?.label}
  </SelectTrigger>
  <SelectContent>
    {OPTIONS.map((opt) => (
      <SelectItem key={opt.value} value={opt.value}>
        {opt.label}
      </SelectItem>
    ))}
  </SelectContent>
</Select>
```

**When this matters:**
- Any Select where value differs from display text (snake_case values, numeric IDs, etc.)
- Select options loaded from an API with id/name pairs

**When SelectValue works fine:**
- Simple cases where value equals the display text (e.g., `value="info"` displays as "info")

### React State Synchronization (Avoiding useEffect for State Sync)

**IMPORTANT**: Do NOT use `useEffect` to synchronize state when props change. This triggers the `react-hooks/set-state-in-effect` lint error and causes unnecessary re-renders. Instead, use the React-recommended "render-time state adjustment" pattern.

**WRONG - Using useEffect to sync state:**
```tsx
function MyComponent({ data }) {
  const [formData, setFormData] = useState(null)

  // ❌ BAD: Causes cascading renders and lint errors
  useEffect(() => {
    if (data) {
      setFormData(data)
    }
  }, [data])
}
```

**CORRECT - Render-time state adjustment:**
```tsx
function MyComponent({ data }) {
  const [formData, setFormData] = useState(null)
  const [prevData, setPrevData] = useState(data)

  // ✅ GOOD: Sync state during render when prop changes
  if (data !== prevData) {
    setPrevData(data)
    if (data) {
      setFormData(data)
    }
  }
}
```

**Common use cases for this pattern:**
- Form initialization from async-loaded data
- Resetting dialog/modal state when opened/closed
- Syncing controlled component values with internal state
- Auto-selecting items when data loads

**For media queries and browser APIs, use `useSyncExternalStore`:**
```tsx
function useIsMobile() {
  return useSyncExternalStore(
    (callback) => {
      const mql = window.matchMedia('(max-width: 767px)')
      mql.addEventListener('change', callback)
      return () => mql.removeEventListener('change', callback)
    },
    () => window.innerWidth < 768,
    () => false // SSR fallback
  )
}
```

**For synchronous browser API checks (e.g., feature detection), compute directly:**
```tsx
function usePasskeySupport() {
  // ✅ Sync check - no state needed
  const isSupported = passkeyApi.isSupported()
  return { isSupported, isLoading: false }
}
```

### Frontend Code Patterns

#### Hook Extraction

Large components must separate logic from presentation. Extract all state, queries, mutations, and handlers into a custom hook. The component becomes a thin rendering shell.

- Hook file: `use-<feature>.ts` in the same directory as the component
- Return a flat object of values and stable handler references
- Keep the component under 50 lines — it should only contain JSX and early returns

```tsx
// use-movie-detail.ts — all logic
export function useMovieDetail() {
  const { id } = useParams({ from: '/movies/$id' })
  const movieId = Number.parseInt(id)
  const query = useMovie(movieId)
  const deleteMutation = useDeleteMovie()
  const handleDelete = () => { deleteMutation.mutate(movieId) }
  return { movie: query.data, isLoading: query.isLoading, handleDelete }
}

// movie-detail-page.tsx — rendering only
export function MovieDetailPage() {
  const { movie, isLoading, handleDelete } = useMovieDetail()
  if (isLoading) return <LoadingState />
  return <MovieDetailView movie={movie} onDelete={handleDelete} />
}
```

#### Async Error Handling

Two patterns, applied uniformly:

1. **Mutations** — use `onError` for user-facing feedback:
```tsx
useMutation({
  mutationFn: deleteMovie,
  onSuccess: () => {
    void queryClient.invalidateQueries({ queryKey: movieKeys.all })
  },
  onError: (err) => { toast.error(err.message) },
})
```

2. **Fire-and-forget** — `void` prefix for intentional no-await:
```tsx
void queryClient.invalidateQueries({ queryKey: movieKeys.all })
```

Never use `.catch(() => {})` — it swallows errors silently. Never make `onSuccess` async just to use `await` — TanStack Query doesn't await these callbacks.

#### Conditional Rendering

Priority order: early returns > component map > single ternary. **Never nest ternaries.**

```tsx
// BEST: early returns
if (isLoading) return <LoadingState />
if (isError) return <ErrorState />
if (!data.length) return <EmptyState />
return <Content data={data} />

// GOOD: component map (when inline in JSX)
const view = { loading: <Spinner />, error: <Error /> } as const
return view[status] ?? <Content />

// ACCEPTABLE: single flat ternary
{isEditing ? <EditForm /> : <ReadView />}

// NEVER: nested ternary
{a ? <X /> : b ? <Y /> : <Z />}
```

#### Null Handling

Use `??` (nullish coalescing) by default. Use `||` (logical OR) **only** when you intentionally need falsy coalescing (0, empty string, NaN should trigger the fallback). When keeping `||`, add a comment explaining why.

```tsx
const name = user.displayName ?? 'Anonymous'       // correct: only null/undefined
const port = Number.parseInt(input) || 3000         // intentional ||: NaN and 0 should fallback
```

### Service Layer
Handlers delegate to service structs (e.g., `movies.Service`, `metadata.Service`) which wrap sqlc queries. Services are injected into handlers during server setup.

### Auto Search & Upgrade Pipeline

Auto search finds and grabs releases for missing or upgradable media. Understanding the full flow is critical when modifying this area — bugs here cause repeated erroneous downloads.

**Key files:**
- `internal/autosearch/service.go` — Core search/grab logic, SearchableItem construction
- `internal/autosearch/scheduled.go` — Scheduled task that collects items and dispatches searches
- `internal/indexer/scoring/scorer.go` — Release scoring and quality matching
- `internal/indexer/search/router.go` — Search routing (Prowlarr vs direct indexers)
- `internal/import/pipeline.go` — Post-download import with quality upgrade validation

**Search flow for TV upgrades (the most complex path):**
1. Scheduled task calls `collectUpgradeEpisodes` which groups upgradable episodes by season
2. If all episodes in a season are upgradable → `SearchSeasonUpgrade` (season pack with upgrade checks)
3. `SearchSeasonUpgrade` creates a season pack `SearchableItem` with `HasFile=true` and `CurrentQualityID` set
4. `selectBestRelease` filters: season pack parsing → quality acceptable check → **upgrade check** (skips if `!HasFile`)
5. If no season pack upgrade found → falls back to individual episodes via `SearchEpisode`
6. `SearchEpisode` has an E01 fallback that retries as a season pack for missing episodes only (`!item.HasFile`)

**Critical invariant — `SearchableItem` must carry file info:**
When an episode/movie has a file, the `SearchableItem` MUST have `HasFile=true` and `CurrentQualityID` set to the highest quality across all file records. Without this, `selectBestRelease` skips the upgrade check entirely and grabs any matching release.

**Common pitfalls:**
- `seriesToSeasonPackItem` does NOT set `HasFile`/`CurrentQualityID` — callers must set these explicitly for upgrade scenarios (see `SearchSeasonUpgrade` lines 285-286)
- Duplicate file records can exist from re-imports — always use the highest `quality_id` across all files, never just `files[0]`
- The E01 season pack fallback in `SearchEpisode` must NOT run for upgradable episodes — the season pack search was already attempted by `SearchSeasonUpgrade` with proper guards
- `selectBestRelease` only checks upgrades when `item.HasFile` is true — if file info is lost when constructing a `SearchableItem`, the upgrade check is silently skipped

**Quality system basics:**
- Each quality has a unique ID and weight (e.g., WEBDL-2160p = ID 15, weight 15)
- `profile.IsUpgrade(currentID, releaseID)` checks if the release quality is strictly better
- `profile.IsAcceptable(qualityID)` checks if a quality is allowed in the profile
- Profile cutoff determines the "upgradable" vs "available" boundary

## Testing Notes

Unit tests are in `*_test.go` files alongside source. Integration tests may use `internal/testutil` helpers. Scanner tests parse media filenames; quality tests validate profile matching logic.

## Developer Mode

SlipStream supports a runtime-toggleable developer mode for testing and debugging features. When enabled:
- Debug buttons are visible in the UI (e.g., download client dialogs)
- Mock metadata providers, indexer, download client, and notification are automatically created
- The application uses a separate development database (`slipstream_dev.db`)
- The `/api/v1/status` endpoint includes `developerMode: true`

### Enabling Developer Mode

Click the hammer icon in the header to toggle developer mode on/off. The toggle:
- Switches between production and development databases
- Creates mock services (indexer, download client, notification, metadata providers)
- Broadcasts state changes via WebSocket to all connected clients

### Backend Check

Use `dbManager.IsDevMode()` to check developer mode status:
```go
if s.dbManager.IsDevMode() {
    // Developer mode is enabled
}
```

### Frontend Hook

Use the `useDeveloperMode()` hook to check developer mode status in React components:
```typescript
import { useDeveloperMode } from '@/hooks'

function MyComponent() {
  const developerMode = useDeveloperMode()

  if (developerMode) {
    // Show debug features
  }
}
```

### Best Practices

- Always check developer mode before exposing debug/testing features
- Mock data should use realistic values from actual media content
- Debug endpoints should return 403 Forbidden when not in developer mode

## External Requests Portal

SlipStream includes an external request portal that allows invited users (friends/family) to request content without having full admin access.

### Architecture

The portal uses a separate authentication system from the main admin interface:
- **Admin auth**: Session-based, stored in cookies, for admin users managing the server
- **Portal auth**: JWT-based, stored in localStorage, for portal users making requests

### API Routes

- `/api/v1/portal/*` - Portal user endpoints (JWT auth required)
  - Authentication: `/login`, `/register`, `/logout`, `/refresh`
  - Requests: `/requests` (CRUD), `/search` (metadata search)
  - Profile: `/profile`, `/notifications`
- `/api/v1/admin/*` - Admin management endpoints (session auth required)
  - Users: `/users` (list, update, enable/disable, delete)
  - Invitations: `/invitations` (create, list, resend, delete)
  - Requests: `/requests` (approve, deny, batch operations)
  - Settings: `/requests/settings` (quotas, rate limits)

### Frontend Routes

Portal and admin interfaces are separate:
- `/portal/*` - Portal user interface (login, search, requests, profile)
- `/settings/requests/*` - Admin interface (queue, users, settings)

### Key Concepts

- **Invitations**: Admins create invitation tokens; users register via `/portal/signup?token=xxx`
- **Quotas**: Per-user limits on movies/seasons/episodes per week (configurable per user or global defaults)
- **Auto-Approve**: Quality profiles can be marked as "Allow Auto-Approve". Portal users assigned such a profile will have their requests auto-approved
- **Request Lifecycle**: `pending` → `approved` → `downloading` → `available` (or `denied`/`cancelled`)

### Quality Profile Auto-Approve

Quality profiles have an `allowAutoApprove` field. When a portal user has a quality profile with this enabled:
1. Their requests skip the approval queue
2. Requests go directly to "approved" status
3. Auto-search begins immediately (based on request settings)

This is useful for trusted users who should have immediate access without admin review.

## Windows-Specific Notes

When running bash commands on Windows, use forward slashes for paths:
```bash
cd c:/Git/SlipStream/web && bun run build 2>&1
```
