package decisioning

import (
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// SelectBestRelease picks the best acceptable release from a scored, sorted list.
// Releases MUST be pre-sorted by score (highest first).
// Returns nil if no acceptable release is found.
func SelectBestRelease(releases []types.TorrentInfo, profile *quality.Profile, item *SearchableItem, logger *zerolog.Logger) *types.TorrentInfo {
	seasonPackCandidates := 0
	seasonPacksFound := 0

	for i := range releases {
		release := &releases[i]

		if item.MediaType == MediaTypeEpisode || item.MediaType == MediaTypeSeason {
			skip, candidates, found := shouldSkipTVRelease(release, item, logger, seasonPackCandidates)
			seasonPackCandidates += candidates
			seasonPacksFound += found
			if skip {
				continue
			}
		}

		releaseQualityID := extractReleaseQualityID(release)

		if shouldSkipForQuality(release, releaseQualityID, item, profile, logger) {
			continue
		}

		return release
	}

	logSeasonPackSummary(item, seasonPacksFound, seasonPackCandidates, len(releases), logger)
	return nil
}

func shouldSkipTVRelease(release *types.TorrentInfo, item *SearchableItem, logger *zerolog.Logger, seasonPackCandidates int) (skip bool, candidates, found int) {
	parsed := scanner.ParseFilename(release.Title)

	if !matchesTargetSeason(parsed, item) {
		return true, 0, 0
	}

	if item.MediaType == MediaTypeEpisode && item.EpisodeNumber > 0 {
		if parsed.Episode != item.EpisodeNumber {
			return true, 0, 0
		}
	}

	if item.MediaType == MediaTypeSeason {
		if !parsed.IsSeasonPack {
			if seasonPackCandidates < 5 {
				logger.Info().
					Str("release", release.Title).
					Int("parsedSeason", parsed.Season).
					Int("parsedEpisode", parsed.Episode).
					Bool("isSeasonPack", parsed.IsSeasonPack).
					Msg("Rejected release for season pack search")
			}
			return true, 1, 0
		}
		logger.Info().
			Str("release", release.Title).
			Int("parsedSeason", parsed.Season).
			Int("parsedEndSeason", parsed.EndSeason).
			Bool("isSeasonPack", parsed.IsSeasonPack).
			Bool("isCompleteSeries", parsed.IsCompleteSeries).
			Msg("Found season pack release")
		return false, 0, 1
	}

	return false, 0, 0
}

func matchesTargetSeason(parsed *scanner.ParsedMedia, item *SearchableItem) bool {
	if item.SeasonNumber == 0 || parsed.Season == 0 {
		return true
	}

	if parsed.IsCompleteSeries && parsed.EndSeason > 0 {
		return item.SeasonNumber >= parsed.Season && item.SeasonNumber <= parsed.EndSeason
	}

	return parsed.Season == item.SeasonNumber
}

func extractReleaseQualityID(release *types.TorrentInfo) int {
	if release.ScoreBreakdown == nil {
		return 0
	}
	return release.ScoreBreakdown.QualityID
}

func shouldSkipForQuality(release *types.TorrentInfo, releaseQualityID int, item *SearchableItem, profile *quality.Profile, logger *zerolog.Logger) bool {
	if releaseQualityID > 0 && !profile.IsAcceptable(releaseQualityID) {
		logger.Debug().
			Str("release", release.Title).
			Int("qualityId", releaseQualityID).
			Str("qualityName", release.ScoreBreakdown.QualityName).
			Msg("Rejected - quality not acceptable")
		return true
	}

	if !item.HasFile {
		return false
	}

	if item.CurrentQualityID == 0 {
		logger.Debug().
			Str("release", release.Title).
			Msg("Skipping release - have file but current quality unknown")
		return true
	}

	if releaseQualityID == 0 {
		logger.Debug().
			Str("release", release.Title).
			Int("currentQualityId", item.CurrentQualityID).
			Msg("Skipping release with unknown quality - already have file")
		return true
	}

	if !profile.IsUpgrade(item.CurrentQualityID, releaseQualityID) {
		logger.Debug().
			Str("release", release.Title).
			Int("currentQualityId", item.CurrentQualityID).
			Int("releaseQualityId", releaseQualityID).
			Msg("Skipping release - not an upgrade")
		return true
	}

	return false
}

func logSeasonPackSummary(item *SearchableItem, seasonPacksFound, seasonPackCandidates, totalReleases int, logger *zerolog.Logger) {
	if item.MediaType != MediaTypeSeason {
		return
	}

	if seasonPacksFound > 0 {
		return
	}

	if seasonPackCandidates > 0 {
		logger.Info().
			Int("totalReleases", totalReleases).
			Int("individualEpisodes", seasonPackCandidates).
			Int("seasonPacksFound", seasonPacksFound).
			Int("targetSeason", item.SeasonNumber).
			Msg("No season pack found - all matching releases were individual episodes")
		return
	}

	logger.Info().
		Int("totalReleases", totalReleases).
		Int("targetSeason", item.SeasonNumber).
		Msg("No releases matched the target season")
}
