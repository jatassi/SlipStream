package requests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/testutil"
)

// Mock implementations for lookup interfaces

type mockMovieLookup struct {
	tmdbIDs map[int64]int64 // movieID → tmdbID
}

func (m *mockMovieLookup) GetTmdbIDByMovieID(_ context.Context, movieID int64) (int64, error) {
	if id, ok := m.tmdbIDs[movieID]; ok {
		return id, nil
	}
	return 0, sql.ErrNoRows
}

type mockEpisodeLookup struct {
	episodes map[int64]episodeInfo // episodeID → info
}

type episodeInfo struct {
	tvdbID     int64
	seasonNum  int
	episodeNum int
}

func (m *mockEpisodeLookup) GetEpisodeInfo(_ context.Context, episodeID int64) (int64, int, int, error) {
	if info, ok := m.episodes[episodeID]; ok {
		return info.tvdbID, info.seasonNum, info.episodeNum, nil
	}
	return 0, 0, 0, sql.ErrNoRows
}

// Helper to create a test portal user and return the user ID.
func createTestUser(t *testing.T, queries *sqlc.Queries) int64 {
	t.Helper()
	user, err := queries.CreatePortalUser(context.Background(), sqlc.CreatePortalUserParams{
		Username:     "testuser",
		PasswordHash: "fakehash",
		Enabled:      1,
	})
	if err != nil {
		t.Fatalf("CreatePortalUser error = %v", err)
	}
	return user.ID
}

// Helper to create a request and approve it.
func createApprovedRequest(t *testing.T, svc *Service, userID int64, input CreateInput) *Request {
	t.Helper()
	ctx := context.Background()
	req, err := svc.Create(ctx, userID, input)
	if err != nil {
		t.Fatalf("Create request error = %v", err)
	}
	approved, err := svc.Approve(ctx, req.ID, userID, ApprovalActionOnly)
	if err != nil {
		t.Fatalf("Approve request error = %v", err)
	}
	return approved
}

// Gap 6: StatusTracker OnDownloadStarted for movie
func TestStatusTracker_OnDownloadStarted_Movie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Test Movie",
	})

	// Link the request to the movie media ID, then restore approved status
	// (LinkMedia sets status=available as a side effect)
	_, err := reqSvc.LinkMedia(ctx, req.ID, movieID)
	if err != nil {
		t.Fatalf("LinkMedia error = %v", err)
	}
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusApproved)

	err = tracker.OnDownloadStarted(ctx, "movie", movieID)
	if err != nil {
		t.Fatalf("OnDownloadStarted error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusDownloading {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusDownloading)
	}
}

// Gap 6: StatusTracker ignores non-approved requests
func TestStatusTracker_OnDownloadStarted_IgnoresNonApproved(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	// Create but do NOT approve - remains pending
	req, _ := reqSvc.Create(ctx, userID, CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Test Movie",
	})
	// LinkMedia sets status=available; reset to pending to simulate a non-approved request
	_, _ = reqSvc.LinkMedia(ctx, req.ID, movieID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusPending)

	err := tracker.OnDownloadStarted(ctx, "movie", movieID)
	if err != nil {
		t.Fatalf("OnDownloadStarted error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusPending {
		t.Errorf("Request status = %q, want %q (should remain pending)", updated.Status, StatusPending)
	}
}

// Gap 6: StatusTracker OnDownloadFailed for movie
func TestStatusTracker_OnDownloadFailed_Movie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Test Movie",
	})
	// LinkMedia sets status=available; set to downloading to test failure path
	_, _ = reqSvc.LinkMedia(ctx, req.ID, movieID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnDownloadFailed(ctx, "movie", movieID)
	if err != nil {
		t.Fatalf("OnDownloadFailed error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusFailed {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusFailed)
	}
}

// Gap 6: StatusTracker OnDownloadFailed for episode
func TestStatusTracker_OnDownloadFailed_Episode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)
	tracker.SetEpisodeLookup(&mockEpisodeLookup{
		episodes: map[int64]episodeInfo{
			200: {tvdbID: 5000, seasonNum: 1, episodeNum: 1},
		},
	})

	userID := createTestUser(t, queries)
	episodeID := int64(200)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType: MediaTypeEpisode,
		TvdbID:    testutil.Int64Ptr(5000),
		Title:     "Test Episode",
		SeasonNumber:  testutil.Int64Ptr(1),
		EpisodeNumber: testutil.Int64Ptr(1),
	})
	// LinkMedia sets status=available; set to downloading to test failure path
	_, _ = reqSvc.LinkMedia(ctx, req.ID, episodeID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnDownloadFailed(ctx, "episode", episodeID)
	if err != nil {
		t.Fatalf("OnDownloadFailed error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusFailed {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusFailed)
	}
}

// Gap 6: StatusTracker OnMovieAvailable
func TestStatusTracker_OnMovieAvailable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	movieID := int64(100)
	tmdbID := int64(42)
	tracker.SetMovieLookup(&mockMovieLookup{
		tmdbIDs: map[int64]int64{movieID: tmdbID},
	})

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    &tmdbID,
		Title:     "Test Movie",
	})
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnMovieAvailable(ctx, movieID)
	if err != nil {
		t.Fatalf("OnMovieAvailable error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusAvailable {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusAvailable)
	}
}

// Gap 6: StatusTracker OnEpisodeAvailable
func TestStatusTracker_OnEpisodeAvailable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	episodeID := int64(200)
	tvdbID := int64(5000)
	tracker.SetEpisodeLookup(&mockEpisodeLookup{
		episodes: map[int64]episodeInfo{
			episodeID: {tvdbID: tvdbID, seasonNum: 1, episodeNum: 3},
		},
	})

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType:     MediaTypeEpisode,
		TvdbID:        &tvdbID,
		Title:         "Test Episode",
		SeasonNumber:  testutil.Int64Ptr(1),
		EpisodeNumber: testutil.Int64Ptr(3),
	})
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnEpisodeAvailable(ctx, episodeID)
	if err != nil {
		t.Fatalf("OnEpisodeAvailable error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusAvailable {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusAvailable)
	}
}

type mockSeriesLookup struct {
	seriesIDs map[int64]int64 // tvdbID → seriesID
	complete  map[int64]bool  // seriesID → allSeasonsComplete
}

func (m *mockSeriesLookup) GetSeriesIDByTvdbID(_ context.Context, tvdbID int64) (int64, error) {
	if id, ok := m.seriesIDs[tvdbID]; ok {
		return id, nil
	}
	return 0, sql.ErrNoRows
}

func (m *mockSeriesLookup) AreSeasonsComplete(_ context.Context, seriesID int64, _ []int64) (bool, error) {
	if complete, ok := m.complete[seriesID]; ok {
		return complete, nil
	}
	return false, nil
}

func TestStatusTracker_OnDownloadStarted_Episode_UpdatesSeriesRequest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	tvdbID := int64(5000)
	episodeID := int64(200)
	tracker.SetEpisodeLookup(&mockEpisodeLookup{
		episodes: map[int64]episodeInfo{
			episodeID: {tvdbID: tvdbID, seasonNum: 1, episodeNum: 1},
		},
	})

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{1},
	})

	err := tracker.OnDownloadStarted(ctx, "episode", episodeID)
	if err != nil {
		t.Fatalf("OnDownloadStarted error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusDownloading {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusDownloading)
	}
}

func TestStatusTracker_OnDownloadFailed_Series_OnlyWhenAllEpisodesFailed(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create a real series with 2 episodes in the DB
	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
		TvdbID:           sql.NullInt64{Int64: 5000, Valid: true},
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    1,
	})
	ep1, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Ep1", Valid: true},
		Status:        "failed",
		Monitored:     1,
	})
	ep2, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 2,
		Title:         sql.NullString{String: "Ep2", Valid: true},
		Status:        "missing",
		Monitored:     1,
	})

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	tvdbID := int64(5000)
	tracker.SetEpisodeLookup(&mockEpisodeLookup{
		episodes: map[int64]episodeInfo{
			ep1.ID: {tvdbID: tvdbID, seasonNum: 1, episodeNum: 1},
			ep2.ID: {tvdbID: tvdbID, seasonNum: 1, episodeNum: 2},
		},
	})
	tracker.SetSeriesLookup(&mockSeriesLookup{
		seriesIDs: map[int64]int64{tvdbID: series.ID},
	})

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{1},
	})
	// Link to series and set to downloading
	_, _ = reqSvc.LinkMedia(ctx, req.ID, series.ID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	// ep1 is failed, ep2 is still missing → request should NOT become failed
	err := tracker.OnDownloadFailed(ctx, "episode", ep1.ID)
	if err != nil {
		t.Fatalf("OnDownloadFailed (first) error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusDownloading {
		t.Errorf("After first episode failed: status = %q, want downloading (ep2 still missing)", updated.Status)
	}

	// Now set ep2 to failed too
	_ = queries.UpdateEpisodeStatus(ctx, sqlc.UpdateEpisodeStatusParams{
		ID:     ep2.ID,
		Status: "failed",
	})

	err = tracker.OnDownloadFailed(ctx, "episode", ep2.ID)
	if err != nil {
		t.Fatalf("OnDownloadFailed (second) error = %v", err)
	}

	updated, _ = reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusFailed {
		t.Errorf("After all episodes failed: status = %q, want failed", updated.Status)
	}
}

func TestStatusTracker_OnDownloadFailed_SkipsDenied(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	req, _ := reqSvc.Create(ctx, userID, CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Denied Movie",
	})
	_, _ = reqSvc.LinkMedia(ctx, req.ID, movieID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDenied)

	err := tracker.OnDownloadFailed(ctx, "movie", movieID)
	if err != nil {
		t.Fatalf("OnDownloadFailed error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusDenied {
		t.Errorf("Request status = %q, want %q (denied requests should not change)", updated.Status, StatusDenied)
	}
}

func TestStatusTracker_OnEpisodeAvailable_CompletesSeasonRequest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create a real series
	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
		TvdbID:           sql.NullInt64{Int64: 6000, Valid: true},
	})

	reqSvc := NewService(queries, tdb.Logger)
	watchersSvc := NewWatchersService(queries, tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, tdb.Logger)

	tvdbID := int64(6000)
	episodeID := int64(300)
	tracker.SetEpisodeLookup(&mockEpisodeLookup{
		episodes: map[int64]episodeInfo{
			episodeID: {tvdbID: tvdbID, seasonNum: 1, episodeNum: 5},
		},
	})
	tracker.SetSeriesLookup(&mockSeriesLookup{
		seriesIDs: map[int64]int64{tvdbID: series.ID},
		complete:  map[int64]bool{series.ID: true},
	})

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{1},
	})
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnEpisodeAvailable(ctx, episodeID)
	if err != nil {
		t.Fatalf("OnEpisodeAvailable error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusAvailable {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusAvailable)
	}
}

// Gap 13: Verify all valid request statuses and key transitions
func TestStatusTracker_RequestStatusConstraint(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, tdb.Logger)
	userID := createTestUser(t, queries)

	// Verify all valid statuses can be set
	validStatuses := []string{
		StatusPending, StatusApproved, StatusDenied,
		StatusDownloading, StatusFailed, StatusAvailable,
	}
	for _, status := range validStatuses {
		t.Run("valid_"+status, func(t *testing.T) {
			req, _ := reqSvc.Create(ctx, userID, CreateInput{
				MediaType: MediaTypeMovie,
				TmdbID:    testutil.Int64Ptr(int64(100 + len(status))), // unique per test
				Title:     "Movie " + status,
			})
			_, err := reqSvc.UpdateStatus(ctx, req.ID, status)
			if err != nil {
				t.Errorf("UpdateStatus(%q) error = %v", status, err)
			}
		})
	}

	// Verify key transitions: approved → downloading → failed
	t.Run("approved_to_downloading_to_failed", func(t *testing.T) {
		req := createApprovedRequest(t, reqSvc, userID, CreateInput{
			MediaType: MediaTypeMovie,
			TmdbID:    testutil.Int64Ptr(999),
			Title:     "Transition Movie",
		})
		if req.Status != StatusApproved {
			t.Fatalf("Expected approved, got %q", req.Status)
		}

		updated, _ := reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)
		if updated.Status != StatusDownloading {
			t.Errorf("Status = %q, want downloading", updated.Status)
		}

		updated, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusFailed)
		if updated.Status != StatusFailed {
			t.Errorf("Status = %q, want failed", updated.Status)
		}
	})

	// Verify key transition: downloading → available
	t.Run("downloading_to_available", func(t *testing.T) {
		req := createApprovedRequest(t, reqSvc, userID, CreateInput{
			MediaType: MediaTypeMovie,
			TmdbID:    testutil.Int64Ptr(998),
			Title:     "Available Movie",
		})
		_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

		updated, _ := reqSvc.UpdateStatus(ctx, req.ID, StatusAvailable)
		if updated.Status != StatusAvailable {
			t.Errorf("Status = %q, want available", updated.Status)
		}
	})
}
