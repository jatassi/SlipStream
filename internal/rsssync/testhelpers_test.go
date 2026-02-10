package rsssync

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/testutil"
)

type testEnv struct {
	tdb       *testutil.TestDB
	queries   *sqlc.Queries
	profileID int64
	profile   *quality.Profile
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	tdb := testutil.NewTestDB(t)
	q := sqlc.New(tdb.Conn)
	qs := quality.NewService(tdb.Conn, tdb.Logger)

	p := quality.HD1080pProfile()
	created, err := qs.Create(context.Background(), quality.CreateProfileInput{
		Name:            p.Name,
		Cutoff:          p.Cutoff,
		UpgradeStrategy: p.UpgradeStrategy,
		Items:           p.Items,
	})
	if err != nil {
		tdb.Close()
		t.Fatalf("create profile: %v", err)
	}

	return &testEnv{
		tdb:       tdb,
		queries:   q,
		profileID: created.ID,
		profile:   created,
	}
}

func (e *testEnv) close() {
	e.tdb.Close()
}

func (e *testEnv) createMovie(t *testing.T, title string, tmdbID int64, year int64, status string) int64 {
	t.Helper()
	m, err := e.queries.CreateMovie(context.Background(), sqlc.CreateMovieParams{
		Title:            title,
		SortTitle:        title,
		Year:             sql.NullInt64{Int64: year, Valid: true},
		TmdbID:           sql.NullInt64{Int64: tmdbID, Valid: true},
		QualityProfileID: sql.NullInt64{Int64: e.profileID, Valid: true},
		Monitored:        1,
		Status:           status,
	})
	if err != nil {
		t.Fatalf("create movie %q: %v", title, err)
	}
	return m.ID
}

func (e *testEnv) createMovieFile(t *testing.T, movieID int64, qualityID int64) {
	t.Helper()
	_, err := e.queries.CreateMovieFile(context.Background(), sqlc.CreateMovieFileParams{
		MovieID:   movieID,
		Path:      fmt.Sprintf("/movies/file_%d.mkv", movieID),
		Size:      1000000,
		QualityID: sql.NullInt64{Int64: qualityID, Valid: true},
	})
	if err != nil {
		t.Fatalf("create movie file: %v", err)
	}
}

func (e *testEnv) createSeries(t *testing.T, title string, tvdbID int64, prodStatus string) int64 {
	t.Helper()
	s, err := e.queries.CreateSeries(context.Background(), sqlc.CreateSeriesParams{
		Title:            title,
		SortTitle:        title,
		TvdbID:           sql.NullInt64{Int64: tvdbID, Valid: true},
		QualityProfileID: sql.NullInt64{Int64: e.profileID, Valid: true},
		Monitored:        1,
		SeasonFolder:     1,
		ProductionStatus: prodStatus,
	})
	if err != nil {
		t.Fatalf("create series %q: %v", title, err)
	}
	return s.ID
}

func (e *testEnv) createSeason(t *testing.T, seriesID int64, seasonNum int64) {
	t.Helper()
	_, err := e.queries.CreateSeason(context.Background(), sqlc.CreateSeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: seasonNum,
		Monitored:    1,
	})
	if err != nil {
		t.Fatalf("create season S%02d: %v", seasonNum, err)
	}
}

func (e *testEnv) createEpisode(t *testing.T, seriesID int64, season, episode int64, status string) int64 {
	t.Helper()
	ep, err := e.queries.CreateEpisode(context.Background(), sqlc.CreateEpisodeParams{
		SeriesID:      seriesID,
		SeasonNumber:  season,
		EpisodeNumber: episode,
		Title:         sql.NullString{String: fmt.Sprintf("Episode %d", episode), Valid: true},
		Monitored:     1,
		Status:        status,
	})
	if err != nil {
		t.Fatalf("create S%02dE%02d: %v", season, episode, err)
	}
	return ep.ID
}

func (e *testEnv) createEpisodeFile(t *testing.T, episodeID int64, qualityID int64) {
	t.Helper()
	_, err := e.queries.CreateEpisodeFile(context.Background(), sqlc.CreateEpisodeFileParams{
		EpisodeID: episodeID,
		Path:      fmt.Sprintf("/tv/file_%d.mkv", episodeID),
		Size:      500000,
		QualityID: sql.NullInt64{Int64: qualityID, Valid: true},
	})
	if err != nil {
		t.Fatalf("create episode file: %v", err)
	}
}

func (e *testEnv) collectWantedItems(t *testing.T) []decisioning.SearchableItem {
	t.Helper()
	collector := &decisioning.Collector{
		Queries:        e.queries,
		Logger:         e.tdb.Logger,
		BackoffChecker: decisioning.NoBackoff{},
	}
	items, err := decisioning.CollectWantedItems(context.Background(), collector)
	if err != nil {
		t.Fatalf("CollectWantedItems: %v", err)
	}
	return items
}

func makeTorrentRelease(title string, source string, resolution int, seeders int) types.TorrentInfo {
	return types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			Title:       title,
			DownloadURL: "https://example.com/" + title,
			Source:      source,
			Resolution:  resolution,
			PublishDate: time.Now().Add(-2 * time.Hour),
		},
		Seeders:  seeders,
		Leechers: 1,
	}
}

func makeTorrentWithIDs(title string, source string, resolution int, seeders int, tmdbID int, tvdbID int) types.TorrentInfo {
	t := makeTorrentRelease(title, source, resolution, seeders)
	t.TmdbID = tmdbID
	t.TvdbID = tvdbID
	return t
}

// buildEpisodeItems manually creates individual episode SearchableItems for a series+season.
// This is needed because CollectWantedItems collapses all-missing/all-upgradable episodes
// into a single season item, but the Matcher requires individual episode items in the index
// so that matchSeasonPack can find candidates and check eligibility.
func buildEpisodeItems(seriesID int64, title string, tvdbID int, profileID int64, season int, count int, hasFile bool, qualityID int) []decisioning.SearchableItem {
	items := make([]decisioning.SearchableItem, count)
	for i := 0; i < count; i++ {
		items[i] = decisioning.SearchableItem{
			MediaType:        decisioning.MediaTypeEpisode,
			MediaID:          seriesID*100 + int64(i+1),
			Title:            title,
			TvdbID:           tvdbID,
			SeriesID:         seriesID,
			SeasonNumber:     season,
			EpisodeNumber:    i + 1,
			QualityProfileID: profileID,
			HasFile:          hasFile,
			CurrentQualityID: qualityID,
		}
	}
	return items
}

func makeNoiseReleases() []types.TorrentInfo {
	return []types.TorrentInfo{
		makeTorrentRelease("Ubuntu.24.04.LTS.Desktop.AMD64", "", 0, 500),
		makeTorrentWithIDs("The.Holdovers.2023.1080p.WEB-DL", "WEB-DL", 1080, 100, 840430, 0),
		makeTorrentRelease("Taylor.Swift.Eras.Tour.2024.1080p", "", 1080, 200),
		makeTorrentWithIDs("Shogun.S01E05.1080p.WEB-DL", "WEB-DL", 1080, 80, 0, 392369),
		makeTorrentWithIDs("Game.of.Thrones.S08E06.2160p.UHD.BluRay", "BluRay", 2160, 60, 0, 121361),
		makeTorrentWithIDs("Oppenheimer.2023.HDCAM", "HDCAM", 0, 150, 872585, 0),
		makeTorrentWithIDs("How.to.Train.Your.Dragon.2025.1080p.WEB-DL", "WEB-DL", 1080, 90, 11999, 0),
	}
}
