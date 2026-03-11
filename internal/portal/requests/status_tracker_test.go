package requests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/testutil"
)

// mockProvisionerLookup implements ModuleProvisionerLookup for testing.
type mockProvisionerLookup struct {
	provisioners    map[string]module.PortalProvisioner
	leafEntityTypes map[string]bool
}

func (m *mockProvisionerLookup) GetProvisionerForEntityType(entityType string) module.PortalProvisioner {
	if m == nil || m.provisioners == nil {
		return nil
	}
	return m.provisioners[entityType]
}

func (m *mockProvisionerLookup) IsLeafEntityType(entityType string) bool {
	if m == nil || m.leafEntityTypes == nil {
		// Default: movie and episode are leaf types (matches real module schemas)
		return entityType == "movie" || entityType == "episode"
	}
	return m.leafEntityTypes[entityType]
}

// mockProvisioner implements module.PortalProvisioner for testing.
type mockProvisioner struct {
	completionResult *module.RequestCompletionResult
	completionErr    error
}

func (m *mockProvisioner) EnsureInLibrary(_ context.Context, _ *module.ProvisionInput) (int64, error) {
	return 0, nil
}

func (m *mockProvisioner) CheckAvailability(_ context.Context, _ *module.AvailabilityCheckInput) (*module.AvailabilityResult, error) {
	return &module.AvailabilityResult{}, nil
}

func (m *mockProvisioner) CheckRequestCompletion(_ context.Context, _ *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) {
	return m.completionResult, m.completionErr
}

func (m *mockProvisioner) ValidateRequest(_ context.Context, _ string, _ map[string]int64) error {
	return nil
}

func (m *mockProvisioner) SupportedEntityTypes() []string {
	return nil
}

// Helper to create a test portal user and return the user ID.
func createTestUser(t *testing.T, queries *sqlc.Queries) int64 {
	t.Helper()
	user, err := queries.CreatePortalUser(context.Background(), sqlc.CreatePortalUserParams{
		Username:     "testuser",
		PasswordHash: "fakehash",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("CreatePortalUser error = %v", err)
	}
	return user.ID
}

// Helper to create a request and approve it.
func createApprovedRequest(t *testing.T, svc *Service, userID int64, input *CreateInput) *Request {
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

func TestStatusTracker_OnDownloadStarted_Movie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Test Movie",
	})

	_, err := reqSvc.LinkMedia(ctx, req.ID, movieID)
	if err != nil {
		t.Fatalf("LinkMedia error = %v", err)
	}

	err = tracker.OnDownloadStarted(ctx, "movie", movieID)
	if err != nil {
		t.Fatalf("OnDownloadStarted error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusDownloading {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusDownloading)
	}
}

func TestStatusTracker_OnDownloadStarted_IgnoresNonApproved(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	// Create but do NOT approve - remains pending
	req, _ := reqSvc.Create(ctx, userID, &CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Test Movie",
	})
	_, _ = reqSvc.LinkMedia(ctx, req.ID, movieID)

	err := tracker.OnDownloadStarted(ctx, "movie", movieID)
	if err != nil {
		t.Fatalf("OnDownloadStarted error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusPending {
		t.Errorf("Request status = %q, want %q (should remain pending)", updated.Status, StatusPending)
	}
}

func TestStatusTracker_OnDownloadFailed_Movie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Test Movie",
	})
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

func TestStatusTracker_OnDownloadFailed_Episode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create a series with an episode so findParentRequests can look it up
	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
		TvdbID:           sql.NullInt64{Int64: 5000, Valid: true},
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})
	episode, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Ep1", Valid: true},
		Status:        "missing",
		Monitored:     true,
	})

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType:     MediaTypeEpisode,
		TvdbID:        testutil.Int64Ptr(5000),
		Title:         "Test Episode",
		SeasonNumber:  testutil.Int64Ptr(1),
		EpisodeNumber: testutil.Int64Ptr(1),
	})
	_, _ = reqSvc.LinkMedia(ctx, req.ID, episode.ID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnDownloadFailed(ctx, "episode", episode.ID)
	if err != nil {
		t.Fatalf("OnDownloadFailed error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusFailed {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusFailed)
	}
}

func TestStatusTracker_OnEntityAvailable_Movie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	movieID := int64(100)

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{
		provisioners: map[string]module.PortalProvisioner{
			"movie": &mockProvisioner{},
		},
	}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType: MediaTypeMovie,
		TmdbID:    testutil.Int64Ptr(42),
		Title:     "Test Movie",
	})
	_, _ = reqSvc.LinkMedia(ctx, req.ID, movieID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnEntityAvailable(ctx, "movie", "movie", movieID)
	if err != nil {
		t.Fatalf("OnEntityAvailable error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusAvailable {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusAvailable)
	}
}

func TestStatusTracker_OnEntityAvailable_Episode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create a series with an episode
	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
		TvdbID:           sql.NullInt64{Int64: 5000, Valid: true},
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})
	episode, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 3,
		Title:         sql.NullString{String: "Ep3", Valid: true},
		Status:        "missing",
		Monitored:     true,
	})

	tvdbID := int64(5000)
	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{
		provisioners: map[string]module.PortalProvisioner{
			"episode": &mockProvisioner{},
		},
	}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)

	// Create an episode-level request linked directly by media_id
	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType:     MediaTypeEpisode,
		TvdbID:        &tvdbID,
		Title:         "Test Episode",
		SeasonNumber:  testutil.Int64Ptr(1),
		EpisodeNumber: testutil.Int64Ptr(3),
	})
	_, _ = reqSvc.LinkMedia(ctx, req.ID, episode.ID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnEntityAvailable(ctx, "tv", "episode", episode.ID)
	if err != nil {
		t.Fatalf("OnEntityAvailable error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusAvailable {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusAvailable)
	}
}

func TestStatusTracker_OnEntityAvailable_EpisodeCompletesSeriesRequest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
		TvdbID:           sql.NullInt64{Int64: 6000, Valid: true},
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})
	episode, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 5,
		Title:         sql.NullString{String: "Ep5", Valid: true},
		Status:        "missing",
		Monitored:     true,
	})

	tvdbID := int64(6000)

	// Mock provisioner returns "should mark available" for the series request
	prov := &mockProvisioner{
		completionResult: &module.RequestCompletionResult{ShouldMarkAvailable: true},
	}
	lookup := &mockProvisionerLookup{
		provisioners: map[string]module.PortalProvisioner{
			"episode": prov,
			"series":  prov,
		},
	}

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{1},
	})
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	err := tracker.OnEntityAvailable(ctx, "tv", "episode", episode.ID)
	if err != nil {
		t.Fatalf("OnEntityAvailable error = %v", err)
	}

	updated, _ := reqSvc.Get(ctx, req.ID)
	if updated.Status != StatusAvailable {
		t.Errorf("Request status = %q, want %q", updated.Status, StatusAvailable)
	}
}

func TestStatusTracker_OnDownloadStarted_Episode_UpdatesSeriesRequest(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	tvdbID := int64(5000)

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
		TvdbID:           sql.NullInt64{Int64: tvdbID, Valid: true},
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})
	episode, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Ep1", Valid: true},
		Status:        "missing",
		Monitored:     true,
	})

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{1},
	})

	err := tracker.OnDownloadStarted(ctx, "episode", episode.ID)
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

	series, _ := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		ProductionStatus: "ended",
		TvdbID:           sql.NullInt64{Int64: 5000, Valid: true},
	})
	_, _ = queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})
	ep1, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Ep1", Valid: true},
		Status:        "failed",
		Monitored:     true,
	})
	ep2, _ := queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 2,
		Title:         sql.NullString{String: "Ep2", Valid: true},
		Status:        "missing",
		Monitored:     true,
	})

	tvdbID := int64(5000)
	prov := &mockProvisioner{}
	lookup := &mockProvisionerLookup{
		provisioners: map[string]module.PortalProvisioner{
			"series":  prov,
			"episode": prov,
		},
	}
	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)

	req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{1},
	})
	_, _ = reqSvc.LinkMedia(ctx, req.ID, series.ID)
	_, _ = reqSvc.UpdateStatus(ctx, req.ID, StatusDownloading)

	// ep1 is failed, ep2 is still missing -> request should NOT become failed
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

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	watchersSvc := NewWatchersService(queries, &tdb.Logger)
	lookup := &mockProvisionerLookup{}
	tracker := NewStatusTracker(queries, reqSvc, watchersSvc, &tdb.Logger, lookup, nil)

	userID := createTestUser(t, queries)
	movieID := int64(100)

	req, _ := reqSvc.Create(ctx, userID, &CreateInput{
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

func TestStatusTracker_RequestStatusConstraint(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	reqSvc := NewService(queries, &tdb.Logger, nil, nil, nil)
	userID := createTestUser(t, queries)

	validStatuses := []string{
		StatusPending, StatusApproved, StatusDenied,
		StatusDownloading, StatusFailed, StatusAvailable,
	}
	for _, status := range validStatuses {
		t.Run("valid_"+status, func(t *testing.T) {
			req, _ := reqSvc.Create(ctx, userID, &CreateInput{
				MediaType: MediaTypeMovie,
				TmdbID:    testutil.Int64Ptr(int64(100 + len(status))),
				Title:     "Movie " + status,
			})
			_, err := reqSvc.UpdateStatus(ctx, req.ID, status)
			if err != nil {
				t.Errorf("UpdateStatus(%q) error = %v", status, err)
			}
		})
	}

	t.Run("approved_to_downloading_to_failed", func(t *testing.T) {
		req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
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

	t.Run("downloading_to_available", func(t *testing.T) {
		req := createApprovedRequest(t, reqSvc, userID, &CreateInput{
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
