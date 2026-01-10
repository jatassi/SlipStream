// Package scoring provides desirability scoring for indexer search results.
package scoring

import (
	"time"

	"github.com/slipstream/slipstream/internal/library/quality"
)

// ScoringConfig holds configurable weights for the scoring algorithm.
type ScoringConfig struct {
	// Quality weights
	MaxQualityPoints     float64 // default: 100
	UnknownQualityFactor float64 // default: 0.5 (50% of max for unknown quality)
	DisallowedPenalty    float64 // default: -1000 (heavy penalty for disallowed quality)

	// Health weights (torrents)
	MaxSeederPoints float64 // default: 20
	MaxRatioPoints  float64 // default: 15
	FreeleechPoints float64 // default: 15

	// Indexer priority
	MaxIndexerPoints float64 // default: 20

	// Match scoring
	YearMatchPoints    float64 // default: 10
	ExactEpisodePoints float64 // default: 20
	SeasonPackPoints   float64 // default: 10

	// Age penalty
	AgePenaltyStartDays int     // default: 7 (no penalty for first 7 days)
	MaxAgePenalty       float64 // default: 20
}

// DefaultConfig returns sensible default scoring weights.
func DefaultConfig() ScoringConfig {
	return ScoringConfig{
		// Quality weights
		MaxQualityPoints:     100,
		UnknownQualityFactor: 0.5,
		DisallowedPenalty:    -1000,

		// Health weights (torrents)
		MaxSeederPoints: 20,
		MaxRatioPoints:  15,
		FreeleechPoints: 15,

		// Indexer priority
		MaxIndexerPoints: 20,

		// Match scoring
		YearMatchPoints:    10,
		ExactEpisodePoints: 20,
		SeasonPackPoints:   10,

		// Age penalty
		AgePenaltyStartDays: 7,
		MaxAgePenalty:       20,
	}
}

// ScoringContext provides context for scoring a release.
type ScoringContext struct {
	// QualityProfile is used to determine quality scores and allowed qualities.
	QualityProfile *quality.Profile

	// SearchYear is the expected year (for movies). Zero means no year matching.
	SearchYear int

	// SearchSeason is the expected season (for TV). Zero means no season matching.
	SearchSeason int

	// SearchEpisode is the expected episode (for TV). Zero means no episode matching.
	SearchEpisode int

	// IndexerPriorities maps indexer ID to priority (1-100, lower = better).
	// If not set, default priority of 50 is assumed.
	IndexerPriorities map[int64]int

	// Now is the current time for age calculations. If zero, time.Now() is used.
	Now time.Time
}

// GetIndexerPriority returns the priority for an indexer, defaulting to 50.
func (ctx *ScoringContext) GetIndexerPriority(indexerID int64) int {
	if ctx.IndexerPriorities == nil {
		return 50
	}
	if priority, ok := ctx.IndexerPriorities[indexerID]; ok {
		return priority
	}
	return 50
}

// GetNow returns the current time for age calculations.
func (ctx *ScoringContext) GetNow() time.Time {
	if ctx.Now.IsZero() {
		return time.Now()
	}
	return ctx.Now
}
