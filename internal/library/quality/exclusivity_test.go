package quality

import (
	"testing"
)

// Helper to create AttributeSettings with per-item modes
func makeSettings(items map[string]AttributeMode) AttributeSettings {
	return AttributeSettings{Items: items}
}

// Helper to create AttributeSettings where all values have the same mode
func makeSettingsWithMode(mode AttributeMode, values []string) AttributeSettings {
	items := make(map[string]AttributeMode)
	for _, v := range values {
		items[v] = mode
	}
	return AttributeSettings{Items: items}
}

// Test helpers to create profiles with specific attribute settings
func profileWithHDR(name string, mode AttributeMode, values []string) *Profile {
	return &Profile{
		ID:                   1,
		Name:                 name,
		HDRSettings:          makeSettingsWithMode(mode, values),
		VideoCodecSettings:   DefaultAttributeSettings(),
		AudioCodecSettings:   DefaultAttributeSettings(),
		AudioChannelSettings: DefaultAttributeSettings(),
	}
}

func profileWithVideoCodec(name string, mode AttributeMode, values []string) *Profile {
	return &Profile{
		ID:                   1,
		Name:                 name,
		HDRSettings:          DefaultAttributeSettings(),
		VideoCodecSettings:   makeSettingsWithMode(mode, values),
		AudioCodecSettings:   DefaultAttributeSettings(),
		AudioChannelSettings: DefaultAttributeSettings(),
	}
}

func profileWithMultipleAttrs(name string, hdr, video AttributeSettings) *Profile {
	return &Profile{
		ID:                   1,
		Name:                 name,
		HDRSettings:          hdr,
		VideoCodecSettings:   video,
		AudioCodecSettings:   DefaultAttributeSettings(),
		AudioChannelSettings: DefaultAttributeSettings(),
	}
}

// Req 3.1.1: Two profiles are mutually exclusive if their required attributes conflict
func TestCheckMutualExclusivity_HDRConflict(t *testing.T) {
	// Profile A requires HDR (DV, HDR10), Profile B requires SDR
	profileA := profileWithHDR("4K HDR", AttributeModeRequired, []string{"DV", "HDR10", "HDR10+"})
	profileB := profileWithHDR("1080p SDR", AttributeModeRequired, []string{"SDR"})

	result := CheckMutualExclusivity(profileA, profileB)

	if !result.AreExclusive {
		t.Error("Req 3.1.1: Expected profiles with conflicting HDR requirements to be mutually exclusive")
	}
	if len(result.ConflictingAttrs) == 0 || result.ConflictingAttrs[0] != "HDR" {
		t.Errorf("Expected HDR to be listed as conflicting attribute, got %v", result.ConflictingAttrs)
	}
}

// Req 3.1.2: Conflict means Profile A requires X, Profile B disallows X
func TestCheckMutualExclusivity_VideoCodecConflict(t *testing.T) {
	// Profile A requires x265, Profile B requires x264
	profileA := profileWithVideoCodec("HEVC Only", AttributeModeRequired, []string{"x265"})
	profileB := profileWithVideoCodec("AVC Only", AttributeModeRequired, []string{"x264"})

	result := CheckMutualExclusivity(profileA, profileB)

	if !result.AreExclusive {
		t.Error("Req 3.1.2: Expected profiles with non-overlapping required video codecs to be mutually exclusive")
	}
	if len(result.ConflictingAttrs) == 0 || result.ConflictingAttrs[0] != "Video Codec" {
		t.Errorf("Expected Video Codec to be conflicting, got %v", result.ConflictingAttrs)
	}
}

// Test required vs notAllowed conflict
func TestCheckMutualExclusivity_RequiredVsNotAllowed(t *testing.T) {
	// Profile A requires DV, Profile B has DV as notAllowed
	profileA := &Profile{
		ID:                   1,
		Name:                 "DV Required",
		HDRSettings:          makeSettings(map[string]AttributeMode{"DV": AttributeModeRequired}),
		VideoCodecSettings:   DefaultAttributeSettings(),
		AudioCodecSettings:   DefaultAttributeSettings(),
		AudioChannelSettings: DefaultAttributeSettings(),
	}
	profileB := &Profile{
		ID:                   2,
		Name:                 "DV Blocked",
		HDRSettings:          makeSettings(map[string]AttributeMode{"DV": AttributeModeNotAllowed, "SDR": AttributeModeRequired}),
		VideoCodecSettings:   DefaultAttributeSettings(),
		AudioCodecSettings:   DefaultAttributeSettings(),
		AudioChannelSettings: DefaultAttributeSettings(),
	}

	result := CheckMutualExclusivity(profileA, profileB)

	if !result.AreExclusive {
		t.Error("Expected profiles where A requires X and B has X as notAllowed to be mutually exclusive")
	}
}

// Req 3.1.3: Preferred attributes do not affect exclusivity calculation
func TestCheckMutualExclusivity_PreferredDoesNotAffect(t *testing.T) {
	// Both profiles prefer different HDR formats - should NOT be exclusive
	profileA := profileWithHDR("Prefer DV", AttributeModePreferred, []string{"DV"})
	profileB := profileWithHDR("Prefer SDR", AttributeModePreferred, []string{"SDR"})

	result := CheckMutualExclusivity(profileA, profileB)

	if result.AreExclusive {
		t.Error("Req 3.1.3: Profiles with only preferred attributes should NOT be mutually exclusive")
	}
}

func TestCheckMutualExclusivity_AcceptableModeNeverExclusive(t *testing.T) {
	profileA := profileWithHDR("Acceptable HDR", AttributeModeAcceptable, nil)
	profileB := profileWithHDR("Require SDR", AttributeModeRequired, []string{"SDR"})

	result := CheckMutualExclusivity(profileA, profileB)

	if result.AreExclusive {
		t.Error("Profile with 'acceptable' mode should never be mutually exclusive with anything")
	}
}

func TestCheckMutualExclusivity_OverlappingRequired(t *testing.T) {
	// Both require HDR10 - they overlap, not exclusive
	profileA := profileWithHDR("HDR10 OK", AttributeModeRequired, []string{"HDR10", "HDR10+"})
	profileB := profileWithHDR("HDR10 Too", AttributeModeRequired, []string{"HDR10", "DV"})

	result := CheckMutualExclusivity(profileA, profileB)

	if result.AreExclusive {
		t.Error("Profiles with overlapping required values should NOT be mutually exclusive")
	}
	if len(result.OverlappingAttrs) == 0 {
		t.Error("Expected overlapping attributes to be reported")
	}
}

func TestCheckMutualExclusivity_MultipleConflicts(t *testing.T) {
	// Both HDR and video codec conflict
	profileA := profileWithMultipleAttrs("4K HDR HEVC",
		makeSettingsWithMode(AttributeModeRequired, []string{"DV", "HDR10"}),
		makeSettingsWithMode(AttributeModeRequired, []string{"x265"}),
	)
	profileB := profileWithMultipleAttrs("1080p SDR AVC",
		makeSettingsWithMode(AttributeModeRequired, []string{"SDR"}),
		makeSettingsWithMode(AttributeModeRequired, []string{"x264"}),
	)

	result := CheckMutualExclusivity(profileA, profileB)

	if !result.AreExclusive {
		t.Error("Expected profiles with multiple conflicting attributes to be mutually exclusive")
	}
	if len(result.ConflictingAttrs) != 2 {
		t.Errorf("Expected 2 conflicting attributes, got %d: %v", len(result.ConflictingAttrs), result.ConflictingAttrs)
	}
}

func TestCheckMutualExclusivity_EmptyValuesNotConflict(t *testing.T) {
	// Required mode with empty values shouldn't count as conflict
	profileA := profileWithHDR("Empty HDR", AttributeModeRequired, []string{})
	profileB := profileWithHDR("SDR Only", AttributeModeRequired, []string{"SDR"})

	result := CheckMutualExclusivity(profileA, profileB)

	if result.AreExclusive {
		t.Error("Required mode with empty values should not create a conflict")
	}
}

// Req 3.1.4: System prevents saving slot configuration if assigned profiles overlap
func TestValidateSlotExclusivity_AllExclusive(t *testing.T) {
	slots := []SlotConfig{
		{SlotNumber: 1, SlotName: "4K HDR", Enabled: true,
			Profile: profileWithHDR("4K HDR", AttributeModeRequired, []string{"DV", "HDR10"})},
		{SlotNumber: 2, SlotName: "1080p SDR", Enabled: true,
			Profile: profileWithHDR("1080p SDR", AttributeModeRequired, []string{"SDR"})},
	}

	errors, valid := ValidateSlotExclusivity(slots)

	if !valid {
		t.Errorf("Req 3.1.4: Expected exclusive slots to pass validation, got errors: %v", errors)
	}
}

func TestValidateSlotExclusivity_OverlappingProfiles(t *testing.T) {
	slots := []SlotConfig{
		{SlotNumber: 1, SlotName: "Primary", Enabled: true,
			Profile: profileWithHDR("Any HDR", AttributeModePreferred, []string{"HDR10"})},
		{SlotNumber: 2, SlotName: "Secondary", Enabled: true,
			Profile: profileWithHDR("Any HDR Too", AttributeModePreferred, []string{"DV"})},
	}

	errors, valid := ValidateSlotExclusivity(slots)

	if valid {
		t.Error("Req 3.1.4: Expected overlapping profiles to fail validation")
	}
	if len(errors) != 1 {
		t.Errorf("Expected exactly 1 error, got %d", len(errors))
	}
	if errors[0].SlotA != 1 || errors[0].SlotB != 2 {
		t.Errorf("Expected error for slots 1 and 2, got %d and %d", errors[0].SlotA, errors[0].SlotB)
	}
}

func TestValidateSlotExclusivity_DisabledSlotsIgnored(t *testing.T) {
	slots := []SlotConfig{
		{SlotNumber: 1, SlotName: "Primary", Enabled: true,
			Profile: profileWithHDR("Any", AttributeModeAcceptable, nil)},
		{SlotNumber: 2, SlotName: "Secondary", Enabled: false, // Disabled
			Profile: profileWithHDR("Any Too", AttributeModeAcceptable, nil)},
	}

	_, valid := ValidateSlotExclusivity(slots)

	if !valid {
		t.Error("Disabled slots should not affect exclusivity validation")
	}
}

func TestValidateSlotExclusivity_NilProfileIgnored(t *testing.T) {
	slots := []SlotConfig{
		{SlotNumber: 1, SlotName: "Primary", Enabled: true,
			Profile: profileWithHDR("HDR Only", AttributeModeRequired, []string{"HDR10"})},
		{SlotNumber: 2, SlotName: "Secondary", Enabled: true, Profile: nil},
	}

	_, valid := ValidateSlotExclusivity(slots)

	if !valid {
		t.Error("Slots without profiles should not affect exclusivity validation")
	}
}

func TestValidateSlotExclusivity_ThreeSlots(t *testing.T) {
	slots := []SlotConfig{
		{SlotNumber: 1, SlotName: "4K DV", Enabled: true,
			Profile: profileWithHDR("DV Only", AttributeModeRequired, []string{"DV"})},
		{SlotNumber: 2, SlotName: "4K HDR10", Enabled: true,
			Profile: profileWithHDR("HDR10 Only", AttributeModeRequired, []string{"HDR10"})},
		{SlotNumber: 3, SlotName: "1080p SDR", Enabled: true,
			Profile: profileWithHDR("SDR Only", AttributeModeRequired, []string{"SDR"})},
	}

	errors, valid := ValidateSlotExclusivity(slots)

	if !valid {
		t.Errorf("Expected 3 mutually exclusive slots to pass, got errors: %v", errors)
	}
}

func TestValidateSlotExclusivity_MultipleOverlaps(t *testing.T) {
	// All three slots overlap with each other
	slots := []SlotConfig{
		{SlotNumber: 1, SlotName: "A", Enabled: true,
			Profile: profileWithHDR("Any 1", AttributeModeAcceptable, nil)},
		{SlotNumber: 2, SlotName: "B", Enabled: true,
			Profile: profileWithHDR("Any 2", AttributeModeAcceptable, nil)},
		{SlotNumber: 3, SlotName: "C", Enabled: true,
			Profile: profileWithHDR("Any 3", AttributeModeAcceptable, nil)},
	}

	errors, valid := ValidateSlotExclusivity(slots)

	if valid {
		t.Error("Expected all-overlapping slots to fail validation")
	}
	// Should have 3 errors: 1-2, 1-3, 2-3
	if len(errors) != 3 {
		t.Errorf("Expected 3 errors for 3 overlapping pairs, got %d", len(errors))
	}
}

func TestHasOverlap(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []string
		expected bool
	}{
		{"Empty slices", []string{}, []string{}, false},
		{"One empty", []string{"a"}, []string{}, false},
		{"No overlap", []string{"a", "b"}, []string{"c", "d"}, false},
		{"Single overlap", []string{"a", "b"}, []string{"b", "c"}, true},
		{"Full overlap", []string{"a", "b"}, []string{"a", "b"}, true},
		{"Subset", []string{"a", "b", "c"}, []string{"b"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOverlap(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("hasOverlap(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestGetProfileExclusivityHints(t *testing.T) {
	// Two profiles with overlapping HDR settings
	profileA := profileWithHDR("Any HDR", AttributeModeAcceptable, nil)
	profileB := profileWithHDR("Any HDR Too", AttributeModeAcceptable, nil)

	hints := GetProfileExclusivityHints(profileA, profileB)

	if len(hints) == 0 {
		t.Error("Expected hints for overlapping profiles")
	}
}

func TestGetProfileExclusivityHints_AlreadyExclusive(t *testing.T) {
	profileA := profileWithHDR("HDR", AttributeModeRequired, []string{"DV", "HDR10"})
	profileB := profileWithHDR("SDR", AttributeModeRequired, []string{"SDR"})

	hints := GetProfileExclusivityHints(profileA, profileB)

	if len(hints) != 0 {
		t.Errorf("Expected no hints for already exclusive profiles, got: %v", hints)
	}
}

// Spec example: Mutually Exclusive
// Profile A: HDR required, 4K required
// Profile B: SDR required (HDR disallowed), 4K required
// Conflict: HDR status
func TestSpecExample_MutuallyExclusive(t *testing.T) {
	profileA := profileWithHDR("4K HDR", AttributeModeRequired, []string{"DV", "HDR10", "HDR10+", "HDR", "HLG"})
	profileB := profileWithHDR("4K SDR", AttributeModeRequired, []string{"SDR"})

	result := CheckMutualExclusivity(profileA, profileB)

	if !result.AreExclusive {
		t.Error("Spec example: HDR vs SDR required should be mutually exclusive")
	}
}

// Spec example: NOT Mutually Exclusive (should be blocked)
// Profile A: 4K preferred, any HDR acceptable
// Profile B: 4K preferred, any HDR acceptable
// No required attribute conflicts
func TestSpecExample_NotMutuallyExclusive(t *testing.T) {
	profileA := profileWithHDR("4K Preferred", AttributeModePreferred, []string{"HDR10"})
	profileB := profileWithHDR("4K Preferred Too", AttributeModePreferred, []string{"DV"})

	result := CheckMutualExclusivity(profileA, profileB)

	if result.AreExclusive {
		t.Error("Spec example: Profiles with only preferred attributes should NOT be exclusive")
	}
}
