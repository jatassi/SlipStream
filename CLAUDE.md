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

### Service Layer
Handlers delegate to service structs (e.g., `movies.Service`, `metadata.Service`) which wrap sqlc queries. Services are injected into handlers during server setup.

## Testing Notes

Unit tests are in `*_test.go` files alongside source. Integration tests may use `internal/testutil` helpers. Scanner tests parse media filenames; quality tests validate profile matching logic.

## Developer Mode

SlipStream supports a runtime-toggleable developer mode for testing and debugging features. When enabled:
- Debug buttons are visible in the UI (e.g., download client dialogs)
- Mock metadata providers, indexer, download client, notification, and root folders are automatically created
- Virtual filesystem with pre-populated library content (no actual disk I/O)
- The application uses a separate development database (`slipstream_dev.db`)
- The `/api/v1/status` endpoint includes `developerMode: true`

### Enabling Developer Mode

Click the hammer icon in the header to toggle developer mode on/off. The toggle:
- Switches between production and development databases
- Creates mock services (indexer, download client, notification, metadata providers)
- Creates mock root folders (`/mock/movies`, `/mock/tv`) with virtual filesystem
- Broadcasts state changes via WebSocket to all connected clients

### Virtual Filesystem

Developer mode includes a virtual filesystem for testing without actual disk I/O:
- **Path prefix**: All virtual paths start with `/mock/`
- **Pre-populated content**: Movies and TV shows aligned with mock metadata/indexer
- **Testing vectors**: Mix of available files (for upgrade testing) and missing files (for search/download testing)

Available virtual content:
- Movies with files: The Matrix, Inception, Dune (1080p - upgradable)
- Movies without files: Oppenheimer, Barbie, Dune Part Two (for search testing)
- TV complete: Breaking Bad, Game of Thrones S1-3 (1080p - upgradable)
- TV partial: Stranger Things S4 (5 of 9 eps), Mandalorian S3 missing
- TV missing: The Boys, The Simpsons (for search testing)

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
