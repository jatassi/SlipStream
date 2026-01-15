package slots

import (
	"testing"

	"github.com/slipstream/slipstream/internal/library/quality"
)

// Helper to create AttributeSettings with per-item modes
func makeAttrSettings(items map[string]quality.AttributeMode) quality.AttributeSettings {
	return quality.AttributeSettings{Items: items}
}

// Helper to create AttributeSettings where all values have the same mode
func makeAttrSettingsWithMode(mode quality.AttributeMode, values []string) quality.AttributeSettings {
	items := make(map[string]quality.AttributeMode)
	for _, v := range values {
		items[v] = mode
	}
	return quality.AttributeSettings{Items: items}
}

func TestGetConflictingAttributes(t *testing.T) {
	tests := []struct {
		name     string
		slots    []quality.SlotConfig
		expected []DifferentiatorAttribute
	}{
		{
			name: "Req 4.1.3: Slots differ only on HDR",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					SlotName:   "4K HDR",
					Enabled:    true,
					Profile: &quality.Profile{
						Name:        "4K HDR",
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV", "HDR10"}),
					},
				},
				{
					SlotNumber: 2,
					SlotName:   "4K SDR",
					Enabled:    true,
					Profile: &quality.Profile{
						Name:        "4K SDR",
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"SDR"}),
					},
				},
			},
			expected: []DifferentiatorAttribute{DifferentiatorHDR},
		},
		{
			name: "Req 4.1.2: Slots differ on video codec",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						VideoCodecSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"x265"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						VideoCodecSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"x264"}),
					},
				},
			},
			expected: []DifferentiatorAttribute{DifferentiatorVideoCodec},
		},
		{
			name: "Multiple conflicting attributes",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings:        makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV"}),
						AudioCodecSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"TrueHD"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings:        makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"SDR"}),
						AudioCodecSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"AAC"}),
					},
				},
			},
			expected: []DifferentiatorAttribute{DifferentiatorHDR, DifferentiatorAudioCodec},
		},
		{
			name: "No conflicts - preferred mode doesn't count",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModePreferred, []string{"DV"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModePreferred, []string{"SDR"}),
					},
				},
			},
			expected: []DifferentiatorAttribute{},
		},
		{
			name: "No conflicts - overlapping required values",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV", "HDR10"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"HDR10", "HDR10+"}),
					},
				},
			},
			expected: []DifferentiatorAttribute{}, // HDR10 overlaps
		},
		{
			name: "Disabled slot ignored",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    false, // Disabled
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"SDR"}),
					},
				},
			},
			expected: []DifferentiatorAttribute{},
		},
		{
			name: "Slot without profile ignored",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile:    nil, // No profile
				},
			},
			expected: []DifferentiatorAttribute{},
		},
		{
			name: "Three slots with HDR and codec conflicts",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings:        makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV"}),
						VideoCodecSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"x265"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings:        makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"HDR10"}),
						VideoCodecSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"x265"}),
					},
				},
				{
					SlotNumber: 3,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings:        makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"SDR"}),
						VideoCodecSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"x264"}),
					},
				},
			},
			expected: []DifferentiatorAttribute{DifferentiatorHDR, DifferentiatorVideoCodec},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetConflictingAttributes(tt.slots)

			if len(result) != len(tt.expected) {
				t.Errorf("GetConflictingAttributes() returned %d attributes, want %d", len(result), len(tt.expected))
				t.Errorf("Got: %v, Want: %v", result, tt.expected)
				return
			}

			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetConflictingAttributes() missing expected attribute %s", expected)
				}
			}
		})
	}
}

func TestValidateFilenameFormat(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		requiredAttrs []DifferentiatorAttribute
		expectValid   bool
		missingCount  int
	}{
		{
			name:          "Req 4.1.1: Format includes required HDR token",
			format:        "{Movie Title} ({Year}) - {Quality Title} {MediaInfo VideoDynamicRangeType}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorHDR},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "Req 4.1.5: Format missing required HDR token",
			format:        "{Movie Title} ({Year}) - {Quality Title}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorHDR},
			expectValid:   false,
			missingCount:  1,
		},
		{
			name:          "Format includes video codec token",
			format:        "{Movie Title} ({Year}) {MediaInfo VideoCodec}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorVideoCodec},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "MediaInfo Simple covers video and audio codec",
			format:        "{Movie Title} ({Year}) {MediaInfo Simple}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorVideoCodec, DifferentiatorAudioCodec},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "MediaInfo Full covers video and audio codec",
			format:        "{Movie Title} ({Year}) {MediaInfo Full}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorVideoCodec, DifferentiatorAudioCodec},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "Format with multiple required tokens - all present",
			format:        "{Movie Title} ({Year}) {MediaInfo VideoDynamicRangeType} {MediaInfo VideoCodec} {MediaInfo AudioCodec}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorHDR, DifferentiatorVideoCodec, DifferentiatorAudioCodec},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "Format with multiple required tokens - one missing",
			format:        "{Movie Title} ({Year}) {MediaInfo VideoDynamicRangeType} {MediaInfo VideoCodec}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorHDR, DifferentiatorVideoCodec, DifferentiatorAudioCodec},
			expectValid:   false,
			missingCount:  1,
		},
		{
			name:          "Format missing audio channels token",
			format:        "{Movie Title} ({Year}) - {Quality Title}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorAudioChannels},
			expectValid:   false,
			missingCount:  1,
		},
		{
			name:          "Format includes audio channels token",
			format:        "{Movie Title} ({Year}) {MediaInfo AudioChannels}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorAudioChannels},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "No required attributes - always valid",
			format:        "{Movie Title} ({Year})",
			requiredAttrs: []DifferentiatorAttribute{},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "Format with separator variations",
			format:        "{Movie.Title} ({Year}) - {MediaInfo.VideoDynamicRangeType}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorHDR},
			expectValid:   true,
			missingCount:  0,
		},
		{
			name:          "Alternative HDR token - VideoDynamicRange",
			format:        "{Movie Title} ({Year}) {MediaInfo VideoDynamicRange}",
			requiredAttrs: []DifferentiatorAttribute{DifferentiatorHDR},
			expectValid:   true,
			missingCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFilenameFormat(tt.format, tt.requiredAttrs)

			if result.Valid != tt.expectValid {
				t.Errorf("ValidateFilenameFormat() valid = %v, want %v", result.Valid, tt.expectValid)
			}

			if len(result.MissingTokens) != tt.missingCount {
				t.Errorf("ValidateFilenameFormat() missing tokens = %d, want %d", len(result.MissingTokens), tt.missingCount)
				for _, mt := range result.MissingTokens {
					t.Logf("  Missing: %s - %s", mt.Attribute, mt.Description)
				}
			}
		})
	}
}

func TestValidateSlotNaming(t *testing.T) {
	tests := []struct {
		name          string
		slots         []quality.SlotConfig
		movieFormat   string
		episodeFormat string
		expectValid   bool
	}{
		{
			name: "Req 4.1.4: Valid configuration with HDR differentiator",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV", "HDR10"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"SDR"}),
					},
				},
			},
			movieFormat:   "{Movie Title} ({Year}) - {Quality Title} {MediaInfo VideoDynamicRangeType}",
			episodeFormat: "{Series Title} - S{season:00}E{episode:00} {MediaInfo VideoDynamicRangeType}",
			expectValid:   true,
		},
		{
			name: "Invalid: movie format missing HDR token",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"SDR"}),
					},
				},
			},
			movieFormat:   "{Movie Title} ({Year}) - {Quality Title}", // Missing HDR token
			episodeFormat: "{Series Title} - S{season:00}E{episode:00} {MediaInfo VideoDynamicRangeType}",
			expectValid:   false,
		},
		{
			name: "Invalid: episode format missing HDR token",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"SDR"}),
					},
				},
			},
			movieFormat:   "{Movie Title} ({Year}) {MediaInfo VideoDynamicRangeType}",
			episodeFormat: "{Series Title} - S{season:00}E{episode:00}", // Missing HDR token
			expectValid:   false,
		},
		{
			name: "Valid: single slot enabled - no differentiators needed",
			slots: []quality.SlotConfig{
				{
					SlotNumber: 1,
					Enabled:    true,
					Profile: &quality.Profile{
						HDRSettings: makeAttrSettingsWithMode(quality.AttributeModeRequired, []string{"DV"}),
					},
				},
				{
					SlotNumber: 2,
					Enabled:    false, // Disabled
					Profile:    nil,
				},
			},
			movieFormat:   "{Movie Title} ({Year})",
			episodeFormat: "{Series Title} - S{season:00}E{episode:00}",
			expectValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSlotNaming(tt.slots, tt.movieFormat, tt.episodeFormat)

			if result.CanProceed != tt.expectValid {
				t.Errorf("ValidateSlotNaming() canProceed = %v, want %v", result.CanProceed, tt.expectValid)
				if !result.MovieFormatValid {
					t.Logf("  Movie format invalid: %v", result.MovieValidation.MissingTokens)
				}
				if !result.EpisodeFormatValid {
					t.Logf("  Episode format invalid: %v", result.EpisodeValidation.MissingTokens)
				}
			}
		})
	}
}

func TestBuildNamingValidationWarnings(t *testing.T) {
	// Create a validation result with missing tokens
	validation := SlotNamingValidation{
		MovieFormatValid:   false,
		EpisodeFormatValid: true,
		MovieValidation: NamingValidationResult{
			Valid: false,
			MissingTokens: []MissingTokenInfo{
				{
					Attribute:      DifferentiatorHDR,
					TokenName:      "HDR Format",
					Description:    "Include {MediaInfo VideoDynamicRangeType} to differentiate HDR/SDR files",
					SuggestedToken: "{MediaInfo VideoDynamicRangeType}",
				},
			},
		},
		EpisodeValidation: NamingValidationResult{Valid: true},
	}

	warnings := BuildNamingValidationWarnings(validation)

	if len(warnings) != 1 {
		t.Errorf("BuildNamingValidationWarnings() returned %d warnings, want 1", len(warnings))
	}

	if len(warnings) > 0 && !contains(warnings[0], "Movie filename format") {
		t.Errorf("Warning should mention 'Movie filename format', got: %s", warnings[0])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
