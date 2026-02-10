package rsssync

import (
	"context"
	"testing"

	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

// TestBuildWantedIndex verifies index construction from wanted items.
func TestBuildWantedIndex_BasicLookups(t *testing.T) {
	items := []decisioning.SearchableItem{
		{MediaType: decisioning.MediaTypeMovie, MediaID: 1, Title: "Dune Part Two", TmdbID: 693134},
		{MediaType: decisioning.MediaTypeEpisode, MediaID: 2, Title: "Breaking Bad", TvdbID: 81189, SeasonNumber: 3, EpisodeNumber: 7, SeriesID: 10},
	}

	idx := BuildWantedIndex(items)

	if _, ok := idx.byTmdbID[693134]; !ok {
		t.Error("expected Dune in byTmdbID index")
	}
	if _, ok := idx.byTvdbID[81189]; !ok {
		t.Error("expected Breaking Bad in byTvdbID index")
	}
}

// TestNoiseReleasesNoMatch verifies all 7 noise releases produce no match.
func TestNoiseReleasesNoMatch(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	// Create a single wanted movie so the index isn't empty
	env.createMovie(t, "Dune Part Two", 693134, 2024, "missing")
	items := env.collectWantedItems(t)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	noise := makeNoiseReleases()
	for _, release := range noise {
		results := matcher.Match(context.Background(), release)
		if len(results) > 0 {
			t.Errorf("noise release %q should not match, got %d results", release.Title, len(results))
		}
	}
}

// Scenario 1B: Missing movie — matched via tmdbID
func TestMatcher_Scenario1B_MissingMovie(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	env.createMovie(t, "Dune Part Two", 693134, 2024, "missing")
	items := env.collectWantedItems(t)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	releases := []types.TorrentInfo{
		makeTorrentWithIDs("Dune.Part.Two.2024.2160p.UHD.BluRay.x265", "BluRay", 2160, 150, 693134, 0),
		makeTorrentWithIDs("Dune.Part.Two.2024.1080p.BluRay.x264", "BluRay", 1080, 100, 693134, 0),
		makeTorrentWithIDs("Dune.Part.Two.2024.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 693134, 0),
		makeTorrentWithIDs("Dune.Part.Two.2024.720p.WEB-DL.x264", "WEB-DL", 720, 60, 693134, 0),
		// Different movie (Dune 2021) — different TMDB ID should not match
		makeTorrentWithIDs("Dune.2021.1080p.BluRay.x264", "BluRay", 1080, 90, 438631, 0),
	}

	var totalMatches int
	for _, release := range releases {
		results := matcher.Match(context.Background(), release)
		totalMatches += len(results)
	}

	if totalMatches != 4 {
		t.Errorf("expected 4 matches for Dune Part Two releases, got %d", totalMatches)
	}
}

// Scenario 4B: Missing episode — exact episode match required
func TestMatcher_Scenario4B_MissingEpisode(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	// Breaking Bad S3: mixed statuses prevent season pack
	bbID := env.createSeries(t, "Breaking Bad", 81189, "ended")
	env.createSeason(t, bbID, 3)
	for i := int64(1); i <= 6; i++ {
		epID := env.createEpisode(t, bbID, 3, i, "available")
		env.createEpisodeFile(t, epID, 10) // WEBDL-1080p
	}
	env.createEpisode(t, bbID, 3, 7, "missing") // wanted
	for i := int64(8); i <= 13; i++ {
		epID := env.createEpisode(t, bbID, 3, i, "available")
		env.createEpisodeFile(t, epID, 10)
	}

	items := env.collectWantedItems(t)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	// S03E07 should match
	r1 := makeTorrentWithIDs("Breaking.Bad.S03E07.1080p.WEB-DL.x264", "WEB-DL", 1080, 70, 0, 81189)
	results := matcher.Match(context.Background(), r1)
	if len(results) != 1 {
		t.Errorf("S03E07 should match, got %d results", len(results))
	}

	// S03E08 should not match (not wanted)
	r2 := makeTorrentWithIDs("Breaking.Bad.S03E08.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 0, 81189)
	results = matcher.Match(context.Background(), r2)
	if len(results) != 0 {
		t.Errorf("S03E08 should not match, got %d results", len(results))
	}

	// Season pack should not match (not all episodes missing)
	r3 := makeTorrentWithIDs("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100, 0, 81189)
	results = matcher.Match(context.Background(), r3)
	if len(results) != 0 {
		t.Errorf("S03 season pack should not match (mixed statuses), got %d results", len(results))
	}
}

// Scenario 7B: All episodes missing — season pack eligible
// The WantedIndex must contain individual episode items for matchSeasonPack to find candidates.
func TestMatcher_Scenario7B_AllMissing_SeasonPack(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	bbID := env.createSeries(t, "Breaking Bad", 81189, "ended")
	env.createSeason(t, bbID, 3)
	for i := int64(1); i <= 13; i++ {
		env.createEpisode(t, bbID, 3, i, "missing")
	}

	// Build index with individual episode items (simulates what the index needs for matching)
	items := buildEpisodeItems(bbID, "Breaking Bad", 81189, env.profileID, 3, 13, false, 0)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	// Season pack should match via eligibility check
	r1 := makeTorrentWithIDs("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100, 0, 81189)
	results := matcher.Match(context.Background(), r1)
	if len(results) != 1 {
		t.Fatalf("season pack should match, got %d results", len(results))
	}
	if !results[0].IsSeason {
		t.Error("result should be marked as season")
	}
	if results[0].WantedItem.MediaType != decisioning.MediaTypeSeason {
		t.Errorf("expected MediaTypeSeason, got %s", results[0].WantedItem.MediaType)
	}

	// Individual episodes should also match
	r2 := makeTorrentWithIDs("Breaking.Bad.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 0, 81189)
	results = matcher.Match(context.Background(), r2)
	if len(results) != 1 {
		t.Errorf("S03E01 should match, got %d results", len(results))
	}
}

// Scenario 8B: All upgradable — season pack upgrade eligible
func TestMatcher_Scenario8B_AllUpgradable_SeasonPackUpgrade(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	bbID := env.createSeries(t, "Breaking Bad", 81189, "ended")
	env.createSeason(t, bbID, 3)
	for i := int64(1); i <= 13; i++ {
		epID := env.createEpisode(t, bbID, 3, i, "upgradable")
		env.createEpisodeFile(t, epID, 6) // WEBDL-720p
	}

	// Build index with individual upgrade episode items
	items := buildEpisodeItems(bbID, "Breaking Bad", 81189, env.profileID, 3, 13, true, 6)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	r1 := makeTorrentWithIDs("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100, 0, 81189)
	results := matcher.Match(context.Background(), r1)
	if len(results) != 1 {
		t.Fatalf("season pack upgrade should match, got %d results", len(results))
	}
	if !results[0].IsSeason {
		t.Error("result should be marked as season")
	}
	if !results[0].WantedItem.HasFile {
		t.Error("upgrade season item should have HasFile=true")
	}
	if results[0].WantedItem.CurrentQualityID != 6 {
		t.Errorf("expected CurrentQualityID=6, got %d", results[0].WantedItem.CurrentQualityID)
	}
}

// Scenario 10B: Continuing season — season pack rejected, individuals match
func TestMatcher_Scenario10B_ContinuingSeason(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	mandoID := env.createSeries(t, "The Mandalorian", 82856, "continuing")
	env.createSeason(t, mandoID, 3)
	for i := int64(1); i <= 5; i++ {
		env.createEpisode(t, mandoID, 3, i, "missing")
	}
	for i := int64(6); i <= 8; i++ {
		env.createEpisode(t, mandoID, 3, i, "unreleased")
	}

	items := env.collectWantedItems(t)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	// Season pack should be rejected (unreleased episodes present)
	r1 := makeTorrentWithIDs("The.Mandalorian.S03.1080p.WEB-DL.x264", "WEB-DL", 1080, 100, 0, 82856)
	results := matcher.Match(context.Background(), r1)
	if len(results) != 0 {
		t.Errorf("season pack should not match with unreleased episodes, got %d results", len(results))
	}

	// Individual episodes E03 and E04 should match
	r2 := makeTorrentWithIDs("The.Mandalorian.S03E03.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 0, 82856)
	results = matcher.Match(context.Background(), r2)
	if len(results) != 1 {
		t.Errorf("S03E03 should match, got %d results", len(results))
	}

	r3 := makeTorrentWithIDs("The.Mandalorian.S03E04.1080p.WEB-DL.x264", "WEB-DL", 1080, 75, 0, 82856)
	results = matcher.Match(context.Background(), r3)
	if len(results) != 1 {
		t.Errorf("S03E04 should match, got %d results", len(results))
	}

	// E07 should not match (unreleased)
	r4 := makeTorrentWithIDs("The.Mandalorian.S03E07.1080p.WEB-DL.x264", "WEB-DL", 1080, 70, 0, 82856)
	results = matcher.Match(context.Background(), r4)
	if len(results) != 0 {
		t.Errorf("S03E07 (unreleased) should not match, got %d results", len(results))
	}
}

// Scenario 11B: Season premiere — only E01 matches
func TestMatcher_Scenario11B_SeasonPremiere(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	hotdID := env.createSeries(t, "House of the Dragon", 94997, "continuing")
	env.createSeason(t, hotdID, 3)
	env.createEpisode(t, hotdID, 3, 1, "missing")
	for i := int64(2); i <= 10; i++ {
		env.createEpisode(t, hotdID, 3, i, "unreleased")
	}

	items := env.collectWantedItems(t)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	r1 := makeTorrentWithIDs("House.of.the.Dragon.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 90, 0, 94997)
	results := matcher.Match(context.Background(), r1)
	if len(results) != 1 {
		t.Errorf("S03E01 should match, got %d results", len(results))
	}

	r2 := makeTorrentWithIDs("House.of.the.Dragon.S03E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 85, 0, 94997)
	results = matcher.Match(context.Background(), r2)
	if len(results) != 0 {
		t.Errorf("S03E02 (unreleased) should not match, got %d results", len(results))
	}
}

// Scenario 14B: Season pack blocked — E01 available, E02-E07 missing
func TestMatcher_Scenario14B_SeasonPackBlocked(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	tlou := env.createSeries(t, "The Last of Us", 100088, "continuing")
	env.createSeason(t, tlou, 2)
	epID := env.createEpisode(t, tlou, 2, 1, "available")
	env.createEpisodeFile(t, epID, 10) // WEBDL-1080p
	for i := int64(2); i <= 7; i++ {
		env.createEpisode(t, tlou, 2, i, "missing")
	}

	items := env.collectWantedItems(t)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	// Season pack rejected
	r1 := makeTorrentWithIDs("The.Last.of.Us.S02.1080p.WEB-DL.x264", "WEB-DL", 1080, 100, 0, 100088)
	results := matcher.Match(context.Background(), r1)
	if len(results) != 0 {
		t.Errorf("season pack should not match (E01 available), got %d results", len(results))
	}

	// S02E02 should match
	r2 := makeTorrentWithIDs("The.Last.of.Us.S02E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 80, 0, 100088)
	results = matcher.Match(context.Background(), r2)
	if len(results) != 1 {
		t.Errorf("S02E02 should match, got %d results", len(results))
	}
}

// Scenario 15B: Title-based matching (no external IDs)
func TestMatcher_Scenario15B_TitleBasedMatching(t *testing.T) {
	env := setupTestEnv(t)
	defer env.close()

	env.createMovie(t, "Dune Part Two", 693134, 2024, "missing")
	items := env.collectWantedItems(t)
	idx := BuildWantedIndex(items)
	matcher := NewMatcher(idx, env.queries, env.tdb.Logger)

	// Title match (no IDs) — should match
	r1 := makeTorrentRelease("Dune.Part.Two.2024.1080p.BluRay.x264", "BluRay", 1080, 100)
	results := matcher.Match(context.Background(), r1)
	if len(results) != 1 {
		t.Errorf("title-based match should work, got %d results", len(results))
	}

	// Different movie with overlapping name but different year — should not match via ID (no ID)
	// Title normalization: "dune" != "dune part two" → no match
	r2 := makeTorrentRelease("Dune.1984.720p.BluRay.x264", "BluRay", 720, 50)
	results = matcher.Match(context.Background(), r2)
	if len(results) != 0 {
		t.Errorf("Dune.1984 should not match 'Dune Part Two', got %d results", len(results))
	}
}

// Test groupMatches function
func TestGroupMatches(t *testing.T) {
	matches := []MatchResult{
		{
			Release:    makeTorrentRelease("Release.1", "BluRay", 1080, 100),
			WantedItem: decisioning.SearchableItem{MediaType: decisioning.MediaTypeMovie, MediaID: 1},
		},
		{
			Release:    makeTorrentRelease("Release.2", "WEB-DL", 1080, 80),
			WantedItem: decisioning.SearchableItem{MediaType: decisioning.MediaTypeMovie, MediaID: 1},
		},
		{
			Release:    makeTorrentRelease("Release.3", "BluRay", 1080, 90),
			WantedItem: decisioning.SearchableItem{MediaType: decisioning.MediaTypeEpisode, MediaID: 2, SeriesID: 10, SeasonNumber: 3},
		},
		{
			Release:    makeTorrentRelease("Release.4", "BluRay", 1080, 95),
			WantedItem: decisioning.SearchableItem{MediaType: decisioning.MediaTypeSeason, MediaID: 10, SeriesID: 10, SeasonNumber: 3},
			IsSeason:   true,
		},
	}

	groups := groupMatches(matches)

	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}

	movieKey := itemKey(matches[0].WantedItem)
	if g, ok := groups[movieKey]; !ok {
		t.Error("missing movie group")
	} else if len(g.releases) != 2 {
		t.Errorf("movie group should have 2 releases, got %d", len(g.releases))
	}

	seasonKey := itemKey(matches[3].WantedItem)
	if g, ok := groups[seasonKey]; !ok {
		t.Error("missing season group")
	} else if !g.isSeason {
		t.Error("season group should have isSeason=true")
	}
}

// Test seasonKeyForEpisode
func TestSeasonKeyForEpisode(t *testing.T) {
	item := decisioning.SearchableItem{
		MediaType:    decisioning.MediaTypeEpisode,
		MediaID:      100,
		SeriesID:     10,
		SeasonNumber: 3,
	}

	key := seasonKeyForEpisode(item)
	expected := "season:10:3"
	if key != expected {
		t.Errorf("expected %q, got %q", expected, key)
	}
}
