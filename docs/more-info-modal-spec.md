# More Info Modal Implementation Plan

**Status: COMPLETED**

## Summary
Add a "More Info" button to search result cards that opens a modal displaying extended metadata: credits, ratings (Rotten Tomatoes, IMDB), awards, and season/episode lists for TV.

## Data Sources
- **TMDB** (existing): Credits (cast/crew with photos), content ratings, production companies
- **OMDb** (new): Rotten Tomatoes scores, IMDB rating, awards text

---

## Phase 1: TMDB Credits & Content Ratings

### 1.1 Add Models (`internal/metadata/tmdb/models.go`)
```go
// Request/Response types
type CreditsResponse struct {
    ID   int          `json:"id"`
    Cast []CastMember `json:"cast"`
    Crew []CrewMember `json:"crew"`
}

type CastMember struct {
    ID, Order       int
    Name, Character string
    ProfilePath     *string `json:"profile_path"`
}

type CrewMember struct {
    ID                        int
    Name, Job, Department     string
    ProfilePath               *string `json:"profile_path"`
}

// Normalized output
type NormalizedCredits struct {
    Directors   []NormalizedPerson `json:"directors,omitempty"`
    Writers     []NormalizedPerson `json:"writers,omitempty"`
    Creators    []NormalizedPerson `json:"creators,omitempty"` // TV showrunners
    Cast        []NormalizedPerson `json:"cast"`
}

type NormalizedPerson struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Role     string `json:"role,omitempty"`
    PhotoURL string `json:"photoUrl,omitempty"`
}
```

### 1.2 Add Client Methods (`internal/metadata/tmdb/client.go`)
- `GetMovieCredits(ctx, id int) (*NormalizedCredits, error)` - `/movie/{id}/credits`
- `GetSeriesCredits(ctx, id int) (*NormalizedCredits, error)` - `/tv/{id}/credits` + `/tv/{id}` for creators
- `GetMovieContentRating(ctx, id int) (string, error)` - `/movie/{id}/release_dates` (US certification)
- `GetSeriesContentRating(ctx, id int) (string, error)` - `/tv/{id}/content_ratings` (US rating)

### 1.3 Update Interface (`internal/metadata/interfaces.go`)
Add new methods to `TMDBClient` interface.

### 1.4 Update Mock (`internal/metadata/mock/tmdb.go`)
Add mock implementations with realistic data for dev mode movies.

---

## Phase 2: OMDb Integration

### 2.1 Add Config (`internal/config/config.go`)
```go
type OMDbConfig struct {
    APIKey  string `mapstructure:"api_key"`
    BaseURL string `mapstructure:"base_url"`
    Timeout int    `mapstructure:"timeout_seconds"`
}

// Add to MetadataConfig
OMDB OMDbConfig `mapstructure:"omdb"`

// Defaults
v.SetDefault("metadata.omdb.api_key", "")
v.SetDefault("metadata.omdb.base_url", "https://www.omdbapi.com")
v.SetDefault("metadata.omdb.timeout_seconds", 15)
```

### 2.2 Create Client (`internal/metadata/omdb/`)
**Files:** `client.go`, `models.go`

```go
// models.go
type Response struct {
    Rated      string   `json:"Rated"`      // PG-13, R, etc.
    Awards     string   `json:"Awards"`     // "Won 3 Oscars..."
    ImdbRating string   `json:"imdbRating"` // "8.5"
    ImdbVotes  string   `json:"imdbVotes"`  // "1,234,567"
    Ratings    []Rating `json:"Ratings"`
}

type NormalizedRatings struct {
    ImdbRating     float64 `json:"imdbRating,omitempty"`
    ImdbVotes      int     `json:"imdbVotes,omitempty"`
    RottenTomatoes int     `json:"rottenTomatoes,omitempty"` // Critic 0-100
    RottenAudience int     `json:"rottenAudience,omitempty"` // Audience 0-100
    Metacritic     int     `json:"metacritic,omitempty"`
    Awards         string  `json:"awards,omitempty"`
}

// client.go
func (c *Client) GetByIMDbID(ctx, imdbID string) (*NormalizedRatings, error)
```

### 2.3 Add Interface & Mock
- `internal/metadata/interfaces.go`: Add `OMDbClient` interface
- `internal/metadata/mock/omdb.go`: Mock with ratings for dev mode movies

### 2.4 Update Service (`internal/metadata/service.go`)
- Add `omdb OMDbClient` field
- Add `SetOMDbClient()` method
- Initialize in `NewService()`
- Register with health service if configured

### 2.5 Update Server (`internal/api/server.go`)
- Store `realOMDbClient` like TMDB/TVDB
- Include in `switchMetadataClients()` for dev mode

---

## Phase 3: Extended Metadata Endpoints

### 3.1 Add Types (`internal/metadata/provider.go`)
```go
type ExtendedMovieResult struct {
    MovieResult
    Credits       *tmdb.NormalizedCredits `json:"credits,omitempty"`
    ContentRating string                  `json:"contentRating,omitempty"`
    Studio        string                  `json:"studio,omitempty"`
    Ratings       *omdb.NormalizedRatings `json:"ratings,omitempty"`
}

type ExtendedSeriesResult struct {
    SeriesResult
    Credits       *tmdb.NormalizedCredits `json:"credits,omitempty"`
    ContentRating string                  `json:"contentRating,omitempty"`
    Ratings       *omdb.NormalizedRatings `json:"ratings,omitempty"`
    Seasons       []SeasonResult          `json:"seasons,omitempty"`
}
```

### 3.2 Add Service Methods (`internal/metadata/service.go`)
```go
func (s *Service) GetExtendedMovie(ctx, id int) (*ExtendedMovieResult, error)
func (s *Service) GetExtendedSeries(ctx, id int) (*ExtendedSeriesResult, error)
```

Logic:
1. Get base details (existing method)
2. Fetch credits from TMDB
3. Fetch content rating from TMDB
4. If IMDB ID exists AND OMDb configured, fetch ratings
5. Combine and return

### 3.3 Add Handlers (`internal/metadata/handlers.go`)
```go
// Routes
g.GET("/movie/:id/extended", h.GetExtendedMovie)
g.GET("/series/:id/extended", h.GetExtendedSeries)
```

---

## Phase 4: Frontend Types & Hooks

### 4.1 Add Types (`web/src/types/metadata.ts`)
```typescript
export interface Person {
  id: number
  name: string
  role?: string
  photoUrl?: string
}

export interface Credits {
  directors?: Person[]
  writers?: Person[]
  creators?: Person[]
  cast: Person[]
}

export interface ExternalRatings {
  imdbRating?: number
  imdbVotes?: number
  rottenTomatoes?: number
  rottenAudience?: number
  metacritic?: number
  awards?: string
}

export interface ExtendedMovieResult extends MovieSearchResult {
  credits?: Credits
  contentRating?: string
  studio?: string
  ratings?: ExternalRatings
}

export interface ExtendedSeriesResult extends SeriesSearchResult {
  credits?: Credits
  contentRating?: string
  ratings?: ExternalRatings
  seasons?: SeasonResult[]
}
```

### 4.2 Add API (`web/src/api/metadata.ts`)
```typescript
getExtendedMovie: (tmdbId: number) => apiFetch(`/metadata/movie/${tmdbId}/extended`)
getExtendedSeries: (tmdbId: number) => apiFetch(`/metadata/series/${tmdbId}/extended`)
```

### 4.3 Add Hooks (`web/src/hooks/useMetadata.ts`)
```typescript
export function useExtendedMovieMetadata(tmdbId: number)
export function useExtendedSeriesMetadata(tmdbId: number)
```

---

## Phase 5: Modal Component

### 5.1 Create Modal (`web/src/components/search/MediaInfoModal.tsx`)

**Layout:**
```
+------------------------------------------+
|                               [X Close]  |
+------------------------------------------+
| +--------+  Title (Year)                 |
| | POSTER |  PG-13 | 2h 28m | Action      |
| |        |  Director: Name               |
| |        |  Studio: Name                 |
| +--------+                               |
+------------------------------------------+
| Overview text...                         |
+------------------------------------------+
| Ratings:                                 |
| [Tomato] 92%  [Popcorn] 87%  [IMDb] 8.5  |
| Awards: Won 3 Oscars...                  |
+------------------------------------------+
| Cast: (horizontal scroll)                |
| [O] [O] [O] [O] [O] [O] ...             |
| Name Name Name Name Name Name            |
+------------------------------------------+
| [For TV: Seasons accordion list]         |
+------------------------------------------+
| [Add to Library]                         |
+------------------------------------------+
```

### 5.2 Sub-components
- `RatingsDisplay` - RT tomato/splat icons, IMDB logo, scores
- `CastList` - Horizontal scroll with circular avatars
- `SeasonsList` - Accordion for TV seasons/episodes

### 5.3 Update External Cards
**Files:** `ExternalMovieCard.tsx`, `ExternalSeriesCard.tsx`

Add info button that opens modal:
```tsx
<Button variant="ghost" size="icon-sm" onClick={() => setInfoOpen(true)}>
  <Info className="size-4" />
</Button>
<MediaInfoModal open={infoOpen} onOpenChange={setInfoOpen} ... />
```

---

## Graceful Degradation

| Scenario | Behavior |
|----------|----------|
| OMDb not configured | Ratings section hidden, no error |
| OMDb request fails | Ratings section hidden, warning logged |
| TMDB credits fail | Credits section hidden, partial result returned |
| No IMDB ID | Skip OMDb call, show TMDB data only |

---

## Files to Modify/Create

### Backend (Go)
| File | Action |
|------|--------|
| `internal/config/config.go` | Add OMDbConfig |
| `internal/metadata/tmdb/models.go` | Add credits types |
| `internal/metadata/tmdb/client.go` | Add 4 new methods |
| `internal/metadata/omdb/client.go` | **Create** |
| `internal/metadata/omdb/models.go` | **Create** |
| `internal/metadata/interfaces.go` | Add interfaces |
| `internal/metadata/mock/tmdb.go` | Add mock credits |
| `internal/metadata/mock/omdb.go` | **Create** |
| `internal/metadata/provider.go` | Add extended types |
| `internal/metadata/service.go` | Add extended methods, OMDb |
| `internal/metadata/handlers.go` | Add 2 endpoints |
| `internal/api/server.go` | OMDb client init |
| `configs/config.example.yaml` | Add omdb section |

### Frontend (TypeScript)
| File | Action |
|------|--------|
| `web/src/types/metadata.ts` | Add extended types |
| `web/src/api/metadata.ts` | Add 2 API methods |
| `web/src/hooks/useMetadata.ts` | Add 2 hooks |
| `web/src/components/search/MediaInfoModal.tsx` | **Create** |
| `web/src/components/search/ExternalMovieCard.tsx` | Add info button |
| `web/src/components/search/ExternalSeriesCard.tsx` | Add info button |

---

## Verification

1. **Backend**: Run `make test` after each phase
2. **Integration**:
   - Search for a movie, click More Info, verify all fields display
   - Search for a TV series, verify seasons/episodes show
   - Disable OMDb key, verify ratings gracefully hidden
3. **Dev Mode**: Toggle developer mode, verify mock data displays correctly
