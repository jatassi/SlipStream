# Prowlarr Integration Implementation Plan

## Overview

This document outlines the implementation plan for Prowlarr integration in SlipStream, based on the [Prowlarr Integration Specification](./Prowlarr-integration-spec.md).

## Assumptions

| # | Assumption | Status | Notes |
|---|------------|--------|-------|
| A1 | Torznab XML parsing can be reused from existing Cardigann code | **DENIED** | Cardigann uses HTML/JSON parsers, not Torznab XML. Must write new Torznab XML parser using Go's `encoding/xml` |
| A2 | The existing search service can be extended with a strategy/adapter pattern | **CONFIRMED** | `SearchService` interface exists in `internal/indexer/search/interfaces.go` with 4 methods. Can implement router pattern |
| A3 | WebSocket events don't need modification (same payload structure) | **CONFIRMED** | Spec 14.1-14.5 states same events broadcast |
| A4 | Notifications don't need modification | **CONFIRMED** | Spec 15.1-15.2 states identical format |
| A5 | Auto-search scheduler doesn't need modification (just routing changes) | **CONFIRMED** | Spec 17.1-17.3 states same logic, different routing |
| A6 | Quality profile scoring algorithm can be reused as-is | **CONFIRMED** | Spec 7.3.3 references SlipStream's full scoring. Scorer in `internal/indexer/scoring/scorer.go` |
| A7 | Grab service can be extended via dependency injection | **CONFIRMED** | `IndexerClientProvider` interface exists. Can inject Prowlarr-aware provider via `SetIndexerService()` |
| A8 | The indexer mode is stored in `prowlarr_config.enabled` | **CONFIRMED** | Spec 4.2.1 defines `enabled` field for this purpose |
| A9 | HTTP client can handle self-signed certs with skip_ssl_verify | **CONFIRMED** | Standard Go pattern: `http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}` |

---

## Phase 1: Database & Core Types

**Dependencies:** None
**Estimated Scope:** Small

### Tasks

#### 1.1 Create Database Migration
**File:** `internal/database/migrations/036_prowlarr_config.sql`

Create the `prowlarr_config` table with all required fields.

**Requirements Covered:**
- 4.1.2: Storage in `prowlarr_config` database table
- 13.1.1: Table name `prowlarr_config`
- 13.1.2.1-13.1.2.12: All database columns (id, enabled, url, api_key, movie_categories, tv_categories, timeout, skip_ssl_verify, capabilities, capabilities_updated_at, created_at, updated_at)

#### 1.2 Create SQLC Queries
**File:** `internal/database/queries/prowlarr_config.sql`

Add queries for CRUD operations on prowlarr_config.

**Requirements Covered:**
- 4.1.2: Storage operations
- 12.1.1: GET prowlarr config
- 12.1.2: PUT prowlarr config

#### 1.3 Define Go Types
**File:** `internal/prowlarr/types.go`

Define core types: `ProwlarrConfig`, `ProwlarrIndexer`, `ProwlarrCapabilities`, `ProwlarrSearchResult`.

**Requirements Covered:**
- 4.2.1-4.2.7: All configuration fields with defaults
- 6.3.1-6.3.4: Indexer display fields (name, protocol, status, capabilities)
- 7.5: Torznab attribute mapping types

---

## Phase 2: Prowlarr Client

**Dependencies:** Phase 1
**Estimated Scope:** Large

### Tasks

#### 2.1 Create HTTP Client Foundation
**File:** `internal/prowlarr/client.go`

Implement base HTTP client with:
- Configurable timeout (default 90s)
- Skip SSL verification option
- API key header injection
- Error handling and logging

**Requirements Covered:**
- 4.2.6: Timeout configuration (default 90)
- 4.2.7: SSL verification toggle (default true = skip)
- 18.1.1: HTTP/HTTPS support
- 18.1.2: SSL permissive by default
- 18.2.1: 90-second timeout
- 11.1.1: Log connection attempts

#### 2.2 Implement Connection Test
**File:** `internal/prowlarr/client.go`

Test connection via capabilities endpoint (`t=caps`).

**Requirements Covered:**
- 4.3.3: Successful connection test required before saving
- 9.1.3: Check type is capabilities (t=caps)
- 9.2.1: Verify Prowlarr is reachable
- 12.1.3: POST test connection endpoint

#### 2.3 Implement Capabilities Fetching
**File:** `internal/prowlarr/capabilities.go`

Parse capabilities XML response, extract categories and search types.

**Requirements Covered:**
- 9.2.2: Fetch supported categories
- 9.2.3: Fetch supported search types
- 9.2.4: Cache for use in searches
- 18.3.2: Standard Torznab/Newznab compliance

#### 2.4 Implement Indexer List Fetching
**File:** `internal/prowlarr/indexers.go`

Fetch and parse Prowlarr's `/api/v1/indexer` endpoint.

**Requirements Covered:**
- 9.3.1: Fetch Prowlarr's indexer list via `/api/v1/indexer`
- 9.3.2: Update displayed indexer statuses
- 12.1.4: GET read-only indexer list from Prowlarr
- 18.3.1: Target Prowlarr API v1

#### 2.5 Implement Search Execution
**File:** `internal/prowlarr/search.go`

Execute searches through Prowlarr's aggregated endpoint.

**Requirements Covered:**
- 3.1.1: Search endpoint `http(s)://<prowlarr-host>:<port>/api`
- 3.1.3: Prowlarr queries all enabled indexers in parallel
- 7.1: Search type mapping (movie → t=movie, TV → t=tvsearch, fallback → t=search)
- 7.2.1: Movie category auto-filtering (2000s)
- 7.2.2: TV category auto-filtering (5000s)
- 7.3.1: Parse Torznab XML response
- 7.3.2: Extract releases with `extended=1`
- 7.3.5: No pagination - single request
- 11.1.2: Log search requests

#### 2.6 Implement Torznab XML Parser
**File:** `internal/prowlarr/torznab.go`

Write new Torznab XML parser using Go's `encoding/xml`. Cannot reuse Cardigann parsers (they handle HTML/JSON, not Torznab XML).

Standard Torznab XML structure:
```xml
<rss><channel><item>
  <title>...</title>
  <guid>...</guid>
  <link>...</link>
  <pubDate>...</pubDate>
  <enclosure url="..." length="..." type="..."/>
  <torznab:attr name="seeders" value="..."/>
  <torznab:attr name="downloadvolumefactor" value="..."/>
  ...
</item></channel></rss>
```

**Requirements Covered:**
- 7.5: All Torznab attributes (title, guid, link, size, pubDate, seeders, peers, category, downloadvolumefactor, uploadvolumefactor, minimumratio, minimumseedtime, indexer)
- 7.6.1: Extract individual indexer name from `indexer` attribute

#### 2.7 Implement Grab/Download
**File:** `internal/prowlarr/grab.go`

Download releases via Prowlarr's download endpoint.

**Requirements Covered:**
- 3.1.2: Download endpoint format
- 8.1.2: Download torrent/NZB via Prowlarr
- 8.1.2.1: URL format `/{indexer_id}/api?t=get&id={guid}` or `/{indexer_id}/download`
- 8.1.2.2: Prowlarr handles indexer authentication
- 11.1.3: Log grab requests

#### 2.8 Implement Rate Limiting
**File:** `internal/prowlarr/ratelimit.go`

Adaptive rate limiting for Prowlarr requests.

**Requirements Covered:**
- 10.2.1: Adaptive rate limiting strategy
- 10.2.2: No initial rate limiting
- 10.2.3: Back off on 429 error
- 10.2.4: Gradually recover after success
- 11.1.4: Log rate limit hits

---

## Phase 3: Service Layer

**Dependencies:** Phase 2
**Estimated Scope:** Medium

### Tasks

#### 3.1 Create Prowlarr Service
**File:** `internal/prowlarr/service.go`

Main service coordinating Prowlarr operations, config management, caching.

**Requirements Covered:**
- 4.1.3: API at `/api/v1/indexers/prowlarr`
- 4.3.1: URL validation
- 4.3.2: API key required validation
- 7.4.1: 5-minute search result cache
- 7.4.2: Cache key is search criteria hash
- 7.4.3: Clear cache on manual refresh
- 18.2.2: 5-minute cache
- 11.1.5: Log configuration changes

#### 3.2 Implement Mode Management
**File:** `internal/prowlarr/mode.go`

Manage indexer mode state (slipstream/prowlarr).

**Requirements Covered:**
- 2.1.2: SlipStream mode default for new installations
- 2.2.2: Prowlarr mode disables internal indexer management
- 5.1.1: Internal indexers hidden/disabled in Prowlarr mode
- 5.2.1: Internal indexers re-enabled on switch back
- 5.2.2: Prowlarr config preserved but inactive
- 5.3.1: Dev mode forces SlipStream mode
- 5.3.2: Mock indexer available in dev mode
- 12.2.1: GET current indexer mode
- 12.2.2: PUT set indexer mode

#### 3.3 Modify Indexer Service for Mode Awareness
**File:** `internal/indexer/service.go` (modify)

Add mode checking to existing indexer operations.

**Requirements Covered:**
- 2.1.3: Current behavior unchanged in SlipStream mode
- 2.1.4: SlipStream manages complete indexer flow via Cardigann
- 5.1.2: Indexer management UI replaced with Prowlarr configuration
- 12.3.1: GET /indexers returns internal indexers (SlipStream) or empty (Prowlarr)

---

## Phase 4: Search Integration

**Dependencies:** Phase 3
**Estimated Scope:** Medium

### Tasks

#### 4.1 Create Search Router/Strategy
**File:** `internal/indexer/search/router.go`

Route searches based on active indexer mode.

**Requirements Covered:**
- 2.2.4: All searches route through Prowlarr's aggregated endpoint
- 12.3.2: GET /search/* routes through appropriate backend
- 17.3: Auto-search routes through Prowlarr

#### 4.2 Integrate Scoring Algorithm
**File:** `internal/prowlarr/scoring.go`

Apply SlipStream's scoring to Prowlarr results.

**Requirements Covered:**
- 7.3.3.1: Quality profile matching
- 7.3.3.2: Seeder count (logarithmic scale)
- 7.3.3.3: Freeleech bonus
- 7.3.3.4: Age scoring
- 7.3.3.5: Language detection from title
- 7.3.4: Trust Prowlarr's deduplication (no additional dedup)

#### 4.3 Implement Search Result Processing
**File:** `internal/prowlarr/results.go`

Transform Prowlarr results to SlipStream format with enrichment.

**Requirements Covered:**
- 7.6.2: Display all standard release information
- 7.6.3: Show freeleech/ratio requirements when available
- 16.1: Both protocols displayed together
- 16.3: Torrent-specific attributes only for torrents

---

## Phase 5: Grab Integration

**Dependencies:** Phase 4
**Estimated Scope:** Small

### Tasks

#### 5.1 Modify Grab Service for Mode Awareness
**File:** `internal/indexer/grab/service.go` (modify)

Route grabs through Prowlarr or internal indexers based on mode.

**Requirements Covered:**
- 8.1.1: User selects release to grab
- 8.1.3: Select download client based on protocol
- 8.1.4: Send to download client
- 8.1.5: Record in grab history with indexer name
- 8.1.6: Broadcast WebSocket event
- 8.3.1: Automatic protocol-based selection
- 8.3.2: Torrent releases → torrent client
- 8.3.3: Usenet releases → usenet client
- 12.3.3: POST /search/grab uses appropriate backend
- 16.2: Download client auto-selected based on protocol

#### 5.2 Implement Grab Retry Logic
**File:** `internal/prowlarr/grab.go` (extend)

Add retry logic for Prowlarr grabs.

**Requirements Covered:**
- 8.2.1: Retry once automatically after short delay
- 8.2.2: If retry fails, show error message to user
- 8.2.3: Let user try alternative releases manually

---

## Phase 6: Health & Monitoring

**Dependencies:** Phase 3
**Estimated Scope:** Medium

### Tasks

#### 6.1 Create Prowlarr Health Checker
**File:** `internal/prowlarr/health.go`

Monitor Prowlarr connection health.

**Requirements Covered:**
- 9.1.1: 15-minute check interval
- 9.1.2: Also triggers on page load, manual refresh
- 9.5.1: Integrated with SlipStream's health monitoring system
- 9.5.4: Health check failures recorded in health service
- 18.2.3: 15-minute capability/indexer refresh

#### 6.2 Register Health Task with Scheduler
**File:** `internal/scheduler/tasks/prowlarrhealth.go`

Schedule periodic Prowlarr health checks.

**Requirements Covered:**
- 9.1.1: Every 15 minutes
- 9.3.3: Every 15 min + on page load + manual refresh

#### 6.3 Implement Post-Search Health Check
**File:** `internal/prowlarr/search.go` (extend)

Check indexer status after each search.

**Requirements Covered:**
- 9.4.1: After each search, check Prowlarr's indexer status API
- 9.4.2: If any indexers are failing/disabled, show warning
- 9.4.3: Warning message text

#### 6.4 Integrate with Health Service
**File:** `internal/health/service.go` (modify)

Add Prowlarr as a health category.

**Requirements Covered:**
- 9.5.1: Prowlarr connection status in health monitoring
- 9.5.2: Toast notifications for connection errors
- 9.5.3: Status indicator on Indexers page
- 12.1.5: GET Prowlarr connection status

---

## Phase 7: API Endpoints

**Dependencies:** Phase 6
**Estimated Scope:** Small

### Tasks

#### 7.1 Create Prowlarr Handlers
**File:** `internal/prowlarr/handlers.go`

HTTP handlers for Prowlarr configuration API.

**Requirements Covered:**
- 12.1.1: GET /api/v1/indexers/prowlarr
- 12.1.2: PUT /api/v1/indexers/prowlarr
- 12.1.3: POST /api/v1/indexers/prowlarr/test
- 12.1.4: GET /api/v1/indexers/prowlarr/indexers
- 12.1.5: GET /api/v1/indexers/prowlarr/status

#### 7.2 Create Mode Handlers
**File:** `internal/prowlarr/handlers.go` (extend)

HTTP handlers for mode management.

**Requirements Covered:**
- 12.2.1: GET /api/v1/indexers/mode
- 12.2.2: PUT /api/v1/indexers/mode

#### 7.3 Register Routes
**File:** `internal/api/server.go` (modify)

Wire up new endpoints under `/api/v1/indexers`.

**Requirements Covered:**
- 4.1.3: API at /api/v1/indexers/prowlarr

---

## Phase 8: Error Handling & Logging

**Dependencies:** Phase 7
**Estimated Scope:** Small

### Tasks

#### 8.1 Implement Connection Error Handling
**File:** `internal/prowlarr/errors.go`

Structured error types for Prowlarr operations.

**Requirements Covered:**
- 10.1.1: Fail gracefully with error message
- 10.1.2: Toast notification + update status indicator
- 10.1.3: No automatic retry or queuing

#### 8.2 Implement Search Error Handling
**File:** `internal/prowlarr/search.go` (extend)

Handle search failures gracefully.

**Requirements Covered:**
- 10.3.1: Show error toast to user
- 10.3.2: Return empty results
- 10.3.3: Log error details (verbose in dev mode)

#### 8.3 Configure Logging Levels
**File:** `internal/prowlarr/logging.go`

Environment-aware logging.

**Requirements Covered:**
- 11 (table): Verbose debug in dev, standard in production
- 11.1.1-11.1.5: All logged events

---

## Phase 9: Frontend - Types & API

**Dependencies:** Phase 7
**Estimated Scope:** Small

### Tasks

#### 9.1 Create TypeScript Types
**File:** `web/src/types/prowlarr.ts`

Define TypeScript interfaces for Prowlarr integration.

**Requirements Covered:**
- 4.2.1-4.2.7: Configuration field types
- 6.3.1-6.3.4: Indexer display types

#### 9.2 Create API Client
**File:** `web/src/api/prowlarr.ts`

API client functions for Prowlarr endpoints.

**Requirements Covered:**
- 12.1.1-12.1.5: All Prowlarr API endpoints
- 12.2.1-12.2.2: Mode endpoints

#### 9.3 Create React Query Hooks
**File:** `web/src/hooks/useProwlarr.ts`

React Query hooks for Prowlarr data fetching and mutations.

**Requirements Covered:**
- All API operations with proper cache invalidation

---

## Phase 10: Frontend - UI Components

**Dependencies:** Phase 9
**Estimated Scope:** Medium

### Tasks

#### 10.1 Create Mode Toggle Component
**File:** `web/src/components/indexers/IndexerModeToggle.tsx`

Radio button toggle for indexer mode selection.

**Requirements Covered:**
- 4.1.1: Mode toggle at top of Indexers page
- 6.1: Mode selection UI (wireframe)
- 2.1.1: SlipStream mode labeled "Experimental"
- 2.2.1: Prowlarr mode no label (unlabeled option)

#### 10.2 Create Prowlarr Config Form
**File:** `web/src/components/indexers/ProwlarrConfigForm.tsx`

Form for Prowlarr configuration settings.

**Requirements Covered:**
- 6.2: Prowlarr mode content layout (wireframe)
- 4.2.1-4.2.7: All configuration fields in form
- 4.2.3: API key displayed in plain text for easy copying
- 4.3.1-4.3.3: Validation and test before save

#### 10.3 Create Prowlarr Indexer List
**File:** `web/src/components/indexers/ProwlarrIndexerList.tsx`

Read-only display of Prowlarr indexers.

**Requirements Covered:**
- 6.2: Indexer table in Prowlarr mode
- 6.3.1: Display indexer name
- 6.3.2: Display protocol (Torrent/Usenet)
- 6.3.3: Display status (Healthy/Warning/Disabled)
- 6.3.4: Display capabilities

#### 10.4 Create Category Selector Component
**File:** `web/src/components/indexers/CategorySelector.tsx`

Multi-select for movie/TV categories.

**Requirements Covered:**
- 4.2.4: Movie categories configuration
- 4.2.5: TV categories configuration
- 6.2: Category configuration in form (wireframe)

#### 10.5 Modify Indexers Page
**File:** `web/src/routes/settings/indexers.tsx` (modify)

Add conditional rendering based on mode.

**Requirements Covered:**
- 5.1.2: Indexer management UI replaced with Prowlarr configuration
- 6.1-6.2: Complete UI layout per wireframes

#### 10.6 Add Status Indicators
**File:** `web/src/components/indexers/ProwlarrStatus.tsx`

Connection status display component.

**Requirements Covered:**
- 9.5.3: Status indicator on Indexers page
- 6.2: Status display in Prowlarr mode (wireframe)

---

## Phase 11: Integration & Testing

**Dependencies:** Phase 10
**Estimated Scope:** Small

### Tasks

#### 11.1 Wire Up All Services
**File:** `internal/api/server.go` (modify)

Connect all new services in server initialization.

**Requirements Covered:**
- All service integration

#### 11.2 Add WebSocket Event Handling
**File:** `web/src/stores/websocket.ts` (modify if needed)

Handle any new Prowlarr-specific events.

**Requirements Covered:**
- 14.1-14.5: WebSocket events (same format)

#### 11.3 Verify Auto-Search Integration
Test that auto-search works correctly in Prowlarr mode.

**Requirements Covered:**
- 17.1: Same scheduling logic
- 17.2: Same quality profile evaluation
- 17.3: Routes through Prowlarr
- 5.1.3: Auto-search schedules continue unchanged
- 5.2.3: Schedules unchanged on mode switch

#### 11.4 Verify Notification Integration
Test that notifications work correctly in Prowlarr mode.

**Requirements Covered:**
- 15.1: External notifications use identical format
- 15.2: No indication of whether release came from Prowlarr

#### 11.5 Verify History Preservation
Test that history is preserved across mode switches.

**Requirements Covered:**
- 5.1.4: All search/grab history preserved
- 5.2.4: History preserved on switch back

---

## Requirement Coverage Audit

| Requirement | Phase | Task | Notes |
|-------------|-------|------|-------|
| 1.1-1.3 | N/A | N/A | Background context only |
| 2.1.1 | 10 | 10.1 | "Experimental" label |
| 2.1.2 | 3 | 3.2 | Default mode |
| 2.1.3 | 3 | 3.3 | Unchanged behavior |
| 2.1.4 | 3 | 3.3 | Cardigann flow |
| 2.2.1 | 10 | 10.1 | No label |
| 2.2.2 | 3 | 3.2 | Disable internal |
| 2.2.3 | 2 | 2.5 | Aggregated mode |
| 2.2.4 | 4 | 4.1 | Route through Prowlarr |
| 3.1.1 | 2 | 2.5 | Search endpoint |
| 3.1.2 | 2 | 2.7 | Download endpoint |
| 3.1.3 | 2 | 2.5 | Parallel queries |
| 3.1.4 | 3 | 3.2 | No individual configs |
| 3.2.1-3.2.4 | N/A | N/A | Rationale only |
| 4.1.1 | 10 | 10.1 | Mode toggle UI |
| 4.1.2 | 1 | 1.1 | Database table |
| 4.1.3 | 7 | 7.1, 7.3 | API endpoints |
| 4.2.1-4.2.7 | 1, 10 | 1.1, 1.3, 10.2 | Config fields |
| 4.3.1-4.3.3 | 3, 10 | 3.1, 10.2 | Validation |
| 5.1.1-5.1.4 | 3 | 3.2, 3.3 | Switch to Prowlarr |
| 5.2.1-5.2.4 | 3 | 3.2 | Switch to SlipStream |
| 5.3.1-5.3.2 | 3 | 3.2 | Dev mode override |
| 6.1-6.2 | 10 | 10.1-10.6 | UI wireframes |
| 6.3.1-6.3.4 | 10 | 10.3 | Indexer display |
| 7.1-7.2.2 | 2, 4 | 2.5, 4.2 | Search types/categories |
| 7.3.1-7.3.5 | 2, 4 | 2.5, 2.6, 4.2 | Result processing |
| 7.4.1-7.4.3 | 3 | 3.1 | Search caching |
| 7.5 | 2 | 2.6 | Torznab attributes |
| 7.6.1-7.6.3 | 4 | 4.3 | Result display |
| 8.1.1-8.1.6 | 5 | 5.1 | Grab flow |
| 8.2.1-8.2.3 | 5 | 5.2 | Retry logic |
| 8.3.1-8.3.3 | 5 | 5.1 | Client selection |
| 9.1.1-9.1.3 | 6 | 6.1, 6.2 | Periodic checks |
| 9.2.1-9.2.4 | 2 | 2.2, 2.3 | Capabilities |
| 9.3.1-9.3.3 | 2, 6 | 2.4, 6.2 | Indexer refresh |
| 9.4.1-9.4.3 | 6 | 6.3 | Post-search check |
| 9.5.1-9.5.4 | 6 | 6.4 | Health integration |
| 10.1.1-10.1.3 | 8 | 8.1 | Connection errors |
| 10.2.1-10.2.4 | 2 | 2.8 | Rate limiting |
| 10.3.1-10.3.3 | 8 | 8.2 | Search errors |
| 11.1.1-11.1.5 | 8 | 8.3 | Logging |
| 12.1.1-12.1.5 | 7 | 7.1 | Prowlarr API |
| 12.2.1-12.2.2 | 7 | 7.2 | Mode API |
| 12.3.1-12.3.3 | 3, 4, 5 | 3.3, 4.1, 5.1 | Mode-aware endpoints |
| 13.1.1-13.1.2.12 | 1 | 1.1 | Database schema |
| 14.1-14.5 | 11 | 11.2 | WebSocket events |
| 15.1-15.2 | 11 | 11.4 | Notifications |
| 16.1-16.3 | 4, 5 | 4.3, 5.1 | Protocol support |
| 17.1-17.3 | 11 | 11.3 | Auto-search |
| 18.1.1-18.1.3 | 2 | 2.1 | Security |
| 18.2.1-18.2.3 | 2, 3, 6 | 2.1, 3.1, 6.2 | Performance |
| 18.3.1-18.3.2 | 2 | 2.3, 2.4 | Compatibility |
| 19.1-20.2 | N/A | N/A | Reference only |

**All numbered requirements are covered.**

---

## File Changes Summary

### New Files
- `internal/database/migrations/036_prowlarr_config.sql`
- `internal/database/queries/prowlarr_config.sql`
- `internal/prowlarr/types.go`
- `internal/prowlarr/client.go`
- `internal/prowlarr/capabilities.go`
- `internal/prowlarr/indexers.go`
- `internal/prowlarr/search.go`
- `internal/prowlarr/torznab.go`
- `internal/prowlarr/grab.go`
- `internal/prowlarr/ratelimit.go`
- `internal/prowlarr/service.go`
- `internal/prowlarr/mode.go`
- `internal/prowlarr/health.go`
- `internal/prowlarr/handlers.go`
- `internal/prowlarr/errors.go`
- `internal/prowlarr/logging.go`
- `internal/prowlarr/scoring.go`
- `internal/prowlarr/results.go`
- `internal/scheduler/tasks/prowlarrhealth.go`
- `internal/indexer/search/router.go`
- `web/src/types/prowlarr.ts`
- `web/src/api/prowlarr.ts`
- `web/src/hooks/useProwlarr.ts`
- `web/src/components/indexers/IndexerModeToggle.tsx`
- `web/src/components/indexers/ProwlarrConfigForm.tsx`
- `web/src/components/indexers/ProwlarrIndexerList.tsx`
- `web/src/components/indexers/CategorySelector.tsx`
- `web/src/components/indexers/ProwlarrStatus.tsx`

### Modified Files
- `internal/indexer/service.go` - Mode awareness
- `internal/indexer/grab/service.go` - Prowlarr grab routing
- `internal/health/service.go` - Prowlarr health category
- `internal/api/server.go` - Route registration, service wiring
- `web/src/routes/settings/indexers.tsx` - Conditional rendering
- `web/src/stores/websocket.ts` - Event handling (if needed)

---

## Implementation Order

```
Phase 1 (Database) ─────────────────────────────────────────────────────┐
                                                                        │
Phase 2 (Prowlarr Client) ──────────────────────────────────────────────┤
                                                                        │
Phase 3 (Service Layer) ────────────────────────────────────────────────┤
          │                                                             │
          ├── Phase 4 (Search Integration) ─────────────────────────────┤
          │                                                             │
          └── Phase 5 (Grab Integration) ───────────────────────────────┤
                    │                                                   │
                    └── Phase 6 (Health & Monitoring) ──────────────────┤
                                  │                                     │
                                  └── Phase 7 (API Endpoints) ──────────┤
                                                │                       │
                                                └── Phase 8 (Errors) ───┤
                                                                        │
Phase 9 (Frontend Types & API) ─────────────────────────────────────────┤
                    │                                                   │
                    └── Phase 10 (Frontend UI) ─────────────────────────┤
                                                                        │
Phase 11 (Integration & Testing) ───────────────────────────────────────┘
```

Phases 1-3 must be sequential. Phases 4-6 can be parallelized after Phase 3. Phase 9 can start after Phase 7. Phase 10 requires Phase 9. Phase 11 is final integration.
