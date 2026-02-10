package availability

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/testutil"
)

// Tests for the availability/status refresh service.
// Spec: docs/status-consolidation.md - "Scheduler Changes > Status Refresh"

func TestRefreshMovies_UnreleasedToMissing(t *testing.T) {
	// Spec: "Movies: Update status from unreleased → missing where
	//   release_date IS NOT NULL AND release_date <= date('now') and status = 'unreleased'"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create a movie with past release date and status = unreleased
	pastDate := time.Now().AddDate(0, 0, -7)
	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Past Movie",
		SortTitle:   "Past Movie",
		Status:      "unreleased",
		Monitored:   1,
		ReleaseDate: sql.NullTime{Time: pastDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	updated, err := service.RefreshMovies(ctx)
	if err != nil {
		t.Fatalf("RefreshMovies() error = %v", err)
	}

	if updated != 1 {
		t.Errorf("RefreshMovies() updated = %d, want 1", updated)
	}
}

func TestRefreshMovies_SameDayRelease(t *testing.T) {
	// Regression: the Go SQLite driver stores time.Time in RFC3339 format with
	// timezone offset (e.g. "2026-02-10T00:00:00-07:00"). A raw string comparison
	// against date('now') ("2026-02-10") fails on the release day because 'T'
	// extends past the date-only string. The fix uses substr(col, 1, 10) to
	// extract the date portion before comparing.
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	today := time.Now()
	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Today Movie",
		SortTitle:   "Today Movie",
		Status:      "unreleased",
		Monitored:   1,
		ReleaseDate: sql.NullTime{Time: today, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	updated, err := service.RefreshMovies(ctx)
	if err != nil {
		t.Fatalf("RefreshMovies() error = %v", err)
	}

	if updated != 1 {
		t.Errorf("RefreshMovies() updated = %d, want 1 (same-day release must transition)", updated)
	}
}

func TestRefreshMovies_FutureNotChanged(t *testing.T) {
	// Movies with future release dates should NOT be changed
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	futureDate := time.Now().AddDate(0, 0, 7)
	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Future Movie",
		SortTitle:   "Future Movie",
		Status:      "unreleased",
		Monitored:   1,
		ReleaseDate: sql.NullTime{Time: futureDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	updated, err := service.RefreshMovies(ctx)
	if err != nil {
		t.Fatalf("RefreshMovies() error = %v", err)
	}

	if updated != 0 {
		t.Errorf("RefreshMovies() should not update future movies, got %d", updated)
	}
}

func TestRefreshMovies_OnlyAffectsUnreleased(t *testing.T) {
	// Spec: Only movies with status='unreleased' should be affected.
	// Movies with status='downloading', 'available', etc. should NOT change.
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(0, 0, -7)

	// Create movie with 'available' status and past date - should NOT be touched
	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Available Movie",
		SortTitle:   "Available Movie",
		Status:      "available",
		Monitored:   1,
		ReleaseDate: sql.NullTime{Time: pastDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	// Create movie with 'downloading' status and past date - should NOT be touched
	_, err = queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Downloading Movie",
		SortTitle:   "Downloading Movie",
		Status:      "downloading",
		Monitored:   1,
		ReleaseDate: sql.NullTime{Time: pastDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	updated, err := service.RefreshMovies(ctx)
	if err != nil {
		t.Fatalf("RefreshMovies() error = %v", err)
	}

	if updated != 0 {
		t.Errorf("RefreshMovies() should not affect non-unreleased movies, updated %d", updated)
	}
}

func TestRefreshMovies_NoReleaseDateNotChanged(t *testing.T) {
	// Movies without release dates should NOT be changed
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:     "No Date Movie",
		SortTitle: "No Date Movie",
		Status:    "unreleased",
		Monitored: 1,
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	updated, err := service.RefreshMovies(ctx)
	if err != nil {
		t.Fatalf("RefreshMovies() error = %v", err)
	}

	if updated != 0 {
		t.Errorf("RefreshMovies() should not affect movies without release dates, updated %d", updated)
	}
}

func TestRefreshEpisodes_UnreleasedToMissing(t *testing.T) {
	// Spec: "Episodes: Update status from unreleased → missing where
	//   air_date IS NOT NULL AND air_date <= datetime('now') and status = 'unreleased'"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create series, season, and episode
	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "Test Series",
		ProductionStatus: "continuing",
		Monitored:        1,
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    1,
	})

	pastDate := time.Now().AddDate(0, 0, -7)
	_, err := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Past Episode", Valid: true},
		Status:        "unreleased",
		AirDate:       sql.NullTime{Time: pastDate, Valid: true},
		Monitored:     1,
	})
	if err != nil {
		t.Fatalf("CreateEpisode error = %v", err)
	}

	updated, err := service.RefreshEpisodes(ctx)
	if err != nil {
		t.Fatalf("RefreshEpisodes() error = %v", err)
	}

	if updated < 1 {
		t.Errorf("RefreshEpisodes() updated = %d, want >= 1", updated)
	}
}

func TestRefreshEpisodes_FutureNotChanged(t *testing.T) {
	// Episodes with future air dates should NOT be changed
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "Test Series",
		ProductionStatus: "continuing",
		Monitored:        1,
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    1,
	})

	futureDate := time.Now().AddDate(0, 0, 7)
	_, err := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Future Episode", Valid: true},
		Status:        "unreleased",
		AirDate:       sql.NullTime{Time: futureDate, Valid: true},
		Monitored:     1,
	})
	if err != nil {
		t.Fatalf("CreateEpisode error = %v", err)
	}

	updated, err := service.RefreshEpisodes(ctx)
	if err != nil {
		t.Fatalf("RefreshEpisodes() error = %v", err)
	}

	if updated != 0 {
		t.Errorf("RefreshEpisodes() should not update future episodes, got %d", updated)
	}
}

func TestRefreshEpisodes_OnlyAffectsUnreleased(t *testing.T) {
	// Spec: Only episodes with status='unreleased' should be affected
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "Test Series",
		ProductionStatus: "continuing",
		Monitored:        1,
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    1,
	})

	pastDate := time.Now().AddDate(0, 0, -7)

	// Create episode with 'available' status
	_, _ = queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Available Episode", Valid: true},
		Status:        "available",
		AirDate:       sql.NullTime{Time: pastDate, Valid: true},
		Monitored:     1,
	})

	updated, err := service.RefreshEpisodes(ctx)
	if err != nil {
		t.Fatalf("RefreshEpisodes() error = %v", err)
	}

	if updated != 0 {
		t.Errorf("RefreshEpisodes() should not affect non-unreleased episodes, updated %d", updated)
	}
}

func TestRefreshAll_BothMoviesAndEpisodes(t *testing.T) {
	// Spec: RefreshAll handles both movies and episodes
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(0, 0, -7)

	// Create a past-release movie
	_, _ = queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Past Movie",
		SortTitle:   "Past Movie",
		Status:      "unreleased",
		Monitored:   1,
		ReleaseDate: sql.NullTime{Time: pastDate, Valid: true},
	})

	// Create a past-air episode
	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "Test Series",
		ProductionStatus: "continuing",
		Monitored:        1,
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    1,
	})
	_, _ = queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Status:        "unreleased",
		AirDate:       sql.NullTime{Time: pastDate, Valid: true},
		Monitored:     1,
	})

	err := service.RefreshAll(ctx)
	if err != nil {
		t.Fatalf("RefreshAll() error = %v", err)
	}

	// Verify movie was updated
	movies, _ := queries.ListMoviesPaginated(ctx, sqlc.ListMoviesPaginatedParams{Limit: 10})
	if len(movies) != 1 {
		t.Fatalf("Expected 1 movie, got %d", len(movies))
	}
	if movies[0].Status != "missing" {
		t.Errorf("Movie status = %q, want %q", movies[0].Status, "missing")
	}

	// Verify episode was updated
	episodes, _ := queries.ListEpisodesBySeries(ctx, series.ID)
	if len(episodes) != 1 {
		t.Fatalf("Expected 1 episode, got %d", len(episodes))
	}
	if episodes[0].Status != "missing" {
		t.Errorf("Episode status = %q, want %q", episodes[0].Status, "missing")
	}
}
