package renamer

import (
	"testing"
)

// ===== CASE MODE =====

func TestApplyCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		mode  CaseMode
		want  string
	}{
		{"default unchanged", "Test String", CaseDefault, "Test String"},
		{"upper", "Test String", CaseUpper, "TEST STRING"},
		{"lower", "Test String", CaseLower, "test string"},
		{"title", "test string", CaseTitle, "Test String"},
		{"empty string", "", CaseUpper, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyCase(tt.input, tt.mode)
			if got != tt.want {
				t.Errorf("ApplyCase(%q, %v) = %q, want %q", tt.input, tt.mode, got, tt.want)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "Hello World"},
		{"HELLO WORLD", "Hello World"},
		{"hello-world", "Hello-World"},
		{"hello_world", "Hello_World"},
		{"hello.world", "Hello.World"},
		{"", ""},
		{"a", "A"},
		{"AB", "Ab"},
		{"123 test", "123 Test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toTitleCaseSimple(tt.input)
			if got != tt.want {
				t.Errorf("toTitleCaseSimple(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToTitleCase_MediaIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"preserves S01E01", "test show - s01e01", "Test Show - S01E01"},
		{"preserves S01E01 uppercase input", "TEST SHOW - S01E01", "Test Show - S01E01"},
		{"preserves multiple identifiers", "s01e01 and s02e05", "S01E01 And S02E05"},
		{"handles S01E01E02 multi-ep", "show - s01e01e02", "Show - S01E01E02"},
		{"handles S10E100 large numbers", "show - s10e100", "Show - S10E100"},
		{"no media identifier", "hello world", "Hello World"},
		{"mixed content", "my show s01e05 episode title", "My Show S01E05 Episode Title"},
		{"at start", "s01e01 - pilot", "S01E01 - Pilot"},
		{"at end", "pilot - s01e01", "Pilot - S01E01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toTitleCase(tt.input)
			if got != tt.want {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== SEPARATOR =====

func TestApplySeparator(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		separator string
		want      string
	}{
		{"space unchanged", "Hello World", " ", "Hello World"},
		{"empty unchanged", "Hello World", "", "Hello World"},
		{"dot separator", "Hello World", ".", "Hello.World"},
		{"dash separator", "Hello World", "-", "Hello-World"},
		{"underscore separator", "Hello World", "_", "Hello_World"},
		{"multiple words", "One Two Three", ".", "One.Two.Three"},
		{"no spaces", "NoSpaces", ".", "NoSpaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplySeparator(tt.input, tt.separator)
			if got != tt.want {
				t.Errorf("ApplySeparator(%q, %q) = %q, want %q", tt.input, tt.separator, got, tt.want)
			}
		})
	}
}

// ===== TRUNCATION =====

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		limit int
		want  string
	}{
		{"no truncation needed", "Short", 10, "Short"},
		{"zero limit unchanged", "Test", 0, "Test"},
		{"truncate from end", "Very Long Title Here", 10, "Very Long…"},
		{"truncate from start", "Very Long Title Here", -10, "…itle Here"},
		{"exact length", "12345", 5, "12345"},
		{"one char", "Test", 1, "…"},
		{"negative one char", "Test", -1, "…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.limit)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.limit, got, tt.want)
			}
		})
	}
}

func TestTruncate_Unicode(t *testing.T) {
	// Unicode should be handled by runes, not bytes
	input := "日本語タイトル" // 7 runes
	got := Truncate(input, 4)
	want := "日本語…"
	if got != want {
		t.Errorf("Truncate(%q, 4) = %q, want %q", input, got, want)
	}
}

// ===== LANGUAGE FORMATTING =====

func TestDefaultLanguageConfig(t *testing.T) {
	cfg := DefaultLanguageConfig()
	if cfg.Separator != "+" {
		t.Errorf("Separator = %q, want +", cfg.Separator)
	}
	if cfg.BracketStyle != "square" {
		t.Errorf("BracketStyle = %q, want square", cfg.BracketStyle)
	}
}

func TestFormatLanguages(t *testing.T) {
	tests := []struct {
		name   string
		langs  []string
		config LanguageConfig
		want   string
	}{
		{
			name:   "default config",
			langs:  []string{"EN", "DE"},
			config: DefaultLanguageConfig(),
			want:   "[EN+DE]",
		},
		{
			name:   "single language",
			langs:  []string{"EN"},
			config: DefaultLanguageConfig(),
			want:   "[EN]",
		},
		{
			name:   "empty list",
			langs:  []string{},
			config: DefaultLanguageConfig(),
			want:   "",
		},
		{
			name:   "round brackets",
			langs:  []string{"EN", "FR"},
			config: LanguageConfig{Separator: "+", BracketStyle: "round"},
			want:   "(EN+FR)",
		},
		{
			name:   "no brackets",
			langs:  []string{"EN", "DE"},
			config: LanguageConfig{Separator: "+", BracketStyle: "none"},
			want:   "EN+DE",
		},
		{
			name:   "custom separator",
			langs:  []string{"EN", "DE", "FR"},
			config: LanguageConfig{Separator: ",", BracketStyle: "square"},
			want:   "[EN,DE,FR]",
		},
		{
			name:   "unknown bracket style defaults to square",
			langs:  []string{"EN"},
			config: LanguageConfig{Separator: "+", BracketStyle: "unknown"},
			want:   "[EN]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatLanguages(tt.langs, tt.config)
			if got != tt.want {
				t.Errorf("FormatLanguages() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== LANGUAGE FILTERING =====

func TestFilterLanguages(t *testing.T) {
	tests := []struct {
		name   string
		langs  []string
		filter string
		want   []string
	}{
		{
			name:   "no filter",
			langs:  []string{"EN", "DE", "FR"},
			filter: "",
			want:   []string{"EN", "DE", "FR"},
		},
		{
			name:   "exclude one",
			langs:  []string{"EN", "DE", "FR"},
			filter: "-DE",
			want:   []string{"EN", "FR"},
		},
		{
			name:   "exclude case insensitive",
			langs:  []string{"EN", "DE", "FR"},
			filter: "-de",
			want:   []string{"EN", "FR"},
		},
		{
			name:   "include specific with plus",
			langs:  []string{"EN", "DE", "FR", "ES"},
			filter: "EN+DE",
			want:   []string{"EN", "DE"},
		},
		{
			name:   "include specific with comma",
			langs:  []string{"EN", "DE", "FR", "ES"},
			filter: "EN,FR",
			want:   []string{"EN", "FR"},
		},
		{
			name:   "include non-matching returns empty",
			langs:  []string{"EN", "DE"},
			filter: "FR+ES",
			want:   []string{},
		},
		{
			name:   "exclude non-existing unchanged",
			langs:  []string{"EN", "DE"},
			filter: "-FR",
			want:   []string{"EN", "DE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterLanguages(tt.langs, tt.filter)
			if len(got) != len(tt.want) {
				t.Errorf("FilterLanguages() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("FilterLanguages()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

// ===== LANGUAGE CODE NORMALIZATION =====

func TestNormalizeLanguageCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// ISO 639-1 (2-letter) codes
		{"2-letter lowercase", "en", "EN"},
		{"2-letter uppercase", "EN", "EN"},
		{"2-letter mixed", "En", "EN"},
		{"german 2-letter", "de", "DE"},
		{"spanish 2-letter", "es", "ES"},

		// ISO 639-2/3 (3-letter) codes
		{"eng to EN", "eng", "EN"},
		{"deu to DE", "deu", "DE"},
		{"ger to DE", "ger", "DE"},
		{"fra to FR", "fra", "FR"},
		{"fre to FR", "fre", "FR"},
		{"spa to ES", "spa", "ES"},
		{"jpn to JA", "jpn", "JA"},
		{"zho to ZH", "zho", "ZH"},
		{"chi to ZH", "chi", "ZH"},

		// Full language names
		{"english", "english", "EN"},
		{"English uppercase", "English", "EN"},
		{"german", "german", "DE"},
		{"German uppercase", "German", "DE"},
		{"french", "french", "FR"},
		{"spanish", "spanish", "ES"},
		{"japanese", "japanese", "JA"},
		{"chinese", "chinese", "ZH"},
		{"korean", "korean", "KO"},
		{"dutch", "dutch", "NL"},
		{"polish", "polish", "PL"},
		{"russian", "russian", "RU"},

		// Whitespace handling
		{"leading/trailing spaces", "  en  ", "EN"},
		{"spaces around name", "  english  ", "EN"},

		// Edge cases
		{"empty string", "", ""},
		{"single char", "a", "A"},
		{"unknown 3-letter fallback", "xyz", "XY"},
		{"unknown long string fallback", "unknownlang", "UN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLanguageCode(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeLanguageCode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== CASE MODE CONSTANTS =====

func TestCaseModeConstants(t *testing.T) {
	if CaseDefault != "default" {
		t.Errorf("CaseDefault = %q, want default", CaseDefault)
	}
	if CaseUpper != "upper" {
		t.Errorf("CaseUpper = %q, want upper", CaseUpper)
	}
	if CaseLower != "lower" {
		t.Errorf("CaseLower = %q, want lower", CaseLower)
	}
	if CaseTitle != "title" {
		t.Errorf("CaseTitle = %q, want title", CaseTitle)
	}
}

// ===== EDGE CASES =====

func TestApplyCase_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		mode  CaseMode
		want  string
	}{
		{"title with dashes", "test-string_here.now", CaseTitle, "Test-String_Here.Now"},
		{"title preserves S01E01", "test-s01e01.now", CaseTitle, "Test-S01E01.Now"},
		{"upper", "Test-String", CaseUpper, "TEST-STRING"},
		{"lower", "Test-String", CaseLower, "test-string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyCase(tt.input, tt.mode)
			if got != tt.want {
				t.Errorf("ApplyCase(%q, %v) = %q, want %q", tt.input, tt.mode, got, tt.want)
			}
		})
	}
}

func TestTruncate_EmptyString(t *testing.T) {
	got := Truncate("", 10)
	if got != "" {
		t.Errorf("Truncate empty = %q, want empty", got)
	}
}

func TestFilterLanguages_EmptyInput(t *testing.T) {
	got := FilterLanguages([]string{}, "EN+DE")
	if len(got) != 0 {
		t.Errorf("FilterLanguages empty input = %v, want empty", got)
	}
}

func TestFilterLanguages_NilInput(t *testing.T) {
	got := FilterLanguages(nil, "EN+DE")
	if len(got) != 0 {
		t.Errorf("FilterLanguages nil input = %v, want empty", got)
	}
}
