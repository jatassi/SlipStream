# Availability Tracking & Task Scheduler Implementation Plan

## Overview
Add release availability tracking for movies and TV (episodes, seasons, series) based on release/air dates, plus a modular task scheduling system using gocron for daily midnight refresh and future scheduled tasks.

**Availability Logic:**
- **Movie**: `released = true` when `release_date <= today`
- **Episode**: `released = true` when `air_date <= today`
- **Season**: `released = true` when ALL episodes in season have aired
- **Series**: `released = true` when ALL seasons are released

This 3-tier TV availability will later govern automated search parameters (individual episode, season box set, or complete series box set).

---

## Phase 1: Database Schema Changes

### Create Migration
**File:** `internal/database/migrations/007_add_availability.sql`

```sql
-- +goose Up
-- Movies: released if release_date is in the past
ALTER TABLE movies ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Episodes: released if air_date is in the past
ALTER TABLE episodes ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Seasons: released if ALL episodes in season have aired
ALTER TABLE seasons ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Series: released if ALL seasons are released
ALTER TABLE series ADD COLUMN released INTEGER NOT NULL DEFAULT 0;

-- Indexes for efficient availability queries
CREATE INDEX idx_movies_released ON movies(released);
CREATE INDEX idx_episodes_released ON episodes(released);
CREATE INDEX idx_seasons_released ON seasons(released);
CREATE INDEX idx_series_released ON series(released);

-- +goose Down
-- SQLite requires table recreation to drop columns (handled separately)
```

### Update SQLC Queries

**File:** `internal/database/queries/movies.sql`
- Update `CreateMovie` to include `released` column
- Add `UpdateMovieReleased` query for bulk updates:
```sql
-- name: UpdateMoviesReleasedByDate :execresult
UPDATE movies SET released = 1, updated_at = CURRENT_TIMESTAMP
WHERE released = 0 AND release_date IS NOT NULL AND release_date <= date('now');
```

**File:** `internal/database/queries/episodes.sql`
- Update `CreateEpisode` to include `released` column
- Add bulk update query:
```sql
-- name: UpdateEpisodesReleasedByDate :execresult
UPDATE episodes SET released = 1
WHERE released = 0 AND air_date IS NOT NULL AND air_date <= date('now');
```

**File:** `internal/database/queries/seasons.sql`
- Add query to update season availability based on episodes:
```sql
-- name: UpdateSeasonReleasedFromEpisodes :exec
UPDATE seasons SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM episodes WHERE episodes.series_id = seasons.series_id
    AND episodes.season_number = seasons.season_number
) WHERE id = ?;

-- name: UpdateAllSeasonsReleased :execresult
UPDATE seasons SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM episodes WHERE episodes.series_id = seasons.series_id
    AND episodes.season_number = seasons.season_number
);
```

**File:** `internal/database/queries/series.sql`
- Add query to update series availability based on seasons:
```sql
-- name: UpdateSeriesReleasedFromSeasons :exec
UPDATE series SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM seasons WHERE seasons.series_id = series.id
), updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateAllSeriesReleased :execresult
UPDATE series SET released = (
    SELECT CASE WHEN COUNT(*) = SUM(released) AND COUNT(*) > 0 THEN 1 ELSE 0 END
    FROM seasons WHERE seasons.series_id = series.id
), updated_at = CURRENT_TIMESTAMP;
```

Run: `sqlc generate`

---

## Phase 2: Task Scheduler Infrastructure

### Install gocron
```bash
go get github.com/go-co-op/gocron/v2
```

### Rewrite Scheduler Package
**File:** `internal/scheduler/scheduler.go` (complete rewrite)

```go
package scheduler

import (
    "context"
    "sync"
    "time"

    "github.com/go-co-op/gocron/v2"
    "github.com/rs/zerolog"
)

type TaskFunc func(ctx context.Context) error

type TaskConfig struct {
    ID          string
    Name        string
    Description string
    Cron        string           // Cron expression: "0 0 * * *" for midnight
    Func        TaskFunc
    RunOnStart  bool             // Execute immediately on startup
}

type TaskInfo struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Cron        string    `json:"cron"`
    LastRun     time.Time `json:"lastRun,omitempty"`
    NextRun     time.Time `json:"nextRun,omitempty"`
    Running     bool      `json:"running"`
}

type Scheduler struct {
    gocron  gocron.Scheduler
    logger  zerolog.Logger
    tasks   map[string]*taskEntry
    mu      sync.RWMutex
}

type taskEntry struct {
    config  TaskConfig
    job     gocron.Job
    lastRun time.Time
    running bool
}

func New(logger zerolog.Logger) (*Scheduler, error)
func (s *Scheduler) RegisterTask(config TaskConfig) error
func (s *Scheduler) Start() error
func (s *Scheduler) Stop() error
func (s *Scheduler) RunNow(taskID string) error
func (s *Scheduler) ListTasks() []TaskInfo
```

---

## Phase 3: Availability Service

### Create Availability Package
**New File:** `internal/availability/service.go`

```go
package availability

type Service struct {
    db      *sql.DB
    queries *sqlc.Queries
    logger  zerolog.Logger
}

func NewService(db *sql.DB, logger zerolog.Logger) *Service

// RefreshAll updates availability for all media (run order matters)
func (s *Service) RefreshAll(ctx context.Context) error {
    // 1. Update movies by release_date
    // 2. Update episodes by air_date
    // 3. Update seasons from episodes
    // 4. Update series from seasons
}

// For single-item updates after add/edit
func (s *Service) SetMovieAvailability(ctx context.Context, movieID int64, releaseDate *time.Time) error
func (s *Service) SetEpisodeAvailability(ctx context.Context, episodeID int64, airDate *time.Time) error
func (s *Service) RecalculateSeasonAvailability(ctx context.Context, seriesID int64, seasonNumber int) error
func (s *Service) RecalculateSeriesAvailability(ctx context.Context, seriesID int64) error
```

### Register Availability Task
**New File:** `internal/scheduler/tasks/availability.go`

```go
package tasks

func RegisterAvailabilityTask(sched *scheduler.Scheduler, svc *availability.Service) error {
    return sched.RegisterTask(scheduler.TaskConfig{
        ID:          "availability-refresh",
        Name:        "Refresh Media Availability",
        Description: "Updates released status based on release/air dates",
        Cron:        "0 0 * * *",  // Midnight daily
        RunOnStart:  true,
        Func:        svc.RefreshAll,
    })
}
```

---

## Phase 4: Update Domain Models

### Movie Model
**File:** `internal/library/movies/movie.go`
- Add `Released bool` field to `Movie` struct

**File:** `internal/library/movies/service.go`
- Update `rowToMovie()` to map `released` field
- Update `Create()` to set initial availability based on release_date

### TV Models
**File:** `internal/library/tv/series.go`
- Add `Released bool` to `Series`, `Season`, `Episode` structs

**File:** `internal/library/tv/service.go`
- Update row mapping functions
- Update `CreateEpisode()` to set initial availability

### Library Manager Integration
**File:** `internal/library/librarymanager/service.go`
- Inject availability service
- After `AddMovie`: calculate and set availability
- After `AddSeries`: cascade availability calculation

---

## Phase 5: Scheduler API Endpoints

### Scheduler Handlers
**New File:** `internal/api/handlers/scheduler.go`

```go
// GET /api/v1/scheduler/tasks - List all scheduled tasks
func (h *Handler) ListTasks(c echo.Context) error

// POST /api/v1/scheduler/tasks/{id}/run - Manually trigger a task
func (h *Handler) RunTask(c echo.Context) error
```

### Server Integration
**File:** `internal/api/server.go`
- Initialize scheduler
- Register availability task
- Start scheduler after all services
- Add `/scheduler` routes
- Stop scheduler on shutdown

**File:** `cmd/slipstream/main.go`
- Ensure graceful shutdown includes scheduler stop

---

## Phase 6: Frontend Updates

### Update Types
**File:** `web/src/types/movie.ts`
```typescript
export interface Movie {
  // ... existing fields
  released: boolean
}
```

**File:** `web/src/types/series.ts`
```typescript
export interface Series {
  released: boolean
  // ...
}

export interface Season {
  released: boolean
  // ...
}

export interface Episode {
  released: boolean
  // ...
}
```

---

## Files Summary

### New Files (4)
- `internal/database/migrations/007_add_availability.sql`
- `internal/availability/service.go`
- `internal/scheduler/tasks/availability.go`
- `internal/api/handlers/scheduler.go`

### Modified Files (13)
- `internal/scheduler/scheduler.go` (complete rewrite)
- `internal/database/queries/movies.sql`
- `internal/database/queries/episodes.sql`
- `internal/database/queries/seasons.sql`
- `internal/database/queries/series.sql`
- `internal/library/movies/movie.go`
- `internal/library/movies/service.go`
- `internal/library/tv/series.go`
- `internal/library/tv/service.go`
- `internal/library/librarymanager/service.go`
- `internal/api/server.go`
- `cmd/slipstream/main.go`
- `go.mod` (add gocron/v2)
- `web/src/types/movie.ts`
- `web/src/types/series.ts`

---