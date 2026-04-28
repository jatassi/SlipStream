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

// sourceCAM is the canonical source identifier for camera-recorded releases
// (CAM, HDCAM, TS, HDTS, TELESYNC). Match against quality.PredefinedQualities
// where Source == sourceCAM.
const sourceCAM = "cam"

// sourceMapping maps parsed source strings to quality source identifiers.
// Keys are lowercase for case-insensitive matching.
var sourceMapping = map[string]string{
	"bluray":   "bluray",
	"blu-ray":  "bluray",
	"bdrip":    "bluray",
	"brrip":    "bluray",
	"bdremux":  "remux",
	"remux":    "remux",
	"web-dl":   "webdl",
	"webdl":    "webdl",
	"webrip":   "webrip",
	"web":      "webdl", // Assume WEB-DL if just "WEB"
	"hdtv":     "tv",
	"sdtv":     "tv",
	"pdtv":     "tv",
	"dsr":      "tv",
	"dvdrip":   "dvd",
	"dvd-r":    "dvd",
	"dvd":      "dvd",
	"cam":      sourceCAM,
	"hdcam":    sourceCAM,
	"ts":       sourceCAM,
	"hdts":     sourceCAM,
	"telesync": sourceCAM,
}

// keywordSourceFallbacks maps a substring to a normalized source. Used when a
// raw source string doesn't match `sourceMapping` exactly (e.g. "BDRemux2.0").
// Order matters: the first match wins, so list the most specific patterns first.
var keywordSourceFallbacks = []struct {
	keyword string
	source  string
}{
	{"remux", "remux"},
	{"bluray", "bluray"},
	{"blu-ray", "bluray"},
	{"telesync", sourceCAM},
	{"hdcam", sourceCAM},
	{"hdts", sourceCAM},
	{"hdtv", "tv"},
	{"dvd", "dvd"},
}

// NormalizeSource converts a parsed source string to a quality source identifier.
func NormalizeSource(source string) string {
	lower := strings.ToLower(source)
	if normalized, ok := sourceMapping[lower]; ok {
		return normalized
	}
	for _, fb := range keywordSourceFallbacks {
		if strings.Contains(lower, fb.keyword) {
			return fb.source
		}
	}
	if strings.Contains(lower, "web") {
		if strings.Contains(lower, "rip") {
			return "webrip"
		}
		return "webdl"
	}
	if strings.Contains(lower, "tv") {
		return "tv"
	}
	return ""
}

// MatchQuality matches a release's source/resolution to a quality.
// Returns the matched quality ID, the quality definition, and a confidence score.
func MatchQuality(source string, resolution int) MatchResult {
	normalizedSource := NormalizeSource(source)

	// CAM/TELESYNC sources map directly to the CAM tier regardless of any
	// resolution tag in the title — a "1080p TELESYNC" is still a camcorder.
	if normalizedSource == sourceCAM {
		if best := bestQualityByField(func(q *quality.Quality) bool { return q.Source == sourceCAM }); best != nil {
			return MatchResult{QualityID: best.ID, Quality: best, Confidence: 1.0}
		}
	}

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
