package tv

import (
	"context"
	"testing"

	fsmock "github.com/slipstream/slipstream/internal/filesystem/mock"
	"github.com/slipstream/slipstream/internal/library/quality"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/testutil"
)

// Dev mode switches the library services to the dev database but leaves the
// module's own connection on the production database, so mock seeding must go
// through the TV service rather than the module's handle.
func TestCreateMockEpisodeFilesSeedsThroughService(t *testing.T) {
	devDB := testutil.NewTestDB(t)
	defer devDB.Close()
	prodDB := testutil.NewTestDB(t)
	defer prodDB.Close()
	ctx := context.Background()
	logger := devDB.Logger

	qs := quality.NewService(devDB.Conn, &logger)
	profile, err := qs.Create(ctx, &quality.CreateProfileInput{Name: "HD", ModuleType: "tv", Cutoff: 11, Items: quality.HD1080pProfile().Items})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	tvSvc := tvlib.NewService(devDB.Conn, nil, &logger, qs, nil)
	m := NewModule(prodDB.Conn, nil, tvSvc, nil, nil, qs, &logger)

	episodeCounts := map[int]int{1: 7, 2: 13}
	var seasons []tvlib.SeasonInput
	for seasonNum, count := range episodeCounts {
		var episodes []tvlib.EpisodeInput
		for e := 1; e <= count; e++ {
			episodes = append(episodes, tvlib.EpisodeInput{EpisodeNumber: e, Title: "Episode", Monitored: true})
		}
		seasons = append(seasons, tvlib.SeasonInput{SeasonNumber: seasonNum, Monitored: true, Episodes: episodes})
	}
	path := fsmock.MockTVPath + "/Breaking Bad"
	series, err := tvSvc.CreateSeries(ctx, &tvlib.CreateSeriesInput{
		Title: "Breaking Bad", Year: 2008, TvdbID: 81189, Path: path,
		QualityProfileID: profile.ID, Monitored: true, SeasonFolder: true, Seasons: seasons,
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	m.createMockEpisodeFiles(ctx, series.ID, path)

	var fileCount int
	if err := devDB.Conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM episode_files").Scan(&fileCount); err != nil {
		t.Fatal(err)
	}
	// Season 1 has two quality tiers per episode in the VFS, season 2 has one.
	wantFiles := episodeCounts[1]*2 + episodeCounts[2]
	if fileCount != wantFiles {
		t.Fatalf("episode_files = %d, want %d", fileCount, wantFiles)
	}

	var upgradable int
	if err := devDB.Conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM episodes WHERE status = 'available'").Scan(&upgradable); err != nil {
		t.Fatal(err)
	}
	if upgradable != episodeCounts[1]+episodeCounts[2] {
		t.Fatalf("available episodes = %d, want %d", upgradable, episodeCounts[1]+episodeCounts[2])
	}
}
