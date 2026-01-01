# Downloads Page Implementation Plan

## Overview
Make the "Queue" page functional by displaying real-time download data from Transmission. Rename to "Downloads", add media type tabs, and show detailed torrent information in a table with quality attributes.

---

## Phase 0: Backend - Extend Parser with Additional Attributes

### File: `internal/library/scanner/parser.go`

**0.1 Add Attributes field to ParsedMedia struct:**
```go
type ParsedMedia struct {
    // ... existing fields ...
    Attributes []string `json:"attributes,omitempty"` // NEW: HDR, Atmos, etc.
}
```

**0.2 Add HDR pattern detection:**
```go
hdrPatterns = map[string]*regexp.Regexp{
    "DV":      regexp.MustCompile(`(?i)(dolby[\.\s]?vision|dovi|dv)`),
    "HDR10+":  regexp.MustCompile(`(?i)hdr10\+`),
    "HDR10":   regexp.MustCompile(`(?i)hdr10`),
    "HDR":     regexp.MustCompile(`(?i)hdr(?!10)`),
    "HLG":     regexp.MustCompile(`(?i)hlg`),
}
```

**0.3 Add Audio pattern detection:**
```go
audioPatterns = map[string]*regexp.Regexp{
    "Atmos":    regexp.MustCompile(`(?i)atmos`),
    "DTS-X":    regexp.MustCompile(`(?i)dts[\.\-]?x`),
    "DTS-HD":   regexp.MustCompile(`(?i)dts[\.\-]?hd([\.\-]?ma)?`),
    "TrueHD":   regexp.MustCompile(`(?i)truehd`),
    "DTS":      regexp.MustCompile(`(?i)dts(?![\.\-]?(x|hd))`),
    "DD+":      regexp.MustCompile(`(?i)(ddp|dd\+|e[\.\-]?ac[\.\-]?3)`),
    "DD":       regexp.MustCompile(`(?i)(dd[25]\.[01]|ac[\.\-]?3)`),
    "AAC":      regexp.MustCompile(`(?i)aac`),
    "FLAC":     regexp.MustCompile(`(?i)flac`),
}
```

**0.4 Update parseQualityInfo() to populate Attributes:**
- Check all HDR patterns and append matches to Attributes
- Check all Audio patterns and append matches to Attributes
- Also add "REMUX" to Attributes if Source is "Remux"

---

## Phase 1: Backend - Extend Transmission Client

### File: `internal/downloader/transmission/client.go`

**1.1 Add new fields to Torrent struct:**
```go
type Torrent struct {
    ID             string  `json:"id"`
    Name           string  `json:"name"`
    Status         string  `json:"status"`
    Progress       float64 `json:"progress"`
    Size           int64   `json:"size"`
    DownloadedSize int64   `json:"downloadedSize"`  // NEW
    DownloadSpeed  int64   `json:"downloadSpeed"`   // NEW
    ETA            int64   `json:"eta"`             // NEW: seconds, -1 if unavailable
    Path           string  `json:"path"`
}
```

**1.2 Update List() to request additional fields:**
- Add to fields array: `"eta"`, `"rateDownload"`, `"downloadedEver"`, `"sizeWhenDone"`

**1.3 Add Stop() method:**
```go
func (c *Client) Stop(id string) error  // Uses "torrent-stop" RPC
```

**1.4 Add RemoveWithData() method:**
```go
func (c *Client) RemoveWithData(id string) error  // delete-local-data: true
```

---

## Phase 2: Backend - Queue Service & API

### File: `internal/downloader/queue.go` (NEW)

**2.1 Create QueueItem struct with parsed metadata:**
```go
type QueueItem struct {
    ID             string   `json:"id"`
    ClientID       int64    `json:"clientId"`
    ClientName     string   `json:"clientName"`
    Title          string   `json:"title"`
    MediaType      string   `json:"mediaType"`      // "movie" or "series"
    Status         string   `json:"status"`
    Progress       float64  `json:"progress"`       // 0-100
    Size           int64    `json:"size"`
    DownloadedSize int64    `json:"downloadedSize"`
    DownloadSpeed  int64    `json:"downloadSpeed"`
    ETA            int64    `json:"eta"`
    Quality        string   `json:"quality"`        // "1080p", "2160p"
    Source         string   `json:"source"`         // "BluRay", "WEB-DL"
    Codec          string   `json:"codec"`          // "x265", "x264"
    Attributes     []string `json:"attributes"`     // HDR, Atmos, REMUX, etc.
    Season         int      `json:"season,omitempty"`
    Episode        int      `json:"episode,omitempty"`
    DownloadPath   string   `json:"downloadPath"`
}
```

**2.2 Implement queue methods:**
- `GetQueue(ctx)` - Query all enabled download clients, parse torrent names, return enriched items
- `PauseDownload(ctx, clientID, torrentID)` - Call client.Stop()
- `ResumeDownload(ctx, clientID, torrentID)` - Call client.Start()
- `RemoveDownload(ctx, clientID, torrentID, deleteFiles)` - Call Remove/RemoveWithData

**2.3 Media type detection:**
- Check if `downloadPath` contains `SlipStream/Movies` → "movie"
- Check if `downloadPath` contains `SlipStream/Series` → "series"

**2.4 Parse torrent name using existing scanner:**
- Reuse `scanner.ParseFilename()` to extract quality, source, codec, season, episode

### File: `internal/api/server.go`

**2.5 Replace placeholder queue handlers:**
- `GET /api/v1/queue` - Return real data from GetQueue()
- `POST /api/v1/queue/:id/pause` - Accept `{clientId}` in body
- `POST /api/v1/queue/:id/resume` - Accept `{clientId}` in body
- `DELETE /api/v1/queue/:id` - Query params: `clientId`, `deleteFiles`

---

## Phase 3: Frontend - Types & Utilities

### File: `web/src/types/queue.ts`

**3.1 Update QueueItem interface:**
```typescript
export interface QueueItem {
  id: string                    // Changed: string (hash)
  clientId: number
  clientName: string
  title: string
  mediaType: 'movie' | 'series'
  status: 'queued' | 'downloading' | 'paused' | 'completed' | 'failed'
  progress: number
  size: number
  downloadedSize: number
  downloadSpeed: number
  eta: number
  quality?: string
  source?: string
  codec?: string
  attributes: string[]          // HDR, Atmos, REMUX, etc.
  season?: number
  episode?: number
  downloadPath: string
}
```

### File: `web/src/lib/formatters.ts`

**3.2 Add reusable series title utility:**
```typescript
export function formatSeriesTitle(
  seriesName: string,
  season?: number,
  episode?: number
): string {
  if (!season || !episode) return seriesName
  return `${seriesName} - ${formatEpisodeNumber(season, episode)}`
}
```

### File: `web/src/components/media/FormatBadges.tsx` (NEW)

**3.3 Create format badges component:**
```typescript
interface FormatBadgesProps {
  source?: string      // BluRay, WEB-DL, etc.
  codec?: string       // x265, x264, etc.
  attributes: string[] // HDR, Atmos, REMUX, etc.
}
```
- Display badges for each attribute (DV, HDR10, Atmos, etc.)
- Display codec badge if present (HEVC shown for x265)
- Use existing Badge component with secondary variant
- Color-code special attributes (e.g., DV/HDR in purple, Atmos in blue)

---

## Phase 4: Frontend - Update Navigation

### File: `web/src/components/layout/Sidebar.tsx`

**4.1 Rename "Queue" to "Downloads" in activityGroup (line 58):**
```typescript
{ title: 'Downloads', href: '/activity', icon: Download },
```

---

## Phase 5: Frontend - Refactor Downloads Page

### File: `web/src/routes/activity/index.tsx`

**5.1 Update page header:**
- Title: "Downloads"
- Description: "Monitor active downloads"
- Icon: Download (already imported in Sidebar)

**5.2 Replace tabs with media type filter:**
```typescript
<TabsList>
  <TabsTrigger value="all">All</TabsTrigger>
  <TabsTrigger value="movies">Movies</TabsTrigger>
  <TabsTrigger value="series">Series</TabsTrigger>
</TabsList>
```

**5.3 Replace QueueItemRow with table layout:**
```
| Title | Quality | Attributes | Progress | Time Left | Speed | Actions |
```

- **Title**: Movie name OR `formatSeriesTitle(name, season, episode)` with Film/Tv icon
- **Quality**: QualityBadge component (e.g., "1080p")
- **Attributes**: FormatBadges component showing codec + attributes (e.g., "HEVC", "DV", "Atmos", "REMUX")
- **Progress**: ProgressBar + `{downloaded}/{total}` text
- **Time Left**: formatEta(eta)
- **Speed**: formatSpeed(downloadSpeed)
- **Actions**: Pause/Play button, Delete button with ConfirmDialog

**5.4 Filter logic:**
- "all" tab: show all items
- "movies" tab: filter by `mediaType === 'movie'`
- "series" tab: filter by `mediaType === 'series'`

### File: `web/src/api/queue.ts`

**5.5 Update API client:**
```typescript
pause: (clientId: number, id: string) =>
  apiFetch(`/queue/${id}/pause`, { method: 'POST', body: JSON.stringify({ clientId }) })

resume: (clientId: number, id: string) =>
  apiFetch(`/queue/${id}/resume`, { method: 'POST', body: JSON.stringify({ clientId }) })

remove: (clientId: number, id: string, deleteFiles = false) =>
  apiFetch(`/queue/${id}?clientId=${clientId}&deleteFiles=${deleteFiles}`, { method: 'DELETE' })
```

### File: `web/src/hooks/useQueue.ts`

**5.6 Update mutation hooks to accept clientId + torrentId**

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/library/scanner/parser.go` | Add Attributes field, HDR/audio pattern detection |
| `internal/downloader/transmission/client.go` | Add ETA/speed fields, Stop(), RemoveWithData() |
| `internal/downloader/queue.go` | NEW: Queue service with parsing |
| `internal/api/server.go` | Implement queue handlers |
| `web/src/types/queue.ts` | Update QueueItem interface with attributes |
| `web/src/lib/formatters.ts` | Add formatSeriesTitle() |
| `web/src/components/media/FormatBadges.tsx` | NEW: Format badges component |
| `web/src/components/layout/Sidebar.tsx` | Rename Queue → Downloads |
| `web/src/routes/activity/index.tsx` | Full refactor to table with tabs |
| `web/src/api/queue.ts` | Update API calls with clientId |
| `web/src/hooks/useQueue.ts` | Update mutation signatures |

---

## Implementation Order

1. Backend: Extend parser with Attributes field + HDR/audio patterns
2. Backend: Extend Transmission client (new fields + methods)
3. Backend: Create queue service with parsing
4. Backend: Implement queue API handlers
5. Frontend: Update types and add formatSeriesTitle
6. Frontend: Create FormatBadges component
7. Frontend: Update Sidebar navigation
8. Frontend: Refactor Downloads page with table
9. Frontend: Update API client and hooks
