# SlipStream Indexer Implementation Plan

## Executive Summary

This document outlines the implementation plan for adding indexer functionality to SlipStream, based on technical specifications derived from Prowlarr. The plan focuses on features relevant to SlipStream as a unified media manager, excluding functionality that is duplicative or unnecessary.

## Scope Analysis

### What SlipStream Needs (In Scope)

| Feature | Priority | Rationale |
|---------|----------|-----------|
| Torznab/Newznab Protocol Support | High | Standard API for 90%+ of indexers |
| Search Aggregation | High | Search across multiple indexers simultaneously |
| Category Mapping | High | Standard Newznab categories for Movies/TV |
| Indexer CRUD Operations | High | Full indexer management |
| Indexer Status Tracking | Medium | Health monitoring, failure backoff |
| Rate Limiting | Medium | Protect indexer accounts |
| Search History | Medium | Track searches and grabs |
| Download Client Expansion | Medium | qBittorrent, SABnzbd support |
| Release Grabbing | High | Send releases to download clients |

### What SlipStream Does NOT Need (Out of Scope)

| Prowlarr Feature | Reason for Exclusion |
|------------------|---------------------|
| Application Sync | SlipStream IS the media manager; no external apps to sync to |
| Cardigann/YAML Definitions | Complex meta-indexer system; Torznab/Newznab covers most indexers |
| Newznab API Endpoints (Server) | SlipStream doesn't need to expose itself as an indexer proxy |
| Notification System | Can be added later; not core to indexer functionality |
| Music/Book Search | SlipStream focuses on Movies and TV only |
| Tag-based Filtering | Overkill for initial implementation |
| OAuth Authentication | Not needed for indexer auth patterns |

---

## Phase 1: Core Indexer Infrastructure

**Goal**: Establish the foundation for indexer management and Torznab/Newznab protocol support.

### 1.1 Database Schema Updates

Create migration `003_indexer_enhancements.sql`:

```sql
-- Enhance indexers table with additional fields
ALTER TABLE indexers ADD COLUMN settings TEXT DEFAULT '{}';  -- JSON for provider-specific settings
ALTER TABLE indexers ADD COLUMN protocol TEXT DEFAULT 'torrent' CHECK (protocol IN ('torrent', 'usenet'));
ALTER TABLE indexers ADD COLUMN privacy TEXT DEFAULT 'public' CHECK (privacy IN ('public', 'semi-private', 'private'));
ALTER TABLE indexers ADD COLUMN supports_search INTEGER DEFAULT 1;
ALTER TABLE indexers ADD COLUMN supports_rss INTEGER DEFAULT 1;
ALTER TABLE indexers ADD COLUMN api_path TEXT DEFAULT '/api';

-- Indexer status tracking (for health/failure management)
CREATE TABLE IF NOT EXISTS indexer_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    indexer_id INTEGER NOT NULL UNIQUE REFERENCES indexers(id) ON DELETE CASCADE,
    initial_failure DATETIME,
    most_recent_failure DATETIME,
    escalation_level INTEGER DEFAULT 0,
    disabled_till DATETIME,
    last_rss_sync DATETIME,
    cookies TEXT,  -- JSON for session cookies
    cookies_expiration DATETIME
);

CREATE INDEX idx_indexer_status_indexer_id ON indexer_status(indexer_id);

-- Search/grab history for indexers
CREATE TABLE IF NOT EXISTS indexer_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    indexer_id INTEGER NOT NULL REFERENCES indexers(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('query', 'rss', 'grab', 'auth')),
    successful INTEGER NOT NULL DEFAULT 1,
    query TEXT,
    categories TEXT,  -- JSON array
    results_count INTEGER,
    elapsed_ms INTEGER,
    data TEXT,  -- JSON for additional data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_indexer_history_indexer_id ON indexer_history(indexer_id);
CREATE INDEX idx_indexer_history_created_at ON indexer_history(created_at);
```

### 1.2 Core Types and Interfaces

Create `internal/indexer/types.go`:

```go
// Protocol represents the download protocol
type Protocol string

const (
    ProtocolTorrent Protocol = "torrent"
    ProtocolUsenet  Protocol = "usenet"
)

// Privacy represents indexer privacy level
type Privacy string

const (
    PrivacyPublic      Privacy = "public"
    PrivacySemiPrivate Privacy = "semi-private"
    PrivacyPrivate     Privacy = "private"
)

// IndexerDefinition represents a configured indexer
type IndexerDefinition struct {
    ID              int64           `json:"id"`
    Name            string          `json:"name"`
    Type            string          `json:"type"`  // torznab, newznab
    URL             string          `json:"url"`
    APIPath         string          `json:"apiPath"`
    APIKey          string          `json:"apiKey,omitempty"`
    Categories      []int           `json:"categories"`
    Protocol        Protocol        `json:"protocol"`
    Privacy         Privacy         `json:"privacy"`
    SupportsMovies  bool            `json:"supportsMovies"`
    SupportsTV      bool            `json:"supportsTV"`
    SupportsSearch  bool            `json:"supportsSearch"`
    SupportsRSS     bool            `json:"supportsRss"`
    Priority        int             `json:"priority"`
    Enabled         bool            `json:"enabled"`
    Settings        json.RawMessage `json:"settings,omitempty"`
}

// SearchCriteria defines search parameters
type SearchCriteria struct {
    Query      string   `json:"query,omitempty"`
    Type       string   `json:"type"`  // search, tvsearch, movie
    Categories []int    `json:"categories,omitempty"`

    // Movie-specific
    ImdbID string `json:"imdbId,omitempty"`
    TmdbID int    `json:"tmdbId,omitempty"`
    Year   int    `json:"year,omitempty"`

    // TV-specific
    TvdbID  int    `json:"tvdbId,omitempty"`
    Season  int    `json:"season,omitempty"`
    Episode int    `json:"episode,omitempty"`

    // Pagination
    Limit  int `json:"limit,omitempty"`
    Offset int `json:"offset,omitempty"`
}

// ReleaseInfo represents a search result
type ReleaseInfo struct {
    GUID        string    `json:"guid"`
    Title       string    `json:"title"`
    Description string    `json:"description,omitempty"`
    DownloadURL string    `json:"downloadUrl"`
    InfoURL     string    `json:"infoUrl,omitempty"`
    Size        int64     `json:"size"`
    PublishDate time.Time `json:"publishDate"`
    Categories  []int     `json:"categories"`

    // Indexer info
    IndexerID   int64    `json:"indexerId"`
    IndexerName string   `json:"indexer"`
    Protocol    Protocol `json:"protocol"`

    // External IDs
    ImdbID int `json:"imdbId,omitempty"`
    TmdbID int `json:"tmdbId,omitempty"`
    TvdbID int `json:"tvdbId,omitempty"`
}

// TorrentInfo extends ReleaseInfo with torrent-specific fields
type TorrentInfo struct {
    ReleaseInfo
    Seeders              int     `json:"seeders"`
    Leechers             int     `json:"leechers"`
    InfoHash             string  `json:"infoHash,omitempty"`
    MagnetURL            string  `json:"magnetUrl,omitempty"`
    MinimumRatio         float64 `json:"minimumRatio,omitempty"`
    MinimumSeedTime      int64   `json:"minimumSeedTime,omitempty"`  // seconds
    DownloadVolumeFactor float64 `json:"downloadVolumeFactor"`       // 0 = freeleech
    UploadVolumeFactor   float64 `json:"uploadVolumeFactor"`         // 2 = double upload
}
```

### 1.3 Indexer Interface

Update `internal/indexer/indexer.go`:

```go
// Indexer defines the interface for search indexers
type Indexer interface {
    // Identity
    Name() string
    Definition() *IndexerDefinition

    // Operations
    Test(ctx context.Context) error
    Search(ctx context.Context, criteria SearchCriteria) ([]ReleaseInfo, error)
    Download(ctx context.Context, url string) ([]byte, error)

    // Capabilities
    Capabilities() *Capabilities
    SupportsSearch() bool
    SupportsRSS() bool
}

// Capabilities describes what an indexer supports
type Capabilities struct {
    SupportsMovies      bool              `json:"supportsMovies"`
    SupportsTV          bool              `json:"supportsTV"`
    SupportsSearch      bool              `json:"supportsSearch"`
    SupportsRSS         bool              `json:"supportsRss"`
    SearchParams        []string          `json:"searchParams"`
    TvSearchParams      []string          `json:"tvSearchParams"`
    MovieSearchParams   []string          `json:"movieSearchParams"`
    Categories          []CategoryMapping `json:"categories"`
    MaxResultsPerSearch int               `json:"maxResultsPerSearch"`
}

// CategoryMapping maps indexer categories to standard Newznab categories
type CategoryMapping struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}
```

### 1.4 Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/indexer/types.go` | Create | Core type definitions |
| `internal/indexer/indexer.go` | Modify | Enhanced interface |
| `internal/indexer/service.go` | Create | Service layer with CRUD operations |
| `internal/indexer/handlers.go` | Create | HTTP handlers |
| `internal/database/queries/indexers.sql` | Modify | Add new queries |
| `internal/database/migrations/003_indexer_enhancements.sql` | Create | Schema updates |

---

## Phase 2: Torznab/Newznab Protocol Implementation

**Goal**: Implement the Torznab/Newznab protocol for communicating with indexers.

### 2.1 Newznab Category Constants

Create `internal/indexer/categories.go`:

```go
// Standard Newznab Categories
const (
    CategoryConsole       = 1000
    CategoryMovies        = 2000
    CategoryMoviesSD      = 2030
    CategoryMoviesHD      = 2040
    CategoryMoviesUHD     = 2045
    CategoryMoviesBluRay  = 2050
    CategoryAudio         = 3000
    CategoryPC            = 4000
    CategoryTV            = 5000
    CategoryTVSD          = 5030
    CategoryTVHD          = 5040
    CategoryTVUHD         = 5045
    CategoryTVAnime       = 5070
    CategoryXXX           = 6000
    CategoryBooks         = 7000
    CategoryOther         = 8000
)

// MovieCategories returns all movie-related categories
func MovieCategories() []int {
    return []int{
        CategoryMovies,
        CategoryMoviesSD,
        CategoryMoviesHD,
        CategoryMoviesUHD,
        CategoryMoviesBluRay,
    }
}

// TVCategories returns all TV-related categories
func TVCategories() []int {
    return []int{
        CategoryTV,
        CategoryTVSD,
        CategoryTVHD,
        CategoryTVUHD,
        CategoryTVAnime,
    }
}
```

### 2.2 Torznab Client Implementation

Create `internal/indexer/torznab/client.go`:

```go
package torznab

// Client implements the Torznab protocol
type Client struct {
    definition *indexer.IndexerDefinition
    httpClient *http.Client
    logger     zerolog.Logger
}

// NewClient creates a new Torznab client
func NewClient(def *indexer.IndexerDefinition, logger zerolog.Logger) *Client

// Test verifies the connection and fetches capabilities
func (c *Client) Test(ctx context.Context) error

// FetchCapabilities retrieves the indexer's capabilities via ?t=caps
func (c *Client) FetchCapabilities(ctx context.Context) (*indexer.Capabilities, error)

// Search performs a search using the Torznab API
func (c *Client) Search(ctx context.Context, criteria indexer.SearchCriteria) ([]indexer.ReleaseInfo, error)

// Download fetches a torrent/nzb file
func (c *Client) Download(ctx context.Context, url string) ([]byte, error)
```

### 2.3 XML Response Parsing

Create `internal/indexer/torznab/parser.go`:

```go
// Parse Newznab/Torznab XML responses
// Handle RSS format with newznab:attr and torznab:attr extensions

type rssResponse struct {
    Channel struct {
        Items []rssItem `xml:"item"`
    } `xml:"channel"`
}

type rssItem struct {
    Title       string       `xml:"title"`
    GUID        string       `xml:"guid"`
    Link        string       `xml:"link"`
    PubDate     string       `xml:"pubDate"`
    Size        int64        `xml:"size"`
    Description string       `xml:"description"`
    Enclosure   rssEnclosure `xml:"enclosure"`
    Attributes  []rssAttr    `xml:"attr"`
}

// ParseSearchResponse parses XML search results into ReleaseInfo objects
func ParseSearchResponse(body []byte) ([]indexer.ReleaseInfo, error)

// ParseCapabilities parses the ?t=caps XML response
func ParseCapabilities(body []byte) (*indexer.Capabilities, error)
```

### 2.4 Request Building

Create `internal/indexer/torznab/request.go`:

```go
// BuildSearchURL constructs the search URL with parameters
func BuildSearchURL(baseURL, apiPath string, criteria indexer.SearchCriteria, apiKey string) string {
    // Base: {baseURL}{apiPath}?t={type}&apikey={key}
    // Add: q, cat, limit, offset
    // TV: season, ep, tvdbid, imdbid
    // Movie: imdbid, tmdbid, year
}

// BuildCapabilitiesURL constructs the caps URL
func BuildCapabilitiesURL(baseURL, apiPath, apiKey string) string {
    // {baseURL}{apiPath}?t=caps&apikey={key}
}
```

### 2.5 Files to Create

| File | Description |
|------|-------------|
| `internal/indexer/categories.go` | Newznab category constants and helpers |
| `internal/indexer/torznab/client.go` | Torznab protocol client |
| `internal/indexer/torznab/parser.go` | XML response parsing |
| `internal/indexer/torznab/request.go` | Request URL building |
| `internal/indexer/newznab/client.go` | Newznab client (extends Torznab) |

---

## Phase 3: Search Service and Aggregation

**Goal**: Implement search across multiple indexers with result aggregation.

### 3.1 Search Service

Create `internal/indexer/search/service.go`:

```go
// Service orchestrates searches across multiple indexers
type Service struct {
    indexerService *indexer.Service
    statusService  *status.Service
    historyService *history.Service
    logger         zerolog.Logger
}

// Search executes a search across enabled indexers
func (s *Service) Search(ctx context.Context, criteria indexer.SearchCriteria) (*SearchResult, error) {
    // 1. Get enabled indexers that support the search type
    // 2. Filter by category support
    // 3. Filter out rate-limited/disabled indexers
    // 4. Dispatch parallel searches
    // 5. Aggregate results
    // 6. Deduplicate by GUID
    // 7. Sort by priority/seeders
    // 8. Record in history
}

// SearchResult contains aggregated search results
type SearchResult struct {
    Releases      []indexer.ReleaseInfo `json:"releases"`
    TotalResults  int                   `json:"totalResults"`
    IndexersUsed  int                   `json:"indexersUsed"`
    IndexerErrors []IndexerError        `json:"indexerErrors,omitempty"`
}

type IndexerError struct {
    IndexerID   int64  `json:"indexerId"`
    IndexerName string `json:"indexerName"`
    Error       string `json:"error"`
}
```

### 3.2 Parallel Search Dispatch

```go
// dispatchSearches runs searches in parallel across indexers
func (s *Service) dispatchSearches(ctx context.Context, indexers []*indexer.IndexerDefinition, criteria indexer.SearchCriteria) (*SearchResult, error) {
    var wg sync.WaitGroup
    resultsChan := make(chan searchTaskResult, len(indexers))

    // Set timeout for individual indexer searches
    searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    for _, idx := range indexers {
        wg.Add(1)
        go func(def *indexer.IndexerDefinition) {
            defer wg.Done()

            result := s.searchIndexer(searchCtx, def, criteria)
            resultsChan <- result
        }(idx)
    }

    // Wait and collect results
    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    return s.aggregateResults(resultsChan)
}
```

### 3.3 Result Aggregation and Deduplication

```go
// aggregateResults combines results from multiple indexers
func (s *Service) aggregateResults(results <-chan searchTaskResult) *SearchResult {
    allReleases := make([]indexer.ReleaseInfo, 0)
    errors := make([]IndexerError, 0)

    for result := range results {
        if result.Error != nil {
            errors = append(errors, IndexerError{
                IndexerID:   result.IndexerID,
                IndexerName: result.IndexerName,
                Error:       result.Error.Error(),
            })
            continue
        }
        allReleases = append(allReleases, result.Releases...)
    }

    // Deduplicate by GUID, prefer higher priority indexer
    deduplicated := deduplicateReleases(allReleases)

    // Sort: seeders descending for torrents, publish date for usenet
    sortReleases(deduplicated)

    return &SearchResult{
        Releases:      deduplicated,
        TotalResults:  len(deduplicated),
        IndexersUsed:  len(results) - len(errors),
        IndexerErrors: errors,
    }
}
```

### 3.4 Files to Create

| File | Description |
|------|-------------|
| `internal/indexer/search/service.go` | Search orchestration service |
| `internal/indexer/search/aggregator.go` | Result aggregation and deduplication |
| `internal/indexer/search/handlers.go` | HTTP handlers for search endpoints |

---

## Phase 4: Indexer Status and Rate Limiting

**Goal**: Track indexer health and implement rate limiting to protect accounts.

### 4.1 Status Service

Create `internal/indexer/status/service.go`:

```go
// Service tracks indexer health and status
type Service struct {
    queries *sqlc.Queries
    logger  zerolog.Logger
}

// GetStatus retrieves the current status for an indexer
func (s *Service) GetStatus(ctx context.Context, indexerID int64) (*IndexerStatus, error)

// RecordSuccess records a successful operation
func (s *Service) RecordSuccess(ctx context.Context, indexerID int64) error

// RecordFailure records a failed operation with backoff
func (s *Service) RecordFailure(ctx context.Context, indexerID int64, err error) error

// IsDisabled checks if an indexer is temporarily disabled
func (s *Service) IsDisabled(ctx context.Context, indexerID int64) (bool, time.Time, error)
```

### 4.2 Backoff Strategy

```go
// Escalation backoff periods
var backoffPeriods = []time.Duration{
    5 * time.Minute,   // Level 1
    15 * time.Minute,  // Level 2
    30 * time.Minute,  // Level 3
    1 * time.Hour,     // Level 4
    3 * time.Hour,     // Level 5+
}

func calculateBackoff(escalationLevel int) time.Duration {
    if escalationLevel >= len(backoffPeriods) {
        return backoffPeriods[len(backoffPeriods)-1]
    }
    return backoffPeriods[escalationLevel]
}
```

### 4.3 Rate Limiting

Create `internal/indexer/ratelimit/limiter.go`:

```go
// Limiter tracks query/grab counts per indexer
type Limiter struct {
    historyService *history.Service
}

// CheckQueryLimit returns whether the indexer has reached its query limit
func (l *Limiter) CheckQueryLimit(ctx context.Context, indexerID int64, limit int, period time.Duration) (bool, error)

// CheckGrabLimit returns whether the indexer has reached its grab limit
func (l *Limiter) CheckGrabLimit(ctx context.Context, indexerID int64, limit int, period time.Duration) (bool, error)
```

### 4.4 Files to Create

| File | Description |
|------|-------------|
| `internal/indexer/status/service.go` | Status tracking service |
| `internal/indexer/status/models.go` | Status model definitions |
| `internal/indexer/ratelimit/limiter.go` | Rate limiting logic |

---

## Phase 5: Release Grabbing

**Goal**: Implement the workflow for grabbing releases and sending them to download clients.

### 5.1 Grab Service

Create `internal/indexer/grab/service.go`:

```go
// Service handles grabbing releases
type Service struct {
    indexerService    *indexer.Service
    downloadService   *downloader.Service
    statusService     *status.Service
    historyService    *history.Service
    logger            zerolog.Logger
}

// GrabRequest represents a request to grab a release
type GrabRequest struct {
    Release       *indexer.ReleaseInfo `json:"release"`
    ClientID      int64                `json:"clientId,omitempty"`      // Optional: specific client
    MediaType     string               `json:"mediaType"`               // movie, episode
    MediaID       int64                `json:"mediaId"`                 // movie_id or episode_id
}

// Grab downloads a release and sends it to a download client
func (s *Service) Grab(ctx context.Context, req GrabRequest) (*GrabResult, error) {
    // 1. Validate the release
    // 2. Check grab limits
    // 3. Download torrent/nzb from indexer
    // 4. Validate torrent/nzb content
    // 5. Select appropriate download client
    // 6. Send to download client
    // 7. Create download record in database
    // 8. Record grab in history
}

// GrabResult contains the result of a grab operation
type GrabResult struct {
    Success    bool   `json:"success"`
    DownloadID string `json:"downloadId,omitempty"`
    ClientID   int64  `json:"clientId"`
    ClientName string `json:"clientName"`
    Error      string `json:"error,omitempty"`
}
```

### 5.2 Torrent Validation

Create `internal/indexer/grab/validation.go`:

```go
// ValidateTorrent checks if the content is a valid torrent file
func ValidateTorrent(content []byte) (string, error) {
    // Check for magnet link disguised as file
    if bytes.HasPrefix(content, []byte("magn")) {
        return extractMagnetURL(content)
    }

    // Parse as bencoded torrent
    // Validate info dictionary exists
    // Extract and return info hash
}

// ValidateNZB checks if the content is a valid NZB file
func ValidateNZB(content []byte) error {
    // Parse as XML
    // Validate structure
    // Check for error responses
}
```

### 5.3 Download Client Selection

```go
// selectDownloadClient chooses the appropriate client for the release
func (s *Service) selectDownloadClient(ctx context.Context, release *indexer.ReleaseInfo, preferredClientID int64) (*downloader.DownloadClient, error) {
    // 1. If preferred client specified and matches protocol, use it
    // 2. Otherwise, get all enabled clients for the protocol
    // 3. Filter out blocked/failed clients
    // 4. Sort by priority
    // 5. Return first available
}
```

### 5.4 Files to Create

| File | Description |
|------|-------------|
| `internal/indexer/grab/service.go` | Grab orchestration service |
| `internal/indexer/grab/validation.go` | Torrent/NZB validation |
| `internal/indexer/grab/handlers.go` | HTTP handlers |

---

## Phase 6: Download Client Expansion

**Goal**: Add support for additional download clients (qBittorrent, SABnzbd).

### 6.1 qBittorrent Client

Create `internal/downloader/qbittorrent/client.go`:

```go
// Client implements qBittorrent Web API
type Client struct {
    config    Config
    sessionID string
    client    *http.Client
    logger    zerolog.Logger
}

type Config struct {
    Host     string
    Port     int
    Username string
    Password string
    UseSSL   bool
    Category string
}

// Authenticate logs in and obtains session cookie
func (c *Client) Authenticate() error

// AddTorrent adds a torrent from URL or file
func (c *Client) AddTorrent(url string, options AddTorrentOptions) (string, error)

// AddTorrentFile adds a torrent from file bytes
func (c *Client) AddTorrentFile(data []byte, options AddTorrentOptions) (string, error)

// GetTorrents lists all torrents
func (c *Client) GetTorrents() ([]Torrent, error)

// SetSeedLimits configures seed ratio/time limits
func (c *Client) SetSeedLimits(hash string, ratio float64, seedTime int64) error
```

### 6.2 SABnzbd Client

Create `internal/downloader/sabnzbd/client.go`:

```go
// Client implements SABnzbd API
type Client struct {
    config Config
    client *http.Client
    logger zerolog.Logger
}

type Config struct {
    Host     string
    Port     int
    APIKey   string
    UseSSL   bool
    Category string
}

// AddURL adds an NZB from URL
func (c *Client) AddURL(url string, category string) (string, error)

// AddFile adds an NZB from file bytes
func (c *Client) AddFile(data []byte, filename, category string) (string, error)

// GetQueue returns current download queue
func (c *Client) GetQueue() ([]QueueItem, error)

// GetHistory returns download history
func (c *Client) GetHistory() ([]HistoryItem, error)
```

### 6.3 Files to Create

| File | Description |
|------|-------------|
| `internal/downloader/qbittorrent/client.go` | qBittorrent client implementation |
| `internal/downloader/qbittorrent/types.go` | qBittorrent type definitions |
| `internal/downloader/sabnzbd/client.go` | SABnzbd client implementation |
| `internal/downloader/sabnzbd/types.go` | SABnzbd type definitions |

---

## Phase 7: API Endpoints and Frontend Integration

**Goal**: Complete REST API implementation and provide frontend integration points.

### 7.1 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/indexers` | List all indexers |
| POST | `/api/v1/indexers` | Create indexer |
| GET | `/api/v1/indexers/:id` | Get indexer by ID |
| PUT | `/api/v1/indexers/:id` | Update indexer |
| DELETE | `/api/v1/indexers/:id` | Delete indexer |
| POST | `/api/v1/indexers/:id/test` | Test indexer connection |
| POST | `/api/v1/indexers/test` | Test indexer config (without saving) |
| GET | `/api/v1/indexers/:id/status` | Get indexer status |
| GET | `/api/v1/indexers/status` | Get all indexer statuses |
| GET | `/api/v1/indexers/schema` | Get available indexer types |
| GET | `/api/v1/search` | Search across indexers |
| POST | `/api/v1/search/grab` | Grab a release |
| POST | `/api/v1/search/grab/bulk` | Grab multiple releases |
| GET | `/api/v1/history/indexer` | Get indexer search/grab history |

### 7.2 Request/Response Examples

**Search Request:**
```http
GET /api/v1/search?query=Matrix&type=movie&tmdbId=603&categories=2000,2040
```

**Search Response:**
```json
{
  "releases": [
    {
      "guid": "abc123",
      "title": "The Matrix 1999 2160p UHD BluRay x265",
      "downloadUrl": "https://indexer.com/download/abc123",
      "size": 45000000000,
      "publishDate": "2024-01-15T12:00:00Z",
      "indexer": "MyIndexer",
      "indexerId": 1,
      "protocol": "torrent",
      "seeders": 150,
      "leechers": 25,
      "categories": [2045]
    }
  ],
  "totalResults": 25,
  "indexersUsed": 3,
  "indexerErrors": []
}
```

**Grab Request:**
```http
POST /api/v1/search/grab
{
  "release": {
    "guid": "abc123",
    "indexerId": 1,
    "downloadUrl": "https://indexer.com/download/abc123",
    "title": "The Matrix 1999 2160p UHD BluRay x265"
  },
  "mediaType": "movie",
  "mediaId": 42
}
```

### 7.3 WebSocket Events

Extend existing WebSocket hub with indexer events:

```go
// Event types for indexer operations
const (
    EventSearchStarted   = "search:started"
    EventSearchCompleted = "search:completed"
    EventGrabStarted     = "grab:started"
    EventGrabCompleted   = "grab:completed"
    EventIndexerStatus   = "indexer:status"
)
```

---

## Implementation Timeline

### Milestone 1: Foundation (Phase 1-2)
- Database migrations
- Core types and interfaces
- Torznab/Newznab protocol implementation
- Basic indexer CRUD operations

### Milestone 2: Search (Phase 3)
- Search service
- Multi-indexer aggregation
- Result deduplication
- Search API endpoints

### Milestone 3: Health & Grabbing (Phase 4-5)
- Indexer status tracking
- Rate limiting
- Release grabbing
- Download client integration

### Milestone 4: Clients & Polish (Phase 6-7)
- qBittorrent client
- SABnzbd client
- Complete API endpoints
- Frontend integration

---

## Testing Strategy

### Unit Tests
- Torznab XML parsing
- Category mapping
- Rate limit calculations
- Backoff calculations

### Integration Tests
- Mock Torznab server for protocol tests
- Database tests for status tracking
- End-to-end search flow

### Manual Testing
- Real indexer connections
- Download client integration
- Error handling scenarios

---

## File Structure Summary

```
internal/indexer/
├── indexer.go              # Core interface
├── types.go                # Type definitions
├── categories.go           # Newznab categories
├── service.go              # Indexer CRUD service
├── handlers.go             # HTTP handlers
├── torznab/
│   ├── client.go           # Torznab client
│   ├── parser.go           # XML parsing
│   └── request.go          # URL building
├── newznab/
│   └── client.go           # Newznab client (usenet)
├── search/
│   ├── service.go          # Search orchestration
│   ├── aggregator.go       # Result aggregation
│   └── handlers.go         # Search API handlers
├── status/
│   ├── service.go          # Status tracking
│   └── models.go           # Status models
├── ratelimit/
│   └── limiter.go          # Rate limiting
├── grab/
│   ├── service.go          # Grab orchestration
│   ├── validation.go       # Torrent/NZB validation
│   └── handlers.go         # Grab API handlers
└── history/
    ├── service.go          # History tracking
    └── models.go           # History models

internal/downloader/
├── client.go               # Client interface
├── service.go              # Existing service
├── queue.go                # Existing queue
├── transmission/
│   └── client.go           # Existing
├── qbittorrent/
│   ├── client.go           # New
│   └── types.go            # New
└── sabnzbd/
    ├── client.go           # New
    └── types.go            # New
```

---

## Dependencies

No new external dependencies required. Uses existing:
- `encoding/xml` - XML parsing
- `net/http` - HTTP client
- `github.com/rs/zerolog` - Logging
- `github.com/labstack/echo/v4` - HTTP server

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Indexer API variations | Medium | Focus on Torznab standard, handle edge cases |
| Rate limiting complexity | Low | Start simple, enhance as needed |
| XML parsing edge cases | Medium | Comprehensive test coverage with real responses |
| Download client auth issues | Medium | Detailed logging, clear error messages |

---

## Success Criteria

1. Users can add/configure Torznab and Newznab indexers
2. Search returns results from multiple indexers
3. Users can grab releases and send to download clients
4. Indexer failures are handled gracefully with backoff
5. Search and grab history is tracked
6. qBittorrent and SABnzbd clients work alongside Transmission
