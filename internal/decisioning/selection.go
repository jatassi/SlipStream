package decisioning

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/module"
)

// ReleaseParser converts a raw release title into a ReleaseForFilter.
// Callers provide an implementation backed by the module registry.
type ReleaseParser func(title string, size int64, categories []int) *module.ReleaseForFilter

// SelectBestRelease picks the best acceptable release from a scored, sorted list.
// Releases MUST be pre-sorted by score (highest first).
// The strategy parameter provides module-specific release filtering.
// The parser converts raw release titles into structured ReleaseForFilter values.
// Returns nil if no acceptable release is found.
func SelectBestRelease(releases []types.TorrentInfo, profile *quality.Profile, item module.SearchableItem, strategy module.SearchStrategy, parser ReleaseParser, logger *zerolog.Logger) *types.TorrentInfo {
	hasFile := module.ItemHasFile(item)
	currentQualityID := module.ItemCurrentQualityID(item)

	for i := range releases {
		release := &releases[i]

		rff := parser(release.Title, release.Size, release.Categories)
		if reject, reason := strategy.FilterRelease(context.Background(), rff, item); reject {
			logger.Debug().
				Str("release", release.Title).
				Str("reason", reason).
				Msg("Module filter rejected release")
			continue
		}

		releaseQualityID := extractReleaseQualityID(release)

		if shouldSkipForQuality(release, releaseQualityID, hasFile, currentQualityID, profile, logger) {
			continue
		}

		return release
	}

	return nil
}

func extractReleaseQualityID(release *types.TorrentInfo) int {
	if release.ScoreBreakdown == nil {
		return 0
	}
	return release.ScoreBreakdown.QualityID
}

func shouldSkipForQuality(release *types.TorrentInfo, releaseQualityID int, hasFile bool, currentQualityID int, profile *quality.Profile, logger *zerolog.Logger) bool {
	if releaseQualityID > 0 && !profile.IsAcceptable(releaseQualityID) {
		logger.Debug().
			Str("release", release.Title).
			Int("qualityId", releaseQualityID).
			Str("qualityName", release.ScoreBreakdown.QualityName).
			Msg("Rejected - quality not acceptable")
		return true
	}

	if !hasFile {
		return false
	}

	if currentQualityID == 0 {
		logger.Debug().
			Str("release", release.Title).
			Msg("Skipping release - have file but current quality unknown")
		return true
	}

	if releaseQualityID == 0 {
		logger.Debug().
			Str("release", release.Title).
			Int("currentQualityId", currentQualityID).
			Msg("Skipping release with unknown quality - already have file")
		return true
	}

	if !profile.IsUpgrade(currentQualityID, releaseQualityID) {
		logger.Debug().
			Str("release", release.Title).
			Int("currentQualityId", currentQualityID).
			Int("releaseQualityId", releaseQualityID).
			Msg("Skipping release - not an upgrade")
		return true
	}

	return false
}
