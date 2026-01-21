# Prowlarr Integration Audit: SlipStream vs Radarr

This document provides a comprehensive comparison between SlipStream's Prowlarr integration and Radarr's implementation, focusing on features and data flow rather than architectural or syntactical differences.

---

## Executive Summary

SlipStream implements an **aggregated mode** Prowlarr integration where all searches go through Prowlarr's search API. Radarr uses a **per-indexer proxy mode** where Prowlarr syncs individual indexer definitions and Radarr queries each one separately via Torznab endpoints.

### Key Differences at a Glance

| Aspect | SlipStream | Radarr |
|--------|------------|--------|
| **Integration Model** | Aggregated (single API endpoint) | Per-indexer proxy (Prowlarr syncs individual indexers) |
| **Indexer Management** | Read-only display from Prowlarr | Full CRUD, Prowlarr pushes configs |
| **Search Routing** | Single request to Prowlarr `/api` | Parallel requests to each indexer's Torznab endpoint |
| **Per-Indexer Control** | None (all-or-nothing) | Enable/disable, priority, tags, download client override |
| **RSS Sync** | Not implemented | Supported per-indexer |
| **Sync Direction** | SlipStream pulls from Prowlarr | Prowlarr pushes to Radarr |

---

## 1. Indexer Sync Mechanism

### Radarr's Approach (Prowlarr Push Model)

Prowlarr acts as the **source of truth** and pushes indexer configurations to Radarr:

1. **Application Registration**: Radarr is registered in Prowlarr as an "Application" with sync settings
2. **Sync Levels**: Prowlarr offers three sync modes:
   - `Disabled` - No sync
   - `AddOnly` - Only add new indexers (never remove)
   - `FullSync` - Add, update, and remove indexers automatically
3. **Individual Indexer Definitions**: Each Prowlarr indexer creates a corresponding indexer in Radarr with:
   - Prowlarr proxy URL as the base URL (e.g., `http://prowlarr:9696/1/api`)
   - Prowlarr API key for authentication
   - Filtered categories based on app configuration
   - Seed ratio/time requirements
4. **Event-Driven Sync**: Changes in Prowlarr trigger automatic updates:
   - Indexer added → syncs to all enabled apps
   - Indexer updated → updates in FullSync apps
   - Indexer deleted → removes from all apps
   - API key changed → re-syncs all indexers

### SlipStream's Approach (Aggregated Pull Model)

SlipStream uses Prowlarr's search API directly without storing individual indexer definitions:

1. **No Application Registration**: SlipStream is not registered as an app in Prowlarr
2. **Read-Only Indexer Display**: Indexers are fetched for display only via `GET /api/v1/indexer`
3. **Single Search Endpoint**: All searches go to Prowlarr's `/api?t=search` endpoint
4. **No Per-Indexer Sync**: Changes in Prowlarr are automatically reflected (no sync needed)

### Gap Analysis

| Feature | Radarr | SlipStream | Impact |
|---------|--------|------------|--------|
| **Prowlarr app registration** | ✅ Required | ❌ Not used | SlipStream cannot receive push notifications |
| **Per-indexer enable/disable** | ✅ Yes | ❌ No | Users cannot disable specific indexers from SlipStream |
| **Per-indexer priority** | ✅ 1-50 scale | ❌ No | Cannot prioritize preferred indexers |
| **Per-indexer tags** | ✅ Yes | ❌ No | Cannot limit indexers to specific content |
| **Download client override** | ✅ Per-indexer | ❌ No | Cannot route specific indexers to specific clients |
| **Automatic sync** | ✅ Event-driven | ❌ Manual refresh | Users must manually refresh to see changes |

### Recommendation

**Consider implementing the Prowlarr application registration flow** to enable:
- Per-indexer control (enable/disable, priority, tags)
- Automatic sync when Prowlarr indexers change
- Better alignment with the *arr ecosystem workflow

---

## 2. Search Flow

### Radarr's Search Flow

```
Movie Search Request
    ↓
Select enabled indexers (based on: EnableAutomaticSearch, tags, status)
    ↓
For each indexer (parallel):
    → Build Torznab request (GET /1/api?t=movie&imdbid=...)
    → Send to Prowlarr's indexer-specific endpoint
    → Prowlarr proxies to actual indexer
    → Parse Torznab XML response
    ↓
Aggregate results from all indexers
    ↓
Deduplicate by GUID (priority-based selection)
    ↓
Apply quality profile scoring
    ↓
Return ranked results
```

### SlipStream's Search Flow

```
Movie Search Request
    ↓
Check indexer mode (SlipStream vs Prowlarr)
    ↓
If Prowlarr mode:
    → Build Torznab request (GET /api?t=movie&imdbid=...)
    → Send single request to Prowlarr aggregate endpoint
    → Prowlarr searches all enabled indexers
    → Returns combined results
    → Parse Torznab XML response
    ↓
Apply quality enrichment (resolution, source parsing)
    ↓
Apply SlipStream scoring algorithm
    ↓
Return ranked results (5-minute cache)
```

### Feature Comparison

| Feature | Radarr | SlipStream | Notes |
|---------|--------|------------|-------|
| **Parallel indexer queries** | ✅ From Radarr | ✅ From Prowlarr | Both parallel, different orchestration point |
| **Per-indexer timeouts** | ✅ Configurable | ❌ Single global timeout | Slow indexers can delay entire search |
| **Per-indexer rate limiting** | ✅ Yes | ✅ Prowlarr handles | SlipStream relies on Prowlarr |
| **Result deduplication** | ✅ By GUID + priority | ✅ Prowlarr handles | SlipStream trusts Prowlarr's dedup |
| **Indexer attribution** | ✅ Per-result | ✅ Per-result | Both show source indexer name |
| **Search type routing** | ✅ t=movie/tvsearch | ✅ t=movie/tvsearch | Both use appropriate search types |
| **ID-based search** | ✅ IMDB/TMDB | ✅ IMDB/TMDB | Both pass metadata IDs |
| **Category filtering** | ✅ Per-indexer | ✅ Global config | SlipStream uses same categories for all |
| **Minimum seeders filter** | ✅ Per-indexer | ❌ Post-filter only | Radarr filters at request time |

### Gap Analysis

| Missing Feature | Impact | Priority |
|-----------------|--------|----------|
| **Per-indexer search enable** | Cannot exclude indexers from automatic search | Medium |
| **Per-indexer categories** | Cannot customize categories per indexer | Low |
| **Per-indexer minimum seeders** | Must filter post-search | Low |
| **Interactive vs automatic search distinction** | Both use same indexers | Low |

---

## 3. Download/Grab Flow

### Radarr's Download Flow

```
User clicks "Grab"
    ↓
Validate release still available (30-min cache)
    ↓
Get indexer instance from IndexerFactory
    ↓
Select download client based on:
    - Indexer's download client override (if set)
    - Protocol (Torrent/Usenet)
    - Client availability
    ↓
Download via Prowlarr proxy URL
    ↓
Prowlarr intercepts → forwards to actual indexer
    ↓
Record success/failure in IndexerStatusService
    ↓
Publish MovieGrabbedEvent
```

### SlipStream's Download Flow

```
User clicks "Grab"
    ↓
Check if IndexerID == 0 (Prowlarr release)
    ↓
If Prowlarr:
    → GrabClient.Download() → Service.Download()
    → Download via release URL directly
    ↓
Select download client based on protocol
    ↓
Send to download client
    ↓
WebSocket broadcast
```

### Feature Comparison

| Feature | Radarr | SlipStream | Notes |
|---------|--------|------------|-------|
| **Release validation** | ✅ 30-min cache | ❌ No validation | SlipStream grabs immediately |
| **Per-indexer download client** | ✅ Override per indexer | ❌ Protocol-based only | Cannot route indexers to specific clients |
| **Download rate limiting** | ✅ 2/sec per host | ❌ No rate limiting | Potential for rapid-fire grabs |
| **Indexer status tracking** | ✅ Success/failure recorded | ✅ Basic tracking | Radarr has more sophisticated tracking |
| **Grab history** | ✅ Full history | ✅ Full history | Both track grab history |
| **Retry logic** | ✅ Built-in | ✅ Single retry | Similar capability |

### Gap Analysis

| Missing Feature | Impact | Priority |
|-----------------|--------|----------|
| **Release validation cache** | Could grab unavailable releases | Low |
| **Per-indexer download client routing** | Cannot separate private/public tracker clients | Medium |
| **Download rate limiting** | Could overwhelm indexers | Low |

---

## 4. RSS Sync

### Radarr's RSS Sync

Radarr supports periodic RSS sync to catch new releases:
- Per-indexer RSS enable/disable
- Tracks last sync timestamp per indexer
- Only fetches releases newer than last sync
- Automatic background sync on schedule

### SlipStream's RSS Sync

**Not implemented for Prowlarr mode.**

### Gap Analysis

| Missing Feature | Impact | Priority |
|-----------------|--------|----------|
| **RSS sync** | Must rely on auto-search schedules | Medium |
| **Recent releases feed** | No quick view of new content | Low |

---

## 5. Health Monitoring & Status Tracking

### Radarr's Indexer Status

```csharp
IndexerStatus {
    LastRssSyncReleaseInfo    // Last RSS sync details
    DisabledTill              // Temporary disable timestamp
    MostRecentFailure         // Last failure details
    EscalationLevel           // Backoff level
    InitialFailure            // First failure in current series
}
```

Features:
- **Exponential backoff** on repeated failures
- **Temporary blocking** with automatic recovery
- **Per-indexer failure tracking**
- **HTTP 429 response handling** with retry-after

### SlipStream's Prowlarr Status

- **Connection-level health only**: Connected/Disconnected
- **15-minute health check interval**
- **No per-indexer status** (relies on Prowlarr)
- **Basic rate limiting** with exponential backoff

### Gap Analysis

| Missing Feature | Impact | Priority |
|-----------------|--------|----------|
| **Per-indexer status tracking** | Cannot identify problematic indexers | Medium |
| **Per-indexer failure history** | Cannot debug search issues | Low |
| **Detailed error messages** | Generic "check Prowlarr" message | Low |

---

## 6. Configuration Options

### Radarr Per-Indexer Settings

| Setting | Purpose |
|---------|---------|
| `EnableRss` | Include in RSS sync |
| `EnableAutomaticSearch` | Include in auto-search |
| `EnableInteractiveSearch` | Include in manual search |
| `Priority` | Dedup preference (1-50) |
| `DownloadClientId` | Override download client |
| `Tags` | Limit to tagged content |
| `MinimumSeeders` | Reject low-seed torrents |
| `SeedRatio` | Required seed ratio |
| `SeedTime` | Required seed time |
| `RequiredFlags` | Required indexer flags (Freeleech, etc.) |
| `Categories` | Search categories |

### SlipStream Global Settings

| Setting | Purpose |
|---------|---------|
| `url` | Prowlarr instance URL |
| `api_key` | Authentication |
| `movie_categories` | Global movie categories |
| `tv_categories` | Global TV categories |
| `timeout` | Request timeout |
| `skip_ssl_verify` | SSL verification |

### Gap Analysis

| Missing Feature | Impact | Priority |
|-----------------|--------|----------|
| **Per-indexer enable/disable** | Cannot exclude indexers | High |
| **Per-indexer priority** | Cannot prefer certain indexers | Medium |
| **Per-indexer categories** | Same categories for all | Low |
| **Per-indexer seed requirements** | Prowlarr handles | Low |
| **Required flags filter** | Cannot filter for freeleech only | Low |
| **Tags** | Cannot limit indexers to content | Medium |

---

## 7. Data Models Comparison

### Release Information

| Field | Radarr | SlipStream | Notes |
|-------|--------|------------|-------|
| `guid` | ✅ | ✅ | Unique identifier |
| `title` | ✅ | ✅ | Release title |
| `size` | ✅ | ✅ | File size |
| `downloadUrl` | ✅ | ✅ | Download URL |
| `infoUrl` | ✅ | ✅ | Info page URL |
| `indexerId` | ✅ | ✅ (always 0) | Indexer ID |
| `indexer` | ✅ | ✅ | Indexer name |
| `indexerPriority` | ✅ | ❌ | Not tracked |
| `seeders` | ✅ | ✅ | Torrent seeders |
| `leechers` | ✅ | ✅ | Torrent leechers |
| `protocol` | ✅ | ✅ | Torrent/Usenet |
| `tmdbId` | ✅ | ✅ | TMDB ID |
| `imdbId` | ✅ | ✅ | IMDB ID |
| `publishDate` | ✅ | ✅ | Release date |
| `languages` | ✅ | ✅ | Detected languages |
| `indexerFlags` | ✅ | ✅ | Freeleech, etc. |
| `minimumRatio` | ✅ | ✅ | Seed ratio requirement |
| `minimumSeedTime` | ✅ | ✅ | Seed time requirement |
| `downloadVolumeFactor` | ✅ | ✅ | Freeleech indicator |
| `uploadVolumeFactor` | ✅ | ✅ | Upload multiplier |

### Gap Analysis

Data models are largely equivalent. The main difference is `indexerPriority` which SlipStream doesn't use because all results come from a single aggregated search.

---

## 8. Recommended Improvements

### High Priority

1. **Per-Indexer Control UI**
   - Display all Prowlarr indexers with enable/disable toggles
   - Store enabled/disabled state in SlipStream database
   - Filter search requests to only include enabled indexers
   - Implementation: Add `prowlarr_indexer_settings` table

2. **Per-Indexer Priority** ✅ IMPLEMENTED
   - Add priority field (1-50) per indexer
   - Use priority when deduplicating results
   - Higher priority indexers' results preferred

### Medium Priority

3. **Per-Indexer Download Client Routing**
   - Allow mapping indexers to specific download clients
   - Useful for separating private/public tracker handling

4. **RSS Sync Support**
   - Implement periodic RSS fetch through Prowlarr
   - Track last sync timestamp
   - Process new releases through auto-search logic

5. **Per-Indexer Tags (Content Type Filtering)** ✅ IMPLEMENTED
   - ~~Allow limiting indexers to specific content~~
   - Implemented as "Content Type" filter: movies-only, series-only, or both
   - Filters which searches use each indexer based on search type

### Low Priority

6. **Per-Indexer Categories** ✅ IMPLEMENTED
   - Allow customizing categories per indexer
   - Custom movie/TV categories supplement global defaults

7. **Per-Indexer Failure Tracking** ✅ IMPLEMENTED
   - Track success/failure rates per indexer
   - Display health status in UI with success/failure counts
   - Shows last failure reason
   - Reset stats capability

8. **Release Validation Cache**
   - Cache releases for 30 minutes
   - Validate before grab

---

## 9. Data Flow Diagrams

### Current SlipStream Flow (Aggregated)

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│ SlipStream  │ ──────► │   Prowlarr   │ ──────► │  Indexers   │
│  (Search)   │  single │  (Aggregates │ parallel│ (1, 2, 3..) │
│             │ request │   + dedup)   │ queries │             │
└─────────────┘         └──────────────┘         └─────────────┘
                               │
                               ▼
                        Combined results
                               │
                               ▼
                        SlipStream scores
                               │
                               ▼
                        User sees results
```

### Radarr Flow (Per-Indexer Proxy)

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Radarr    │ ──────► │   Prowlarr   │ ──────► │  Indexer 1  │
│  (Search)   │ parallel│ (Proxy each  │         │             │
│             │ requests│  indexer)    │         │             │
└─────────────┘         └──────────────┘         └─────────────┘
      │                        │
      │                        ├────────────────► Indexer 2
      │                        │
      │                        ├────────────────► Indexer 3
      │                        │
      ▼                        ▼
Radarr aggregates       Prowlarr proxies
+ dedup + score         + rate limits
```

### Recommended SlipStream Flow (Hybrid)

```
┌─────────────┐  Store enabled/  ┌──────────────┐         ┌─────────────┐
│ SlipStream  │  priority/tags   │   Prowlarr   │ ──────► │  Indexers   │
│  Config DB  │ ◄──────────────► │  (Sync push) │ sync    │ (1, 2, 3..) │
└─────────────┘                  └──────────────┘         └─────────────┘
      │
      │ Filter by enabled indexers
      ▼
┌─────────────┐         ┌──────────────┐
│ SlipStream  │ ──────► │   Prowlarr   │
│  (Search)   │ request │ (Aggregated) │
│ w/ indexers │ w/ ids  │              │
└─────────────┘         └──────────────┘
      │
      ▼
Apply priority dedup
      │
      ▼
SlipStream scoring
```

---

## 10. Summary

SlipStream's Prowlarr integration provides a functional aggregated search experience but lacks the granular control that Radarr users expect. The main gaps are:

1. **No per-indexer control** (enable/disable, priority, tags)
2. **No RSS sync support**
3. **No per-indexer download client routing**
4. **Limited status tracking** (connection-level only)

The aggregated approach is simpler but less flexible. For users migrating from Radarr/Sonarr, the lack of per-indexer control may be surprising.

### Recommended Path Forward

1. **Short term**: Add per-indexer enable/disable and priority
2. **Medium term**: Add per-indexer tags and download client routing
3. **Long term**: Consider Prowlarr application registration for push-based sync

---

*Audit completed: January 2026*
