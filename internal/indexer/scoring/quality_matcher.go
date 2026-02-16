package scoring

import (
	"strings"

	"github.com/slipstream/slipstream/internal/library/quality"
)

// MatchResult contains the result of quality matching.
type MatchResult struct {
	QualityID  int
	Quality    *quality.Quality
	Confidence float64 // 0.0-1.0, how confident we are in the match
}

// sourceMapping maps parsed source strings to quality source identifiers.
// Keys are lowercase for case-insensitive matching.
var sourceMapping = map[string]string{
	"bluray":  "bluray",
	"blu-ray": "bluray",
	"bdrip":   "bluray",
	"brrip":   "bluray",
	"bdremux": "remux",
	"remux":   "remux",
	"web-dl":  "webdl",
	"webdl":   "webdl",
	"webrip":  "webrip",
	"web":     "webdl", // Assume WEB-DL if just "WEB"
	"hdtv":    "tv",
	"sdtv":    "tv",
	"pdtv":    "tv",
	"dsr":     "tv",
	"dvdrip":  "dvd",
	"dvd-r":   "dvd",
	"dvd":     "dvd",
}

// NormalizeSource converts a parsed source string to a quality source identifier.
func NormalizeSource(source string) string {
	lower := strings.ToLower(source)
	if normalized, ok := sourceMapping[lower]; ok {
		return normalized
	}
	// If source contains certain keywords, try to match
	if strings.Contains(lower, "remux") {
		return "remux"
	}
	if strings.Contains(lower, "bluray") || strings.Contains(lower, "blu-ray") {
		return "bluray"
	}
	if strings.Contains(lower, "web") {
		if strings.Contains(lower, "rip") {
			return "webrip"
		}
		return "webdl"
	}
	if strings.Contains(lower, "hdtv") || strings.Contains(lower, "tv") {
		return "tv"
	}
	if strings.Contains(lower, "dvd") {
		return "dvd"
	}
	return ""
}

// MatchQuality matches a release's source/resolution to a quality.
// Returns the matched quality ID, the quality definition, and a confidence score.
func MatchQuality(source string, resolution int) MatchResult {
	normalizedSource := NormalizeSource(source)

	if normalizedSource != "" && resolution > 0 {
		if result, ok := matchExact(normalizedSource, resolution); ok {
			return result
		}
	}

	if resolution > 0 {
		if best := bestQualityByField(func(q *quality.Quality) bool { return q.Resolution == resolution }); best != nil {
			return MatchResult{QualityID: best.ID, Quality: best, Confidence: 0.7}
		}
	}

	if normalizedSource != "" {
		if best := bestQualityByField(func(q *quality.Quality) bool { return q.Source == normalizedSource }); best != nil {
			return MatchResult{QualityID: best.ID, Quality: best, Confidence: 0.5}
		}
	}

	return MatchResult{}
}

func matchExact(source string, resolution int) (MatchResult, bool) {
	for _, q := range quality.PredefinedQualities {
		if q.Source == source && q.Resolution == resolution {
			return MatchResult{QualityID: q.ID, Quality: &q, Confidence: 1.0}, true
		}
	}
	return MatchResult{}, false
}

func bestQualityByField(matches func(q *quality.Quality) bool) *quality.Quality {
	var best *quality.Quality
	for i := range quality.PredefinedQualities {
		q := &quality.PredefinedQualities[i]
		if matches(q) && (best == nil || q.Weight > best.Weight) {
			best = q
		}
	}
	return best
}

// EstimateQualityWeightFromResolution provides a rough quality weight estimate
// when we can't determine the exact quality. Used for scoring unknown qualities.
func EstimateQualityWeightFromResolution(resolution int) int {
	switch {
	case resolution >= 2160:
		return 15 // Roughly WEBDL-2160p
	case resolution >= 1080:
		return 10 // Roughly WEBDL-1080p
	case resolution >= 720:
		return 6 // Roughly WEBDL-720p
	case resolution >= 480:
		return 2 // Roughly DVD
	default:
		return 1 // SDTV
	}
}
