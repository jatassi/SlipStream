package shared

import "github.com/slipstream/slipstream/internal/module"

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
