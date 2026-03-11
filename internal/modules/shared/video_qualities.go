package shared

import (
	"strings"

	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

// sourceNormMap maps parseutil source strings (lowercased) to quality source identifiers.
var sourceNormMap = map[string]string{
	"remux":   "remux",
	"bluray":  "bluray",
	"blu-ray": "bluray",
	"bdrip":   "bluray",
	"brrip":   "bluray",
	"webrip":  "webrip",
	"web-dl":  "webdl",
	"webdl":   "webdl",
	"web":     "webdl",
	"hdtv":    "tv",
	"sdtv":    "tv",
	"pdtv":    "tv",
	"dsr":     "tv",
	"dvdrip":  "dvd",
	"dvd-r":   "dvd",
	"dvd":     "dvd",
	"cam":     "",
}

// resolutionMap maps resolution strings to integer values.
var resolutionMap = map[string]int{
	"2160p": 2160,
	"1080p": 1080,
	"720p":  720,
	"480p":  480,
}

// normalizeSource converts a parseutil source string to a quality source identifier.
func normalizeSource(source string) string {
	if normalized, ok := sourceNormMap[strings.ToLower(source)]; ok {
		return normalized
	}
	return ""
}

// matchQualityItem finds the best matching QualityItem for the given source and resolution.
func matchQualityItem(normalizedSource string, resolution int) *module.QualityItem {
	items := VideoQualityItems()

	if match := matchExact(items, normalizedSource, resolution); match != nil {
		return match
	}
	if match := matchByField(items, func(q *module.QualityItem) bool { return q.Resolution == resolution }, resolution > 0); match != nil {
		return match
	}
	return matchByField(items, func(q *module.QualityItem) bool { return q.Source == normalizedSource }, normalizedSource != "")
}

func matchExact(items []module.QualityItem, source string, resolution int) *module.QualityItem {
	if source == "" || resolution == 0 {
		return nil
	}
	for i := range items {
		if items[i].Source == source && items[i].Resolution == resolution {
			return &items[i]
		}
	}
	return nil
}

func matchByField(items []module.QualityItem, predicate func(*module.QualityItem) bool, enabled bool) *module.QualityItem {
	if !enabled {
		return nil
	}
	var best *module.QualityItem
	for i := range items {
		if predicate(&items[i]) && (best == nil || items[i].Weight > best.Weight) {
			best = &items[i]
		}
	}
	return best
}

// ParseVideoQuality parses a release title and returns the matched QualityResult.
// Delegates to parseutil for title parsing and matches against the standard video quality items.
func ParseVideoQuality(releaseTitle string) (*module.QualityResult, error) {
	qualityStr, source, _ := parseutil.ParseVideoQuality(releaseTitle)

	resolution := resolutionMap[qualityStr]
	normalizedSource := normalizeSource(source)

	result := &module.QualityResult{}
	if matched := matchQualityItem(normalizedSource, resolution); matched != nil {
		result.Quality = *matched
	}

	result.Proper = strings.EqualFold(parseutil.ParseRevision(releaseTitle), "proper")
	return result, nil
}

// VideoQualityItems returns the 17 standard video quality tiers used by
// both the Movie and TV modules. Future non-video modules (music, books)
// would define their own quality items.
func VideoQualityItems() []module.QualityItem {
	return []module.QualityItem{
		{ID: 1, Name: "SDTV", Source: "tv", Resolution: 480, Weight: 1},
		{ID: 2, Name: "DVD", Source: "dvd", Resolution: 480, Weight: 2},
		{ID: 3, Name: "WEBRip-480p", Source: "webrip", Resolution: 480, Weight: 3},
		{ID: 4, Name: "HDTV-720p", Source: "tv", Resolution: 720, Weight: 4},
		{ID: 5, Name: "WEBRip-720p", Source: "webrip", Resolution: 720, Weight: 5},
		{ID: 6, Name: "WEBDL-720p", Source: "webdl", Resolution: 720, Weight: 6},
		{ID: 7, Name: "Bluray-720p", Source: "bluray", Resolution: 720, Weight: 7},
		{ID: 8, Name: "HDTV-1080p", Source: "tv", Resolution: 1080, Weight: 8},
		{ID: 9, Name: "WEBRip-1080p", Source: "webrip", Resolution: 1080, Weight: 9},
		{ID: 10, Name: "WEBDL-1080p", Source: "webdl", Resolution: 1080, Weight: 10},
		{ID: 11, Name: "Bluray-1080p", Source: "bluray", Resolution: 1080, Weight: 11},
		{ID: 12, Name: "Remux-1080p", Source: "remux", Resolution: 1080, Weight: 12},
		{ID: 13, Name: "HDTV-2160p", Source: "tv", Resolution: 2160, Weight: 13},
		{ID: 14, Name: "WEBRip-2160p", Source: "webrip", Resolution: 2160, Weight: 14},
		{ID: 15, Name: "WEBDL-2160p", Source: "webdl", Resolution: 2160, Weight: 15},
		{ID: 16, Name: "Bluray-2160p", Source: "bluray", Resolution: 2160, Weight: 16},
		{ID: 17, Name: "Remux-2160p", Source: "remux", Resolution: 2160, Weight: 17},
	}
}
