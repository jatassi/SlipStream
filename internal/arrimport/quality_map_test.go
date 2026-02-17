package arrimport

import (
	"testing"
)

// TestRadarrQualityMap verifies every entry from spec §2.4
func TestRadarrQualityMap(t *testing.T) {
	tests := []struct {
		radarrID     int
		name         string
		expectedSSID *int
	}{
		// Direct matches (spec §2.4, rows 1-17)
		{1, "SDTV", intPtr(1)},
		{2, "DVD", intPtr(2)},
		{8, "WEBDL-480p", intPtr(3)}, // nearest: WEBRip-480p
		{12, "WEBRip-480p", intPtr(3)},
		{4, "HDTV-720p", intPtr(4)},
		{5, "WEBDL-720p", intPtr(6)},
		{14, "WEBRip-720p", intPtr(5)},
		{6, "Bluray-720p", intPtr(7)},
		{9, "HDTV-1080p", intPtr(8)},
		{3, "WEBDL-1080p", intPtr(10)},
		{15, "WEBRip-1080p", intPtr(9)},
		{7, "Bluray-1080p", intPtr(11)},
		{30, "Remux-1080p", intPtr(12)},
		{16, "HDTV-2160p", intPtr(13)},
		{18, "WEBDL-2160p", intPtr(15)},
		{17, "WEBRip-2160p", intPtr(14)},
		{19, "Bluray-2160p", intPtr(16)},
		{31, "Remux-2160p", intPtr(17)},
		// Approximate matches
		{20, "Bluray-480p", intPtr(2)}, // nearest: DVD
		{21, "Bluray-576p", intPtr(2)}, // nearest: DVD
		{23, "DVD-R", intPtr(2)},       // nearest: DVD
		// Unmapped (should return nil)
		{0, "Unknown", nil},
		{10, "Raw-HD", nil},
		{22, "BR-DISK", nil},
		{24, "WORKPRINT", nil},
		{25, "CAM", nil},
		{26, "TELESYNC", nil},
		{27, "TELECINE", nil},
		{28, "DVDSCR", nil},
		{29, "REGIONAL", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapRadarrQualityID(tt.radarrID)
			if tt.expectedSSID == nil {
				if result != nil {
					t.Errorf("radarr ID %d (%s): expected nil, got %d", tt.radarrID, tt.name, *result)
				}
			} else {
				if result == nil {
					t.Errorf("radarr ID %d (%s): expected %d, got nil", tt.radarrID, tt.name, *tt.expectedSSID)
				} else if *result != int64(*tt.expectedSSID) {
					t.Errorf("radarr ID %d (%s): expected %d, got %d", tt.radarrID, tt.name, *tt.expectedSSID, *result)
				}
			}
		})
	}
}

// TestSonarrQualityMap verifies Sonarr-specific additions from spec §3.6
func TestSonarrQualityMap(t *testing.T) {
	sonarrOnlyTests := []struct {
		sonarrID     int
		name         string
		expectedSSID *int
	}{
		{13, "Bluray-480p", intPtr(2)},         // nearest: DVD
		{20, "Bluray-1080p Remux", intPtr(12)}, // Remux-1080p
		{21, "Bluray-2160p Remux", intPtr(17)}, // Remux-2160p
		{22, "Bluray-576p", intPtr(2)},         // nearest: DVD
	}

	for _, tt := range sonarrOnlyTests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapSonarrQualityID(tt.sonarrID)
			if tt.expectedSSID == nil {
				if result != nil {
					t.Errorf("sonarr ID %d (%s): expected nil, got %d", tt.sonarrID, tt.name, *result)
				}
			} else {
				if result == nil {
					t.Errorf("sonarr ID %d (%s): expected %d, got nil", tt.sonarrID, tt.name, *tt.expectedSSID)
				} else if *result != int64(*tt.expectedSSID) {
					t.Errorf("sonarr ID %d (%s): expected %d, got %d", tt.sonarrID, tt.name, *tt.expectedSSID, *result)
				}
			}
		})
	}

	// Verify Sonarr includes all Radarr mappings EXCEPT overridden IDs 20, 21
	radarrMapped := []int{1, 2, 8, 12, 4, 5, 14, 6, 9, 3, 15, 7, 30, 16, 18, 17, 19, 31, 23}
	sonarrOverrides := map[int]bool{20: true, 21: true} // These differ between Radarr and Sonarr
	for _, id := range radarrMapped {
		if sonarrOverrides[id] {
			continue // Skip IDs that are intentionally different
		}
		r := mapRadarrQualityID(id)
		s := mapSonarrQualityID(id)
		if r == nil || s == nil {
			continue // Skip unmapped
		}
		if *r != *s {
			t.Errorf("ID %d: Radarr maps to %d but Sonarr maps to %d (should match)", id, *r, *s)
		}
	}

	// Verify Sonarr overrides are different from Radarr
	r20 := mapRadarrQualityID(20)
	s20 := mapSonarrQualityID(20)
	if r20 != nil && s20 != nil && *r20 == *s20 {
		t.Errorf("ID 20: Radarr and Sonarr should differ (Radarr=Bluray-480p→DVD, Sonarr=Remux-1080p)")
	}
	r21 := mapRadarrQualityID(21)
	s21 := mapSonarrQualityID(21)
	if r21 != nil && s21 != nil && *r21 == *s21 {
		t.Errorf("ID 21: Radarr and Sonarr should differ (Radarr=Bluray-576p→DVD, Sonarr=Remux-2160p)")
	}
}

func intPtr(v int) *int { return &v }
