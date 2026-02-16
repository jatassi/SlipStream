package tv

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

// Tests for the unified status consolidation spec requirements on TV series.
// Spec: docs/status-consolidation.md

func TestEpisodeStatus_InitialStatus_Unreleased(t *testing.T) {
	// Spec: Episodes with no air date or future air date should be "unreleased"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	futureDate := time.Now().AddDate(1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "No Air Date", Monitored: true},
					{EpisodeNumber: 2, Title: "Future Air Date", AirDate: &futureDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, testutil.IntPtr(1))

	for _, ep := range episodes {
		if ep.Status != "unreleased" {
			t.Errorf("Episode %q: status = %q, want %q", ep.Title, ep.Status, "unreleased")
		}
	}
}

func TestEpisodeStatus_InitialStatus_Missing(t *testing.T) {
	// Spec: Episodes with air date in the past should be "missing"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Aired Episode", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, testutil.IntPtr(1))
	if len(episodes) != 1 {
		t.Fatalf("Expected 1 episode, got %d", len(episodes))
	}

	if episodes[0].Status != "missing" {
		t.Errorf("Episode with past air date: status = %q, want %q", episodes[0].Status, "missing")
	}
}

func TestEpisodeStatus_AddFile_SetsAvailable(t *testing.T) {
	// Spec: File imported at or above quality cutoff → "available"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	ep := episodes[0]

	_, err := service.AddEpisodeFile(ctx, ep.ID, &CreateEpisodeFileInput{
		Path: "/tv/Test/S01E01.mkv",
		Size: 500000000,
	})
	if err != nil {
		t.Fatalf("AddEpisodeFile() error = %v", err)
	}

	updated, _ := service.GetEpisode(ctx, ep.ID)
	if updated.Status != "available" {
		t.Errorf("Episode status after file add = %q, want %q", updated.Status, "available")
	}
}

func TestEpisodeStatus_RemoveFile_SetsMissingAndUnmonitors(t *testing.T) {
	// Spec: "available → missing: File removed manually via UI → Sets monitored = false"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	file, _ := service.AddEpisodeFile(ctx, episodes[0].ID, &CreateEpisodeFileInput{
		Path: "/tv/Test/S01E01.mkv",
		Size: 1000,
	})

	err := service.RemoveEpisodeFile(ctx, file.ID)
	if err != nil {
		t.Fatalf("RemoveEpisodeFile() error = %v", err)
	}

	updated, _ := service.GetEpisode(ctx, episodes[0].ID)
	if updated.Status != "missing" {
		t.Errorf("Episode status after file removal = %q, want %q", updated.Status, "missing")
	}
	if updated.Monitored {
		t.Error("Episode should be unmonitored after manual file removal")
	}
}

func TestEpisodeStatus_NewFields(t *testing.T) {
	// Spec: New fields statusMessage and activeDownloadId on Episode
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	ep, _ := service.GetEpisode(ctx, episodes[0].ID)

	if ep.StatusMessage != nil {
		t.Errorf("StatusMessage should be nil, got %v", ep.StatusMessage)
	}
	if ep.ActiveDownloadID != nil {
		t.Errorf("ActiveDownloadID should be nil, got %v", ep.ActiveDownloadID)
	}
}

func TestSeriesStatus_ProductionStatus(t *testing.T) {
	// Spec: "status → production_status (same CHECK constraint: continuing, ended, upcoming)"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	tests := []struct {
		name             string
		productionStatus string
		expected         string
	}{
		{"Default is continuing", "", "continuing"},
		{"Explicit continuing", "continuing", "continuing"},
		{"Ended", "ended", "ended"},
		{"Upcoming", "upcoming", "upcoming"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := service.CreateSeries(ctx, &CreateSeriesInput{
				Title:            "Test " + tt.name,
				ProductionStatus: tt.productionStatus,
			})
			if err != nil {
				t.Fatalf("CreateSeries() error = %v", err)
			}
			if series.ProductionStatus != tt.expected {
				t.Errorf("ProductionStatus = %q, want %q", series.ProductionStatus, tt.expected)
			}
		})
	}
}

func TestSeriesStatus_StatusCounts(t *testing.T) {
	// Spec: "Series statusCounts: Aggregate status values of all episodes across all seasons"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	futureDate := time.Now().AddDate(1, 0, 0)

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Aired Ep 1", AirDate: &pastDate, Monitored: true},
					{EpisodeNumber: 2, Title: "Aired Ep 2", AirDate: &pastDate, Monitored: true},
				},
			},
			{
				SeasonNumber: 2,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Unreleased", AirDate: &futureDate, Monitored: true},
				},
			},
		},
	})

	// Add a file to S01E01 to make it available
	episodes, _ := service.ListEpisodes(ctx, series.ID, testutil.IntPtr(1))
	_, _ = service.AddEpisodeFile(ctx, episodes[0].ID, &CreateEpisodeFileInput{
		Path: "/tv/Test/S01E01.mkv",
		Size: 1000,
	})

	// Re-fetch series with counts
	fetched, err := service.GetSeries(ctx, series.ID)
	if err != nil {
		t.Fatalf("GetSeries() error = %v", err)
	}

	counts := fetched.StatusCounts
	if counts.Total != 3 {
		t.Errorf("Total = %d, want 3", counts.Total)
	}
	if counts.Available != 1 {
		t.Errorf("Available = %d, want 1", counts.Available)
	}
	if counts.Missing != 1 {
		t.Errorf("Missing = %d, want 1", counts.Missing)
	}
	if counts.Unreleased != 1 {
		t.Errorf("Unreleased = %d, want 1", counts.Unreleased)
	}
	if counts.Downloading != 0 {
		t.Errorf("Downloading = %d, want 0", counts.Downloading)
	}
	if counts.Failed != 0 {
		t.Errorf("Failed = %d, want 0", counts.Failed)
	}
	if counts.Upgradable != 0 {
		t.Errorf("Upgradable = %d, want 0", counts.Upgradable)
	}
}

func TestSeasonStatus_StatusCounts(t *testing.T) {
	// Spec: "Season statusCounts: Aggregate status values of all episodes in the season"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	futureDate := time.Now().AddDate(1, 0, 0)

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Aired", AirDate: &pastDate, Monitored: true},
					{EpisodeNumber: 2, Title: "Future", AirDate: &futureDate, Monitored: true},
					{EpisodeNumber: 3, Title: "Also Aired", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	// Add file to episode 1
	episodes, _ := service.ListEpisodes(ctx, series.ID, testutil.IntPtr(1))
	_, _ = service.AddEpisodeFile(ctx, episodes[0].ID, &CreateEpisodeFileInput{
		Path: "/tv/Test/S01E01.mkv",
		Size: 1000,
	})

	seasons, err := service.ListSeasons(ctx, series.ID)
	if err != nil {
		t.Fatalf("ListSeasons() error = %v", err)
	}

	if len(seasons) != 1 {
		t.Fatalf("Expected 1 season, got %d", len(seasons))
	}

	counts := seasons[0].StatusCounts
	if counts.Total != 3 {
		t.Errorf("Total = %d, want 3", counts.Total)
	}
	if counts.Available != 1 {
		t.Errorf("Available = %d, want 1 (episode with file)", counts.Available)
	}
	if counts.Missing != 1 {
		t.Errorf("Missing = %d, want 1 (aired episode without file)", counts.Missing)
	}
	if counts.Unreleased != 1 {
		t.Errorf("Unreleased = %d, want 1 (future episode)", counts.Unreleased)
	}
}

func TestEpisodeStatus_AirDateDatetime(t *testing.T) {
	// Spec: "air_date type: DATE → DATETIME (to support air time, not just date)"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	// Test with full datetime including time
	airTime := time.Date(2024, 3, 15, 21, 0, 0, 0, time.UTC)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", AirDate: &airTime, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	if episodes[0].AirDate == nil {
		t.Fatal("AirDate should not be nil")
	}
	if episodes[0].AirDate.Hour() != 21 {
		t.Errorf("AirDate hour = %d, want 21 (datetime should preserve time)", episodes[0].AirDate.Hour())
	}
}

func TestComputeEpisodeStatus(t *testing.T) {
	// Unit test for computeEpisodeStatus helper
	tests := []struct {
		name     string
		airDate  *time.Time
		expected string
	}{
		{"nil air date → unreleased", nil, "unreleased"},
		{"future air date → unreleased", func() *time.Time { t := time.Now().AddDate(1, 0, 0); return &t }(), "unreleased"},
		{"past air date → missing", func() *time.Time { t := time.Now().AddDate(-1, 0, 0); return &t }(), "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeEpisodeStatus(tt.airDate)
			if got != tt.expected {
				t.Errorf("computeEpisodeStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSeriesStatus_StatusCountsAllAvailable(t *testing.T) {
	// Spec: Frontend display "All available → Available"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Complete Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Ep 1", AirDate: &pastDate, Monitored: true},
					{EpisodeNumber: 2, Title: "Ep 2", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	for _, ep := range episodes {
		_, _ = service.AddEpisodeFile(ctx, ep.ID, &CreateEpisodeFileInput{
			Path: "/tv/" + ep.Title + ".mkv",
			Size: 1000,
		})
	}

	fetched, _ := service.GetSeries(ctx, series.ID)
	if fetched.StatusCounts.Available != 2 {
		t.Errorf("Available = %d, want 2", fetched.StatusCounts.Available)
	}
	if fetched.StatusCounts.Missing != 0 {
		t.Errorf("Missing = %d, want 0", fetched.StatusCounts.Missing)
	}
}

func TestSeriesStatus_StatusCountsAllUnreleased(t *testing.T) {
	// Spec: Frontend display "All unreleased → Unreleased"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	futureDate := time.Now().AddDate(1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Upcoming Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Ep 1", AirDate: &futureDate, Monitored: true},
					{EpisodeNumber: 2, Title: "Ep 2", AirDate: &futureDate, Monitored: true},
				},
			},
		},
	})

	fetched, _ := service.GetSeries(ctx, series.ID)
	if fetched.StatusCounts.Unreleased != 2 {
		t.Errorf("Unreleased = %d, want 2", fetched.StatusCounts.Unreleased)
	}
	if fetched.StatusCounts.Total != 2 {
		t.Errorf("Total = %d, want 2", fetched.StatusCounts.Total)
	}
}

// Gap 1: Episode download lifecycle
func TestEpisodeStatus_DownloadLifecycle(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	ep := episodes[0]
	if ep.Status != "missing" {
		t.Fatalf("Pre-condition: status = %q, want missing", ep.Status)
	}

	// Transition to downloading via direct DB
	_ = queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:               ep.ID,
		Status:           "downloading",
		ActiveDownloadID: sql.NullString{String: "dl-ep-1", Valid: true},
	})

	updated, _ := service.GetEpisode(ctx, ep.ID)
	if updated.Status != "downloading" {
		t.Errorf("Status = %q, want downloading", updated.Status)
	}
	if updated.ActiveDownloadID == nil || *updated.ActiveDownloadID != "dl-ep-1" {
		t.Errorf("ActiveDownloadID = %v, want dl-ep-1", updated.ActiveDownloadID)
	}

	// Import file → should transition to available
	_, err := service.AddEpisodeFile(ctx, ep.ID, &CreateEpisodeFileInput{
		Path: "/tv/Test/S01E01.mkv",
		Size: 500000000,
	})
	if err != nil {
		t.Fatalf("AddEpisodeFile() error = %v", err)
	}

	updated, _ = service.GetEpisode(ctx, ep.ID)
	if updated.Status != "available" {
		t.Errorf("Status after import = %q, want available", updated.Status)
	}
}

// Gap 2 + 12: Episode import with quality evaluation
func TestEpisodeStatus_ImportWithQualityEvaluation(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := quality.NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	profile, err := qs.Create(ctx, &quality.CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11,
		Items:  quality.HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	service.SetQualityService(qs)

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title:            "Test Series",
		QualityProfileID: profile.ID,
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Ep1", AirDate: &pastDate, Monitored: true},
					{EpisodeNumber: 2, Title: "Ep2", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)

	// Below cutoff → upgradable
	belowCutoff := int64(4) // HDTV-720p
	_, err = service.AddEpisodeFile(ctx, episodes[0].ID, &CreateEpisodeFileInput{
		Path:      "/tv/Test/S01E01.mkv",
		Size:      500000000,
		QualityID: &belowCutoff,
	})
	if err != nil {
		t.Fatalf("AddEpisodeFile() error = %v", err)
	}

	ep1, _ := service.GetEpisode(ctx, episodes[0].ID)
	if ep1.Status != "upgradable" {
		t.Errorf("Episode with below-cutoff quality: status = %q, want upgradable", ep1.Status)
	}

	// At cutoff → available
	atCutoff := int64(11) // Bluray-1080p
	_, err = service.AddEpisodeFile(ctx, episodes[1].ID, &CreateEpisodeFileInput{
		Path:      "/tv/Test/S01E02.mkv",
		Size:      500000000,
		QualityID: &atCutoff,
	})
	if err != nil {
		t.Fatalf("AddEpisodeFile() error = %v", err)
	}

	ep2, _ := service.GetEpisode(ctx, episodes[1].ID)
	if ep2.Status != "available" {
		t.Errorf("Episode with at-cutoff quality: status = %q, want available", ep2.Status)
	}
}

// Gap 11: Season 0 (specials) excluded from series StatusCounts
func TestSeriesStatus_Season0Exclusion(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 0, // Specials
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Special 1", AirDate: &pastDate, Monitored: true},
					{EpisodeNumber: 2, Title: "Special 2", AirDate: &pastDate, Monitored: true},
				},
			},
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Ep 1", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	fetched, _ := service.GetSeries(ctx, series.ID)

	// StatusCounts should exclude season 0 episodes
	// Only season 1 episode (1 missing) should be counted
	if fetched.StatusCounts.Total != 1 {
		t.Errorf("Total = %d, want 1 (season 0 should be excluded)", fetched.StatusCounts.Total)
	}
	if fetched.StatusCounts.Missing != 1 {
		t.Errorf("Missing = %d, want 1", fetched.StatusCounts.Missing)
	}
}

func TestEpisodeStatus_AddFileWithUpgradesDisabled(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := quality.NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	upgradesDisabled := false
	profile, err := qs.Create(ctx, &quality.CreateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          11,
		UpgradesEnabled: &upgradesDisabled,
		Items:           quality.HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	service.SetQualityService(qs)

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title:            "Test Series",
		QualityProfileID: profile.ID,
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)

	belowCutoff := int64(4) // HDTV-720p, below cutoff 11
	_, err = service.AddEpisodeFile(ctx, episodes[0].ID, &CreateEpisodeFileInput{
		Path:      "/tv/Test/S01E01.mkv",
		Size:      500000000,
		QualityID: &belowCutoff,
	})
	if err != nil {
		t.Fatalf("AddEpisodeFile() error = %v", err)
	}

	ep, _ := service.GetEpisode(ctx, episodes[0].ID)
	if ep.Status != "available" {
		t.Errorf("Status with below-cutoff quality but upgrades disabled = %q, want available", ep.Status)
	}
}

func TestEpisodeStatus_ManualRetryClearsStatusMessage(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	pastDate := time.Now().AddDate(-1, 0, 0)
	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", AirDate: &pastDate, Monitored: true},
				},
			},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	ep := episodes[0]

	// Set to failed with status message
	_ = queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:               ep.ID,
		Status:           "failed",
		StatusMessage:    sql.NullString{String: "Download timeout", Valid: true},
		ActiveDownloadID: sql.NullString{String: "dl-ep-1", Valid: true},
	})

	// Manual retry: reset to missing, clear message and download ID
	_ = queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:               ep.ID,
		Status:           "missing",
		StatusMessage:    sql.NullString{},
		ActiveDownloadID: sql.NullString{},
	})

	updated, _ := service.GetEpisode(ctx, ep.ID)
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

// Gap 7: Removed fields on Episode and Series
func TestEpisodeStatus_RemovedFields(t *testing.T) {
	episodeType := reflect.TypeOf(Episode{})
	removedEpFields := []string{"Released", "HasFile"}
	for _, field := range removedEpFields {
		if _, found := episodeType.FieldByName(field); found {
			t.Errorf("Episode struct should NOT have field %q", field)
		}
	}

	seriesType := reflect.TypeOf(Series{})
	removedSeriesFields := []string{"Released", "AvailabilityStatus", "EpisodeCount", "EpisodeFileCount"}
	for _, field := range removedSeriesFields {
		if _, found := seriesType.FieldByName(field); found {
			t.Errorf("Series struct should NOT have field %q", field)
		}
	}
}
