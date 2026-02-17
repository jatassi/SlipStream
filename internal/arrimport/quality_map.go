package arrimport

import (
	"github.com/slipstream/slipstream/internal/library/quality"
)

// radarrQualityMap maps Radarr quality IDs to SlipStream quality IDs.
// Entries from spec §2.4.
var radarrQualityMap = map[int]int64{
	1:  1,  // SDTV → SDTV
	2:  2,  // DVD → DVD
	8:  3,  // WEBDL-480p → WEBRip-480p (nearest)
	12: 3,  // WEBRip-480p → WEBRip-480p
	4:  4,  // HDTV-720p → HDTV-720p
	5:  6,  // WEBDL-720p → WEBDL-720p
	14: 5,  // WEBRip-720p → WEBRip-720p
	6:  7,  // Bluray-720p → Bluray-720p
	9:  8,  // HDTV-1080p → HDTV-1080p
	3:  10, // WEBDL-1080p → WEBDL-1080p
	15: 9,  // WEBRip-1080p → WEBRip-1080p
	7:  11, // Bluray-1080p → Bluray-1080p
	30: 12, // Remux-1080p → Remux-1080p
	16: 13, // HDTV-2160p → HDTV-2160p
	18: 15, // WEBDL-2160p → WEBDL-2160p
	17: 14, // WEBRip-2160p → WEBRip-2160p
	19: 16, // Bluray-2160p → Bluray-2160p
	31: 17, // Remux-2160p → Remux-2160p
	20: 2,  // Bluray-480p → DVD (nearest)
	21: 2,  // Bluray-576p → DVD (nearest)
	23: 2,  // DVD-R → DVD
	// Unmapped: 0 (Unknown), 10 (Raw-HD), 22 (BR-DISK), 24 (WORKPRINT),
	// 25 (CAM), 26 (TELESYNC), 27 (TELECINE), 28 (DVDSCR), 29 (REGIONAL)
}

// sonarrQualityMap maps Sonarr quality IDs to SlipStream quality IDs.
// Starts with all Radarr entries, then overrides/adds Sonarr-specific entries.
var sonarrQualityMap = map[int]int64{
	// Same as Radarr
	1:  1,  // SDTV → SDTV
	2:  2,  // DVD → DVD
	8:  3,  // WEBDL-480p → WEBRip-480p (nearest)
	12: 3,  // WEBRip-480p → WEBRip-480p
	4:  4,  // HDTV-720p → HDTV-720p
	5:  6,  // WEBDL-720p → WEBDL-720p
	14: 5,  // WEBRip-720p → WEBRip-720p
	6:  7,  // Bluray-720p → Bluray-720p
	9:  8,  // HDTV-1080p → HDTV-1080p
	3:  10, // WEBDL-1080p → WEBDL-1080p
	15: 9,  // WEBRip-1080p → WEBRip-1080p
	7:  11, // Bluray-1080p → Bluray-1080p
	30: 12, // Remux-1080p → Remux-1080p
	16: 13, // HDTV-2160p → HDTV-2160p
	18: 15, // WEBDL-2160p → WEBDL-2160p
	17: 14, // WEBRip-2160p → WEBRip-2160p
	19: 16, // Bluray-2160p → Bluray-2160p
	31: 17, // Remux-2160p → Remux-2160p
	23: 2,  // DVD-R → DVD
	// Sonarr-only additions
	13: 2, // Bluray-480p → DVD (nearest)
	22: 2, // Bluray-576p → DVD (nearest)
	// OVERRIDES (different meaning than Radarr for same IDs)
	20: 12, // Sonarr: Bluray-1080p Remux → Remux-1080p (Radarr: Bluray-480p → DVD)
	21: 17, // Sonarr: Bluray-2160p Remux → Remux-2160p (Radarr: Bluray-576p → DVD)
}

// mapRadarrQualityID maps a Radarr quality ID to a SlipStream quality ID.
func mapRadarrQualityID(radarrID int) *int64 {
	if ssID, ok := radarrQualityMap[radarrID]; ok {
		return &ssID
	}
	return nil
}

// mapSonarrQualityID maps a Sonarr quality ID to a SlipStream quality ID.
func mapSonarrQualityID(sonarrID int) *int64 {
	if ssID, ok := sonarrQualityMap[sonarrID]; ok {
		return &ssID
	}
	return nil
}

// MapQualityID maps a source quality ID to a SlipStream quality ID.
// It first checks the static map for the source type, then falls back
// to matching by quality name via quality.GetQualityByName.
func MapQualityID(sourceType SourceType, sourceQualityID int, sourceQualityName string) *int64 {
	var result *int64
	switch sourceType {
	case SourceTypeRadarr:
		result = mapRadarrQualityID(sourceQualityID)
	case SourceTypeSonarr:
		result = mapSonarrQualityID(sourceQualityID)
	}
	if result != nil {
		return result
	}
	// Fallback: try matching by name
	if q, ok := quality.GetQualityByName(sourceQualityName); ok {
		id := int64(q.ID)
		return &id
	}
	return nil
}
