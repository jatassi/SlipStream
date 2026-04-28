package decisioning

import (
	"context"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/indexer/scoring"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

// Test-local TV parsing patterns for building ReleaseForFilter in unit tests.
var (
	testTVSE         = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[.\s_-]*(.*)$`)
	testTVSeasonPack = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})(?:[.\s_-]|$)(.*)$`)
)

// testReleaseParser is a self-contained parser for unit tests that doesn't
// depend on the module registry. It recognizes basic SxxExx and Sxx patterns.
func testReleaseParser(rawTitle string, size int64, categories []int) *module.ReleaseForFilter {
	name := strings.TrimSuffix(rawTitle, filepath.Ext(rawTitle))
	rff := &module.ReleaseForFilter{Size: size, Categories: categories}

	if match := testTVSE.FindStringSubmatch(name); match != nil {
		rff.Title = parseutil.CleanTitle(match[1])
		rff.Season, _ = strconv.Atoi(match[2])
		rff.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			rff.EndEpisode, _ = strconv.Atoi(match[4])
		}
		rff.IsTV = true
		return rff
	}
	if match := testTVSeasonPack.FindStringSubmatch(name); match != nil {
		rff.Title = parseutil.CleanTitle(match[1])
		rff.Season, _ = strconv.Atoi(match[2])
		rff.IsTV = true
		rff.IsSeasonPack = true
		return rff
	}

	rff.Title = parseutil.CleanTitle(name)
	return rff
}

// testTVStrategy is a test-local SearchStrategy that replicates TV module filtering
// for use in unit tests (avoids importing modules/tv which would create a circular dep).
type testTVStrategy struct{ DefaultStrategy }

func (testTVStrategy) FilterRelease(_ context.Context, release *module.ReleaseForFilter, item module.SearchableItem) (reject bool, reason string) {
	seasonNumber, _ := item.GetSearchParams().Extra["seasonNumber"].(int)
	episodeNumber, _ := item.GetSearchParams().Extra["episodeNumber"].(int)

	// Season match
	if seasonNumber > 0 && release.Season > 0 {
		if release.IsCompleteSeries && release.EndSeason > 0 {
			if seasonNumber < release.Season || seasonNumber > release.EndSeason {
				return true, "wrong season range"
			}
		} else if release.Season != seasonNumber {
			return true, "wrong season"
		}
	}

	switch item.GetMediaType() {
	case string(MediaTypeEpisode):
		if episodeNumber > 0 {
			if release.IsSeasonPack || release.IsCompleteSeries {
				return true, "season pack during episode search"
			}
			if release.Episode > 0 && release.Episode != episodeNumber {
				return true, "wrong episode"
			}
		}
	case string(MediaTypeSeason):
		if !release.IsSeasonPack && !release.IsCompleteSeries {
			return true, "not a season pack"
		}
	}

	return false, ""
}

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

func scoreAndSort(releases []types.TorrentInfo, profile *quality.Profile, item module.SearchableItem) {
	scorer := scoring.NewDefaultScorer()
	ctx := scoring.ScoringContext{
		QualityProfile: profile,
		Now:            time.Now(),
	}
	if item.GetMediaType() == string(MediaTypeMovie) {
		ctx.SearchYear = module.ItemYear(item)
	} else {
		ctx.SearchSeason = module.ItemSeasonNumber(item)
		ctx.SearchEpisode = module.ItemEpisodeNumber(item)
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

var (
	movieStrategy = DefaultStrategy{}
	tvStrategy    = testTVStrategy{}
)

func testMovieItem(mediaID int64, title string, year, tmdbID int, profileID int64, currentQualityID *int64) module.SearchableItem {
	extIDs := map[string]string{}
	if tmdbID != 0 {
		extIDs["tmdbId"] = strconv.Itoa(tmdbID)
	}
	extra := map[string]any{"year": year}
	return module.NewWantedItem(module.TypeMovie, string(MediaTypeMovie), mediaID, title, extIDs, profileID, currentQualityID, module.SearchParams{Extra: extra})
}

func testEpisodeItem(mediaID, seriesID int64, title string, season, episode int, profileID int64, currentQualityID *int64) module.SearchableItem {
	extra := map[string]any{
		"seriesId":      seriesID,
		"seasonNumber":  season,
		"episodeNumber": episode,
	}
	return module.NewWantedItem(module.TypeTV, string(MediaTypeEpisode), mediaID, title, nil, profileID, currentQualityID, module.SearchParams{Extra: extra})
}

func testSeasonItem(seriesID int64, title string, season int, profileID int64, currentQualityID *int64) module.SearchableItem {
	extra := map[string]any{
		"seriesId":     seriesID,
		"seasonNumber": season,
	}
	return module.NewWantedItem(module.TypeTV, string(MediaTypeSeason), seriesID, title, nil, profileID, currentQualityID, module.SearchParams{Extra: extra})
}

func int64Ptr(v int64) *int64 { return &v }

// Scenario 1A: Missing movie — Bluray-1080p selected, 2160p rejected by IsAcceptable
func TestSelectBestRelease_Scenario1A_MissingMovie(t *testing.T) {
	profile := hd1080pProfile()
	item := testMovieItem(1, "Dune Part Two", 2024, 693134, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("Dune.Part.Two.2024.2160p.UHD.BluRay.x265", "BluRay", 2160, 150),
		makeTorrent("Dune.Part.Two.2024.1080p.BluRay.x264", "BluRay", 1080, 100),
		makeTorrent("Dune.Part.Two.2024.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Dune.Part.Two.2024.720p.WEB-DL.x264", "WEB-DL", 720, 60),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, movieStrategy, testReleaseParser, logger)

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
	item := testMovieItem(2, "Inception", 2010, 27205, profile.ID, int64Ptr(6))

	releases := []types.TorrentInfo{
		makeTorrent("Inception.2010.1080p.BluRay.x264", "BluRay", 1080, 120),
		makeTorrent("Inception.2010.720p.BluRay.x264", "BluRay", 720, 90),
		makeTorrent("Inception.2010.720p.WEB-DL.x264", "WEB-DL", 720, 60),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, movieStrategy, testReleaseParser, logger)

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
	item := testMovieItem(2, "Inception", 2010, 27205, profile.ID, int64Ptr(6))

	// Only same-quality or lower releases — balanced strategy blocks same-res non-disc
	releases := []types.TorrentInfo{
		makeTorrent("Inception.2010.720p.WEB-DL.x264", "WEB-DL", 720, 80),
		makeTorrent("Inception.2010.720p.WEBRip.x264", "WEBRip", 720, 60),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, movieStrategy, testReleaseParser, logger)

	if best != nil {
		t.Errorf("expected nil (no upgrade), got %s (quality ID %d)", best.Title, safeQualityID(best))
	}
}

// Scenario 4A: Missing episode — exact episode match required, season packs and wrong episodes rejected
func TestSelectBestRelease_Scenario4A_MissingEpisode(t *testing.T) {
	profile := hd1080pProfile()
	item := testEpisodeItem(100, 10, "Breaking Bad", 3, 7, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),   // season pack
		makeTorrent("Breaking.Bad.S03E08.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // wrong episode
		makeTorrent("Breaking.Bad.S03E07.1080p.WEB-DL.x264", "WEB-DL", 1080, 70), // correct
		makeTorrent("Breaking.Bad.S03E07.720p.HDTV.x264", "HDTV", 720, 50),       // correct, lower quality
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testEpisodeItem(200, 20, "Stranger Things", 4, 5, profile.ID, int64Ptr(8))

	releases := []types.TorrentInfo{
		makeTorrent("Stranger.Things.S04E05.1080p.BluRay.x264", "BluRay", 1080, 80),
		makeTorrent("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 70),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testEpisodeItem(200, 20, "Stranger Things", 4, 5, profile.ID, int64Ptr(8))

	releases := []types.TorrentInfo{
		makeTorrent("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Stranger.Things.S04E05.1080p.WEBRip.x264", "WEBRip", 1080, 60),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

	if best != nil {
		t.Errorf("expected nil (non-disc not an upgrade at same resolution), got %s", best.Title)
	}
}

// Scenario 7A: Season pack search — season pack selected, individual episodes rejected
func TestSelectBestRelease_Scenario7A_SeasonPackSelected(t *testing.T) {
	profile := hd1080pProfile()
	item := testSeasonItem(10, "Breaking Bad", 3, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Breaking.Bad.S03E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 75),
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testSeasonItem(10, "Breaking Bad", 3, profile.ID, int64Ptr(6))

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),
		makeTorrent("Breaking.Bad.S03.720p.WEB-DL.x264", "WEB-DL", 720, 80),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testEpisodeItem(110, 10, "Breaking Bad", 3, 1, profile.ID, int64Ptr(6))

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03.1080p.BluRay.x264", "BluRay", 1080, 100),   // season pack — rejected for episode search
		makeTorrent("Breaking.Bad.S03E01.1080p.BluRay.x264", "BluRay", 1080, 90), // correct
		makeTorrent("Breaking.Bad.S03E01.720p.WEB-DL.x264", "WEB-DL", 720, 60),   // not upgrade (balanced: same res non-disc)
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testEpisodeItem(300, 30, "The Mandalorian", 3, 3, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("The.Mandalorian.S03.1080p.WEB-DL.x264", "WEB-DL", 1080, 100),   // season pack
		makeTorrent("The.Mandalorian.S03E03.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // correct
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testEpisodeItem(400, 40, "House of the Dragon", 3, 1, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("House.of.the.Dragon.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 90),
		makeTorrent("House.of.the.Dragon.S03E01.720p.HDTV.x264", "HDTV", 720, 60),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testEpisodeItem(500, 20, "Stranger Things", 4, 5, profile.ID, int64Ptr(8))

	releases := []types.TorrentInfo{
		makeTorrent("Stranger.Things.S04E05.1080p.BluRay.x264", "BluRay", 1080, 80),
		makeTorrent("Stranger.Things.S04E05.1080p.WEB-DL.x264", "WEB-DL", 1080, 70),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testEpisodeItem(600, 50, "The Last of Us", 2, 2, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("The.Last.of.Us.S02.1080p.WEB-DL.x264", "WEB-DL", 1080, 100),   // season pack
		makeTorrent("The.Last.of.Us.S02E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // correct
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
	item := testSeasonItem(10, "Breaking Bad", 3, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S03E01.1080p.WEB-DL.x264", "WEB-DL", 1080, 80),
		makeTorrent("Breaking.Bad.S03E02.1080p.WEB-DL.x264", "WEB-DL", 1080, 75),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

	if best != nil {
		t.Errorf("expected nil for season search with no season packs, got %s", best.Title)
	}
}

// Edge case: HasFile but CurrentQualityID == 0 returns nil
func TestSelectBestRelease_HasFileButUnknownQuality(t *testing.T) {
	profile := hd1080pProfile()
	// currentQualityID=0 but non-nil means "has file but quality unknown"
	// We represent this by using a zero-valued int64 pointer.
	// However, module.ItemHasFile checks if GetCurrentQualityID() != nil.
	// For this edge case, we need HasFile=true but CurrentQualityID=0.
	// We use int64Ptr(0) to indicate has file with unknown quality.
	item := testMovieItem(99, "Test Movie", 0, 0, profile.ID, int64Ptr(0))

	releases := []types.TorrentInfo{
		makeTorrent("Test.Movie.2024.1080p.BluRay.x264", "BluRay", 1080, 100),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, movieStrategy, testReleaseParser, logger)

	if best != nil {
		t.Errorf("expected nil when HasFile=true but CurrentQualityID=0, got %s", best.Title)
	}
}

// Edge case: Empty releases list returns nil
func TestSelectBestRelease_EmptyReleases(t *testing.T) {
	profile := hd1080pProfile()
	item := testMovieItem(1, "Test Movie", 0, 0, profile.ID, nil)

	best := SelectBestRelease(nil, profile, item, movieStrategy, testReleaseParser, logger)
	if best != nil {
		t.Errorf("expected nil for empty releases, got %s", best.Title)
	}
}

// Regression: a TELESYNC release tagged "1080p" must not win selection in an
// HD-1080p profile that does not allow CAM. Previously it parsed to
// Remux-1080p (highest 1080p) and was grabbed for in-theater movies.
func TestSelectBestRelease_TelesyncRejected(t *testing.T) {
	profile := hd1080pProfile()
	item := testMovieItem(1, "Michael", 2026, 936075, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("Michael.2026.1080p.TELESYNC.V2.x264-SyncUP", "CAM", 1080, 200),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, movieStrategy, testReleaseParser, logger)

	if best != nil {
		t.Errorf("expected TELESYNC release to be rejected, got %s (quality ID %d)", best.Title, safeQualityID(best))
	}
}

// When CAM is explicitly allowed in a profile, the release should be selected
// as the CAM tier — not silently upgraded to Remux-1080p.
func TestSelectBestRelease_CAMAllowedExplicitly(t *testing.T) {
	profile := hd1080pProfile()
	for i := range profile.Items {
		if profile.Items[i].Quality.ID == quality.CAMQualityID {
			profile.Items[i].Allowed = true
		}
	}
	item := testMovieItem(1, "Michael", 2026, 936075, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("Michael.2026.1080p.TELESYNC.V2.x264-SyncUP", "CAM", 1080, 200),
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, movieStrategy, testReleaseParser, logger)

	if best == nil {
		t.Fatal("expected release to be selected when CAM is allowed")
	}
	if best.ScoreBreakdown == nil || best.ScoreBreakdown.QualityID != quality.CAMQualityID {
		t.Errorf("expected CAM quality (id %d), got id=%d", quality.CAMQualityID, safeQualityID(best))
	}
}

// Edge case: Wrong season for episode search
func TestSelectBestRelease_WrongSeason(t *testing.T) {
	profile := hd1080pProfile()
	item := testEpisodeItem(100, 10, "Breaking Bad", 3, 7, profile.ID, nil)

	releases := []types.TorrentInfo{
		makeTorrent("Breaking.Bad.S02E07.1080p.WEB-DL.x264", "WEB-DL", 1080, 80), // wrong season
	}

	scoreAndSort(releases, profile, item)
	best := SelectBestRelease(releases, profile, item, tvStrategy, testReleaseParser, logger)

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
