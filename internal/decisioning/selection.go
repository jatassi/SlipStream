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
func SelectBestRelease(releases []types.TorrentInfo, profile *quality.Profile, item SearchableItem, logger zerolog.Logger) *types.TorrentInfo {
	seasonPackCandidates := 0
	seasonPacksFound := 0

	for i := range releases {
		release := &releases[i]

		// For TV searches, verify the release matches the target season/episode
		if item.MediaType == MediaTypeEpisode || item.MediaType == MediaTypeSeason {
			parsed := scanner.ParseFilename(release.Title)

			// Check if the release covers the target season
			if item.SeasonNumber > 0 && parsed.Season > 0 {
				if parsed.IsCompleteSeries && parsed.EndSeason > 0 {
					if item.SeasonNumber < parsed.Season || item.SeasonNumber > parsed.EndSeason {
						continue
					}
				} else if parsed.Season != item.SeasonNumber {
					continue
				}
			}

			// For specific episode searches, require exact episode match (no season packs)
			if item.MediaType == MediaTypeEpisode && item.EpisodeNumber > 0 {
				if parsed.Episode != item.EpisodeNumber {
					continue
				}
			}

			// For season pack searches, require actual season pack
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
					seasonPackCandidates++
					continue
				}
				seasonPacksFound++
				logger.Info().
					Str("release", release.Title).
					Int("parsedSeason", parsed.Season).
					Int("parsedEndSeason", parsed.EndSeason).
					Bool("isSeasonPack", parsed.IsSeasonPack).
					Bool("isCompleteSeries", parsed.IsCompleteSeries).
					Msg("Found season pack release")
			}
		}

		// Determine release quality
		releaseQualityID := 0
		if release.ScoreBreakdown != nil {
			releaseQualityID = release.ScoreBreakdown.QualityID
		}

		// Skip if quality is not acceptable
		if releaseQualityID > 0 && !profile.IsAcceptable(releaseQualityID) {
			logger.Debug().
				Str("release", release.Title).
				Int("qualityId", releaseQualityID).
				Str("qualityName", release.ScoreBreakdown.QualityName).
				Msg("Rejected - quality not acceptable")
			continue
		}

		// If item already has a file, only grab upgrades
		if item.HasFile {
			if item.CurrentQualityID == 0 {
				logger.Debug().
					Str("release", release.Title).
					Msg("Skipping release - have file but current quality unknown")
				continue
			}
			if releaseQualityID == 0 {
				logger.Debug().
					Str("release", release.Title).
					Int("currentQualityId", item.CurrentQualityID).
					Msg("Skipping release with unknown quality - already have file")
				continue
			}
			if !profile.IsUpgrade(item.CurrentQualityID, releaseQualityID) {
				logger.Debug().
					Str("release", release.Title).
					Int("currentQualityId", item.CurrentQualityID).
					Int("releaseQualityId", releaseQualityID).
					Msg("Skipping release - not an upgrade")
				continue
			}
		}

		return release
	}

	// Log season pack search summary
	if item.MediaType == MediaTypeSeason {
		if seasonPacksFound == 0 && seasonPackCandidates > 0 {
			logger.Info().
				Int("totalReleases", len(releases)).
				Int("individualEpisodes", seasonPackCandidates).
				Int("seasonPacksFound", seasonPacksFound).
				Int("targetSeason", item.SeasonNumber).
				Msg("No season pack found - all matching releases were individual episodes")
		} else if seasonPacksFound == 0 {
			logger.Info().
				Int("totalReleases", len(releases)).
				Int("targetSeason", item.SeasonNumber).
				Msg("No releases matched the target season")
		}
	}

	return nil
}
