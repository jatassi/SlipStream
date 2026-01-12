package renamer

import (
	"testing"
)

// ===== MULTI-EPISODE STYLE CONSTANTS =====

func TestMultiEpisodeStyleConstants(t *testing.T) {
	tests := []struct {
		style MultiEpisodeStyle
		want  string
	}{
		{StyleExtend, "extend"},
		{StyleDuplicate, "duplicate"},
		{StyleRepeat, "repeat"},
		{StyleScene, "scene"},
		{StyleRange, "range"},
		{StylePrefixedRange, "prefixed_range"},
	}

	for _, tt := range tests {
		t.Run(string(tt.style), func(t *testing.T) {
			if string(tt.style) != tt.want {
				t.Errorf("style = %q, want %q", tt.style, tt.want)
			}
		})
	}
}

// ===== FORMAT MULTI-EPISODE =====

func TestFormatMultiEpisode_StyleExtend(t *testing.T) {
	tests := []struct {
		name     string
		season   int
		episodes []int
		padding  int
		want     string
	}{
		{"two episodes", 1, []int{1, 2}, 2, "S01E01-02"},
		{"three episodes", 1, []int{1, 2, 3}, 2, "S01E01-02-03"},
		{"non-consecutive", 1, []int{1, 3, 5}, 2, "S01E01-03-05"},
		{"single episode", 1, []int{5}, 2, "S01E05"},
		{"3-digit padding", 1, []int{1, 2}, 3, "S01E001-002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultiEpisode(tt.season, tt.episodes, StyleExtend, tt.padding)
			if got != tt.want {
				t.Errorf("FormatMultiEpisode(Extend) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMultiEpisode_StyleDuplicate(t *testing.T) {
	tests := []struct {
		name     string
		season   int
		episodes []int
		padding  int
		want     string
	}{
		{"two episodes", 1, []int{1, 2}, 2, "S01E01.S01E02"},
		{"three episodes", 2, []int{5, 6, 7}, 2, "S02E05.S02E06.S02E07"},
		{"single episode", 1, []int{1}, 2, "S01E01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultiEpisode(tt.season, tt.episodes, StyleDuplicate, tt.padding)
			if got != tt.want {
				t.Errorf("FormatMultiEpisode(Duplicate) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMultiEpisode_StyleRepeat(t *testing.T) {
	tests := []struct {
		name     string
		season   int
		episodes []int
		padding  int
		want     string
	}{
		{"two episodes", 1, []int{1, 2}, 2, "S01E01E02"},
		{"three episodes", 2, []int{5, 6, 7}, 2, "S02E05E06E07"},
		{"single episode", 1, []int{1}, 2, "S01E01"},
		{"3-digit padding", 1, []int{1, 2}, 3, "S01E001E002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultiEpisode(tt.season, tt.episodes, StyleRepeat, tt.padding)
			if got != tt.want {
				t.Errorf("FormatMultiEpisode(Repeat) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMultiEpisode_StyleScene(t *testing.T) {
	tests := []struct {
		name     string
		season   int
		episodes []int
		padding  int
		want     string
	}{
		{"two episodes", 1, []int{1, 2}, 2, "S01E01-E02"},
		{"three episodes", 2, []int{5, 6, 7}, 2, "S02E05-E06-E07"},
		{"single episode", 1, []int{1}, 2, "S01E01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultiEpisode(tt.season, tt.episodes, StyleScene, tt.padding)
			if got != tt.want {
				t.Errorf("FormatMultiEpisode(Scene) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMultiEpisode_StyleRange(t *testing.T) {
	tests := []struct {
		name     string
		season   int
		episodes []int
		padding  int
		want     string
	}{
		{"consecutive", 1, []int{1, 2, 3}, 2, "S01E01-03"},
		{"two consecutive", 2, []int{5, 6}, 2, "S02E05-06"},
		{"non-consecutive falls back to extend", 1, []int{1, 3, 5}, 2, "S01E01-03-05"},
		{"single episode", 1, []int{1}, 2, "S01E01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultiEpisode(tt.season, tt.episodes, StyleRange, tt.padding)
			if got != tt.want {
				t.Errorf("FormatMultiEpisode(Range) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMultiEpisode_StylePrefixedRange(t *testing.T) {
	tests := []struct {
		name     string
		season   int
		episodes []int
		padding  int
		want     string
	}{
		{"consecutive", 1, []int{1, 2, 3}, 2, "S01E01-E03"},
		{"two consecutive", 2, []int{5, 6}, 2, "S02E05-E06"},
		{"non-consecutive falls back to scene", 1, []int{1, 3, 5}, 2, "S01E01-E03-E05"},
		{"single episode", 1, []int{1}, 2, "S01E01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultiEpisode(tt.season, tt.episodes, StylePrefixedRange, tt.padding)
			if got != tt.want {
				t.Errorf("FormatMultiEpisode(PrefixedRange) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMultiEpisode_DefaultStyle(t *testing.T) {
	// Unknown style should default to extend
	got := FormatMultiEpisode(1, []int{1, 2}, MultiEpisodeStyle("unknown"), 2)
	want := "S01E01-02"
	if got != want {
		t.Errorf("FormatMultiEpisode(unknown) = %q, want %q (extend default)", got, want)
	}
}

func TestFormatMultiEpisode_EmptyEpisodes(t *testing.T) {
	got := FormatMultiEpisode(1, []int{}, StyleExtend, 2)
	if got != "" {
		t.Errorf("FormatMultiEpisode(empty) = %q, want empty", got)
	}
}

func TestFormatMultiEpisode_DefaultPadding(t *testing.T) {
	// Padding < 1 should default to 2
	got := FormatMultiEpisode(1, []int{5}, StyleExtend, 0)
	want := "S01E05"
	if got != want {
		t.Errorf("FormatMultiEpisode(padding 0) = %q, want %q", got, want)
	}

	got = FormatMultiEpisode(1, []int{5}, StyleExtend, -1)
	if got != want {
		t.Errorf("FormatMultiEpisode(padding -1) = %q, want %q", got, want)
	}
}

func TestFormatMultiEpisode_UnsortedEpisodes(t *testing.T) {
	// Episodes should be sorted internally
	got := FormatMultiEpisode(1, []int{3, 1, 2}, StyleExtend, 2)
	want := "S01E01-02-03"
	if got != want {
		t.Errorf("FormatMultiEpisode(unsorted) = %q, want %q", got, want)
	}
}

// ===== IS CONSECUTIVE =====

func TestIsConsecutive(t *testing.T) {
	tests := []struct {
		name     string
		episodes []int
		want     bool
	}{
		{"empty", []int{}, true},
		{"single", []int{1}, true},
		{"two consecutive", []int{1, 2}, true},
		{"three consecutive", []int{5, 6, 7}, true},
		{"two non-consecutive", []int{1, 3}, false},
		{"three non-consecutive", []int{1, 2, 5}, false},
		{"gap in middle", []int{1, 2, 4, 5}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConsecutive(tt.episodes)
			if got != tt.want {
				t.Errorf("isConsecutive(%v) = %v, want %v", tt.episodes, got, tt.want)
			}
		})
	}
}

// ===== FORMAT HELPERS =====

func TestFormatSeasonEpisode(t *testing.T) {
	tests := []struct {
		season  int
		episode int
		padding int
		want    string
	}{
		{1, 1, 2, "S01E01"},
		{10, 5, 2, "S10E05"},
		{1, 100, 3, "S01E100"},
		{1, 5, 3, "S01E005"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSeasonEpisode(tt.season, tt.episode, tt.padding)
			if got != tt.want {
				t.Errorf("formatSeasonEpisode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatEpisodeOnly(t *testing.T) {
	tests := []struct {
		episode int
		padding int
		want    string
	}{
		{1, 2, "01"},
		{10, 2, "10"},
		{5, 3, "005"},
		{100, 3, "100"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatEpisodeOnly(tt.episode, tt.padding)
			if got != tt.want {
				t.Errorf("formatEpisodeOnly() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== MULTI-EPISODE TITLES =====

func TestFormatMultiEpisodeTitles(t *testing.T) {
	tests := []struct {
		name      string
		titles    []string
		separator string
		want      string
	}{
		{"two titles", []string{"Part 1", "Part 2"}, " + ", "Part 1 + Part 2"},
		{"three titles", []string{"A", "B", "C"}, " - ", "A - B - C"},
		{"default separator", []string{"X", "Y"}, "", "X + Y"},
		{"empty titles filtered", []string{"A", "", "B"}, " + ", "A + B"},
		{"all empty", []string{"", ""}, " + ", ""},
		{"single title", []string{"Only One"}, " + ", "Only One"},
		{"empty list", []string{}, " + ", ""},
		{"whitespace titles trimmed", []string{"  A  ", "  B  "}, " + ", "A + B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultiEpisodeTitles(tt.titles, tt.separator)
			if got != tt.want {
				t.Errorf("FormatMultiEpisodeTitles() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== PARSE STYLE =====

func TestParseMultiEpisodeStyle(t *testing.T) {
	tests := []struct {
		input string
		want  MultiEpisodeStyle
	}{
		{"extend", StyleExtend},
		{"EXTEND", StyleExtend},
		{"  extend  ", StyleExtend},
		{"duplicate", StyleDuplicate},
		{"repeat", StyleRepeat},
		{"scene", StyleScene},
		{"range", StyleRange},
		{"prefixed_range", StylePrefixedRange},
		{"prefixedrange", StylePrefixedRange},
		{"unknown", StyleExtend},
		{"", StyleExtend},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseMultiEpisodeStyle(tt.input)
			if got != tt.want {
				t.Errorf("ParseMultiEpisodeStyle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== GET STYLE OPTIONS =====

func TestGetMultiEpisodeStyleOptions(t *testing.T) {
	options := GetMultiEpisodeStyleOptions()

	if len(options) != 6 {
		t.Errorf("GetMultiEpisodeStyleOptions() returned %d options, want 6", len(options))
	}

	// Verify all expected styles are present
	styles := map[MultiEpisodeStyle]bool{
		StyleExtend:        false,
		StyleDuplicate:     false,
		StyleRepeat:        false,
		StyleScene:         false,
		StyleRange:         false,
		StylePrefixedRange: false,
	}

	for _, opt := range options {
		styles[opt.Value] = true
		if opt.Label == "" {
			t.Errorf("style %v has empty label", opt.Value)
		}
		if opt.Example == "" {
			t.Errorf("style %v has empty example", opt.Value)
		}
	}

	for style, found := range styles {
		if !found {
			t.Errorf("style %v not found in options", style)
		}
	}
}

// ===== EDGE CASES =====

func TestFormatMultiEpisode_LargeNumbers(t *testing.T) {
	got := FormatMultiEpisode(99, []int{998, 999, 1000}, StyleExtend, 4)
	want := "S99E0998-0999-1000"
	if got != want {
		t.Errorf("FormatMultiEpisode(large) = %q, want %q", got, want)
	}
}

func TestFormatMultiEpisode_SeasonZero(t *testing.T) {
	// Specials (season 0)
	got := FormatMultiEpisode(0, []int{1, 2}, StyleExtend, 2)
	want := "S00E01-02"
	if got != want {
		t.Errorf("FormatMultiEpisode(season 0) = %q, want %q", got, want)
	}
}
