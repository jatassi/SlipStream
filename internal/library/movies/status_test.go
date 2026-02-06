package movies

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/testutil"
)

// Tests for the unified status consolidation spec requirements on movies.
// Spec: docs/status-consolidation.md

func TestMovieStatus_InitialStatus_Unreleased(t *testing.T) {
	// Spec: "Determine initial status based on release date"
	// When release date is in the future or not set, status should be "unreleased"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// No release date → unreleased
	movie, err := service.Create(ctx, CreateMovieInput{
		Title: "Future Movie No Date",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie.Status != "unreleased" {
		t.Errorf("Movie without release date: status = %q, want %q", movie.Status, "unreleased")
	}

	// Future release date → unreleased
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	movie2, err := service.Create(ctx, CreateMovieInput{
		Title:       "Future Movie With Date",
		ReleaseDate: futureDate,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie2.Status != "unreleased" {
		t.Errorf("Movie with future release date: status = %q, want %q", movie2.Status, "unreleased")
	}
}

func TestMovieStatus_InitialStatus_Missing(t *testing.T) {
	// Spec: When release date is in the past, status should be "missing"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, err := service.Create(ctx, CreateMovieInput{
		Title:       "Released Movie",
		ReleaseDate: pastDate,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie.Status != "missing" {
		t.Errorf("Movie with past release date: status = %q, want %q", movie.Status, "missing")
	}
}

func TestMovieStatus_AddFile_SetsAvailable(t *testing.T) {
	// Spec: "downloading → available: File imported at or above quality cutoff"
	// Without quality profiles, AddFile defaults to "available"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
	})

	if movie.Status != "missing" {
		t.Fatalf("Pre-condition: movie should be missing, got %q", movie.Status)
	}

	_, err := service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path: "/movies/Test Movie/test.mkv",
		Size: 1500000000,
	})
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "available" {
		t.Errorf("Movie status after file add = %q, want %q", updated.Status, "available")
	}
}

func TestMovieStatus_RemoveFile_SetsMissingAndUnmonitors(t *testing.T) {
	// Spec: "available → missing: File removed manually via UI → Sets monitored = false"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
		Monitored:   true,
	})

	file, _ := service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path: "/movies/test.mkv",
		Size: 1000,
	})

	err := service.RemoveFile(ctx, file.ID)
	if err != nil {
		t.Fatalf("RemoveFile() error = %v", err)
	}

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "missing" {
		t.Errorf("Movie status after file removal = %q, want %q", updated.Status, "missing")
	}
	if updated.Monitored {
		t.Error("Movie should be unmonitored after manual file removal")
	}
}

func TestMovieStatus_UpdateReleaseDateRecalculates(t *testing.T) {
	// Spec: Status transitions when release date changes
	// "unreleased → missing: release date passes" and "missing → unreleased: date pushed to future"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create unreleased movie
	movie, _ := service.Create(ctx, CreateMovieInput{Title: "Test Movie"})
	if movie.Status != "unreleased" {
		t.Fatalf("Pre-condition: status = %q, want unreleased", movie.Status)
	}

	// Set release date to past → should become missing
	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	updated, err := service.Update(ctx, movie.ID, UpdateMovieInput{
		ReleaseDate: &pastDate,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Status != "missing" {
		t.Errorf("After setting past release date: status = %q, want %q", updated.Status, "missing")
	}

	// Push release date to future → should become unreleased
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	updated2, err := service.Update(ctx, movie.ID, UpdateMovieInput{
		ReleaseDate: &futureDate,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated2.Status != "unreleased" {
		t.Errorf("After pushing to future: status = %q, want %q", updated2.Status, "unreleased")
	}
}

func TestMovieStatus_UpdateReleaseDateNoEffectOnAvailable(t *testing.T) {
	// Spec: Only unreleased/missing status is affected by release date changes
	// Available/downloading/failed should not change
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Available Movie",
		ReleaseDate: pastDate,
	})
	_, _ = service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path: "/movies/test.mkv",
		Size: 1000,
	})

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "available" {
		t.Fatalf("Pre-condition: status = %q, want available", updated.Status)
	}

	// Changing release date to future should NOT change available status
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	updated2, err := service.Update(ctx, movie.ID, UpdateMovieInput{
		ReleaseDate: &futureDate,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated2.Status != "available" {
		t.Errorf("Available movie with future date: status = %q, want %q", updated2.Status, "available")
	}
}

func TestMovieStatus_NewFields(t *testing.T) {
	// Spec: New fields statusMessage and activeDownloadId should be present
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	movie, _ := service.Create(ctx, CreateMovieInput{Title: "Test Movie"})

	// Initially nil
	if movie.StatusMessage != nil {
		t.Errorf("StatusMessage should be nil, got %v", movie.StatusMessage)
	}
	if movie.ActiveDownloadID != nil {
		t.Errorf("ActiveDownloadID should be nil, got %v", movie.ActiveDownloadID)
	}

	// Verify the fields appear correctly on Get too
	fetched, _ := service.Get(ctx, movie.ID)
	if fetched.StatusMessage != nil {
		t.Errorf("Fetched StatusMessage should be nil, got %v", fetched.StatusMessage)
	}
	if fetched.ActiveDownloadID != nil {
		t.Errorf("Fetched ActiveDownloadID should be nil, got %v", fetched.ActiveDownloadID)
	}
}

func TestMovieStatus_ReleaseDateParsing(t *testing.T) {
	// Spec: Release dates should be stored and returned correctly
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	movie, err := service.Create(ctx, CreateMovieInput{
		Title:               "Test Movie",
		ReleaseDate:         "2020-07-16",
		PhysicalReleaseDate: "2020-12-07",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, _ := service.Get(ctx, movie.ID)

	if fetched.ReleaseDate == nil {
		t.Fatal("ReleaseDate should not be nil")
	}
	if fetched.ReleaseDate.Format("2006-01-02") != "2020-07-16" {
		t.Errorf("ReleaseDate = %s, want 2020-07-16", fetched.ReleaseDate.Format("2006-01-02"))
	}

	if fetched.PhysicalReleaseDate == nil {
		t.Fatal("PhysicalReleaseDate should not be nil")
	}
	if fetched.PhysicalReleaseDate.Format("2006-01-02") != "2020-12-07" {
		t.Errorf("PhysicalReleaseDate = %s, want 2020-12-07", fetched.PhysicalReleaseDate.Format("2006-01-02"))
	}
}

func TestMovieStatus_MultipleFileRemoval(t *testing.T) {
	// When a movie has multiple files, removing one should not change status
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
		Monitored:   true,
	})

	file1, _ := service.AddFile(ctx, movie.ID, CreateMovieFileInput{Path: "/path1.mkv", Size: 1000})
	_, _ = service.AddFile(ctx, movie.ID, CreateMovieFileInput{Path: "/path2.mkv", Size: 2000})

	// Remove first file - movie still has a file
	_ = service.RemoveFile(ctx, file1.ID)

	updated, _ := service.Get(ctx, movie.ID)
	// Status should still be available since there's still a file
	if updated.Status != "available" {
		t.Errorf("Status after removing one of two files = %q, want %q", updated.Status, "available")
	}
}

// Gap 1: Download lifecycle - missing → downloading
func TestMovieStatus_DownloadLifecycle_MissingToDownloading(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
	})
	if movie.Status != "missing" {
		t.Fatalf("Pre-condition: status = %q, want missing", movie.Status)
	}

	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "dl-1", Valid: true},
	})

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "downloading" {
		t.Errorf("Status = %q, want downloading", updated.Status)
	}
	if updated.ActiveDownloadID == nil || *updated.ActiveDownloadID != "dl-1" {
		t.Errorf("ActiveDownloadID = %v, want dl-1", updated.ActiveDownloadID)
	}
}

// Gap 1: Download lifecycle - downloading → available via file import
func TestMovieStatus_DownloadLifecycle_DownloadingToAvailable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
	})

	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "dl-1", Valid: true},
	})

	_, err := service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path: "/movies/test.mkv",
		Size: 1500000000,
	})
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "available" {
		t.Errorf("Status after import = %q, want available", updated.Status)
	}
}

// Gap 1: Download lifecycle - downloading → failed
func TestMovieStatus_DownloadLifecycle_DownloadingToFailed(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
	})

	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "dl-1", Valid: true},
	})

	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:            movie.ID,
		Status:        "failed",
		StatusMessage: sql.NullString{String: "Download stalled", Valid: true},
	})

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "failed" {
		t.Errorf("Status = %q, want failed", updated.Status)
	}
	if updated.StatusMessage == nil || *updated.StatusMessage != "Download stalled" {
		t.Errorf("StatusMessage = %v, want 'Download stalled'", updated.StatusMessage)
	}
}

// Gap 2 + 12: Import with quality evaluation - file below cutoff sets upgradable
func TestMovieStatus_ImportWithQualityEvaluation_Upgradable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := quality.NewService(tdb.Conn, tdb.Logger)
	ctx := context.Background()

	profile, err := qs.Create(ctx, quality.CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11, // Bluray-1080p
		Items:  quality.HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	service := NewService(tdb.Conn, nil, tdb.Logger)
	service.SetQualityService(qs)

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:            "Test Movie",
		ReleaseDate:      pastDate,
		QualityProfileID: profile.ID,
	})

	qualityID := int64(4) // HDTV-720p (weight=4, below cutoff weight=11)
	_, err = service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		Quality:   "HDTV-720p",
		QualityID: &qualityID,
	})
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "upgradable" {
		t.Errorf("Status with below-cutoff quality = %q, want upgradable", updated.Status)
	}
}

// Gap 2: Import with quality evaluation - file at cutoff sets available
func TestMovieStatus_ImportWithQualityEvaluation_Available(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := quality.NewService(tdb.Conn, tdb.Logger)
	ctx := context.Background()

	profile, err := qs.Create(ctx, quality.CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11,
		Items:  quality.HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	service := NewService(tdb.Conn, nil, tdb.Logger)
	service.SetQualityService(qs)

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:            "Test Movie",
		ReleaseDate:      pastDate,
		QualityProfileID: profile.ID,
	})

	qualityID := int64(11) // Bluray-1080p (at cutoff)
	_, err = service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		Quality:   "Bluray-1080p",
		QualityID: &qualityID,
	})
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "available" {
		t.Errorf("Status with at-cutoff quality = %q, want available", updated.Status)
	}
}

// Gap 3: Disappeared download
func TestMovieStatus_DisappearedDownload(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
	})

	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "dl-disappeared", Valid: true},
	})

	// Simulate download disappearing: set failed with message, clear download ID
	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:            movie.ID,
		Status:        "failed",
		StatusMessage: sql.NullString{String: "Download removed from client", Valid: true},
	})

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "failed" {
		t.Errorf("Status = %q, want failed", updated.Status)
	}
	if updated.StatusMessage == nil || *updated.StatusMessage != "Download removed from client" {
		t.Errorf("StatusMessage = %v, want 'Download removed from client'", updated.StatusMessage)
	}
}

// Gap 5: Manual retry - failed → missing
func TestMovieStatus_ManualRetry(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	queries := sqlc.New(tdb.Conn)
	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
	})

	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:            movie.ID,
		Status:        "failed",
		StatusMessage: sql.NullString{String: "Some error", Valid: true},
	})

	// Manual retry resets back to missing
	_ = queries.UpdateMovieStatus(ctx, sqlc.UpdateMovieStatusParams{
		ID:     movie.ID,
		Status: "missing",
	})

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "missing" {
		t.Errorf("Status after manual retry = %q, want missing", updated.Status)
	}
}

// Gap 8: Upgradable → missing on file removal
func TestMovieStatus_UpgradableToMissingOnFileRemoval(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := quality.NewService(tdb.Conn, tdb.Logger)
	ctx := context.Background()

	profile, err := qs.Create(ctx, quality.CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11,
		Items:  quality.HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	service := NewService(tdb.Conn, nil, tdb.Logger)
	service.SetQualityService(qs)

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:            "Test Movie",
		ReleaseDate:      pastDate,
		QualityProfileID: profile.ID,
		Monitored:        true,
	})

	qualityID := int64(4) // HDTV-720p, below cutoff
	file, _ := service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		QualityID: &qualityID,
	})

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "upgradable" {
		t.Fatalf("Pre-condition: status = %q, want upgradable", updated.Status)
	}

	_ = service.RemoveFile(ctx, file.ID)

	updated, _ = service.Get(ctx, movie.ID)
	if updated.Status != "missing" {
		t.Errorf("Status after file removal from upgradable = %q, want missing", updated.Status)
	}
}

func TestMovieStatus_AddFileWithUpgradesDisabled(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := quality.NewService(tdb.Conn, tdb.Logger)
	ctx := context.Background()

	upgradesDisabled := false
	profile, err := qs.Create(ctx, quality.CreateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          11,
		UpgradesEnabled: &upgradesDisabled,
		Items:           quality.HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	service := NewService(tdb.Conn, nil, tdb.Logger)
	service.SetQualityService(qs)

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:            "Test Movie",
		ReleaseDate:      pastDate,
		QualityProfileID: profile.ID,
	})

	qualityID := int64(4) // HDTV-720p, below cutoff 11
	_, err = service.AddFile(ctx, movie.ID, CreateMovieFileInput{
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		Quality:   "HDTV-720p",
		QualityID: &qualityID,
	})
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "available" {
		t.Errorf("Status with below-cutoff quality but upgrades disabled = %q, want available", updated.Status)
	}
}

func TestMovieStatus_ManualRetryClearsStatusMessage(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	movie, _ := service.Create(ctx, CreateMovieInput{
		Title:       "Test Movie",
		ReleaseDate: pastDate,
	})

	// Set to failed with status message
	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "failed",
		StatusMessage:    sql.NullString{String: "Download timeout", Valid: true},
		ActiveDownloadID: sql.NullString{String: "dl-1", Valid: true},
	})

	// Manual retry: reset to missing, clear message and download ID
	_ = queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movie.ID,
		Status:           "missing",
		StatusMessage:    sql.NullString{},
		ActiveDownloadID: sql.NullString{},
	})

	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "missing" {
		t.Errorf("Status after retry = %q, want missing", updated.Status)
	}
	if updated.StatusMessage != nil {
		t.Errorf("StatusMessage after retry = %v, want nil", updated.StatusMessage)
	}
	if updated.ActiveDownloadID != nil {
		t.Errorf("ActiveDownloadID after retry = %v, want nil", updated.ActiveDownloadID)
	}
}

// Gap 7: Removed fields - verify Movie struct has no legacy fields
func TestMovieStatus_RemovedFields(t *testing.T) {
	movieType := reflect.TypeOf(Movie{})

	removedFields := []string{"Released", "AvailabilityStatus", "HasFile"}
	for _, field := range removedFields {
		if _, found := movieType.FieldByName(field); found {
			t.Errorf("Movie struct should NOT have field %q (removed by status consolidation)", field)
		}
	}
}

// Theatrical release date tests: verify the priority chain digital → physical → theatrical+90

func TestMovieStatus_TheatricalOnly_RecentPast_Unreleased(t *testing.T) {
	// Movie with only a theatrical date 30 days ago → still unreleased (within 90-day window)
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	theatricalDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	movie, err := service.Create(ctx, CreateMovieInput{
		Title:                 "Recent Theatrical Movie",
		TheatricalReleaseDate: theatricalDate,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie.Status != "unreleased" {
		t.Errorf("Movie with theatrical 30 days ago (no digital): status = %q, want unreleased", movie.Status)
	}
}

func TestMovieStatus_TheatricalOnly_Over90Days_Missing(t *testing.T) {
	// Movie with theatrical date > 90 days ago → missing
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	theatricalDate := time.Now().AddDate(0, 0, -91).Format("2006-01-02")
	movie, err := service.Create(ctx, CreateMovieInput{
		Title:                 "Old Theatrical Movie",
		TheatricalReleaseDate: theatricalDate,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie.Status != "missing" {
		t.Errorf("Movie with theatrical 91 days ago: status = %q, want missing", movie.Status)
	}
}

func TestMovieStatus_DigitalPast_OverridesTheatrical(t *testing.T) {
	// Movie with digital date in past → missing, regardless of theatrical
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	digitalDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	theatricalDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	movie, err := service.Create(ctx, CreateMovieInput{
		Title:                 "Digital Released Movie",
		ReleaseDate:           digitalDate,
		TheatricalReleaseDate: theatricalDate,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie.Status != "missing" {
		t.Errorf("Movie with past digital date: status = %q, want missing", movie.Status)
	}
}

func TestMovieStatus_PhysicalPast_OverridesTheatrical(t *testing.T) {
	// Movie with physical date in past → missing, regardless of theatrical window
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	physicalDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	theatricalDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	movie, err := service.Create(ctx, CreateMovieInput{
		Title:                 "Physical Released Movie",
		PhysicalReleaseDate:   physicalDate,
		TheatricalReleaseDate: theatricalDate,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie.Status != "missing" {
		t.Errorf("Movie with past physical date: status = %q, want missing", movie.Status)
	}
}

func TestMovieStatus_NoDates_Unreleased(t *testing.T) {
	// Movie with no dates at all → unreleased
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	movie, err := service.Create(ctx, CreateMovieInput{
		Title: "No Date Movie",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if movie.Status != "unreleased" {
		t.Errorf("Movie with no dates: status = %q, want unreleased", movie.Status)
	}
}

func TestMovieStatus_TheatricalDateParsing(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	movie, err := service.Create(ctx, CreateMovieInput{
		Title:                 "Test Movie",
		ReleaseDate:           "2020-07-16",
		PhysicalReleaseDate:   "2020-12-07",
		TheatricalReleaseDate: "2020-03-15",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, _ := service.Get(ctx, movie.ID)
	if fetched.TheatricalReleaseDate == nil {
		t.Fatal("TheatricalReleaseDate should not be nil")
	}
	if fetched.TheatricalReleaseDate.Format("2006-01-02") != "2020-03-15" {
		t.Errorf("TheatricalReleaseDate = %s, want 2020-03-15", fetched.TheatricalReleaseDate.Format("2006-01-02"))
	}
}

func TestMovieStatus_UpdateTheatricalDateRecalculates(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create movie with no dates (unreleased)
	movie, _ := service.Create(ctx, CreateMovieInput{Title: "Test Movie"})
	if movie.Status != "unreleased" {
		t.Fatalf("Pre-condition: status = %q, want unreleased", movie.Status)
	}

	// Set theatrical date to >90 days ago → should become missing
	oldTheatrical := time.Now().AddDate(0, 0, -91).Format("2006-01-02")
	updated, err := service.Update(ctx, movie.ID, UpdateMovieInput{
		TheatricalReleaseDate: &oldTheatrical,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Status != "missing" {
		t.Errorf("After setting old theatrical date: status = %q, want missing", updated.Status)
	}

	// Set theatrical date to recent (within 90 days) → should become unreleased
	recentTheatrical := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	updated2, err := service.Update(ctx, movie.ID, UpdateMovieInput{
		TheatricalReleaseDate: &recentTheatrical,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated2.Status != "unreleased" {
		t.Errorf("After setting recent theatrical: status = %q, want unreleased", updated2.Status)
	}
}
