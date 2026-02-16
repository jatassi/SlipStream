package renamer

import (
	"testing"
	"time"
)

// fullContext returns a TokenContext with all fields populated for comprehensive testing.
func fullContext() *TokenContext {
	return &TokenContext{
		SeriesTitle:       "Breaking Bad",
		SeriesYear:        2008,
		SeriesType:        "standard",
		SeasonNumber:      2,
		EpisodeNumber:     5,
		EpisodeNumbers:    nil,
		AbsoluteNumber:    18,
		EpisodeTitle:      "Breakage",
		AirDate:           time.Date(2009, 4, 12, 0, 0, 0, 0, time.UTC),
		Quality:           "1080p",
		Source:            "BluRay",
		Codec:             "x265",
		Revision:          "Proper",
		VideoCodec:        "HEVC",
		VideoBitDepth:     10,
		VideoDynamicRange: "HDR10",
		AudioCodec:        "TrueHD",
		AudioChannels:     "7.1",
		AudioLanguages:    []string{"EN", "DE", "FR"},
		SubtitleLanguages: []string{"EN", "ES"},
		ReleaseGroup:      "SPARKS",
		EditionTags:       "Director's Cut",
		OriginalTitle:     "Breaking.Bad.S02E05",
		OriginalFile:      "breaking.bad.s02e05.mkv",
		CustomFormats:     []string{"Remux", "HDR"},
		ReleaseVersion:    2,
		MovieTitle:        "The Matrix",
		MovieYear:         1999,
	}
}

// ===== SERIES TOKENS =====

func TestToken_SeriesTitle(t *testing.T) {
	ctx := fullContext()

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"basic", "{Series Title}", "Breaking Bad"},
		{"with separator dot", "{.Series Title}", "Breaking.Bad"},        // Separator prefix at start
		{"with separator dash", "{-Series Title}", "Breaking-Bad"},       // Separator prefix at start
		{"with separator underscore", "{_Series Title}", "Breaking_Bad"}, // Separator prefix at start
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			if len(tokens) != 1 {
				t.Fatalf("expected 1 token, got %d", len(tokens))
			}
			got := tokens[0].Resolve(ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_SeriesTitleYear(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "with year",
			ctx:     fullContext(),
			pattern: "{Series TitleYear}",
			want:    "Breaking Bad (2008)",
		},
		{
			name:    "without year",
			ctx:     &TokenContext{SeriesTitle: "Test Show"},
			pattern: "{Series TitleYear}",
			want:    "Test Show",
		},
		{
			name:    "with separator",
			ctx:     fullContext(),
			pattern: "{.Series TitleYear}",
			want:    "Breaking.Bad.(2008)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_SeriesCleanTitle(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "removes illegal chars",
			ctx:     &TokenContext{SeriesTitle: "What If...?"},
			pattern: "{Series CleanTitle}",
			want:    "What If...",
		},
		{
			name:    "removes colons",
			ctx:     &TokenContext{SeriesTitle: "Star Trek: Discovery"},
			pattern: "{Series CleanTitle}",
			want:    "Star Trek Discovery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_SeriesCleanTitleYear(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "with year",
			ctx:     &TokenContext{SeriesTitle: "Star Trek: Discovery", SeriesYear: 2017},
			pattern: "{Series CleanTitleYear}",
			want:    "Star Trek Discovery 2017",
		},
		{
			name:    "without year",
			ctx:     &TokenContext{SeriesTitle: "Test Show"},
			pattern: "{Series CleanTitleYear}",
			want:    "Test Show",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== SEASON TOKENS =====

func TestToken_Season(t *testing.T) {
	ctx := &TokenContext{SeasonNumber: 5}

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"no padding", "{season}", "5"},
		{"no padding explicit", "{season:0}", "5"},
		{"2-digit padding", "{season:00}", "05"},
		{"3-digit padding", "{season:000}", "005"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_SeasonLargeNumber(t *testing.T) {
	ctx := &TokenContext{SeasonNumber: 12}

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"no padding", "{season}", "12"},
		{"2-digit padding", "{season:00}", "12"},
		{"3-digit padding", "{season:000}", "012"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== EPISODE TOKENS =====

func TestToken_Episode(t *testing.T) {
	ctx := &TokenContext{EpisodeNumber: 7}

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"no padding", "{episode}", "7"},
		{"no padding explicit", "{episode:0}", "7"},
		{"2-digit padding", "{episode:00}", "07"},
		{"3-digit padding", "{episode:000}", "007"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_EpisodeMulti(t *testing.T) {
	ctx := &TokenContext{
		EpisodeNumber:  1,
		EpisodeNumbers: []int{1, 2, 3},
	}

	tokens := ParseTokens("{episode:00}")
	got := tokens[0].Resolve(ctx)
	// Multi-episode returns first episode number (rest handled by multi-episode formatter)
	if got != "01" {
		t.Errorf("Resolve() = %q, want %q", got, "01")
	}
}

func TestToken_EpisodeTitle(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "basic",
			ctx:     &TokenContext{EpisodeTitle: "The Pilot"},
			pattern: "{Episode Title}",
			want:    "The Pilot",
		},
		{
			name:    "with separator",
			ctx:     &TokenContext{EpisodeTitle: "The Pilot Episode"},
			pattern: "{.Episode Title}", // Separator prefix at start
			want:    "The.Pilot.Episode",
		},
		{
			name:    "truncate from end",
			ctx:     &TokenContext{EpisodeTitle: "This Is A Very Long Episode Title"},
			pattern: "{Episode Title:20}",
			want:    "This Is A Very Long…",
		},
		{
			name:    "truncate from start",
			ctx:     &TokenContext{EpisodeTitle: "This Is A Very Long Episode Title"},
			pattern: "{Episode Title:-20}",
			want:    "… Long Episode Title", // Includes space at position 14
		},
		{
			name:    "short title no truncation",
			ctx:     &TokenContext{EpisodeTitle: "Pilot"},
			pattern: "{Episode Title:20}",
			want:    "Pilot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_EpisodeTitleDailyFallback(t *testing.T) {
	ctx := &TokenContext{
		SeriesType:   "daily",
		EpisodeTitle: "",
		AirDate:      time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
	}

	tokens := ParseTokens("{Episode Title}")
	got := tokens[0].Resolve(ctx)
	want := "March 15, 2024"
	if got != want {
		t.Errorf("Resolve() = %q, want %q", got, want)
	}
}

func TestToken_EpisodeCleanTitle(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "removes illegal chars",
			ctx:     &TokenContext{EpisodeTitle: "What's Next?"},
			pattern: "{Episode CleanTitle}",
			want:    "What's Next",
		},
		{
			name:    "with truncation",
			ctx:     &TokenContext{EpisodeTitle: "A Very Long Title: The Beginning"},
			pattern: "{Episode CleanTitle:15}",
			want:    "A Very Long Ti…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== AIR DATE TOKENS =====

func TestToken_AirDate(t *testing.T) {
	ctx := &TokenContext{
		AirDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"with dashes", "{Air-Date}", "2024-03-15"},
		{"with spaces", "{Air Date}", "2024 03 15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_AirDateEmpty(t *testing.T) {
	ctx := &TokenContext{}

	tests := []struct {
		name    string
		pattern string
	}{
		{"with dashes", "{Air-Date}"},
		{"with spaces", "{Air Date}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(ctx)
			if got != "" {
				t.Errorf("Resolve() = %q, want empty string", got)
			}
		})
	}
}

// ===== QUALITY TOKENS =====

func TestToken_QualityFull(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "source quality revision",
			ctx:     &TokenContext{Source: "BluRay", Quality: "1080p", Revision: "Proper"},
			pattern: "{Quality Full}",
			want:    "BluRay-1080p Proper",
		},
		{
			name:    "source quality no revision",
			ctx:     &TokenContext{Source: "WEBDL", Quality: "720p"},
			pattern: "{Quality Full}",
			want:    "WEBDL-720p",
		},
		{
			name:    "source only",
			ctx:     &TokenContext{Source: "BluRay"},
			pattern: "{Quality Full}",
			want:    "BluRay",
		},
		{
			name:    "quality only",
			ctx:     &TokenContext{Quality: "1080p"},
			pattern: "{Quality Full}",
			want:    "1080p",
		},
		{
			name:    "quality with revision",
			ctx:     &TokenContext{Quality: "1080p", Revision: "REPACK"},
			pattern: "{Quality Full}",
			want:    "1080p REPACK",
		},
		{
			name:    "all empty",
			ctx:     &TokenContext{},
			pattern: "{Quality Full}",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_QualityTitle(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "source and quality",
			ctx:     &TokenContext{Source: "BluRay", Quality: "1080p", Revision: "Proper"},
			pattern: "{Quality Title}",
			want:    "BluRay-1080p",
		},
		{
			name:    "source only",
			ctx:     &TokenContext{Source: "WEBDL"},
			pattern: "{Quality Title}",
			want:    "WEBDL",
		},
		{
			name:    "quality only",
			ctx:     &TokenContext{Quality: "720p"},
			pattern: "{Quality Title}",
			want:    "720p",
		},
		{
			name:    "empty",
			ctx:     &TokenContext{},
			pattern: "{Quality Title}",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== MEDIAINFO TOKENS =====

func TestToken_MediaInfoSimple(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "video and audio",
			ctx:     &TokenContext{VideoCodec: "x265", AudioCodec: "DTS"},
			pattern: "{MediaInfo Simple}",
			want:    "x265 DTS",
		},
		{
			name:    "video only",
			ctx:     &TokenContext{VideoCodec: "x264"},
			pattern: "{MediaInfo Simple}",
			want:    "x264",
		},
		{
			name:    "audio only",
			ctx:     &TokenContext{AudioCodec: "AAC"},
			pattern: "{MediaInfo Simple}",
			want:    "AAC",
		},
		{
			name:    "empty",
			ctx:     &TokenContext{},
			pattern: "{MediaInfo Simple}",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoFull(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "all info",
			ctx:     &TokenContext{VideoCodec: "x265", AudioCodec: "DTS", AudioLanguages: []string{"EN", "DE"}},
			pattern: "{MediaInfo Full}",
			want:    "x265 DTS [EN+DE]",
		},
		{
			name:    "no languages",
			ctx:     &TokenContext{VideoCodec: "x265", AudioCodec: "DTS"},
			pattern: "{MediaInfo Full}",
			want:    "x265 DTS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoVideoCodec(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"has value", &TokenContext{VideoCodec: "HEVC"}, "HEVC"},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{MediaInfo VideoCodec}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoVideoBitDepth(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"10-bit", &TokenContext{VideoBitDepth: 10}, "10"},
		{"8-bit", &TokenContext{VideoBitDepth: 8}, "8"},
		{"zero", &TokenContext{VideoBitDepth: 0}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{MediaInfo VideoBitDepth}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoVideoDynamicRange(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"HDR10", &TokenContext{VideoDynamicRange: "HDR10"}, "HDR"},
		{"DV", &TokenContext{VideoDynamicRange: "DV"}, "HDR"},
		{"SDR", &TokenContext{VideoDynamicRange: "SDR"}, ""},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{MediaInfo VideoDynamicRange}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoVideoDynamicRangeType(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"HDR10", &TokenContext{VideoDynamicRange: "HDR10"}, "HDR10"},
		{"DV HDR10", &TokenContext{VideoDynamicRange: "DV HDR10"}, "DV HDR10"},
		{"SDR", &TokenContext{VideoDynamicRange: "SDR"}, ""},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{MediaInfo VideoDynamicRangeType}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoAudioCodec(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"TrueHD", &TokenContext{AudioCodec: "TrueHD Atmos"}, "TrueHD Atmos"},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{MediaInfo AudioCodec}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoAudioChannels(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"7.1", &TokenContext{AudioChannels: "7.1"}, "7.1"},
		{"5.1", &TokenContext{AudioChannels: "5.1"}, "5.1"},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{MediaInfo AudioChannels}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoAudioLanguages(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "multiple",
			ctx:     &TokenContext{AudioLanguages: []string{"EN", "DE", "FR"}},
			pattern: "{MediaInfo AudioLanguages}",
			want:    "[EN+DE+FR]",
		},
		{
			name:    "single",
			ctx:     &TokenContext{AudioLanguages: []string{"EN"}},
			pattern: "{MediaInfo AudioLanguages}",
			want:    "[EN]",
		},
		{
			name:    "empty",
			ctx:     &TokenContext{},
			pattern: "{MediaInfo AudioLanguages}",
			want:    "",
		},
		{
			name:    "with exclusion filter",
			ctx:     &TokenContext{AudioLanguages: []string{"EN", "DE", "FR"}},
			pattern: "{MediaInfo AudioLanguages:-DE}",
			want:    "[EN+FR]",
		},
		{
			name:    "with inclusion filter",
			ctx:     &TokenContext{AudioLanguages: []string{"EN", "DE", "FR", "ES"}},
			pattern: "{MediaInfo AudioLanguages:EN+DE}",
			want:    "[EN+DE]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MediaInfoSubtitleLanguages(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "multiple",
			ctx:     &TokenContext{SubtitleLanguages: []string{"EN", "ES"}},
			pattern: "{MediaInfo SubtitleLanguages}",
			want:    "[EN+ES]",
		},
		{
			name:    "empty",
			ctx:     &TokenContext{},
			pattern: "{MediaInfo SubtitleLanguages}",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== OTHER TOKENS =====

func TestToken_ReleaseGroup(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"has value", &TokenContext{ReleaseGroup: "SPARKS"}, "SPARKS"},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{Release Group}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_Revision(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"Proper", &TokenContext{Revision: "Proper"}, "Proper"},
		{"REPACK", &TokenContext{Revision: "REPACK"}, "REPACK"},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{Revision}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_EditionTags(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"Directors Cut", &TokenContext{EditionTags: "Director's Cut"}, "Director's Cut"},
		{"Extended IMAX", &TokenContext{EditionTags: "Extended IMAX"}, "Extended IMAX"},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{Edition Tags}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_CustomFormats(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"multiple", &TokenContext{CustomFormats: []string{"Remux", "HDR"}}, "Remux HDR"},
		{"single", &TokenContext{CustomFormats: []string{"x265"}}, "x265"},
		{"empty", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{Custom Formats}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_CustomFormatSpecific(t *testing.T) {
	ctx := &TokenContext{CustomFormats: []string{"Remux", "HDR", "DV"}}

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"matching format", "{Custom Format:Remux}", "Remux"},
		{"matching format case insensitive", "{Custom Format:hdr}", "HDR"},
		{"non-matching format", "{Custom Format:x265}", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_OriginalTitle(t *testing.T) {
	ctx := &TokenContext{OriginalTitle: "The.Matrix.1999.1080p"}

	tokens := ParseTokens("{Original Title}")
	got := tokens[0].Resolve(ctx)
	if got != "The.Matrix.1999.1080p" {
		t.Errorf("Resolve() = %q, want %q", got, "The.Matrix.1999.1080p")
	}
}

func TestToken_OriginalFilename(t *testing.T) {
	ctx := &TokenContext{OriginalFile: "the.matrix.1999.mkv"}

	tokens := ParseTokens("{Original Filename}")
	got := tokens[0].Resolve(ctx)
	if got != "the.matrix.1999.mkv" {
		t.Errorf("Resolve() = %q, want %q", got, "the.matrix.1999.mkv")
	}
}

// ===== ANIME TOKENS =====

func TestToken_Absolute(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "no padding",
			ctx:     &TokenContext{AbsoluteNumber: 42},
			pattern: "{absolute}",
			want:    "42",
		},
		{
			name:    "3-digit padding",
			ctx:     &TokenContext{AbsoluteNumber: 42},
			pattern: "{absolute:000}",
			want:    "042",
		},
		{
			name:    "4-digit padding",
			ctx:     &TokenContext{AbsoluteNumber: 5},
			pattern: "{absolute:0000}",
			want:    "0005",
		},
		{
			name:    "zero returns empty",
			ctx:     &TokenContext{AbsoluteNumber: 0},
			pattern: "{absolute:000}",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_Version(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"v2", &TokenContext{ReleaseVersion: 2}, "v2"},
		{"v3", &TokenContext{ReleaseVersion: 3}, "v3"},
		{"v1 returns empty", &TokenContext{ReleaseVersion: 1}, ""},
		{"v0 returns empty", &TokenContext{ReleaseVersion: 0}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{version}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== MOVIE TOKENS =====

func TestToken_MovieTitle(t *testing.T) {
	ctx := &TokenContext{MovieTitle: "The Matrix"}

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"basic", "{Movie Title}", "The Matrix"},
		{"with dot separator", "{.Movie Title}", "The.Matrix"}, // Separator prefix at start
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MovieTitleYear(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *TokenContext
		pattern string
		want    string
	}{
		{
			name:    "with year",
			ctx:     &TokenContext{MovieTitle: "The Matrix", MovieYear: 1999},
			pattern: "{Movie TitleYear}",
			want:    "The Matrix (1999)",
		},
		{
			name:    "without year",
			ctx:     &TokenContext{MovieTitle: "The Matrix"},
			pattern: "{Movie TitleYear}",
			want:    "The Matrix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_MovieCleanTitle(t *testing.T) {
	ctx := &TokenContext{MovieTitle: "Star Trek: The Motion Picture"}

	tokens := ParseTokens("{Movie CleanTitle}")
	got := tokens[0].Resolve(ctx)
	want := "Star Trek The Motion Picture"
	if got != want {
		t.Errorf("Resolve() = %q, want %q", got, want)
	}
}

func TestToken_MovieCleanTitleYear(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{
			name: "with year",
			ctx:  &TokenContext{MovieTitle: "Star Trek: The Motion Picture", MovieYear: 1979},
			want: "Star Trek The Motion Picture 1979",
		},
		{
			name: "without year",
			ctx:  &TokenContext{MovieTitle: "Star Trek: TMP"},
			want: "Star Trek TMP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{Movie CleanTitleYear}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToken_Year(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TokenContext
		want string
	}{
		{"movie year", &TokenContext{MovieYear: 1999}, "1999"},
		{"series year fallback", &TokenContext{SeriesYear: 2008}, "2008"},
		{"movie year takes precedence", &TokenContext{MovieYear: 1999, SeriesYear: 2008}, "1999"},
		{"no year", &TokenContext{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens("{Year}")
			got := tokens[0].Resolve(tt.ctx)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== TOKEN PARSING =====

func TestParseTokens(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{"single token", "{Series Title}", 1},
		{"multiple tokens", "{Series Title} - S{season:00}E{episode:00}", 3},
		{"no tokens", "plain text", 0},
		{"mixed content", "Show - {Episode Title}.mkv", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			if len(tokens) != tt.expected {
				t.Errorf("ParseTokens() returned %d tokens, want %d", len(tokens), tt.expected)
			}
		})
	}
}

func TestParseTokenContent(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		wantName      string
		wantSeparator string
		wantModifier  string
	}{
		{
			name:          "simple",
			pattern:       "{Series Title}",
			wantName:      "Series Title",
			wantSeparator: " ",
			wantModifier:  "",
		},
		{
			name:          "dot separator",
			pattern:       "{.Series Title}",
			wantName:      "Series Title",
			wantSeparator: ".",
			wantModifier:  "",
		},
		{
			name:          "dash separator",
			pattern:       "{-Series Title}",
			wantName:      "Series Title",
			wantSeparator: "-",
			wantModifier:  "",
		},
		{
			name:          "underscore separator",
			pattern:       "{_Series Title}",
			wantName:      "Series Title",
			wantSeparator: "_",
			wantModifier:  "",
		},
		{
			name:          "with modifier",
			pattern:       "{season:00}",
			wantName:      "season",
			wantSeparator: " ",
			wantModifier:  "00",
		},
		{
			name:          "separator and modifier",
			pattern:       "{.Episode Title:30}",
			wantName:      "Episode Title",
			wantSeparator: ".",
			wantModifier:  "30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseTokens(tt.pattern)
			if len(tokens) != 1 {
				t.Fatalf("expected 1 token, got %d", len(tokens))
			}
			token := tokens[0]
			if token.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", token.Name, tt.wantName)
			}
			if token.Separator != tt.wantSeparator {
				t.Errorf("Separator = %q, want %q", token.Separator, tt.wantSeparator)
			}
			if token.Modifier != tt.wantModifier {
				t.Errorf("Modifier = %q, want %q", token.Modifier, tt.wantModifier)
			}
		})
	}
}

// ===== TOKEN VALIDATION =====

func TestIsValidTokenName(t *testing.T) {
	validTokens := []string{
		"Series Title", "series title", "SERIES TITLE",
		"Series TitleYear", "Series CleanTitle", "Series CleanTitleYear",
		"season", "episode",
		"Air-Date", "Air Date",
		"Episode Title", "Episode CleanTitle",
		"Quality Full", "Quality Title",
		"MediaInfo Simple", "MediaInfo Full", "MediaInfo VideoCodec",
		"MediaInfo VideoBitDepth", "MediaInfo VideoDynamicRange", "MediaInfo VideoDynamicRangeType",
		"MediaInfo AudioCodec", "MediaInfo AudioChannels",
		"MediaInfo AudioLanguages", "MediaInfo SubtitleLanguages",
		"Release Group", "Edition Tags", "Custom Formats", "Original Title", "Original Filename", "Revision",
		"absolute", "version",
		"Movie Title", "Movie TitleYear", "Movie CleanTitle", "Movie CleanTitleYear", "Year",
		"Custom Format:Remux",
	}

	for _, name := range validTokens {
		t.Run(name, func(t *testing.T) {
			if !IsValidTokenName(name) {
				t.Errorf("IsValidTokenName(%q) = false, want true", name)
			}
		})
	}

	invalidTokens := []string{
		"Invalid Token",
		"Fake",
		"",
		"Random Text",
	}

	for _, name := range invalidTokens {
		t.Run("invalid_"+name, func(t *testing.T) {
			if IsValidTokenName(name) {
				t.Errorf("IsValidTokenName(%q) = true, want false", name)
			}
		})
	}
}

func TestGetAllTokenNames(t *testing.T) {
	names := GetAllTokenNames()
	if len(names) == 0 {
		t.Error("GetAllTokenNames() returned empty list")
	}

	// Verify some expected tokens are present
	expected := []string{"Series Title", "season", "episode", "Movie Title", "Year"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetAllTokenNames() missing expected token %q", exp)
		}
	}
}

// ===== CASE INSENSITIVITY =====

func TestTokenCaseInsensitivity(t *testing.T) {
	ctx := &TokenContext{SeriesTitle: "Test Show", SeasonNumber: 1}

	patterns := []string{
		"{Series Title}",
		"{series title}",
		"{SERIES TITLE}",
		"{Series title}",
	}

	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			tokens := ParseTokens(pattern)
			got := tokens[0].Resolve(ctx)
			if got != "Test Show" {
				t.Errorf("Resolve() = %q, want %q", got, "Test Show")
			}
		})
	}
}

// ===== UNKNOWN TOKENS =====

func TestUnknownToken(t *testing.T) {
	ctx := fullContext()

	tokens := ParseTokens("{Unknown Token}")
	got := tokens[0].Resolve(ctx)
	if got != "" {
		t.Errorf("Resolve() = %q, want empty string for unknown token", got)
	}
}
