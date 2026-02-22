package requests

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/testutil"
)

func createTestRootFolder(t *testing.T, q *sqlc.Queries, mediaType string) int64 {
	t.Helper()
	rf, err := q.CreateRootFolder(context.Background(), sqlc.CreateRootFolderParams{
		Path:      "/test/" + mediaType,
		Name:      "Test " + mediaType,
		MediaType: mediaType,
	})
	if err != nil {
		t.Fatalf("CreateRootFolder error = %v", err)
	}
	return rf.ID
}

func createTestQualityProfile(t *testing.T, q *sqlc.Queries) int64 {
	t.Helper()
	items := []map[string]interface{}{
		{"id": 4, "allowed": true, "name": "HDTV-720p"},
	}
	itemsJSON, _ := json.Marshal(items)
	profile, err := q.CreateQualityProfile(context.Background(), sqlc.CreateQualityProfileParams{
		Name:                 "Test Profile",
		Cutoff:               4,
		Items:                string(itemsJSON),
		HdrSettings:          "[]",
		VideoCodecSettings:   "[]",
		AudioCodecSettings:   "[]",
		AudioChannelSettings: "[]",
		UpgradeStrategy:      "quality",
	})
	if err != nil {
		t.Fatalf("CreateQualityProfile error = %v", err)
	}
	return profile.ID
}

type testSeason struct {
	Number           int
	Monitored        bool
	Episodes         int
	AiredEpisodes    int
	WithFiles        int
	UnairedMonitored int
}

func createTestSeriesWithEpisodes(t *testing.T, q *sqlc.Queries, tvdbID int, seasons []testSeason) {
	t.Helper()
	ctx := context.Background()

	rfID := createTestRootFolder(t, q, "tv")
	qpID := createTestQualityProfile(t, q)

	series, err := q.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		TvdbID:           sql.NullInt64{Int64: int64(tvdbID), Valid: true},
		QualityProfileID: sql.NullInt64{Int64: qpID, Valid: true},
		RootFolderID:     sql.NullInt64{Int64: rfID, Valid: true},
		Monitored:        1,
		ProductionStatus: "ended",
	})
	if err != nil {
		t.Fatalf("CreateSeries error = %v", err)
	}

	slotID := int64(1) // Primary slot from migration

	for _, s := range seasons {
		mon := int64(0)
		if s.Monitored {
			mon = 1
		}
		_, err := q.CreateSeason(ctx, sqlc.CreateSeasonParams{
			SeriesID:     series.ID,
			SeasonNumber: int64(s.Number),
			Monitored:    mon,
		})
		if err != nil {
			t.Fatalf("CreateSeason error = %v", err)
		}

		pastDate := time.Now().AddDate(0, -1, 0)
		futureDate := time.Now().AddDate(0, 1, 0)

		for ep := 1; ep <= s.Episodes; ep++ {
			aired := ep <= s.AiredEpisodes
			airDate := futureDate
			if aired {
				airDate = pastDate
			}

			epMon := int64(0)
			if aired || ep-s.AiredEpisodes <= s.UnairedMonitored {
				epMon = 1
			}
			if !aired && s.UnairedMonitored == 0 {
				epMon = 0
			}

			status := "unreleased"
			if aired {
				status = "missing"
			}

			episode, err := q.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
				SeriesID:      series.ID,
				SeasonNumber:  int64(s.Number),
				EpisodeNumber: int64(ep),
				Title:         sql.NullString{String: "Episode " + string(rune('0'+ep)), Valid: true},
				AirDate:       sql.NullTime{Time: airDate, Valid: true},
				Monitored:     epMon,
				Status:        status,
			})
			if err != nil {
				t.Fatalf("CreateEpisode error = %v", err)
			}

			if aired && ep <= s.WithFiles {
				ef, err := q.CreateEpisodeFile(ctx, sqlc.CreateEpisodeFileParams{
					EpisodeID: episode.ID,
					Path:      "/test/file.mkv",
					Size:      1000,
				})
				if err != nil {
					t.Fatalf("CreateEpisodeFile error = %v", err)
				}

				_, err = q.CreateEpisodeSlotAssignment(ctx, sqlc.CreateEpisodeSlotAssignmentParams{
					EpisodeID: episode.ID,
					SlotID:    slotID,
					FileID:    sql.NullInt64{Int64: ef.ID, Valid: true},
					Monitored: 1,
					Status:    "available",
				})
				if err != nil {
					t.Fatalf("CreateEpisodeSlotAssignment error = %v", err)
				}
			}
		}
	}

}

func TestCheckMovieAvailability_NoFiles(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	rfID := createTestRootFolder(t, q, "movie")
	qpID := createTestQualityProfile(t, q)

	movie, err := q.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:            "Test Movie",
		SortTitle:        "test movie",
		TmdbID:           sql.NullInt64{Int64: 12345, Valid: true},
		QualityProfileID: sql.NullInt64{Int64: qpID, Valid: true},
		RootFolderID:     sql.NullInt64{Int64: rfID, Valid: true},
		Monitored:        1,
		Status:           "missing",
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	// Create slot assignment with NO file
	_, err = q.CreateMovieSlotAssignment(ctx, sqlc.CreateMovieSlotAssignmentParams{
		MovieID:   movie.ID,
		SlotID:    1,
		Monitored: 1,
		Status:    "missing",
	})
	if err != nil {
		t.Fatalf("CreateMovieSlotAssignment error = %v", err)
	}

	checker := NewLibraryChecker(q, &tdb.Logger)
	result, err := checker.CheckMovieAvailability(ctx, 12345, nil)
	if err != nil {
		t.Fatalf("CheckMovieAvailability error = %v", err)
	}

	if result.InLibrary {
		t.Error("expected InLibrary = false for movie with no files")
	}
	if !result.CanRequest {
		t.Error("expected CanRequest = true for movie with no files")
	}
	if result.MediaID == nil {
		t.Error("expected MediaID to be populated")
	}
}

func TestCheckMovieAvailability_HasFiles(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	rfID := createTestRootFolder(t, q, "movie")
	qpID := createTestQualityProfile(t, q)

	movie, err := q.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:            "Test Movie",
		SortTitle:        "test movie",
		TmdbID:           sql.NullInt64{Int64: 12345, Valid: true},
		QualityProfileID: sql.NullInt64{Int64: qpID, Valid: true},
		RootFolderID:     sql.NullInt64{Int64: rfID, Valid: true},
		Monitored:        1,
		Status:           "available",
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	mf, err := q.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
		MovieID: movie.ID,
		Path:    "/test/movie.mkv",
		Size:    5000,
	})
	if err != nil {
		t.Fatalf("CreateMovieFile error = %v", err)
	}

	_, err = q.CreateMovieSlotAssignment(ctx, sqlc.CreateMovieSlotAssignmentParams{
		MovieID:   movie.ID,
		SlotID:    1,
		FileID:    sql.NullInt64{Int64: mf.ID, Valid: true},
		Monitored: 1,
		Status:    "available",
	})
	if err != nil {
		t.Fatalf("CreateMovieSlotAssignment error = %v", err)
	}

	checker := NewLibraryChecker(q, &tdb.Logger)
	result, err := checker.CheckMovieAvailability(ctx, 12345, nil)
	if err != nil {
		t.Fatalf("CheckMovieAvailability error = %v", err)
	}

	if !result.InLibrary {
		t.Error("expected InLibrary = true for movie with files")
	}
}

func TestCheckSeriesAvailability_ZeroFiles(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	createTestSeriesWithEpisodes(t, q, 1000, []testSeason{
		{Number: 1, Monitored: true, Episodes: 5, AiredEpisodes: 5, WithFiles: 0},
		{Number: 2, Monitored: true, Episodes: 5, AiredEpisodes: 5, WithFiles: 0},
	})

	checker := NewLibraryChecker(q, &tdb.Logger)
	result, err := checker.CheckSeriesAvailability(ctx, 1000, nil)
	if err != nil {
		t.Fatalf("CheckSeriesAvailability error = %v", err)
	}

	if result.InLibrary {
		t.Error("expected InLibrary = false for series with zero files")
	}
	if !result.CanRequest {
		t.Error("expected CanRequest = true for series with zero files")
	}
	if len(result.SeasonAvailability) != 2 {
		t.Errorf("expected 2 season availability entries, got %d", len(result.SeasonAvailability))
	}
	for _, sa := range result.SeasonAvailability {
		if sa.HasAnyFiles {
			t.Errorf("expected HasAnyFiles = false for season %d", sa.SeasonNumber)
		}
	}
}

func TestCheckSeriesAvailability_PartiallyAvailable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	createTestSeriesWithEpisodes(t, q, 1001, []testSeason{
		{Number: 1, Monitored: true, Episodes: 10, AiredEpisodes: 10, WithFiles: 10},
		{Number: 2, Monitored: true, Episodes: 10, AiredEpisodes: 10, WithFiles: 5},
		{Number: 3, Monitored: true, Episodes: 8, AiredEpisodes: 5, WithFiles: 5, UnairedMonitored: 3},
	})

	checker := NewLibraryChecker(q, &tdb.Logger)
	result, err := checker.CheckSeriesAvailability(ctx, 1001, nil)
	if err != nil {
		t.Fatalf("CheckSeriesAvailability error = %v", err)
	}

	if !result.InLibrary {
		t.Error("expected InLibrary = true")
	}
	if !result.CanRequest {
		t.Error("expected CanRequest = true (S2 not fully available)")
	}
	if len(result.SeasonAvailability) != 3 {
		t.Fatalf("expected 3 season availability entries, got %d", len(result.SeasonAvailability))
	}

	s1 := result.SeasonAvailability[0]
	if !s1.Available {
		t.Error("S1 should be Available (all aired with files)")
	}
	if s1.AiredEpisodesWithFiles != 10 {
		t.Errorf("S1 AiredEpisodesWithFiles = %d, want 10", s1.AiredEpisodesWithFiles)
	}

	s2 := result.SeasonAvailability[1]
	if s2.Available {
		t.Error("S2 should NOT be Available (only 5/10 with files)")
	}
	if s2.AiredEpisodesWithFiles != 5 {
		t.Errorf("S2 AiredEpisodesWithFiles = %d, want 5", s2.AiredEpisodesWithFiles)
	}

	s3 := result.SeasonAvailability[2]
	if !s3.Available {
		t.Error("S3 should be Available (all aired with files, unaired monitored)")
	}
}

func TestCheckSeriesAvailability_FullyAvailable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	createTestSeriesWithEpisodes(t, q, 1002, []testSeason{
		{Number: 1, Monitored: true, Episodes: 10, AiredEpisodes: 10, WithFiles: 10},
		{Number: 2, Monitored: true, Episodes: 8, AiredEpisodes: 8, WithFiles: 8},
	})

	checker := NewLibraryChecker(q, &tdb.Logger)
	result, err := checker.CheckSeriesAvailability(ctx, 1002, nil)
	if err != nil {
		t.Fatalf("CheckSeriesAvailability error = %v", err)
	}

	if !result.InLibrary {
		t.Error("expected InLibrary = true")
	}
	if result.CanRequest {
		t.Error("expected CanRequest = false (fully available)")
	}
}

func TestSeasonAvailable_UnairedNotMonitored(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	createTestSeriesWithEpisodes(t, q, 1003, []testSeason{
		{Number: 1, Monitored: true, Episodes: 8, AiredEpisodes: 5, WithFiles: 5, UnairedMonitored: 0},
	})

	checker := NewLibraryChecker(q, &tdb.Logger)
	result, err := checker.CheckSeriesAvailability(ctx, 1003, nil)
	if err != nil {
		t.Fatalf("CheckSeriesAvailability error = %v", err)
	}

	if len(result.SeasonAvailability) != 1 {
		t.Fatalf("expected 1 season, got %d", len(result.SeasonAvailability))
	}

	s1 := result.SeasonAvailability[0]
	if s1.Available {
		t.Error("season should NOT be available (unaired episodes not monitored)")
	}
}

func TestGetCoveredSeasons_CrossType(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	userID := createTestUser(t, q)

	seasonsJSON, _ := json.Marshal([]int64{1, 2, 3})
	_, err := q.CreateRequest(ctx, sqlc.CreateRequestParams{
		UserID:           userID,
		MediaType:        MediaTypeSeries,
		TvdbID:           sql.NullInt64{Int64: 2000, Valid: true},
		Title:            "Test Series",
		Status:           StatusPending,
		RequestedSeasons: sql.NullString{String: string(seasonsJSON), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateRequest (series) error = %v", err)
	}

	_, err = q.CreateRequest(ctx, sqlc.CreateRequestParams{
		UserID:       userID,
		MediaType:    MediaTypeSeason,
		TvdbID:       sql.NullInt64{Int64: 2000, Valid: true},
		Title:        "Test Series S5",
		Status:       StatusPending,
		SeasonNumber: sql.NullInt64{Int64: 5, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateRequest (season) error = %v", err)
	}

	checker := NewLibraryChecker(q, &tdb.Logger)
	covered, err := checker.GetCoveredSeasons(ctx, 2000, userID)
	if err != nil {
		t.Fatalf("GetCoveredSeasons error = %v", err)
	}

	expectedSeasons := []int{1, 2, 3, 5}
	for _, sn := range expectedSeasons {
		if _, ok := covered[sn]; !ok {
			t.Errorf("expected season %d to be covered", sn)
		}
	}

	if _, ok := covered[4]; ok {
		t.Error("season 4 should NOT be covered")
	}
}

func TestCreateRequest_CrossTypeDuplicate(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	userID := createTestUser(t, q)
	svc := NewService(q, &tdb.Logger)

	tvdbID := int64(3000)
	_, err := svc.Create(ctx, userID, &CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("Create series request error = %v", err)
	}

	// Season 2 is covered by the series request → should fail
	sn := int64(2)
	_, err = svc.Create(ctx, userID, &CreateInput{
		MediaType:    MediaTypeSeason,
		TvdbID:       &tvdbID,
		Title:        "Test Series S2",
		SeasonNumber: &sn,
	})
	if !errors.Is(err, ErrAlreadyRequested) {
		t.Errorf("expected ErrAlreadyRequested for covered season, got %v", err)
	}

	// Season 4 is NOT covered → should succeed
	sn4 := int64(4)
	_, err = svc.Create(ctx, userID, &CreateInput{
		MediaType:    MediaTypeSeason,
		TvdbID:       &tvdbID,
		Title:        "Test Series S4",
		SeasonNumber: &sn4,
	})
	if err != nil {
		t.Errorf("expected season 4 request to succeed, got %v", err)
	}
}

func TestCreateRequest_PartialOverlap(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	userID := createTestUser(t, q)
	svc := NewService(q, &tdb.Logger)

	tvdbID := int64(4000)
	sn := int64(2)
	_, err := svc.Create(ctx, userID, &CreateInput{
		MediaType:    MediaTypeSeason,
		TvdbID:       &tvdbID,
		Title:        "Test Series S2",
		SeasonNumber: &sn,
	})
	if err != nil {
		t.Fatalf("Create season request error = %v", err)
	}

	// Series request with seasons [2, 3, 4] — only season 2 overlaps, 3-4 are new
	// This should succeed (partial overlap allowed)
	_, err = svc.Create(ctx, userID, &CreateInput{
		MediaType:        MediaTypeSeries,
		TvdbID:           &tvdbID,
		Title:            "Test Series",
		RequestedSeasons: []int64{2, 3, 4},
	})
	if err != nil {
		t.Errorf("expected partial overlap to succeed, got %v", err)
	}
}
