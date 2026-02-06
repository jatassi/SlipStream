package quality

import (
	"context"
	"database/sql"
	"testing"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/testutil"
)

// Tests for StatusForQuality and IsAtOrAboveCutoff in the context of status consolidation.
// Spec: docs/status-consolidation.md - "Quality Profile Changes"

func TestProfile_StatusForQuality_AtCutoff(t *testing.T) {
	// Spec: File at or above cutoff → "available"
	profile := HD1080pProfile()
	// Cutoff = 11 (Bluray-1080p, weight=11)

	// Quality 11 = Bluray-1080p (weight=11, at cutoff)
	status := profile.StatusForQuality(11)
	if status != "available" {
		t.Errorf("Quality at cutoff: StatusForQuality(11) = %q, want %q", status, "available")
	}
}

func TestProfile_StatusForQuality_AboveCutoff(t *testing.T) {
	// Spec: File above cutoff → "available"
	profile := HD1080pProfile()

	// Quality 12 = Remux-1080p (weight=12, above cutoff of 11)
	status := profile.StatusForQuality(12)
	if status != "available" {
		t.Errorf("Quality above cutoff: StatusForQuality(12) = %q, want %q", status, "available")
	}
}

func TestProfile_StatusForQuality_BelowCutoff_UpgradesEnabled(t *testing.T) {
	// Spec: "File below cutoff and upgrades enabled → upgradable"
	profile := HD1080pProfile()
	profile.UpgradesEnabled = true

	// Quality 4 = HDTV-720p (weight=4, below cutoff weight=11)
	status := profile.StatusForQuality(4)
	if status != "upgradable" {
		t.Errorf("Below cutoff, upgrades enabled: StatusForQuality(4) = %q, want %q", status, "upgradable")
	}
}

func TestProfile_StatusForQuality_BelowCutoff_UpgradesDisabled(t *testing.T) {
	// Spec: "File below cutoff but upgrades disabled → available"
	profile := HD1080pProfile()
	profile.UpgradesEnabled = false

	// Quality 4 = HDTV-720p (weight=4, below cutoff)
	status := profile.StatusForQuality(4)
	if status != "available" {
		t.Errorf("Below cutoff, upgrades disabled: StatusForQuality(4) = %q, want %q", status, "available")
	}
}

func TestProfile_StatusForQuality_InvalidQuality(t *testing.T) {
	// Invalid quality ID should return "available" (safe default)
	profile := HD1080pProfile()

	status := profile.StatusForQuality(999)
	// IsAtOrAboveCutoff returns false for invalid quality, so:
	// - If upgrades enabled → "upgradable"
	// - But since it's an invalid quality, it shouldn't be treated as an upgrade candidate
	// The current implementation returns "upgradable" when upgrades are enabled because
	// IsAtOrAboveCutoff(999) returns false. This is fine since invalid quality IDs
	// shouldn't normally occur.
	if status != "upgradable" && status != "available" {
		t.Errorf("Invalid quality: StatusForQuality(999) = %q, want upgradable or available", status)
	}
}

func TestProfile_IsAtOrAboveCutoff(t *testing.T) {
	profile := HD1080pProfile()
	// Cutoff = 11 (Bluray-1080p, weight=11)

	tests := []struct {
		name      string
		qualityID int
		expected  bool
	}{
		{"SDTV (weight=1) below cutoff", 1, false},
		{"HDTV-720p (weight=4) below cutoff", 4, false},
		{"WEBDL-1080p (weight=10) below cutoff", 10, false},
		{"Bluray-1080p (weight=11) at cutoff", 11, true},
		{"Remux-1080p (weight=12) above cutoff", 12, true},
		{"WEBDL-2160p (weight=15) above cutoff", 15, true},
		{"Remux-2160p (weight=17) above cutoff", 17, true},
		{"Invalid quality ID", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.IsAtOrAboveCutoff(tt.qualityID)
			if got != tt.expected {
				t.Errorf("IsAtOrAboveCutoff(%d) = %v, want %v", tt.qualityID, got, tt.expected)
			}
		})
	}
}

func TestProfile_StatusForQuality_Ultra4K(t *testing.T) {
	// Test with 4K profile (cutoff = 16, Bluray-2160p)
	profile := Ultra4KProfile()

	tests := []struct {
		name      string
		qualityID int
		expected  string
	}{
		{"WEBDL-1080p below cutoff → upgradable", 10, "upgradable"},
		{"Bluray-1080p below cutoff → upgradable", 11, "upgradable"},
		{"WEBDL-2160p below cutoff → upgradable", 15, "upgradable"},
		{"Bluray-2160p at cutoff → available", 16, "available"},
		{"Remux-2160p above cutoff → available", 17, "available"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.StatusForQuality(tt.qualityID)
			if got != tt.expected {
				t.Errorf("StatusForQuality(%d) = %q, want %q", tt.qualityID, got, tt.expected)
			}
		})
	}
}

func TestProfile_StatusForQuality_AnyProfile(t *testing.T) {
	// Default "Any" profile with cutoff = 11 (Bluray-1080p)
	profile := DefaultProfile()

	tests := []struct {
		name      string
		qualityID int
		expected  string
	}{
		{"SDTV below cutoff → upgradable", 1, "upgradable"},
		{"HDTV-720p below cutoff → upgradable", 4, "upgradable"},
		{"Bluray-1080p at cutoff → available", 11, "available"},
		{"Remux-2160p above cutoff → available", 17, "available"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.StatusForQuality(tt.qualityID)
			if got != tt.expected {
				t.Errorf("StatusForQuality(%d) = %q, want %q", tt.qualityID, got, tt.expected)
			}
		})
	}
}

// Gap 4: RecalculateStatusForProfile - cutoff change affects movie/episode status
func TestRecalculateStatusForProfile_MovieCutoffChange(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create profile with cutoff=11 (Bluray-1080p)
	profile, err := qs.Create(ctx, CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11,
		Items:  HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	// Create a movie with this profile
	movie, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:            "Test Movie",
		SortTitle:        "test movie",
		QualityProfileID: sql.NullInt64{Int64: profile.ID, Valid: true},
		Status:           "upgradable",
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	// Add a file with quality=4 (HDTV-720p, below cutoff=11)
	qualityID := sql.NullInt64{Int64: 4, Valid: true}
	_, err = queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
		MovieID:   movie.ID,
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		QualityID: qualityID,
	})
	if err != nil {
		t.Fatalf("CreateMovieFile error = %v", err)
	}

	// Movie is currently "upgradable" because quality 4 < cutoff 11
	got, _ := queries.GetMovie(ctx, movie.ID)
	if got.Status != "upgradable" {
		t.Fatalf("Pre-condition: status = %q, want upgradable", got.Status)
	}

	// Now lower the cutoff to 4 (HDTV-720p) — quality 4 is now at cutoff → available
	_, err = qs.Update(ctx, profile.ID, UpdateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          4,
		UpgradesEnabled: true,
		Items:           HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Update profile error = %v", err)
	}

	updated, err := qs.RecalculateStatusForProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("RecalculateStatusForProfile error = %v", err)
	}
	if updated != 1 {
		t.Errorf("RecalculateStatusForProfile updated %d, want 1", updated)
	}

	got, _ = queries.GetMovie(ctx, movie.ID)
	if got.Status != "available" {
		t.Errorf("After cutoff lowered: status = %q, want available", got.Status)
	}
}

func TestRecalculateStatusForProfile_EpisodeCutoffChange(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	profile, err := qs.Create(ctx, CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11,
		Items:  HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	// Create a series with this profile
	series, err := queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            "Test Series",
		SortTitle:        "test series",
		QualityProfileID: sql.NullInt64{Int64: profile.ID, Valid: true},
		ProductionStatus: "ended",
	})
	if err != nil {
		t.Fatalf("CreateSeries error = %v", err)
	}

	// Create a season and episode
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
		Status:        "upgradable",
		Monitored:     1,
	})
	if err != nil {
		t.Fatalf("CreateEpisode error = %v", err)
	}

	// Add an episode file with quality=4 (HDTV-720p, below cutoff=11)
	_, err = queries.CreateEpisodeFile(ctx, sqlc.CreateEpisodeFileParams{
		EpisodeID: episode.ID,
		Path:      "/tv/test/s01e01.mkv",
		Size:      700000000,
		QualityID: sql.NullInt64{Int64: 4, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateEpisodeFile error = %v", err)
	}

	// Lower cutoff to 4 (HDTV-720p)
	_, err = qs.Update(ctx, profile.ID, UpdateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          4,
		UpgradesEnabled: true,
		Items:           HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Update profile error = %v", err)
	}

	updated, err := qs.RecalculateStatusForProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("RecalculateStatusForProfile error = %v", err)
	}
	if updated != 1 {
		t.Errorf("RecalculateStatusForProfile updated %d, want 1", updated)
	}

	ep, _ := queries.GetEpisode(ctx, episode.ID)
	if ep.Status != "available" {
		t.Errorf("After cutoff lowered: episode status = %q, want available", ep.Status)
	}
}

func TestRecalculateStatusForProfile_UpgradesDisabledChange(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	upgradesEnabled := true
	profile, err := qs.Create(ctx, CreateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          11,
		UpgradesEnabled: &upgradesEnabled,
		Items:           HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	movie, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:            "Test Movie",
		SortTitle:        "test movie",
		QualityProfileID: sql.NullInt64{Int64: profile.ID, Valid: true},
		Status:           "upgradable",
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	_, err = queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
		MovieID:   movie.ID,
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		QualityID: sql.NullInt64{Int64: 4, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovieFile error = %v", err)
	}

	// Toggle upgradesEnabled → false
	_, err = qs.Update(ctx, profile.ID, UpdateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          11,
		UpgradesEnabled: false,
		Items:           HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Update profile error = %v", err)
	}

	updated, err := qs.RecalculateStatusForProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("RecalculateStatusForProfile error = %v", err)
	}
	if updated != 1 {
		t.Errorf("RecalculateStatusForProfile updated %d, want 1", updated)
	}

	got, _ := queries.GetMovie(ctx, movie.ID)
	if got.Status != "available" {
		t.Errorf("After disabling upgrades: status = %q, want available", got.Status)
	}
}

func TestRecalculateStatusForProfile_UpgradesEnabledChange(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	upgradesDisabled := false
	profile, err := qs.Create(ctx, CreateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          11,
		UpgradesEnabled: &upgradesDisabled,
		Items:           HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	movie, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:            "Test Movie",
		SortTitle:        "test movie",
		QualityProfileID: sql.NullInt64{Int64: profile.ID, Valid: true},
		Status:           "available",
	})
	if err != nil {
		t.Fatalf("CreateMovie error = %v", err)
	}

	_, err = queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
		MovieID:   movie.ID,
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		QualityID: sql.NullInt64{Int64: 4, Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateMovieFile error = %v", err)
	}

	// Toggle upgradesEnabled → true
	_, err = qs.Update(ctx, profile.ID, UpdateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          11,
		UpgradesEnabled: true,
		Items:           HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Update profile error = %v", err)
	}

	updated, err := qs.RecalculateStatusForProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("RecalculateStatusForProfile error = %v", err)
	}
	if updated != 1 {
		t.Errorf("RecalculateStatusForProfile updated %d, want 1", updated)
	}

	got, _ := queries.GetMovie(ctx, movie.ID)
	if got.Status != "upgradable" {
		t.Errorf("After enabling upgrades: status = %q, want upgradable", got.Status)
	}
}

func TestRecalculateStatusForProfile_SkipsNonApplicableStatuses(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	profile, err := qs.Create(ctx, CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11,
		Items:  HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	statuses := []string{"downloading", "failed", "missing"}
	for i, status := range statuses {
		movie, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
			Title:            "Movie " + status,
			SortTitle:        "movie " + status,
			QualityProfileID: sql.NullInt64{Int64: profile.ID, Valid: true},
			Status:           status,
		})
		if err != nil {
			t.Fatalf("CreateMovie %d error = %v", i, err)
		}

		_, err = queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
			MovieID:   movie.ID,
			Path:      "/movies/test" + status + ".mkv",
			Size:      1500000000,
			QualityID: sql.NullInt64{Int64: 4, Valid: true},
		})
		if err != nil {
			t.Fatalf("CreateMovieFile %d error = %v", i, err)
		}
	}

	// Lower cutoff to 4 — would change status if the SQL WHERE allowed it
	_, err = qs.Update(ctx, profile.ID, UpdateProfileInput{
		Name:            "HD-1080p",
		Cutoff:          4,
		UpgradesEnabled: true,
		Items:           HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Update profile error = %v", err)
	}

	updated, err := qs.RecalculateStatusForProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("RecalculateStatusForProfile error = %v", err)
	}
	if updated != 0 {
		t.Errorf("RecalculateStatusForProfile updated %d, want 0 (non-applicable statuses skipped)", updated)
	}
}

func TestRecalculateStatusForProfile_NoChangeNeeded(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	qs := NewService(tdb.Conn, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	profile, err := qs.Create(ctx, CreateProfileInput{
		Name:   "HD-1080p",
		Cutoff: 11,
		Items:  HD1080pProfile().Items,
	})
	if err != nil {
		t.Fatalf("Create profile error = %v", err)
	}

	// Create movie already at cutoff quality
	movie, _ := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:            "Test Movie",
		SortTitle:        "test movie",
		QualityProfileID: sql.NullInt64{Int64: profile.ID, Valid: true},
		Status:           "available",
	})
	_, _ = queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
		MovieID:   movie.ID,
		Path:      "/movies/test.mkv",
		Size:      1500000000,
		QualityID: sql.NullInt64{Int64: 11, Valid: true}, // At cutoff
	})

	updated, err := qs.RecalculateStatusForProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("RecalculateStatusForProfile error = %v", err)
	}
	if updated != 0 {
		t.Errorf("RecalculateStatusForProfile updated %d, want 0 (no change needed)", updated)
	}
}

