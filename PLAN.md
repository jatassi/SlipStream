# Phase 3: Metadata Integration - Implementation Plan

## Overview

Implement TMDB and TVDB API clients to fetch movie and TV show metadata, with caching and artwork downloading capabilities.

---

## Components to Implement

### 1. Configuration Updates

**File: `internal/config/config.go`**
- Add `MetadataConfig` struct with TMDB and TVDB settings
- Include API keys, base URLs, timeouts, cache duration, and image base URLs

**File: `configs/config.example.yaml`**
- Add metadata configuration section with documented options

### 2. TMDB Client

**Directory: `internal/metadata/tmdb/`**

| File | Purpose |
|------|---------|
| `client.go` | HTTP client, API calls (SearchMovies, GetMovie, SearchTV, GetTV) |
| `models.go` | API response structs (search results, movie details, TV details) |
| `client_test.go` | Unit tests with mocked HTTP responses |

**Key Features:**
- Implements `metadata.Provider` interface
- Rate limiting awareness (TMDB allows 40 req/10sec)
- Context support for cancellation
- Proper error handling for API errors, rate limits, invalid keys

### 3. TVDB Client

**Directory: `internal/metadata/tvdb/`**

| File | Purpose |
|------|---------|
| `client.go` | HTTP client with JWT auth, API calls |
| `models.go` | API response structs |
| `client_test.go` | Unit tests with mocked HTTP responses |

**Key Features:**
- JWT token authentication (login, token refresh)
- Implements `metadata.Provider` interface
- Token caching to avoid repeated logins

### 4. Metadata Service (Orchestration Layer)

**File: `internal/metadata/service.go`**

- Combines TMDB + TVDB providers
- Caching layer (in-memory with TTL)
- Fallback logic (try TMDB first, fall back to TVDB for TV)
- Artwork URL generation

### 5. Artwork Downloader

**File: `internal/metadata/artwork.go`**

- Download poster/backdrop images
- Save to configurable directory (`data/artwork/`)
- Naming convention: `{type}_{id}_{size}.jpg`
- Resize/optimization (optional, defer if complex)

### 6. API Handlers

**File: `internal/metadata/handlers.go`**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/metadata/movie/search` | GET | Search movies by query |
| `/api/v1/metadata/movie/:id` | GET | Get movie details by TMDB ID |
| `/api/v1/metadata/series/search` | GET | Search TV series by query |
| `/api/v1/metadata/series/:id` | GET | Get series details by TVDB/TMDB ID |

### 7. Integration with Existing Services

- Update `movies.Service` to optionally fetch metadata on create
- Update `tv.Service` to optionally fetch metadata on create
- Add refresh metadata endpoints

---

## File Structure

```
internal/metadata/
├── provider.go          # Existing interface (already exists)
├── service.go           # Orchestration service
├── artwork.go           # Artwork downloader
├── handlers.go          # HTTP handlers
├── cache.go             # In-memory cache with TTL
├── tmdb/
│   ├── client.go        # TMDB API client
│   ├── models.go        # TMDB response types
│   └── client_test.go   # Tests
└── tvdb/
    ├── client.go        # TVDB API client
    ├── models.go        # TVDB response types
    └── client_test.go   # Tests
```

---

## Implementation Order

1. **Config** - Add metadata configuration structs and defaults
2. **TMDB Client** - Implement client with tests (movies + TV support)
3. **TVDB Client** - Implement client with tests (TV-focused)
4. **Cache** - Simple TTL cache for API responses
5. **Service** - Orchestration layer combining providers
6. **Artwork** - Image downloading functionality
7. **Handlers** - REST API endpoints
8. **Wire Up** - Integrate into server.go and main.go
9. **Update TODO.md** - Mark phase complete

---

## API Details

### TMDB API (v3)
- Base URL: `https://api.themoviedb.org/3`
- Auth: API key as query parameter (`?api_key=XXX`)
- Image Base: `https://image.tmdb.org/t/p/`
- Key endpoints:
  - `GET /search/movie?query=...`
  - `GET /movie/{id}`
  - `GET /search/tv?query=...`
  - `GET /tv/{id}`

### TVDB API (v4)
- Base URL: `https://api4.thetvdb.com/v4`
- Auth: Bearer token (obtained via POST /login)
- Key endpoints:
  - `POST /login` (get token)
  - `GET /search?query=...&type=series`
  - `GET /series/{id}`
  - `GET /series/{id}/episodes`

---

## Testing Strategy

- **Unit Tests**: Mock HTTP responses for API clients
- **Integration Tests**: Test service layer with mocked providers
- **Test Coverage**: Target 70%+ for new code

---

## Estimated Scope

- ~15-20 new files
- ~1500-2000 lines of Go code
- Full test coverage for API clients
