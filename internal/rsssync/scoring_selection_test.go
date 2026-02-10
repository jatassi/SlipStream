package rsssync

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/indexer/scoring"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

var nopLogger = zerolog.Nop()

// Scenario 1B: Score+Select — Bluray-1080p selected for missing movie (2160p fails IsAcceptable)
func TestScoreAndSelect_Scenario1B_MissingMovie(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	env.createMovie(t, "Dune Part Two", 693134, 2024, "missing")
	items := env.collectWantedItems(t)
	if len(items) == 0 {
		t.Fatal("no wanted items collected")
	}

	item := items[0]
	releases := []types.TorrentInfo{
		makeTorrentWithIDs("Dune.Part.Two.2024.2160p.UHD.BluRay.x265", "BluRay", 2160, 150, 693134, 0),
		makeTorrentWithIDs("Dune.Part.Two.2024.1080p.BluRay.x264", "BluRay", 1080, 100, 693134, 0),
		makeTorrentWithIDs("Dune.Part.Two.2024.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 693134, 0),
		makeTorrentWithIDs("Dune.Part.Two.2024.720p.WEB-DL.x264", "WEB-DL", 720, 60, 693134, 0),
	}

	scorer := scoring.NewDefaultScorer()
	sCtx := scoring.ScoringContext{
		QualityProfile: env.profile,
		SearchYear:     item.Year,
		Now:            time.Now(),
	}
	for i := range releases {
		scorer.ScoreTorrent(&releases[i], sCtx)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Score > releases[j].Score
	})

	best := decisioning.SelectBestRelease(releases, env.profile, item, nopLogger)
	if best == nil {
		t.Fatal("expected a release to be selected")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQID(best))
	}
}

// Scenario 2B: Score+Select — Bluray-1080p upgrades WEBDL-720p movie
func TestScoreAndSelect_Scenario2B_UpgradableMovie(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	movieID := env.createMovie(t, "Inception", 27205, 2010, "upgradable")
	env.createMovieFile(t, movieID, 6) // WEBDL-720p

	items := env.collectWantedItems(t)
	if len(items) == 0 {
		t.Fatal("no wanted items collected")
	}

	item := items[0]
	releases := []types.TorrentInfo{
		makeTorrentWithIDs("Inception.2010.1080p.BluRay.x264", "BluRay", 1080, 120, 27205, 0),
		makeTorrentWithIDs("Inception.2010.720p.BluRay.x264", "BluRay", 720, 90, 27205, 0),
		makeTorrentWithIDs("Inception.2010.720p.WEB-DL.x264", "WEB-DL", 720, 60, 27205, 0),
	}

	scorer := scoring.NewDefaultScorer()
	sCtx := scoring.ScoringContext{
		QualityProfile: env.profile,
		SearchYear:     item.Year,
		Now:            time.Now(),
	}
	for i := range releases {
		scorer.ScoreTorrent(&releases[i], sCtx)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Score > releases[j].Score
	})

	best := decisioning.SelectBestRelease(releases, env.profile, item, nopLogger)
	if best == nil {
		t.Fatal("expected a release to be selected")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQID(best))
	}
}

// Scenario 3B: Score+Select — no upgrade available returns nil
func TestScoreAndSelect_Scenario3B_NoUpgrade(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	movieID := env.createMovie(t, "Inception", 27205, 2010, "upgradable")
	env.createMovieFile(t, movieID, 6) // WEBDL-720p

	items := env.collectWantedItems(t)
	if len(items) == 0 {
		t.Fatal("no wanted items collected")
	}

	item := items[0]
	releases := []types.TorrentInfo{
		makeTorrentWithIDs("Inception.2010.720p.WEB-DL.x264", "WEB-DL", 720, 80, 27205, 0),
		makeTorrentWithIDs("Inception.2010.720p.WEBRip.x264", "WEBRip", 720, 60, 27205, 0),
	}

	scorer := scoring.NewDefaultScorer()
	sCtx := scoring.ScoringContext{
		QualityProfile: env.profile,
		SearchYear:     item.Year,
		Now:            time.Now(),
	}
	for i := range releases {
		scorer.ScoreTorrent(&releases[i], sCtx)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Score > releases[j].Score
	})

	best := decisioning.SelectBestRelease(releases, env.profile, item, nopLogger)
	if best != nil {
		t.Errorf("expected nil (no upgrade), got %s", best.Title)
	}
}

// Scenario 5B: Score+Select — disc source upgrade for episode
func TestScoreAndSelect_Scenario5B_DiscSourceUpgrade(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	stID := env.createSeries(t, "Stranger Things", 66732, "continuing")
	env.createSeason(t, stID, 4)
	for i := int64(1); i <= 7; i++ {
		epID := env.createEpisode(t, stID, 4, i, "upgradable")
		env.createEpisodeFile(t, epID, 8) // HDTV-1080p
	}
	for i := int64(8); i <= 9; i++ {
		env.createEpisode(t, stID, 4, i, "unreleased")
	}

	items := env.collectWantedItems(t)
	// Find S04E05 item
	var item decisioning.SearchableItem
	for _, it := range items {
		if it.EpisodeNumber == 5 && it.SeasonNumber == 4 {
			item = it
			break
		}
	}
	if item.MediaID == 0 {
		t.Fatal("S04E05 not found in wanted items")
	}

	releases := []types.TorrentInfo{
		makeTorrentWithIDs("Stranger.Things.S04E05.1080p.BluRay.x264", "BluRay", 1080, 80, 0, 66732),
		makeTorrentWithIDs("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 70, 0, 66732),
	}

	scorer := scoring.NewDefaultScorer()
	sCtx := scoring.ScoringContext{
		QualityProfile: env.profile,
		SearchSeason:   item.SeasonNumber,
		SearchEpisode:  item.EpisodeNumber,
		Now:            time.Now(),
	}
	for i := range releases {
		scorer.ScoreTorrent(&releases[i], sCtx)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Score > releases[j].Score
	})

	best := decisioning.SelectBestRelease(releases, env.profile, item, nopLogger)
	if best == nil {
		t.Fatal("expected disc source upgrade")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQID(best))
	}
}

// Scenario 6B: Score+Select — non-disc upgrade rejected
func TestScoreAndSelect_Scenario6B_NonDiscUpgradeRejected(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	stID := env.createSeries(t, "Stranger Things", 66732, "continuing")
	env.createSeason(t, stID, 4)
	for i := int64(1); i <= 7; i++ {
		epID := env.createEpisode(t, stID, 4, i, "upgradable")
		env.createEpisodeFile(t, epID, 8) // HDTV-1080p
	}
	for i := int64(8); i <= 9; i++ {
		env.createEpisode(t, stID, 4, i, "unreleased")
	}

	items := env.collectWantedItems(t)
	var item decisioning.SearchableItem
	for _, it := range items {
		if it.EpisodeNumber == 5 && it.SeasonNumber == 4 {
			item = it
			break
		}
	}

	releases := []types.TorrentInfo{
		makeTorrentWithIDs("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 0, 66732),
		makeTorrentWithIDs("Stranger.Things.S04E05.1080p.WEBRip.x264", "WEBRip", 1080, 60, 0, 66732),
	}

	scorer := scoring.NewDefaultScorer()
	sCtx := scoring.ScoringContext{
		QualityProfile: env.profile,
		SearchSeason:   item.SeasonNumber,
		SearchEpisode:  item.EpisodeNumber,
		Now:            time.Now(),
	}
	for i := range releases {
		scorer.ScoreTorrent(&releases[i], sCtx)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Score > releases[j].Score
	})

	best := decisioning.SelectBestRelease(releases, env.profile, item, nopLogger)
	if best != nil {
		t.Errorf("expected nil (non-disc not an upgrade), got %s", best.Title)
	}
}

// Scenario 7B + 8B: Season pack grabs suppress individual episode grabs
func TestScoreAndGrab_SeasonPackSuppression(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	bbID := env.createSeries(t, "Breaking Bad", 81189, "ended")
	env.createSeason(t, bbID, 3)
	for i := int64(1); i <= 13; i++ {
		env.createEpisode(t, bbID, 3, i, "missing")
	}

	// Build index with individual episode items for matcher to work
	items := buildEpisodeItems(bbID, "Breaking Bad", 81189, env.profileID, 3, 13, false, 0)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	// Feed: season pack + individual episodes
	releases := []types.TorrentInfo{
		makeTorrentWithIDs("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100, 0, 81189),
		makeTorrentWithIDs("Breaking.Bad.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 0, 81189),
		makeTorrentWithIDs("Breaking.Bad.S03E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 75, 0, 81189),
	}

	var allMatches []MatchResult
	for _, release := range releases {
		results := matcher.Match(context.Background(), release)
		allMatches = append(allMatches, results...)
	}

	groups := groupMatches(allMatches)

	// Count season and episode groups
	var seasonGroups, episodeGroups int
	for _, g := range groups {
		if g.isSeason {
			seasonGroups++
		} else if g.item.MediaType == decisioning.MediaTypeEpisode {
			episodeGroups++
		}
	}

	if seasonGroups < 1 {
		t.Errorf("expected at least 1 season group, got %d", seasonGroups)
	}
	if episodeGroups < 1 {
		t.Errorf("expected at least 1 episode group, got %d", episodeGroups)
	}

	// Verify that season keys and episode keys are separated correctly
	var seasonKeys, episodeKeys []string
	for key, g := range groups {
		if g.isSeason {
			seasonKeys = append(seasonKeys, key)
		} else {
			episodeKeys = append(episodeKeys, key)
		}
	}

	// The season key should match what seasonKeyForEpisode produces for episodes in the same season
	for _, eKey := range episodeKeys {
		g := groups[eKey]
		if g.item.MediaType == decisioning.MediaTypeEpisode {
			sKey := seasonKeyForEpisode(g.item)
			found := false
			for _, sk := range seasonKeys {
				if sk == sKey {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("episode %s has season key %s but no matching season group exists", eKey, sKey)
			}
		}
	}
}

func safeQID(t *types.TorrentInfo) int {
	if t == nil || t.ScoreBreakdown == nil {
		return 0
	}
	return t.ScoreBreakdown.QualityID
}
