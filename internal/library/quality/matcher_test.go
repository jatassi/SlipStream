package quality

import (
	"testing"
)

// Helper to create AttributeSettings with per-item modes
func makeAttrSettings(items map[string]AttributeMode) AttributeSettings {
	return AttributeSettings{Items: items}
}

// Helper to create AttributeSettings where all values have the same mode
func makeAttrSettingsWithMode(mode AttributeMode, values []string) AttributeSettings {
	items := make(map[string]AttributeMode)
	for _, v := range values {
		items[v] = mode
	}
	return AttributeSettings{Items: items}
}

func TestMatchAttribute(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		settings    AttributeSettings
		wantMatches bool
		wantScore   float64
	}{
		// Default (no settings)
		{
			name:        "default settings always pass",
			value:       "x265",
			settings:    DefaultAttributeSettings(),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "default settings pass for unknown",
			value:       "unknown",
			settings:    DefaultAttributeSettings(),
			wantMatches: true,
			wantScore:   0,
		},

		// Mode: required - Req 2.5.1
		{
			name:        "required mode passes when value matches",
			value:       "x265",
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"x264", "x265"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "required mode fails when value doesn't match",
			value:       "xvid",
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"x264", "x265"}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:        "required mode fails for empty value (Req 2.5.1)",
			value:       "",
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"x264", "x265"}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:        "required mode fails for unknown value (Req 2.5.1)",
			value:       "unknown",
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"x264", "x265"}),
			wantMatches: false,
			wantScore:   0,
		},

		// Mode: preferred - Req 2.5.2
		{
			name:        "preferred mode passes with bonus when matched",
			value:       "x265",
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"x264", "x265"}),
			wantMatches: true,
			wantScore:   1.0,
		},
		{
			name:        "preferred mode passes without bonus when not matched",
			value:       "xvid",
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"x264", "x265"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "preferred mode passes without bonus for unknown (Req 2.5.2)",
			value:       "unknown",
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"x264", "x265"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "preferred mode passes without bonus for empty",
			value:       "",
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"x264", "x265"}),
			wantMatches: true,
			wantScore:   0,
		},

		// Mode: notAllowed
		{
			name:        "notAllowed mode rejects matching value",
			value:       "xvid",
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"xvid"}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:        "notAllowed mode passes non-matching value",
			value:       "x265",
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"xvid"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "notAllowed mode passes empty value",
			value:       "",
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"xvid"}),
			wantMatches: true,
			wantScore:   0,
		},

		// Mixed modes per item
		{
			name:  "mixed modes: value matches required",
			value: "x265",
			settings: makeAttrSettings(map[string]AttributeMode{
				"x265": AttributeModeRequired,
				"xvid": AttributeModeNotAllowed,
			}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:  "mixed modes: value matches notAllowed",
			value: "xvid",
			settings: makeAttrSettings(map[string]AttributeMode{
				"x265": AttributeModeRequired,
				"xvid": AttributeModeNotAllowed,
			}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:  "mixed modes: preferred only (no required)",
			value: "x264",
			settings: makeAttrSettings(map[string]AttributeMode{
				"x264": AttributeModePreferred,
				"xvid": AttributeModeNotAllowed,
			}),
			wantMatches: true,
			wantScore:   1.0,
		},
		{
			name:  "mixed modes: preferred value when required not met",
			value: "x264",
			settings: makeAttrSettings(map[string]AttributeMode{
				"x265": AttributeModeRequired,
				"x264": AttributeModePreferred,
			}),
			wantMatches: false,
			wantScore:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchAttribute(tt.value, tt.settings)
			if result.Matches != tt.wantMatches {
				t.Errorf("MatchAttribute() Matches = %v, want %v", result.Matches, tt.wantMatches)
			}
			if result.Score != tt.wantScore {
				t.Errorf("MatchAttribute() Score = %v, want %v", result.Score, tt.wantScore)
			}
		})
	}
}

func TestMatchHDRAttribute(t *testing.T) {
	tests := []struct {
		name        string
		formats     []string
		settings    AttributeSettings
		wantMatches bool
		wantScore   float64
	}{
		// Default (no settings)
		{
			name:        "default settings always pass",
			formats:     []string{"DV", "HDR10"},
			settings:    DefaultAttributeSettings(),
			wantMatches: true,
			wantScore:   0,
		},

		// Mode: required - Req 2.3.2
		{
			name:        "required mode passes when any format matches (Req 2.3.2)",
			formats:     []string{"DV", "HDR10"},
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"HDR10"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "required mode fails when no format matches",
			formats:     []string{"DV", "HDR10"},
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"HDR10+", "HLG"}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:        "required mode fails for empty formats",
			formats:     []string{},
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"HDR10"}),
			wantMatches: false,
			wantScore:   0,
		},

		// Mode: preferred
		{
			name:        "preferred mode passes with bonus when any format matches",
			formats:     []string{"DV", "HDR10"},
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"DV"}),
			wantMatches: true,
			wantScore:   1.0,
		},
		{
			name:        "preferred mode passes without bonus when no format matches",
			formats:     []string{"SDR"},
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"DV", "HDR10"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "preferred mode passes without bonus for empty formats",
			formats:     []string{},
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"DV"}),
			wantMatches: true,
			wantScore:   0,
		},

		// Mode: notAllowed
		{
			name:        "notAllowed mode rejects when any format is blocked",
			formats:     []string{"DV", "HDR10"},
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"DV"}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:        "notAllowed mode passes when no format is blocked",
			formats:     []string{"HDR10", "HLG"},
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"DV"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "notAllowed mode passes for empty formats",
			formats:     []string{},
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"DV"}),
			wantMatches: true,
			wantScore:   0,
		},

		// Mixed modes
		{
			name:    "mixed modes: required satisfied, notAllowed avoided",
			formats: []string{"HDR10"},
			settings: makeAttrSettings(map[string]AttributeMode{
				"HDR10": AttributeModeRequired,
				"DV":    AttributeModeNotAllowed,
			}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:    "mixed modes: required satisfied but notAllowed present",
			formats: []string{"HDR10", "DV"},
			settings: makeAttrSettings(map[string]AttributeMode{
				"HDR10": AttributeModeRequired,
				"DV":    AttributeModeNotAllowed,
			}),
			wantMatches: false,
			wantScore:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchHDRAttribute(tt.formats, tt.settings)
			if result.Matches != tt.wantMatches {
				t.Errorf("MatchHDRAttribute() Matches = %v, want %v", result.Matches, tt.wantMatches)
			}
			if result.Score != tt.wantScore {
				t.Errorf("MatchHDRAttribute() Score = %v, want %v", result.Score, tt.wantScore)
			}
		})
	}
}

func TestMatchAudioAttribute(t *testing.T) {
	tests := []struct {
		name        string
		tracks      []string
		settings    AttributeSettings
		wantMatches bool
		wantScore   float64
	}{
		// Default (no settings)
		{
			name:        "default settings always pass",
			tracks:      []string{"TrueHD", "AAC"},
			settings:    DefaultAttributeSettings(),
			wantMatches: true,
			wantScore:   0,
		},

		// Mode: required - Req 2.4.1
		{
			name:        "required mode passes when any track matches (Req 2.4.1)",
			tracks:      []string{"TrueHD", "AAC"},
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"TrueHD", "DTS-HD MA"}),
			wantMatches: true,
			wantScore:   0,
		},
		{
			name:        "required mode fails when no track matches",
			tracks:      []string{"AAC", "MP3"},
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"TrueHD", "DTS-HD MA"}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:        "required mode fails for empty tracks",
			tracks:      []string{},
			settings:    makeAttrSettingsWithMode(AttributeModeRequired, []string{"TrueHD"}),
			wantMatches: false,
			wantScore:   0,
		},

		// Mode: preferred
		{
			name:        "preferred mode passes with bonus when any track matches",
			tracks:      []string{"TrueHD", "AAC"},
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"TrueHD"}),
			wantMatches: true,
			wantScore:   1.0,
		},
		{
			name:        "preferred mode passes without bonus when no track matches",
			tracks:      []string{"AAC"},
			settings:    makeAttrSettingsWithMode(AttributeModePreferred, []string{"TrueHD", "DTS-HD MA"}),
			wantMatches: true,
			wantScore:   0,
		},

		// Mode: notAllowed
		{
			name:        "notAllowed mode rejects when any track is blocked",
			tracks:      []string{"TrueHD", "AAC"},
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"AAC"}),
			wantMatches: false,
			wantScore:   0,
		},
		{
			name:        "notAllowed mode passes when no track is blocked",
			tracks:      []string{"TrueHD", "DTS"},
			settings:    makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"AAC", "MP3"}),
			wantMatches: true,
			wantScore:   0,
		},

		// Mixed modes
		{
			name:   "mixed modes: required satisfied, notAllowed avoided",
			tracks: []string{"TrueHD"},
			settings: makeAttrSettings(map[string]AttributeMode{
				"TrueHD": AttributeModeRequired,
				"AAC":    AttributeModeNotAllowed,
			}),
			wantMatches: true,
			wantScore:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchAudioAttribute(tt.tracks, tt.settings)
			if result.Matches != tt.wantMatches {
				t.Errorf("MatchAudioAttribute() Matches = %v, want %v", result.Matches, tt.wantMatches)
			}
			if result.Score != tt.wantScore {
				t.Errorf("MatchAudioAttribute() Score = %v, want %v", result.Score, tt.wantScore)
			}
		})
	}
}

func TestMatchProfileAttributes(t *testing.T) {
	tests := []struct {
		name           string
		release        ReleaseAttributes
		profile        *Profile
		wantAllMatch   bool
		wantTotalScore float64
	}{
		{
			name: "all attributes pass with default settings",
			release: ReleaseAttributes{
				HDRFormats:    []string{"DV"},
				VideoCodec:    "x265",
				AudioCodecs:   []string{"TrueHD"},
				AudioChannels: []string{"7.1"},
			},
			profile: &Profile{
				HDRSettings:          DefaultAttributeSettings(),
				VideoCodecSettings:   DefaultAttributeSettings(),
				AudioCodecSettings:   DefaultAttributeSettings(),
				AudioChannelSettings: DefaultAttributeSettings(),
			},
			wantAllMatch:   true,
			wantTotalScore: 0,
		},
		{
			name: "all attributes pass with preferred matches",
			release: ReleaseAttributes{
				HDRFormats:    []string{"DV", "HDR10"},
				VideoCodec:    "x265",
				AudioCodecs:   []string{"TrueHD", "AAC"},
				AudioChannels: []string{"7.1", "2.0"},
			},
			profile: &Profile{
				HDRSettings:          makeAttrSettingsWithMode(AttributeModePreferred, []string{"DV"}),
				VideoCodecSettings:   makeAttrSettingsWithMode(AttributeModePreferred, []string{"x265"}),
				AudioCodecSettings:   makeAttrSettingsWithMode(AttributeModePreferred, []string{"TrueHD"}),
				AudioChannelSettings: makeAttrSettingsWithMode(AttributeModePreferred, []string{"7.1"}),
			},
			wantAllMatch:   true,
			wantTotalScore: 4.0,
		},
		{
			name: "one required attribute fails",
			release: ReleaseAttributes{
				HDRFormats:    []string{"SDR"},
				VideoCodec:    "x265",
				AudioCodecs:   []string{"TrueHD"},
				AudioChannels: []string{"7.1"},
			},
			profile: &Profile{
				HDRSettings:          makeAttrSettingsWithMode(AttributeModeRequired, []string{"DV", "HDR10"}),
				VideoCodecSettings:   DefaultAttributeSettings(),
				AudioCodecSettings:   DefaultAttributeSettings(),
				AudioChannelSettings: DefaultAttributeSettings(),
			},
			wantAllMatch:   false,
			wantTotalScore: 0,
		},
		{
			name: "mixed modes with partial preferred matches",
			release: ReleaseAttributes{
				HDRFormats:    []string{"HDR10"},
				VideoCodec:    "x264",
				AudioCodecs:   []string{"DTS-HD MA"},
				AudioChannels: []string{"5.1"},
			},
			profile: &Profile{
				HDRSettings:          makeAttrSettingsWithMode(AttributeModeRequired, []string{"HDR10", "DV"}),
				VideoCodecSettings:   makeAttrSettingsWithMode(AttributeModePreferred, []string{"x265"}),
				AudioCodecSettings:   makeAttrSettingsWithMode(AttributeModePreferred, []string{"DTS-HD MA"}),
				AudioChannelSettings: makeAttrSettingsWithMode(AttributeModePreferred, []string{"7.1"}),
			},
			wantAllMatch:   true,
			wantTotalScore: 1.0,
		},
		{
			name: "notAllowed attribute blocks release",
			release: ReleaseAttributes{
				HDRFormats:    []string{"DV", "HDR10"},
				VideoCodec:    "x265",
				AudioCodecs:   []string{"TrueHD"},
				AudioChannels: []string{"7.1"},
			},
			profile: &Profile{
				HDRSettings:          makeAttrSettingsWithMode(AttributeModeNotAllowed, []string{"DV"}),
				VideoCodecSettings:   DefaultAttributeSettings(),
				AudioCodecSettings:   DefaultAttributeSettings(),
				AudioChannelSettings: DefaultAttributeSettings(),
			},
			wantAllMatch:   false,
			wantTotalScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchProfileAttributes(tt.release, tt.profile)
			if result.AllMatch != tt.wantAllMatch {
				t.Errorf("MatchProfileAttributes() AllMatch = %v, want %v", result.AllMatch, tt.wantAllMatch)
			}
			if result.TotalScore != tt.wantTotalScore {
				t.Errorf("MatchProfileAttributes() TotalScore = %v, want %v", result.TotalScore, tt.wantTotalScore)
			}
		})
	}
}

func TestParseHDRFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single format",
			input: "HDR10",
			want:  []string{"HDR10"},
		},
		{
			name:  "combo DV HDR10 (Req 2.3.2)",
			input: "DV HDR10",
			want:  []string{"DV", "HDR10"},
		},
		{
			name:  "combo with multiple spaces",
			input: "DV  HDR10  HLG",
			want:  []string{"DV", "HDR10", "HLG"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "input containing HDR detected as generic HDR",
			input: "CustomHDR",
			want:  []string{"HDR"},
		},
		{
			name:  "unrecognized format returns SDR",
			input: "Unknown",
			want:  []string{"SDR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHDRFormats(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseHDRFormats() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseHDRFormats()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestProfile_IsAttributeMatch(t *testing.T) {
	profile := &Profile{
		HDRSettings:          makeAttrSettingsWithMode(AttributeModeRequired, []string{"DV", "HDR10"}),
		VideoCodecSettings:   DefaultAttributeSettings(),
		AudioCodecSettings:   makeAttrSettingsWithMode(AttributeModePreferred, []string{"TrueHD"}),
		AudioChannelSettings: DefaultAttributeSettings(),
	}

	tests := []struct {
		name    string
		release ReleaseAttributes
		want    bool
	}{
		{
			name: "matches required HDR",
			release: ReleaseAttributes{
				HDRFormats:    []string{"DV"},
				VideoCodec:    "x264",
				AudioCodecs:   []string{"AAC"},
				AudioChannels: []string{"2.0"},
			},
			want: true,
		},
		{
			name: "fails required HDR",
			release: ReleaseAttributes{
				HDRFormats:    []string{"HLG"},
				VideoCodec:    "x265",
				AudioCodecs:   []string{"TrueHD"},
				AudioChannels: []string{"7.1"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := profile.IsAttributeMatch(tt.release); got != tt.want {
				t.Errorf("Profile.IsAttributeMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfile_GetAttributeScore(t *testing.T) {
	profile := &Profile{
		HDRSettings:          makeAttrSettingsWithMode(AttributeModePreferred, []string{"DV"}),
		VideoCodecSettings:   makeAttrSettingsWithMode(AttributeModePreferred, []string{"x265"}),
		AudioCodecSettings:   DefaultAttributeSettings(),
		AudioChannelSettings: makeAttrSettingsWithMode(AttributeModePreferred, []string{"7.1"}),
	}

	tests := []struct {
		name    string
		release ReleaseAttributes
		want    float64
	}{
		{
			name: "all preferred match",
			release: ReleaseAttributes{
				HDRFormats:    []string{"DV"},
				VideoCodec:    "x265",
				AudioCodecs:   []string{"TrueHD"},
				AudioChannels: []string{"7.1"},
			},
			want: 3.0,
		},
		{
			name: "some preferred match",
			release: ReleaseAttributes{
				HDRFormats:    []string{"HDR10"},
				VideoCodec:    "x265",
				AudioCodecs:   []string{"AAC"},
				AudioChannels: []string{"5.1"},
			},
			want: 1.0,
		},
		{
			name: "no preferred match",
			release: ReleaseAttributes{
				HDRFormats:    []string{"SDR"},
				VideoCodec:    "x264",
				AudioCodecs:   []string{"AAC"},
				AudioChannels: []string{"2.0"},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := profile.GetAttributeScore(tt.release); got != tt.want {
				t.Errorf("Profile.GetAttributeScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
