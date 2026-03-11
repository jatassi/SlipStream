package availability

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/testutil"
)

// Tests for the underlying SQL queries used by module ReleaseDateResolver
// implementations (UpdateUnreleasedMoviesToMissing, UpdateUnreleasedEpisodesToMissing).
// Spec: docs/status-consolidation.md - "Scheduler Changes > Status Refresh"

func TestUpdateUnreleasedMoviesToMissing(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(0, 0, -7)
	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Past Movie",
		SortTitle:   "Past Movie",
		Status:      "unreleased",
		Monitored:   true,
		ReleaseDate: sql.NullTime{Time: pastDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	result, err := queries.UpdateUnreleasedMoviesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedMoviesToMissing() error = %v", err)
	}

	updated, _ := result.RowsAffected()
	if updated != 1 {
		t.Errorf("UpdateUnreleasedMoviesToMissing() updated = %d, want 1", updated)
	}
}

func TestUpdateUnreleasedMoviesToMissing_SameDayRelease(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	today := time.Now()
	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Today Movie",
		SortTitle:   "Today Movie",
		Status:      "unreleased",
		Monitored:   true,
		ReleaseDate: sql.NullTime{Time: today, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	result, err := queries.UpdateUnreleasedMoviesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedMoviesToMissing() error = %v", err)
	}

	updated, _ := result.RowsAffected()
	if updated != 1 {
		t.Errorf("UpdateUnreleasedMoviesToMissing() updated = %d, want 1 (same-day release must transition)", updated)
	}
}

func TestUpdateUnreleasedMoviesToMissing_FutureNotChanged(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	futureDate := time.Now().AddDate(0, 0, 7)
	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Future Movie",
		SortTitle:   "Future Movie",
		Status:      "unreleased",
		Monitored:   true,
		ReleaseDate: sql.NullTime{Time: futureDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	result, err := queries.UpdateUnreleasedMoviesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedMoviesToMissing() error = %v", err)
	}

	updated, _ := result.RowsAffected()
	if updated != 0 {
		t.Errorf("UpdateUnreleasedMoviesToMissing() should not update future movies, got %d", updated)
	}
}

func TestUpdateUnreleasedMoviesToMissing_OnlyAffectsUnreleased(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(0, 0, -7)

	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Available Movie",
		SortTitle:   "Available Movie",
		Status:      "available",
		Monitored:   true,
		ReleaseDate: sql.NullTime{Time: pastDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	_, err = queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:       "Downloading Movie",
		SortTitle:   "Downloading Movie",
		Status:      "downloading",
		Monitored:   true,
		ReleaseDate: sql.NullTime{Time: pastDate, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	result, err := queries.UpdateUnreleasedMoviesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedMoviesToMissing() error = %v", err)
	}

	updated, _ := result.RowsAffected()
	if updated != 0 {
		t.Errorf("UpdateUnreleasedMoviesToMissing() should not affect non-unreleased movies, updated %d", updated)
	}
}

func TestUpdateUnreleasedMoviesToMissing_NoReleaseDateNotChanged(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	_, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:     "No Date Movie",
		SortTitle: "No Date Movie",
		Status:    "unreleased",
		Monitored: true,
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	result, err := queries.UpdateUnreleasedMoviesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedMoviesToMissing() error = %v", err)
	}

	updated, _ := result.RowsAffected()
	if updated != 0 {
		t.Errorf("UpdateUnreleasedMoviesToMissing() should not affect movies without release dates, updated %d", updated)
	}
}

func TestUpdateUnreleasedEpisodesToMissing(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "Test Series",
		ProductionStatus: "continuing",
		Monitored:        true,
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})

	pastDate := time.Now().AddDate(0, 0, -7)
	_, err := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Past Episode", Valid: true},
		Status:        "unreleased",
		AirDate:       sql.NullTime{Time: pastDate, Valid: true},
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("CreateEpisode error = %v", err)
	}

	result, err := queries.UpdateUnreleasedEpisodesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedEpisodesToMissing() error = %v", err)
	}
	rows1, _ := result.RowsAffected()

	result, err = queries.UpdateUnreleasedEpisodesToMissingDateOnly(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedEpisodesToMissingDateOnly() error = %v", err)
	}
	rows2, _ := result.RowsAffected()

	if rows1+rows2 < 1 {
		t.Errorf("UpdateUnreleasedEpisodesToMissing() updated = %d, want >= 1", rows1+rows2)
	}
}

func TestUpdateUnreleasedEpisodesToMissing_FutureNotChanged(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "Test Series",
		ProductionStatus: "continuing",
		Monitored:        true,
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})

	futureDate := time.Now().AddDate(0, 0, 7)
	_, err := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Future Episode", Valid: true},
		Status:        "unreleased",
		AirDate:       sql.NullTime{Time: futureDate, Valid: true},
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("CreateEpisode error = %v", err)
	}

	result, err := queries.UpdateUnreleasedEpisodesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedEpisodesToMissing() error = %v", err)
	}
	rows1, _ := result.RowsAffected()

	result, err = queries.UpdateUnreleasedEpisodesToMissingDateOnly(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedEpisodesToMissingDateOnly() error = %v", err)
	}
	rows2, _ := result.RowsAffected()

	if rows1+rows2 != 0 {
		t.Errorf("UpdateUnreleasedEpisodesToMissing() should not update future episodes, got %d", rows1+rows2)
	}
}

func TestUpdateUnreleasedEpisodesToMissing_OnlyAffectsUnreleased(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "Test Series",
		ProductionStatus: "continuing",
		Monitored:        true,
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})

	pastDate := time.Now().AddDate(0, 0, -7)

	_, _ = queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Available Episode", Valid: true},
		Status:        "available",
		AirDate:       sql.NullTime{Time: pastDate, Valid: true},
		Monitored:     true,
	})

	result, err := queries.UpdateUnreleasedEpisodesToMissing(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedEpisodesToMissing() error = %v", err)
	}
	rows1, _ := result.RowsAffected()

	result, err = queries.UpdateUnreleasedEpisodesToMissingDateOnly(ctx)
	if err != nil {
		t.Fatalf("UpdateUnreleasedEpisodesToMissingDateOnly() error = %v", err)
	}
	rows2, _ := result.RowsAffected()

	if rows1+rows2 != 0 {
		t.Errorf("UpdateUnreleasedEpisodesToMissing() should not affect non-unreleased episodes, updated %d", rows1+rows2)
	}
}
