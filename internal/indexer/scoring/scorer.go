package scoring

import (
	"math"
	"sort"

	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

const (
	// MaxQualityWeight is the maximum quality weight from predefined qualities.
	MaxQualityWeight = 17 // Remux-2160p
)

// Scorer calculates desirability scores for releases.
type Scorer struct {
	config ScoringConfig
}

// NewScorer creates a new scorer with the given config.
func NewScorer(config ScoringConfig) *Scorer {
	return &Scorer{config: config}
}

// NewDefaultScorer creates a scorer with default configuration.
func NewDefaultScorer() *Scorer {
	return NewScorer(DefaultConfig())
}

// ScoreTorrent calculates the desirability score for a torrent.
// It modifies the torrent in place, setting Score, NormalizedScore, and ScoreBreakdown.
func (s *Scorer) ScoreTorrent(torrent *types.TorrentInfo, ctx ScoringContext) {
	breakdown := &types.ScoreBreakdown{}

	// Calculate each score component
	breakdown.QualityScore = s.calculateQualityScore(torrent, ctx, breakdown)
	breakdown.HealthScore = s.calculateHealthScore(torrent)
	breakdown.IndexerScore = s.calculateIndexerScore(torrent, ctx)
	breakdown.MatchScore = s.calculateMatchScore(torrent, ctx)
	breakdown.AgeScore = s.calculateAgeScore(torrent, ctx)
	breakdown.LanguageScore = s.calculateLanguageScore(torrent, ctx)

	// Total score
	torrent.Score = breakdown.QualityScore + breakdown.HealthScore +
		breakdown.IndexerScore + breakdown.MatchScore + breakdown.AgeScore +
		breakdown.LanguageScore

	// Normalized score (0-100), clamped
	// Max theoretical positive score: 100 (quality) + 65 (health: 35+15+15) + 20 (indexer) + 30 (match) = 215
	// Normalize to 0-100 range
	normalized := (torrent.Score / 215.0) * 100
	if normalized < 0 {
		normalized = 0
	} else if normalized > 100 {
		normalized = 100
	}
	torrent.NormalizedScore = int(math.Round(normalized))

	torrent.ScoreBreakdown = breakdown
}

// ScoreTorrents scores and sorts a slice of torrents by desirability.
// Torrents are sorted by score descending (highest first).
func (s *Scorer) ScoreTorrents(torrents []types.TorrentInfo, ctx ScoringContext) {
	for i := range torrents {
		s.ScoreTorrent(&torrents[i], ctx)
	}

	// Sort by score descending
	sort.Slice(torrents, func(i, j int) bool {
		return torrents[i].Score > torrents[j].Score
	})
}

// calculateQualityScore calculates the quality component of the score.
// Returns 0-100 for allowed qualities, or DisallowedPenalty for disallowed ones.
func (s *Scorer) calculateQualityScore(torrent *types.TorrentInfo, ctx ScoringContext, breakdown *types.ScoreBreakdown) float64 {
	// Try to match quality from source and resolution
	matchResult := MatchQuality(torrent.Source, torrent.Resolution)

	if matchResult.Quality != nil {
		breakdown.QualityID = matchResult.QualityID
		breakdown.QualityName = matchResult.Quality.Name

		// Check if this quality is allowed in the profile
		if ctx.QualityProfile != nil && !ctx.QualityProfile.IsAcceptable(matchResult.QualityID) {
			return s.config.DisallowedPenalty
		}

		// Score based on quality weight
		score := (float64(matchResult.Quality.Weight) / float64(MaxQualityWeight)) * s.config.MaxQualityPoints
		return score * matchResult.Confidence
	}

	// Unknown quality - estimate from resolution if available
	if torrent.Resolution > 0 {
		estimatedWeight := EstimateQualityWeightFromResolution(torrent.Resolution)
		return (float64(estimatedWeight) / float64(MaxQualityWeight)) * s.config.MaxQualityPoints * s.config.UnknownQualityFactor
	}

	// Completely unknown quality - give minimum score
	return s.config.MaxQualityPoints * s.config.UnknownQualityFactor * 0.5
}

// calculateHealthScore calculates the health component for torrents.
// Returns 0-50 based on seeders, ratio, and freeleech status.
func (s *Scorer) calculateHealthScore(torrent *types.TorrentInfo) float64 {
	var score float64

	// Seeder score: logarithmic, capped at MaxSeederPoints
	// log2(1+1) = 1, log2(10+1) ≈ 3.5, log2(100+1) ≈ 6.7, log2(1000+1) ≈ 10
	if torrent.Seeders > 0 {
		seederScore := 5 * math.Log2(float64(torrent.Seeders+1))
		if seederScore > s.config.MaxSeederPoints {
			seederScore = s.config.MaxSeederPoints
		}
		score += seederScore
	}

	// Ratio score: seeders / leechers, capped at MaxRatioPoints
	leechers := torrent.Leechers
	if leechers < 1 {
		leechers = 1
	}
	ratio := float64(torrent.Seeders) / float64(leechers)
	ratioScore := ratio * 3 // 3 points per ratio point
	if ratioScore > s.config.MaxRatioPoints {
		ratioScore = s.config.MaxRatioPoints
	}
	score += ratioScore

	// Freeleech bonus
	if torrent.DownloadVolumeFactor == 0 {
		score += s.config.FreeleechPoints
	}

	return score
}

// calculateIndexerScore calculates the indexer priority component.
// Returns 0-20 based on indexer priority (lower priority number = higher score).
func (s *Scorer) calculateIndexerScore(torrent *types.TorrentInfo, ctx ScoringContext) float64 {
	priority := ctx.GetIndexerPriority(torrent.IndexerID)

	// Priority is 1-100, lower is better
	// Score = MaxIndexerPoints * (1 - (priority - 1) / 99)
	// Priority 1 -> 20, Priority 50 -> ~10, Priority 100 -> 0
	score := s.config.MaxIndexerPoints * (1 - float64(priority-1)/99)
	if score < 0 {
		score = 0
	}
	return score
}

// calculateMatchScore calculates the content matching component.
// Returns 0-30 based on year match and episode match.
func (s *Scorer) calculateMatchScore(torrent *types.TorrentInfo, ctx ScoringContext) float64 {
	var score float64

	// Parse the release title to extract year and episode info
	parsed := scanner.ParseFilename(torrent.Title)

	// Year matching (movies)
	if ctx.SearchYear > 0 {
		if parsed.Year == 0 {
			// Year not present in release - no penalty, give full points
			score += s.config.YearMatchPoints
		} else if parsed.Year == ctx.SearchYear {
			// Exact year match
			score += s.config.YearMatchPoints
		}
		// Else: year mismatch - no points (but no penalty)
	} else {
		// No year filter - give full year points
		score += s.config.YearMatchPoints
	}

	// Episode matching (TV)
	if ctx.SearchSeason > 0 {
		if parsed.Season == ctx.SearchSeason {
			if ctx.SearchEpisode > 0 {
				// Looking for specific episode
				if parsed.Episode == ctx.SearchEpisode {
					// Exact episode match
					score += s.config.ExactEpisodePoints
				} else if parsed.Episode == 0 {
					// Season pack (no episode number) - partial points
					score += s.config.SeasonPackPoints
				}
				// Else: wrong episode - no points
			} else {
				// Looking for whole season - season packs are good
				if parsed.Episode == 0 {
					// Season pack
					score += s.config.ExactEpisodePoints
				} else {
					// Individual episode when looking for season - partial points
					score += s.config.SeasonPackPoints
				}
			}
		}
		// Else: wrong season - no points
	}

	return score
}

// calculateAgeScore calculates the age penalty component.
// Returns 0 to -20 based on how old the release is.
func (s *Scorer) calculateAgeScore(torrent *types.TorrentInfo, ctx ScoringContext) float64 {
	now := ctx.GetNow()
	age := now.Sub(torrent.PublishDate)
	days := age.Hours() / 24

	// No penalty for releases within AgePenaltyStartDays
	if days <= float64(s.config.AgePenaltyStartDays) {
		return 0
	}

	// Gradual penalty after that
	// -5 points per 30 days after the grace period
	daysOverGrace := days - float64(s.config.AgePenaltyStartDays)
	penalty := (daysOverGrace / 30) * 5

	if penalty > s.config.MaxAgePenalty {
		penalty = s.config.MaxAgePenalty
	}

	return -penalty
}

// calculateLanguageScore calculates the language penalty component.
// Returns 0 for matching language or releases without explicit language tags.
// Returns a negative penalty for releases tagged with non-preferred languages.
func (s *Scorer) calculateLanguageScore(torrent *types.TorrentInfo, ctx ScoringContext) float64 {
	// No languages detected means we assume English (default)
	if len(torrent.Languages) == 0 {
		return 0
	}

	preferredLang := ctx.GetPreferredLanguage()

	// Check if any detected language matches the preferred language
	for _, lang := range torrent.Languages {
		if lang == preferredLang {
			return 0 // Preferred language found, no penalty
		}
	}

	// Release has explicit non-preferred language(s) - apply penalty
	return s.config.LanguageMismatchPenalty
}

// GetQualityForRelease returns the matched quality for a release.
// Useful for external callers who need quality info without full scoring.
func GetQualityForRelease(source string, resolution int) (*quality.Quality, bool) {
	result := MatchQuality(source, resolution)
	if result.Quality != nil {
		return result.Quality, true
	}
	return nil, false
}
