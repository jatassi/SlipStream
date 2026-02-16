package quality

import (
	"encoding/json"
	"testing"
)

func TestGetQualityByID(t *testing.T) {
	tests := []struct {
		id       int
		wantName string
		wantOK   bool
	}{
		{1, "SDTV", true},
		{2, "DVD", true},
		{3, "WEBRip-480p", true},
		{4, "HDTV-720p", true},
		{5, "WEBRip-720p", true},
		{6, "WEBDL-720p", true},
		{7, "Bluray-720p", true},
		{8, "HDTV-1080p", true},
		{9, "WEBRip-1080p", true},
		{10, "WEBDL-1080p", true},
		{11, "Bluray-1080p", true},
		{12, "Remux-1080p", true},
		{13, "HDTV-2160p", true},
		{14, "WEBRip-2160p", true},
		{15, "WEBDL-2160p", true},
		{16, "Bluray-2160p", true},
		{17, "Remux-2160p", true},
		{0, "", false},
		{-1, "", false},
		{100, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			q, ok := GetQualityByID(tt.id)
			if ok != tt.wantOK {
				t.Errorf("GetQualityByID(%d) ok = %v, want %v", tt.id, ok, tt.wantOK)
			}
			if ok && q.Name != tt.wantName {
				t.Errorf("GetQualityByID(%d).Name = %q, want %q", tt.id, q.Name, tt.wantName)
			}
		})
	}
}

func TestGetQualityByName(t *testing.T) {
	tests := []struct {
		name   string
		wantID int
		wantOK bool
	}{
		{"SDTV", 1, true},
		{"Bluray-1080p", 11, true},
		{"Remux-2160p", 17, true},
		{"Unknown", 0, false},
		{"", 0, false},
		{"bluray-1080p", 0, false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, ok := GetQualityByName(tt.name)
			if ok != tt.wantOK {
				t.Errorf("GetQualityByName(%q) ok = %v, want %v", tt.name, ok, tt.wantOK)
			}
			if ok && q.ID != tt.wantID {
				t.Errorf("GetQualityByName(%q).ID = %d, want %d", tt.name, q.ID, tt.wantID)
			}
		})
	}
}

func TestPredefinedQualities(t *testing.T) {
	// Verify we have 17 quality tiers
	if len(PredefinedQualities) != 17 {
		t.Errorf("PredefinedQualities has %d entries, want 17", len(PredefinedQualities))
	}

	// Verify weights are sequential and unique
	weights := make(map[int]bool)
	for _, q := range PredefinedQualities {
		if weights[q.Weight] {
			t.Errorf("Duplicate weight %d found for quality %s", q.Weight, q.Name)
		}
		weights[q.Weight] = true

		if q.Weight < 1 || q.Weight > 17 {
			t.Errorf("Quality %s has weight %d, expected 1-17", q.Name, q.Weight)
		}
	}

	// Verify IDs match weights (they should be the same in this implementation)
	for _, q := range PredefinedQualities {
		if q.ID != q.Weight {
			t.Errorf("Quality %s has ID %d but weight %d", q.Name, q.ID, q.Weight)
		}
	}

	// Verify resolution values are valid
	validResolutions := map[int]bool{480: true, 720: true, 1080: true, 2160: true}
	for _, q := range PredefinedQualities {
		if !validResolutions[q.Resolution] {
			t.Errorf("Quality %s has invalid resolution %d", q.Name, q.Resolution)
		}
	}
}

func TestDefaultProfile(t *testing.T) {
	profile := DefaultProfile()

	if profile.Name != "Any" {
		t.Errorf("DefaultProfile().Name = %q, want %q", profile.Name, "Any")
	}

	if profile.Cutoff != 11 {
		t.Errorf("DefaultProfile().Cutoff = %d, want %d", profile.Cutoff, 11)
	}

	if len(profile.Items) != len(PredefinedQualities) {
		t.Errorf("DefaultProfile().Items has %d entries, want %d", len(profile.Items), len(PredefinedQualities))
	}

	// All qualities should be allowed in default profile
	for _, item := range profile.Items {
		if !item.Allowed {
			t.Errorf("DefaultProfile() has disallowed quality: %s", item.Quality.Name)
		}
	}
}

func TestHD1080pProfile(t *testing.T) {
	profile := HD1080pProfile()

	if profile.Name != "HD-1080p" {
		t.Errorf("HD1080pProfile().Name = %q, want %q", profile.Name, "HD-1080p")
	}

	if profile.Cutoff != 11 {
		t.Errorf("HD1080pProfile().Cutoff = %d, want %d", profile.Cutoff, 11)
	}

	// Check that only 720p and 1080p qualities are allowed
	for _, item := range profile.Items {
		expectedAllowed := item.Quality.Resolution >= 720 && item.Quality.Resolution <= 1080
		if item.Allowed != expectedAllowed {
			t.Errorf("HD1080pProfile() quality %s: Allowed = %v, want %v",
				item.Quality.Name, item.Allowed, expectedAllowed)
		}
	}
}

func TestUltra4KProfile(t *testing.T) {
	profile := Ultra4KProfile()

	if profile.Name != "Ultra-HD" {
		t.Errorf("Ultra4KProfile().Name = %q, want %q", profile.Name, "Ultra-HD")
	}

	if profile.Cutoff != 16 {
		t.Errorf("Ultra4KProfile().Cutoff = %d, want %d", profile.Cutoff, 16)
	}

	// Check that only 1080p and 2160p qualities are allowed
	for _, item := range profile.Items {
		expectedAllowed := item.Quality.Resolution >= 1080
		if item.Allowed != expectedAllowed {
			t.Errorf("Ultra4KProfile() quality %s: Allowed = %v, want %v",
				item.Quality.Name, item.Allowed, expectedAllowed)
		}
	}
}

func TestProfile_IsAcceptable(t *testing.T) {
	profile := HD1080pProfile()

	tests := []struct {
		qualityID int
		want      bool
	}{
		{1, false},   // SDTV - 480p, not allowed
		{2, false},   // DVD - 480p, not allowed
		{4, true},    // HDTV-720p - allowed
		{6, true},    // WEBDL-720p - allowed
		{10, true},   // WEBDL-1080p - allowed
		{11, true},   // Bluray-1080p - allowed
		{15, false},  // WEBDL-2160p - 2160p, not allowed
		{17, false},  // Remux-2160p - 2160p, not allowed
		{0, false},   // Invalid ID
		{100, false}, // Invalid ID
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := profile.IsAcceptable(tt.qualityID)
			if got != tt.want {
				q, _ := GetQualityByID(tt.qualityID)
				t.Errorf("IsAcceptable(%d/%s) = %v, want %v", tt.qualityID, q.Name, got, tt.want)
			}
		})
	}
}

func TestProfile_IsUpgrade(t *testing.T) {
	profile := HD1080pProfile()

	tests := []struct {
		name        string
		currentID   int
		candidateID int
		want        bool
	}{
		{
			name:        "720p HDTV to 1080p BluRay is upgrade",
			currentID:   4,  // HDTV-720p
			candidateID: 11, // Bluray-1080p
			want:        true,
		},
		{
			name:        "720p to 720p same quality not upgrade",
			currentID:   4, // HDTV-720p
			candidateID: 4, // HDTV-720p
			want:        false,
		},
		{
			name:        "1080p BluRay to 720p HDTV is not upgrade (downgrade)",
			currentID:   11, // Bluray-1080p
			candidateID: 4,  // HDTV-720p
			want:        false,
		},
		{
			name:        "at cutoff, no upgrade allowed",
			currentID:   11, // Bluray-1080p (cutoff)
			candidateID: 12, // Remux-1080p
			want:        false,
		},
		{
			name:        "above cutoff, no upgrade allowed",
			currentID:   12, // Remux-1080p (above cutoff)
			candidateID: 17, // Remux-2160p
			want:        false,
		},
		{
			name:        "candidate not allowed in profile",
			currentID:   4,  // HDTV-720p
			candidateID: 15, // WEBDL-2160p (not allowed in HD1080p profile)
			want:        false,
		},
		{
			name:        "720p WEBDL to 1080p WEBDL is upgrade",
			currentID:   6,  // WEBDL-720p
			candidateID: 10, // WEBDL-1080p
			want:        true,
		},
		{
			name:        "invalid current quality",
			currentID:   0,
			candidateID: 10,
			want:        false,
		},
		{
			name:        "invalid candidate quality",
			currentID:   4,
			candidateID: 0,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.IsUpgrade(tt.currentID, tt.candidateID)
			if got != tt.want {
				t.Errorf("IsUpgrade(%d, %d) = %v, want %v", tt.currentID, tt.candidateID, got, tt.want)
			}
		})
	}
}

func TestProfile_getCutoffWeight(t *testing.T) {
	tests := []struct {
		name       string
		cutoff     int
		wantWeight int
	}{
		{"valid cutoff", 11, 11},
		{"high cutoff", 17, 17},
		{"low cutoff", 1, 1},
		{"invalid cutoff", 0, 0},
		{"invalid cutoff high", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Profile{Cutoff: tt.cutoff}
			got := profile.getCutoffWeight()
			if got != tt.wantWeight {
				t.Errorf("getCutoffWeight() = %d, want %d", got, tt.wantWeight)
			}
		})
	}
}

func TestSerializeItems(t *testing.T) {
	items := []QualityItem{
		{Quality: Quality{ID: 1, Name: "SDTV", Weight: 1}, Allowed: true},
		{Quality: Quality{ID: 2, Name: "DVD", Weight: 2}, Allowed: false},
	}

	jsonStr, err := SerializeItems(items)
	if err != nil {
		t.Fatalf("SerializeItems() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed []QualityItem
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("SerializeItems() produced invalid JSON: %v", err)
	}

	if len(parsed) != len(items) {
		t.Errorf("SerializeItems() produced %d items, want %d", len(parsed), len(items))
	}
}

func TestDeserializeItems(t *testing.T) {
	jsonStr := `[{"quality":{"id":1,"name":"SDTV","weight":1},"allowed":true},{"quality":{"id":2,"name":"DVD","weight":2},"allowed":false}]`

	items, err := DeserializeItems(jsonStr)
	if err != nil {
		t.Fatalf("DeserializeItems() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("DeserializeItems() returned %d items, want 2", len(items))
	}

	if items[0].Quality.ID != 1 || items[0].Quality.Name != "SDTV" {
		t.Errorf("First item = %+v, want ID=1, Name=SDTV", items[0])
	}

	if !items[0].Allowed {
		t.Error("First item Allowed = false, want true")
	}

	if items[1].Allowed {
		t.Error("Second item Allowed = true, want false")
	}
}

func TestDeserializeItems_InvalidJSON(t *testing.T) {
	invalidJSON := `{invalid json}`

	_, err := DeserializeItems(invalidJSON)
	if err == nil {
		t.Error("DeserializeItems() with invalid JSON should return error")
	}
}

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	original := DefaultProfile().Items

	serialized, err := SerializeItems(original)
	if err != nil {
		t.Fatalf("SerializeItems() error = %v", err)
	}

	deserialized, err := DeserializeItems(serialized)
	if err != nil {
		t.Fatalf("DeserializeItems() error = %v", err)
	}

	if len(deserialized) != len(original) {
		t.Fatalf("Round trip changed item count: %d -> %d", len(original), len(deserialized))
	}

	for i := range original {
		if deserialized[i].Quality.ID != original[i].Quality.ID {
			t.Errorf("Item %d Quality.ID: %d != %d", i, deserialized[i].Quality.ID, original[i].Quality.ID)
		}
		if deserialized[i].Quality.Name != original[i].Quality.Name {
			t.Errorf("Item %d Quality.Name: %q != %q", i, deserialized[i].Quality.Name, original[i].Quality.Name)
		}
		if deserialized[i].Allowed != original[i].Allowed {
			t.Errorf("Item %d Allowed: %v != %v", i, deserialized[i].Allowed, original[i].Allowed)
		}
	}
}

func TestQualityWeightOrdering(t *testing.T) {
	// Verify that higher resolution qualities have higher weights
	// This ensures upgrade logic works correctly

	resolutionGroups := map[int][]Quality{
		480:  {},
		720:  {},
		1080: {},
		2160: {},
	}

	for _, q := range PredefinedQualities {
		resolutionGroups[q.Resolution] = append(resolutionGroups[q.Resolution], q)
	}

	// All 480p qualities should have lower weight than all 720p qualities
	for _, q480 := range resolutionGroups[480] {
		for _, q720 := range resolutionGroups[720] {
			if q480.Weight >= q720.Weight {
				t.Errorf("%s (weight %d) should be less than %s (weight %d)",
					q480.Name, q480.Weight, q720.Name, q720.Weight)
			}
		}
	}

	// All 720p qualities should have lower weight than all 1080p qualities
	for _, q720 := range resolutionGroups[720] {
		for _, q1080 := range resolutionGroups[1080] {
			if q720.Weight >= q1080.Weight {
				t.Errorf("%s (weight %d) should be less than %s (weight %d)",
					q720.Name, q720.Weight, q1080.Name, q1080.Weight)
			}
		}
	}

	// All 1080p qualities should have lower weight than all 2160p qualities
	for _, q1080 := range resolutionGroups[1080] {
		for _, q2160 := range resolutionGroups[2160] {
			if q1080.Weight >= q2160.Weight {
				t.Errorf("%s (weight %d) should be less than %s (weight %d)",
					q1080.Name, q1080.Weight, q2160.Name, q2160.Weight)
			}
		}
	}
}

func TestModalityTier(t *testing.T) {
	tests := []struct {
		source string
		want   int
	}{
		{"tv", 1},
		{"dvd", 2},
		{"webrip", 3},
		{"webdl", 4},
		{"bluray", 5},
		{"remux", 6},
		{"unknown", 0},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := ModalityTier(tt.source)
			if got != tt.want {
				t.Errorf("ModalityTier(%q) = %d, want %d", tt.source, got, tt.want)
			}
		})
	}
}

func TestIsDiscSource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"bluray", true},
		{"remux", true},
		{"tv", false},
		{"dvd", false},
		{"webrip", false},
		{"webdl", false},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := IsDiscSource(tt.source); got != tt.want {
				t.Errorf("IsDiscSource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestIsValidUpgradeStrategy(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"aggressive", true},
		{"balanced", true},
		{"resolution_only", true},
		{"", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := IsValidUpgradeStrategy(tt.s); got != tt.want {
				t.Errorf("IsValidUpgradeStrategy(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// allAllowedProfile returns a profile with all qualities allowed and given settings.
func allAllowedProfile(cutoff int, strategy UpgradeStrategy) Profile {
	items := make([]QualityItem, len(PredefinedQualities))
	for i, q := range PredefinedQualities {
		items[i] = QualityItem{Quality: q, Allowed: true}
	}
	return Profile{
		Cutoff:          cutoff,
		UpgradesEnabled: true,
		UpgradeStrategy: strategy,
		Items:           items,
	}
}

func TestProfile_IsUpgrade_Aggressive(t *testing.T) {
	profile := allAllowedProfile(16, StrategyAggressive) // Bluray-2160p cutoff

	tests := []struct {
		name        string
		currentID   int
		candidateID int
		want        bool
	}{
		{"HDTV-720p→WEBRip-720p is upgrade (higher weight)", 4, 5, true},
		{"WEBRip-1080p→WEBDL-1080p is upgrade", 9, 10, true},
		{"WEBDL-1080p→WEBRip-1080p is not upgrade (lower weight)", 10, 9, false},
		{"same quality not upgrade", 10, 10, false},
		{"720p→1080p is upgrade", 4, 8, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.IsUpgrade(tt.currentID, tt.candidateID)
			if got != tt.want {
				t.Errorf("IsUpgrade(%d, %d) = %v, want %v", tt.currentID, tt.candidateID, got, tt.want)
			}
		})
	}
}

func TestProfile_IsUpgrade_Balanced(t *testing.T) {
	profile := allAllowedProfile(16, StrategyBalanced) // Bluray-2160p cutoff

	tests := []struct {
		name        string
		currentID   int
		candidateID int
		want        bool
	}{
		// Resolution upgrades always accepted
		{"720p→1080p resolution upgrade", 4, 8, true},
		{"1080p→2160p resolution upgrade", 8, 13, true},
		{"Bluray-1080p→HDTV-2160p resolution upgrade wins", 11, 13, true},

		// Non-disc → disc at same resolution = upgrade
		{"WEBRip-1080p→Bluray-1080p non-disc to disc", 9, 11, true},
		{"WEBDL-1080p→Bluray-1080p non-disc to disc", 10, 11, true},
		{"HDTV-1080p→Remux-1080p non-disc to disc", 8, 12, true},
		{"WEBDL-720p→Bluray-720p non-disc to disc", 6, 7, true},

		// Within same tier at same resolution = not upgrade
		{"WEBRip-1080p→WEBDL-1080p within non-disc", 9, 10, false},
		{"HDTV-1080p→WEBRip-1080p within non-disc", 8, 9, false},
		{"Bluray-1080p→Remux-1080p within disc", 11, 12, false},

		// Disc → non-disc at same resolution = not upgrade
		{"WEBDL-1080p→WEBRip-1080p not upgrade", 10, 9, false},
		{"Bluray-1080p→WEBDL-1080p disc to non-disc", 11, 10, false},

		// Other rejections
		{"same quality not upgrade", 10, 10, false},
		{"HDTV-720p→HDTV-720p same", 4, 4, false},
		{"2160p→1080p downgrade", 13, 8, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.IsUpgrade(tt.currentID, tt.candidateID)
			if got != tt.want {
				t.Errorf("IsUpgrade(%d, %d) = %v, want %v", tt.currentID, tt.candidateID, got, tt.want)
			}
		})
	}
}

func TestProfile_IsUpgrade_ResolutionOnly(t *testing.T) {
	profile := allAllowedProfile(16, StrategyResolutionOnly) // Bluray-2160p cutoff

	tests := []struct {
		name        string
		currentID   int
		candidateID int
		want        bool
	}{
		{"720p→1080p resolution upgrade", 4, 8, true},
		{"1080p→2160p resolution upgrade", 8, 13, true},
		{"WEBRip-1080p→WEBDL-1080p not upgrade (same resolution)", 9, 10, false},
		{"WEBRip-1080p→Bluray-1080p not upgrade (same resolution)", 9, 11, false},
		{"HDTV-1080p→Remux-1080p not upgrade (same resolution)", 8, 12, false},
		{"same quality not upgrade", 10, 10, false},
		{"2160p→1080p downgrade", 13, 8, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.IsUpgrade(tt.currentID, tt.candidateID)
			if got != tt.want {
				t.Errorf("IsUpgrade(%d, %d) = %v, want %v", tt.currentID, tt.candidateID, got, tt.want)
			}
		})
	}
}

func TestProfile_IsUpgrade_CutoffOverridesStrategy(t *testing.T) {
	profile := allAllowedProfile(12, StrategyBalanced) // Remux-1080p cutoff
	profile.CutoffOverridesStrategy = true

	tests := []struct {
		name        string
		currentID   int
		candidateID int
		want        bool
	}{
		// Candidate matches cutoff exactly → override allows it
		{"Bluray-1080p→Remux-1080p override grabs cutoff", 11, 12, true},
		{"WEBDL-1080p→Remux-1080p override grabs cutoff", 10, 12, true},
		{"HDTV-1080p→Remux-1080p override grabs cutoff", 8, 12, true},

		// Already at cutoff → no upgrade
		{"Remux-1080p→Remux-2160p already at cutoff", 12, 17, false},

		// Above cutoff → no upgrade
		{"HDTV-2160p→Remux-2160p above cutoff", 13, 17, false},

		// Candidate is NOT the cutoff quality → normal strategy applies
		{"WEBDL-1080p→Bluray-1080p non-disc to disc (balanced allows)", 10, 11, true},
		{"WEBRip-1080p→WEBDL-1080p within non-disc (balanced blocks)", 9, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.IsUpgrade(tt.currentID, tt.candidateID)
			if got != tt.want {
				t.Errorf("IsUpgrade(%d, %d) = %v, want %v", tt.currentID, tt.candidateID, got, tt.want)
			}
		})
	}

	// Verify same scenarios without override
	profile.CutoffOverridesStrategy = false
	// Bluray-1080p → Remux-1080p: both disc, balanced blocks within-disc
	if profile.IsUpgrade(11, 12) {
		t.Error("Without override, Bluray→Remux should be blocked by balanced strategy")
	}
}

func TestProfile_IsUpgrade_DefaultStrategy(t *testing.T) {
	// Empty strategy should default to aggressive behavior
	profile := allAllowedProfile(16, "")

	// This tests that empty/unrecognized falls back to aggressive (weight-based)
	// WEBRip-1080p (weight 9) → WEBDL-1080p (weight 10) is upgrade in aggressive
	got := profile.IsUpgrade(9, 10)
	if !got {
		t.Error("Empty strategy should default to aggressive: IsUpgrade(9, 10) = false, want true")
	}

	// Same with unrecognized strategy
	profile.UpgradeStrategy = "bogus"
	got = profile.IsUpgrade(9, 10)
	if !got {
		t.Error("Unrecognized strategy should default to aggressive: IsUpgrade(9, 10) = false, want true")
	}
}

func TestDefaultProfile_UpgradeStrategy(t *testing.T) {
	if DefaultProfile().UpgradeStrategy != StrategyBalanced {
		t.Error("DefaultProfile should use balanced strategy")
	}
	if HD1080pProfile().UpgradeStrategy != StrategyBalanced {
		t.Error("HD1080pProfile should use balanced strategy")
	}
	if Ultra4KProfile().UpgradeStrategy != StrategyBalanced {
		t.Error("Ultra4KProfile should use balanced strategy")
	}
}
