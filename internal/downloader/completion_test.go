package downloader

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/testutil"
)

func TestDisappearedDownload_MovieMarkedFailed(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	movie, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:     "Test Movie",
		SortTitle: "test movie",
		Status:    "downloading",
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "dl-abc", Valid: true},
	})

	// Simulate disappearance: mark failed, clear download ID, set message
	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "failed",
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    sql.NullString{String: "Download removed from client", Valid: true},
	})

	got, _ := queries.GetMovie(ctx, movie.ID)
	if got.Status != "failed" {
		t.Errorf("Status = %q, want failed", got.Status)
	}
	if !got.StatusMessage.Valid || got.StatusMessage.String != "Download removed from client" {
		t.Errorf("StatusMessage = %v, want 'Download removed from client'", got.StatusMessage)
	}
	if got.ActiveDownloadID.Valid {
		t.Errorf("ActiveDownloadID should be null, got %q", got.ActiveDownloadID.String)
	}

	// Should no longer appear in downloading list
	downloading, _ := queries.ListDownloadingMovies(ctx)
	for _, m := range downloading {
		if m.ID == movie.ID {
			t.Error("Movie should not appear in ListDownloadingMovies after being marked failed")
		}
	}
}

func TestDisappearedDownload_EpisodeMarkedFailed(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	series, err := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
	})
	if err != nil {
		t.Fatalf("CreateSeries error = %v", err)
	}

	_, err = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    1,
	})
	if err != nil {
		t.Fatalf("CreateSeason error = %v", err)
	}

	episode, err := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Pilot", Valid: true},
		Status:        "downloading",
		Monitored:     1,
	})
	if err != nil {
		t.Fatalf("CreateEpisode error = %v", err)
	}

	_ = queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:               episode.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "dl-ep-abc", Valid: true},
	})

	// Simulate disappearance
	_ = queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:               episode.ID,
		Status:           "failed",
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    sql.NullString{String: "Download removed from client", Valid: true},
	})

	got, _ := queries.GetEpisode(ctx, episode.ID)
	if got.Status != "failed" {
		t.Errorf("Status = %q, want failed", got.Status)
	}
	if !got.StatusMessage.Valid || got.StatusMessage.String != "Download removed from client" {
		t.Errorf("StatusMessage = %v, want 'Download removed from client'", got.StatusMessage)
	}

	downloading, _ := queries.ListDownloadingEpisodes(ctx)
	for _, ep := range downloading {
		if ep.ID == episode.ID {
			t.Error("Episode should not appear in ListDownloadingEpisodes after being marked failed")
		}
	}
}

func TestDisappearedDownload_MockDownloadsSkipped(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create two downloading movies: one mock, one real
	mockMovie, _ := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:     "Mock Movie",
		SortTitle: "mock movie",
		Status:    "downloading",
	})
	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               mockMovie.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "mock-dl-123", Valid: true},
	})

	realMovie, _ := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:     "Real Movie",
		SortTitle: "real movie",
		Status:    "downloading",
	})
	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               realMovie.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "real-dl-456", Valid: true},
	})

	// Simulate the mock-prefix check from CheckForDisappearedDownloads:
	// only process non-mock downloads
	movies, _ := queries.ListDownloadingMovies(ctx)
	for _, m := range movies {
		if !m.ActiveDownloadID.Valid {
			continue
		}
		downloadID := m.ActiveDownloadID.String
		if strings.HasPrefix(downloadID, "mock-") {
			continue
		}
		// Mark non-mock as failed (simulating disappeared download)
		_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
			ID:               m.ID,
			Status:           "failed",
			ActiveDownloadID: sql.NullString{},
			StatusMessage:    sql.NullString{String: "Download removed from client", Valid: true},
		})
	}

	// Mock movie should still be downloading
	gotMock, _ := queries.GetMovie(ctx, mockMovie.ID)
	if gotMock.Status != "downloading" {
		t.Errorf("Mock movie status = %q, want downloading", gotMock.Status)
	}

	// Real movie should be failed
	gotReal, _ := queries.GetMovie(ctx, realMovie.ID)
	if gotReal.Status != "failed" {
		t.Errorf("Real movie status = %q, want failed", gotReal.Status)
	}
}
