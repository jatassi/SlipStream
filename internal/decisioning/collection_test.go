package decisioning

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/testutil"
)

// createProfile creates an HD-1080p quality profile in DB and returns its ID.
func createProfile(t *testing.T, tdb *testutil.TestDB) int64 {
	t.Helper()
	qs := quality.NewService(tdb.Conn, tdb.Logger)
	p := quality.HD1080pProfile()
	created, err := qs.Create(context.Background(), quality.CreateProfileInput{
		Name:            p.Name,
		Cutoff:          p.Cutoff,
		UpgradeStrategy: p.UpgradeStrategy,
		Items:           p.Items,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	return created.ID
}

// createMovie creates a movie in the DB.
func createMovie(t *testing.T, q *sqlc.Queries, title string, tmdbID int64, year int64, profileID int64, status string) int64 {
	t.Helper()
	movie, err := q.CreateMovie(context.Background(), sqlc.CreateMovieParams{
		Title:            title,
		SortTitle:        title,
		Year:             sql.NullInt64{Int64: year, Valid: true},
		TmdbID:           sql.NullInt64{Int64: tmdbID, Valid: true},
		QualityProfileID: sql.NullInt64{Int64: profileID, Valid: true},
		Monitored:        1,
		Status:           status,
	})
	if err != nil {
		t.Fatalf("create movie %q: %v", title, err)
	}
	return movie.ID
}

// createMovieFile creates a movie file record.
func createMovieFile(t *testing.T, q *sqlc.Queries, movieID int64, qualityID int64) {
	t.Helper()
	_, err := q.CreateMovieFile(context.Background(), sqlc.CreateMovieFileParams{
		MovieID:   movieID,
		Path:      fmt.Sprintf("/movies/file_%d.mkv", movieID),
		Size:      1000000,
		QualityID: sql.NullInt64{Int64: qualityID, Valid: true},
	})
	if err != nil {
		t.Fatalf("create movie file: %v", err)
	}
}

// createSeries creates a series in the DB.
func createSeries(t *testing.T, q *sqlc.Queries, title string, tvdbID int64, profileID int64, prodStatus string) int64 {
	t.Helper()
	series, err := q.CreateSeries(context.Background(), sqlc.CreateSeriesParams{
		Title:            title,
		SortTitle:        title,
		TvdbID:           sql.NullInt64{Int64: tvdbID, Valid: true},
		QualityProfileID: sql.NullInt64{Int64: profileID, Valid: true},
		Monitored:        1,
		SeasonFolder:     1,
		ProductionStatus: prodStatus,
	})
	if err != nil {
		t.Fatalf("create series %q: %v", title, err)
	}
	return series.ID
}

// createSeason creates a season record.
func createSeason(t *testing.T, q *sqlc.Queries, seriesID int64, seasonNum int64) {
	t.Helper()
	_, err := q.CreateSeason(context.Background(), sqlc.CreateSeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: seasonNum,
		Monitored:    1,
	})
	if err != nil {
		t.Fatalf("create season S%02d: %v", seasonNum, err)
	}
}

// createEpisode creates an episode in the DB.
func createEpisode(t *testing.T, q *sqlc.Queries, seriesID int64, season, episode int64, status string) int64 {
	t.Helper()
	ep, err := q.CreateEpisode(context.Background(), sqlc.CreateEpisodeParams{
		SeriesID:      seriesID,
		SeasonNumber:  season,
		EpisodeNumber: episode,
		Title:         sql.NullString{String: fmt.Sprintf("Episode %d", episode), Valid: true},
		Monitored:     1,
		Status:        status,
	})
	if err != nil {
		t.Fatalf("create episode S%02dE%02d: %v", season, episode, err)
	}
	return ep.ID
}

// createEpisodeFile creates an episode file record.
func createEpisodeFile(t *testing.T, q *sqlc.Queries, episodeID int64, qualityID int64) {
	t.Helper()
	_, err := q.CreateEpisodeFile(context.Background(), sqlc.CreateEpisodeFileParams{
		EpisodeID: episodeID,
		Path:      fmt.Sprintf("/tv/file_%d.mkv", episodeID),
		Size:      500000,
		QualityID: sql.NullInt64{Int64: qualityID, Valid: true},
	})
	if err != nil {
		t.Fatalf("create episode file: %v", err)
	}
}

// Scenario 7: All 13 episodes missing — season pack eligible
func TestIsSeasonPackEligible_AllMissing(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "Breaking Bad", 81189, profileID, "ended")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 13; i++ {
		createEpisode(t, q, seriesID, 3, i, "missing")
	}

	if !IsSeasonPackEligible(ctx, q, tdb.Logger, seriesID, 3) {
		t.Error("expected season pack eligible when all 13 episodes are missing")
	}
}

// Scenario 8: All 13 episodes upgradable — season pack upgrade eligible
func TestIsSeasonPackUpgradeEligible_AllUpgradable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "Breaking Bad", 81189, profileID, "ended")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 13; i++ {
		epID := createEpisode(t, q, seriesID, 3, i, "upgradable")
		createEpisodeFile(t, q, epID, 6) // WEBDL-720p
	}

	if !IsSeasonPackUpgradeEligible(ctx, q, tdb.Logger, seriesID, 3) {
		t.Error("expected season pack upgrade eligible when all 13 episodes are upgradable")
	}
}

// Scenario 9: 10 upgradable + 3 at cutoff — NOT eligible for season pack
func TestIsSeasonPackUpgradeEligible_MixedUpgradableAndCutoff(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "Breaking Bad", 81189, profileID, "ended")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 10; i++ {
		epID := createEpisode(t, q, seriesID, 3, i, "upgradable")
		createEpisodeFile(t, q, epID, 6) // WEBDL-720p
	}
	for i := int64(11); i <= 13; i++ {
		epID := createEpisode(t, q, seriesID, 3, i, "available")
		createEpisodeFile(t, q, epID, 11) // Bluray-1080p (at cutoff)
	}

	if IsSeasonPackUpgradeEligible(ctx, q, tdb.Logger, seriesID, 3) {
		t.Error("season pack upgrade should NOT be eligible when some episodes are at cutoff")
	}
}

// Scenario 10: 5 missing + 3 unreleased — NOT eligible for season pack
func TestIsSeasonPackEligible_MissingPlusUnreleased(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "The Mandalorian", 82856, profileID, "continuing")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 5; i++ {
		createEpisode(t, q, seriesID, 3, i, "missing")
	}
	for i := int64(6); i <= 8; i++ {
		createEpisode(t, q, seriesID, 3, i, "unreleased")
	}

	if IsSeasonPackEligible(ctx, q, tdb.Logger, seriesID, 3) {
		t.Error("season pack should NOT be eligible when unreleased episodes present")
	}
}

// Scenario 11: 1 missing + 9 unreleased — NOT eligible
func TestIsSeasonPackEligible_SingleMissingPlusUnreleased(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "House of the Dragon", 94997, profileID, "continuing")
	createSeason(t, q, seriesID, 3)
	createEpisode(t, q, seriesID, 3, 1, "missing")
	for i := int64(2); i <= 10; i++ {
		createEpisode(t, q, seriesID, 3, i, "unreleased")
	}

	if IsSeasonPackEligible(ctx, q, tdb.Logger, seriesID, 3) {
		t.Error("season pack should NOT be eligible with unreleased episodes")
	}
}

// Scenario 14: 1 available + 6 missing — NOT eligible (not all missing)
func TestIsSeasonPackEligible_PartiallyAvailable(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "The Last of Us", 100088, profileID, "continuing")
	createSeason(t, q, seriesID, 2)
	epID := createEpisode(t, q, seriesID, 2, 1, "available")
	createEpisodeFile(t, q, epID, 10) // WEBDL-1080p
	for i := int64(2); i <= 7; i++ {
		createEpisode(t, q, seriesID, 2, i, "missing")
	}

	if IsSeasonPackEligible(ctx, q, tdb.Logger, seriesID, 2) {
		t.Error("season pack should NOT be eligible when E01 is available")
	}
}

// Single episode doesn't qualify for season pack (needs >1)
func TestIsSeasonPackEligible_SingleEpisode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "Special", 99999, profileID, "ended")
	createSeason(t, q, seriesID, 1)
	createEpisode(t, q, seriesID, 1, 1, "missing")

	if IsSeasonPackEligible(ctx, q, tdb.Logger, seriesID, 1) {
		t.Error("season pack should NOT be eligible with only 1 episode")
	}
}

// CollectWantedItems: missing movie appears
func TestCollectWantedItems_MissingMovie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	createMovie(t, q, "Dune Part Two", 693134, 2024, profileID, "missing")

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	var found bool
	for _, item := range items {
		if item.MediaType == MediaTypeMovie && item.Title == "Dune Part Two" {
			found = true
			if item.HasFile {
				t.Error("missing movie should not have HasFile=true")
			}
		}
	}
	if !found {
		t.Error("missing movie not found in wanted items")
	}
}

// CollectWantedItems: upgradable movie appears with correct CurrentQualityID
func TestCollectWantedItems_UpgradableMovie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	movieID := createMovie(t, q, "Inception", 27205, 2010, profileID, "upgradable")
	createMovieFile(t, q, movieID, 6) // WEBDL-720p

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	var found bool
	for _, item := range items {
		if item.MediaType == MediaTypeMovie && item.Title == "Inception" {
			found = true
			if !item.HasFile {
				t.Error("upgradable movie should have HasFile=true")
			}
			if item.CurrentQualityID != 6 {
				t.Errorf("expected CurrentQualityID=6, got %d", item.CurrentQualityID)
			}
		}
	}
	if !found {
		t.Error("upgradable movie not found in wanted items")
	}
}

// CollectWantedItems: Scenario 7 — all missing episodes produce season-level item
func TestCollectWantedItems_AllMissingEpisodes_SeasonItem(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "Breaking Bad", 81189, profileID, "ended")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 13; i++ {
		createEpisode(t, q, seriesID, 3, i, "missing")
	}

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	var seasonItems, episodeItems int
	for _, item := range items {
		if item.Title == "Breaking Bad" {
			if item.MediaType == MediaTypeSeason {
				seasonItems++
				if item.SeasonNumber != 3 {
					t.Errorf("expected SeasonNumber=3, got %d", item.SeasonNumber)
				}
			} else if item.MediaType == MediaTypeEpisode {
				episodeItems++
			}
		}
	}
	if seasonItems != 1 {
		t.Errorf("expected 1 season item, got %d", seasonItems)
	}
	if episodeItems != 0 {
		t.Errorf("expected 0 individual episode items (collapsed into season), got %d", episodeItems)
	}
}

// CollectWantedItems: Scenario 8 — all upgradable episodes produce season-level item
func TestCollectWantedItems_AllUpgradableEpisodes_SeasonItem(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "Breaking Bad", 81189, profileID, "ended")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 13; i++ {
		epID := createEpisode(t, q, seriesID, 3, i, "upgradable")
		createEpisodeFile(t, q, epID, 6) // WEBDL-720p
	}

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	var seasonItems int
	for _, item := range items {
		if item.Title == "Breaking Bad" && item.MediaType == MediaTypeSeason {
			seasonItems++
			if !item.HasFile {
				t.Error("season upgrade item should have HasFile=true")
			}
			if item.CurrentQualityID != 6 {
				t.Errorf("expected CurrentQualityID=6, got %d", item.CurrentQualityID)
			}
		}
	}
	if seasonItems != 1 {
		t.Errorf("expected 1 season item, got %d", seasonItems)
	}
}

// CollectWantedItems: Scenario 9 — 10 upgradable + 3 at cutoff → 10 individual items
func TestCollectWantedItems_MixedUpgradableAndCutoff_IndividualItems(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "Breaking Bad", 81189, profileID, "ended")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 10; i++ {
		epID := createEpisode(t, q, seriesID, 3, i, "upgradable")
		createEpisodeFile(t, q, epID, 6)
	}
	for i := int64(11); i <= 13; i++ {
		epID := createEpisode(t, q, seriesID, 3, i, "available")
		createEpisodeFile(t, q, epID, 11)
	}

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	var epCount, seasonCount int
	for _, item := range items {
		if item.Title == "Breaking Bad" {
			if item.MediaType == MediaTypeEpisode {
				epCount++
			} else if item.MediaType == MediaTypeSeason {
				seasonCount++
			}
		}
	}
	if seasonCount != 0 {
		t.Errorf("expected 0 season items, got %d", seasonCount)
	}
	if epCount != 10 {
		t.Errorf("expected 10 individual episode items, got %d", epCount)
	}
}

// CollectWantedItems: Scenario 10 — 5 missing + 3 unreleased → 5 individual items
func TestCollectWantedItems_MissingPlusUnreleased_IndividualItems(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "The Mandalorian", 82856, profileID, "continuing")
	createSeason(t, q, seriesID, 3)
	for i := int64(1); i <= 5; i++ {
		createEpisode(t, q, seriesID, 3, i, "missing")
	}
	for i := int64(6); i <= 8; i++ {
		createEpisode(t, q, seriesID, 3, i, "unreleased")
	}

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	var epCount int
	for _, item := range items {
		if item.Title == "The Mandalorian" && item.MediaType == MediaTypeEpisode {
			epCount++
		}
	}
	if epCount != 5 {
		t.Errorf("expected 5 individual episode items, got %d", epCount)
	}
}

// CollectWantedItems: Scenario 11 — 1 missing + 9 unreleased → 1 individual item
func TestCollectWantedItems_SingleMissingPlusUnreleased(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "House of the Dragon", 94997, profileID, "continuing")
	createSeason(t, q, seriesID, 3)
	createEpisode(t, q, seriesID, 3, 1, "missing")
	for i := int64(2); i <= 10; i++ {
		createEpisode(t, q, seriesID, 3, i, "unreleased")
	}

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	var epCount int
	for _, item := range items {
		if item.Title == "House of the Dragon" && item.MediaType == MediaTypeEpisode {
			epCount++
			if item.EpisodeNumber != 1 {
				t.Errorf("expected episode 1, got %d", item.EpisodeNumber)
			}
		}
	}
	if epCount != 1 {
		t.Errorf("expected 1 individual episode item, got %d", epCount)
	}
}

// CollectWantedItems: failed movies are excluded
func TestCollectWantedItems_FailedMovieExcluded(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	createMovie(t, q, "Failed Movie", 999999, 2024, profileID, "failed")

	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}

	for _, item := range items {
		if item.Title == "Failed Movie" {
			t.Error("failed movie should not appear in wanted items")
		}
	}
}

// Scenario 13: Transition test — Phase A (with unreleased) then Phase B (all missing)
func TestCollectWantedItems_SeasonTransition(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()
	profileID := createProfile(t, tdb)

	seriesID := createSeries(t, q, "The Last of Us", 100088, profileID, "continuing")
	createSeason(t, q, seriesID, 2)
	for i := int64(1); i <= 6; i++ {
		createEpisode(t, q, seriesID, 2, i, "missing")
	}
	createEpisode(t, q, seriesID, 2, 7, "unreleased")

	// Phase A: E07 unreleased → no season pack
	collector := &Collector{
		Queries:        q,
		Logger:         tdb.Logger,
		BackoffChecker: NoBackoff{},
	}
	items, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("Phase A: %v", err)
	}

	var seasonCount, epCount int
	for _, item := range items {
		if item.Title == "The Last of Us" {
			if item.MediaType == MediaTypeSeason {
				seasonCount++
			} else if item.MediaType == MediaTypeEpisode {
				epCount++
			}
		}
	}
	if seasonCount != 0 {
		t.Errorf("Phase A: expected 0 season items, got %d", seasonCount)
	}
	if epCount != 6 {
		t.Errorf("Phase A: expected 6 episode items, got %d", epCount)
	}

	// Phase B: E07 transitions to missing → season pack eligible
	episodes, _ := q.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: 2,
	})
	for _, ep := range episodes {
		if ep.EpisodeNumber == 7 {
			_ = q.UpdateEpisodeStatus(ctx, sqlc.UpdateEpisodeStatusParams{
				ID:     ep.ID,
				Status: "missing",
			})
		}
	}

	items2, err := CollectWantedItems(ctx, collector)
	if err != nil {
		t.Fatalf("Phase B: %v", err)
	}

	seasonCount = 0
	epCount = 0
	for _, item := range items2 {
		if item.Title == "The Last of Us" {
			if item.MediaType == MediaTypeSeason {
				seasonCount++
			} else if item.MediaType == MediaTypeEpisode {
				epCount++
			}
		}
	}
	if seasonCount != 1 {
		t.Errorf("Phase B: expected 1 season item, got %d", seasonCount)
	}
	if epCount != 0 {
		t.Errorf("Phase B: expected 0 individual episode items, got %d", epCount)
	}
}
