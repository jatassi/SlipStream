package decisioning

import (
	"sort"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/indexer/scoring"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
)

func hd1080pProfile() *quality.Profile {
	p := quality.HD1080pProfile()
	p.ID = 1
	return &p
}

func makeTorrent(title, source string, resolution, seeders int) types.TorrentInfo {
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

func scoreAndSort(releases []types.TorrentInfo, profile *quality.Profile, item *SearchableItem) {
	scorer := scoring.NewDefaultScorer()
	ctx := scoring.ScoringContext{
		QualityProfile: profile,
		Now:            time.Now(),
	}
	if item.MediaType == MediaTypeMovie {
		ctx.SearchYear = item.Year
	} else {
		ctx.SearchSeason = item.SeasonNumber
		ctx.SearchEpisode = item.EpisodeNumber
	}
	for i := range releases {
		scorer.ScoreTorrent(&releases[i], &ctx)
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Score > releases[j].Score
	})
}

var logger = func() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}()

// Scenario 1A: Missing movie — Bluray-1080p selected, 2160p rejected by IsAcceptable
func TestSelectBestRelease_Scenario1A_MissingMovie(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeMovie,
		MediaID:          1,
		Title:            "Dune Part Two",
		Year:             2024,
		TmdbID:           693134,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("Dune.Part.Two.2024.2160p.UHD.BluRay.x265", "BluRay", 2160, 150),
		makeTorrent("Dune.Part.Two.2024.1080p.BluRay.x264", "BluRay", 1080, 100),
		makeTorrent("Dune.Part.Two.2024.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Dune.Part.Two.2024.720p.WEB-DL.x264", "WEB-DL", 720, 60),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected a release to be selected, got nil")
	}
	// 2160p should be rejected (not acceptable in HD-1080p profile), Bluray-1080p should win
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d (%s)",
			safeQualityID(best), best.Title)
	}
}

// Scenario 2A: Upgradable movie — Bluray-1080p selected as upgrade from WEBDL-720p
func TestSelectBestRelease_Scenario2A_UpgradableMovie(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeMovie,
		MediaID:          2,
		Title:            "Inception",
		Year:             2010,
		TmdbID:           27205,
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 6, // WEBDL-720p
	}

	releases := []types.TorrentInfo{
		makeTorrent("Inception.2010.1080p.BluRay.x264", "BluRay", 1080, 120),
		makeTorrent("Inception.2010.720p.BluRay.x264", "BluRay", 720, 90),
		makeTorrent("Inception.2010.720p.WEB-DL.x264", "WEB-DL", 720, 60),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected a release to be selected, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 3A: Upgradable movie, no upgrade available — returns nil
func TestSelectBestRelease_Scenario3A_NoUpgradeAvailable(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeMovie,
		MediaID:          2,
		Title:            "Inception",
		Year:             2010,
		TmdbID:           27205,
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 6, // WEBDL-720p
	}

	// Only same-quality or lower releases — balanced strategy blocks same-res non-disc
	releases := []types.TorrentInfo{
		makeTorrent("Inception.2010.720p.WEB-DL.x264", "WEB-DL", 720, 80),
		makeTorrent("Inception.2010.720p.WEBRip.x264", "WEBRip", 720, 60),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best != nil {
		t.Errorf("expected nil (no upgrade), got %s (quality ID %d)", best.Title, safeQualityID(best))
	}
}

// Scenario 4A: Missing episode — exact episode match required, season packs and wrong episodes rejected
func TestSelectBestRelease_Scenario4A_MissingEpisode(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          100,
		Title:            "Breaking Bad",
		SeriesID:         10,
		SeasonNumber:     3,
		EpisodeNumber:    7,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),   // season pack
		makeTorrent("Breaking.Bad.S03E08.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // wrong episode
		makeTorrent("Breaking.Bad.S03E07.1080p.WEB-DL.x264", "WEB-DL", 1080, 70), // correct
		makeTorrent("Breaking.Bad.S03E07.720p.HDTV.x264", "HDTV", 720, 50),       // correct, lower quality
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected a release to be selected, got nil")
	}
	// Should pick S03E07 WEB-DL 1080p (season pack rejected, S03E08 rejected)
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 10 {
		t.Errorf("expected WEBDL-1080p (ID 10), got quality ID %d (%s)", safeQualityID(best), best.Title)
	}
}

// Scenario 5A: Upgradable episode — disc source upgrade (Bluray-1080p upgrades HDTV-1080p)
func TestSelectBestRelease_Scenario5A_DiscSourceUpgrade(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          200,
		Title:            "Stranger Things",
		SeriesID:         20,
		SeasonNumber:     4,
		EpisodeNumber:    5,
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 8, // HDTV-1080p
	}

	releases := []types.TorrentInfo{
		makeTorrent("Stranger.Things.S04E05.1080p.BluRay.x264", "BluRay", 1080, 80),
		makeTorrent("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 70),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected a release to be selected, got nil")
	}
	// Balanced strategy: disc source replaces non-disc at same resolution
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 6A: Upgradable episode — non-disc upgrade rejected (balanced strategy)
func TestSelectBestRelease_Scenario6A_NonDiscUpgradeRejected(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          200,
		Title:            "Stranger Things",
		SeriesID:         20,
		SeasonNumber:     4,
		EpisodeNumber:    5,
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 8, // HDTV-1080p
	}

	releases := []types.TorrentInfo{
		makeTorrent("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Stranger.Things.S04E05.1080p.WEBRip.x264", "WEBRip", 1080, 60),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best != nil {
		t.Errorf("expected nil (non-disc not an upgrade at same resolution), got %s", best.Title)
	}
}

// Scenario 7A: Season pack search — season pack selected, individual episodes rejected
func TestSelectBestRelease_Scenario7A_SeasonPackSelected(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeSeason,
		MediaID:          10,
		Title:            "Breaking Bad",
		SeriesID:         10,
		SeasonNumber:     3,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Breaking.Bad.S03E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 75),
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected a season pack release, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p season pack (ID 11), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 8A: Season pack upgrade — Bluray-1080p upgrades WEBDL-720p season
func TestSelectBestRelease_Scenario8A_SeasonPackUpgrade(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeSeason,
		MediaID:          10,
		Title:            "Breaking Bad",
		SeriesID:         10,
		SeasonNumber:     3,
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 6, // WEBDL-720p
	}

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),
		makeTorrent("Breaking.Bad.S03.720p.WEB-DL.x264", "WEB-DL", 720, 80),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected season pack upgrade, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 9A: Individual episode upgrade when season pack is ineligible
func TestSelectBestRelease_Scenario9A_IndividualEpisodeUpgrade(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          110,
		Title:            "Breaking Bad",
		SeriesID:         10,
		SeasonNumber:     3,
		EpisodeNumber:    1,
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 6, // WEBDL-720p
	}

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),   // season pack — rejected for episode search
		makeTorrent("Breaking.Bad.S03E01.1080p.BluRay.x264", "BluRay", 1080, 90), // correct
		makeTorrent("Breaking.Bad.S03E01.720p.WEB-DL.x264", "WEB-DL", 720, 60),   // not upgrade (balanced: same res non-disc)
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected individual episode upgrade, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 10A: Continuing season — episode search only (season pack rejected)
func TestSelectBestRelease_Scenario10A_ContinuingSeason(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          300,
		Title:            "The Mandalorian",
		SeriesID:         30,
		SeasonNumber:     3,
		EpisodeNumber:    3,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("The.Mandalorian.S03.1080p.WEB-DL.x264", "WEB-DL", 1080, 100),   // season pack
		makeTorrent("The.Mandalorian.S03E03.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // correct
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected individual episode, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 10 {
		t.Errorf("expected WEBDL-1080p (ID 10), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 11A: Season premiere — single episode search
func TestSelectBestRelease_Scenario11A_SeasonPremiere(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          400,
		Title:            "House of the Dragon",
		SeriesID:         40,
		SeasonNumber:     3,
		EpisodeNumber:    1,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("House.of.the.Dragon.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 90),
		makeTorrent("House.of.the.Dragon.S03E01.720p.HDTV.x264", "HDTV", 720, 60),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected episode release, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 10 {
		t.Errorf("expected WEBDL-1080p (ID 10), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 12A: Upgradable episode with unreleased siblings — disc upgrade selected
func TestSelectBestRelease_Scenario12A_UpgradableEpisodeWithUnreleased(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          500,
		Title:            "Stranger Things",
		SeriesID:         20,
		SeasonNumber:     4,
		EpisodeNumber:    5,
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 8, // HDTV-1080p
	}

	releases := []types.TorrentInfo{
		makeTorrent("Stranger.Things.S04E05.1080p.BluRay.x264", "BluRay", 1080, 80),
		makeTorrent("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 70),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected disc upgrade, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 11 {
		t.Errorf("expected Bluray-1080p (ID 11), got quality ID %d", safeQualityID(best))
	}
}

// Scenario 14A: Season pack blocked — individual episode search for missing episodes
func TestSelectBestRelease_Scenario14A_SeasonPackBlocked(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          600,
		Title:            "The Last of Us",
		SeriesID:         50,
		SeasonNumber:     2,
		EpisodeNumber:    2,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("The.Last.of.Us.S02.1080p.WEB-DL.x264", "WEB-DL", 1080, 100),   // season pack
		makeTorrent("The.Last.of.Us.S02E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // correct
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best == nil {
		t.Fatal("expected individual episode, got nil")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != 10 {
		t.Errorf("expected WEBDL-1080p (ID 10), got quality ID %d", safeQualityID(best))
	}
}

// Edge case: Season pack search with no season packs returns nil
func TestSelectBestRelease_SeasonSearch_NoSeasonPacks(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeSeason,
		MediaID:          10,
		Title:            "Breaking Bad",
		SeriesID:         10,
		SeasonNumber:     3,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Breaking.Bad.S03E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 75),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best != nil {
		t.Errorf("expected nil for season search with no season packs, got %s", best.Title)
	}
}

// Edge case: HasFile but CurrentQualityID == 0 returns nil
func TestSelectBestRelease_HasFileButUnknownQuality(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeMovie,
		MediaID:          99,
		Title:            "Test Movie",
		QualityProfileID: profile.ID,
		HasFile:          true,
		CurrentQualityID: 0,
	}

	releases := []types.TorrentInfo{
		makeTorrent("Test.Movie.2024.1080p.BluRay.x264", "BluRay", 1080, 100),
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best != nil {
		t.Errorf("expected nil when HasFile=true but CurrentQualityID=0, got %s", best.Title)
	}
}

// Edge case: Empty releases list returns nil
func TestSelectBestRelease_EmptyReleases(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeMovie,
		MediaID:          1,
		Title:            "Test Movie",
		QualityProfileID: profile.ID,
	}

	best := SelectBestRelease(nil, profile, &item, logger)
	if best != nil {
		t.Errorf("expected nil for empty releases, got %s", best.Title)
	}
}

// Edge case: Wrong season for episode search
func TestSelectBestRelease_WrongSeason(t *testing.T) {
	profile := hd1080pProfile()
	item := SearchableItem{
		MediaType:        MediaTypeEpisode,
		MediaID:          100,
		Title:            "Breaking Bad",
		SeriesID:         10,
		SeasonNumber:     3,
		EpisodeNumber:    7,
		QualityProfileID: profile.ID,
	}

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S02E07.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // wrong season
	}

	scoreAndSort(releases, profile, &item)
	best := SelectBestRelease(releases, profile, &item, logger)

	if best != nil {
		t.Errorf("expected nil for wrong season, got %s", best.Title)
	}
}

func safeQualityID(t *types.TorrentInfo) int {
	if t == nil || t.ScoreBreakdown == nil {
		return 0
	}
	return t.ScoreBreakdown.QualityID
}
