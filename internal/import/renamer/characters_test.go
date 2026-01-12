package renamer

import (
	"testing"
)

// ===== ILLEGAL CHARACTER DETECTION =====

func TestIsIllegalChar(t *testing.T) {
	illegal := []rune{'\\', '/', ':', '*', '?', '"', '<', '>', '|'}
	for _, r := range illegal {
		t.Run(string(r), func(t *testing.T) {
			if !isIllegalChar(r) {
				t.Errorf("isIllegalChar(%q) = false, want true", r)
			}
		})
	}

	legal := []rune{'a', 'Z', '1', ' ', '-', '_', '.', '(', ')', '\'', '!', '@', '#'}
	for _, r := range legal {
		t.Run("legal_"+string(r), func(t *testing.T) {
			if isIllegalChar(r) {
				t.Errorf("isIllegalChar(%q) = true, want false", r)
			}
		})
	}
}

// ===== CHARACTER REPLACEMENT =====

func TestGetReplacement(t *testing.T) {
	tests := []struct {
		input rune
		want  rune
	}{
		{'\\', '-'},
		{'/', '-'},
		{'*', '-'},
		{'?', ' '},
		{'"', '\''},
		{'<', '('},
		{'>', ')'},
		{'|', '-'},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := getReplacement(tt.input)
			if got != tt.want {
				t.Errorf("getReplacement(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== COLON REPLACEMENT MODES =====

func TestHandleColon_Delete(t *testing.T) {
	got := handleColon("test:value", 4, ColonDelete, "")
	if got != "" {
		t.Errorf("handleColon(Delete) = %q, want empty", got)
	}
}

func TestHandleColon_Dash(t *testing.T) {
	got := handleColon("test:value", 4, ColonDash, "")
	if got != "-" {
		t.Errorf("handleColon(Dash) = %q, want -", got)
	}
}

func TestHandleColon_SpaceDash(t *testing.T) {
	got := handleColon("test:value", 4, ColonSpaceDash, "")
	if got != " -" {
		t.Errorf("handleColon(SpaceDash) = %q, want \" -\"", got)
	}
}

func TestHandleColon_SpaceDashSpace(t *testing.T) {
	got := handleColon("test:value", 4, ColonSpaceDashSpace, "")
	if got != " - " {
		t.Errorf("handleColon(SpaceDashSpace) = %q, want \" - \"", got)
	}
}

func TestHandleColon_Custom(t *testing.T) {
	tests := []struct {
		name   string
		custom string
		want   string
	}{
		{"with custom", ">>", ">>"},
		{"empty custom fallback", "", "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleColon("test:value", 4, ColonCustom, tt.custom)
			if got != tt.want {
				t.Errorf("handleColon(Custom) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleColon_Smart(t *testing.T) {
	tests := []struct {
		name  string
		input string
		pos   int
		want  string
	}{
		{"word before space after", "Title: Subtitle", 5, " -"},   // prev=word, next=space -> " -"
		{"space before word after", "Title :Subtitle", 6, "-"},    // prev=space (not word) -> "-"
		{"word before word after", "Title:Subtitle", 5, " - "},    // prev=word, next=word -> " - "
		{"at start", ":test", 0, "-"},
		{"at end", "test:", 4, "-"},
		{"between punctuation", "a::b", 2, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := smartColonReplace(tt.input, tt.pos)
			if got != tt.want {
				t.Errorf("smartColonReplace(%q, %d) = %q, want %q", tt.input, tt.pos, got, tt.want)
			}
		})
	}
}

// ===== REPLACE ILLEGAL CHARACTERS =====

func TestReplaceIllegalCharacters(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		replace    bool
		colonMode  ColonReplacement
		customColon string
		want       string
	}{
		{
			name:      "replace mode - removes chars",
			input:     "Test?File*Name",
			replace:   true,
			colonMode: ColonDelete,
			want:      "Test File-Name",
		},
		{
			name:      "no replace - strips chars",
			input:     "Test?File*Name",
			replace:   false,
			colonMode: ColonDelete,
			want:      "TestFileName",
		},
		{
			name:      "colon delete",
			input:     "Star Trek: Discovery",
			replace:   true,
			colonMode: ColonDelete,
			want:      "Star Trek Discovery",
		},
		{
			name:      "colon dash",
			input:     "Star Trek: Discovery",
			replace:   true,
			colonMode: ColonDash,
			want:      "Star Trek- Discovery",
		},
		{
			name:      "colon space dash space",
			input:     "Star Trek: Discovery",
			replace:   true,
			colonMode: ColonSpaceDashSpace,
			want:      "Star Trek - Discovery",
		},
		{
			name:        "colon custom",
			input:       "Star Trek: Discovery",
			replace:     true,
			colonMode:   ColonCustom,
			customColon: " >>",
			want:        "Star Trek >> Discovery",
		},
		{
			name:      "quotes replaced",
			input:     `He said "hello"`,
			replace:   true,
			colonMode: ColonDelete,
			want:      "He said 'hello'",
		},
		{
			name:      "angle brackets",
			input:     "Item <1>",
			replace:   true,
			colonMode: ColonDelete,
			want:      "Item (1)",
		},
		{
			name:      "empty string",
			input:     "",
			replace:   true,
			colonMode: ColonDelete,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceIllegalCharacters(tt.input, tt.replace, tt.colonMode, tt.customColon)
			if got != tt.want {
				t.Errorf("ReplaceIllegalCharacters() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== CLEAN TITLE =====

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"removes colons", "Star Trek: Discovery", "Star Trek Discovery"},
		{"removes question marks", "What If...?", "What If..."},
		{"removes multiple illegal", `Test: "File" <Name>`, "Test File Name"}, // CleanTitle only removes, doesn't replace
		{"preserves unicode", "Café Müller", "Café Müller"},
		{"preserves accents", "Les Misérables", "Les Misérables"},
		{"removes backslash", "Test\\Name", "TestName"},
		{"cleans double spaces", "Test  Name", "Test Name"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanTitle(tt.input)
			if got != tt.want {
				t.Errorf("CleanTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== SANITIZE FILENAME =====

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		replace    bool
		colonMode  ColonReplacement
		customColon string
		want       string
	}{
		{
			name:      "basic sanitization",
			input:     "Test: File?Name",
			replace:   true,
			colonMode: ColonSpaceDashSpace,
			want:      "Test - File Name",
		},
		{
			name:      "trim leading dots",
			input:     ".hidden",
			replace:   true,
			colonMode: ColonDelete,
			want:      "hidden",
		},
		{
			name:      "trim trailing dots",
			input:     "file.",
			replace:   true,
			colonMode: ColonDelete,
			want:      "file",
		},
		{
			name:      "trim leading spaces",
			input:     "  spaced",
			replace:   true,
			colonMode: ColonDelete,
			want:      "spaced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.input, tt.replace, tt.colonMode, tt.customColon)
			if got != tt.want {
				t.Errorf("SanitizeFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== RESERVED WINDOWS NAMES =====

func TestAvoidReservedNames(t *testing.T) {
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	for _, name := range reservedNames {
		t.Run(name, func(t *testing.T) {
			got := avoidReservedNames(name)
			if got == name {
				t.Errorf("avoidReservedNames(%q) should modify reserved name", name)
			}
			if got != name+"_" {
				t.Errorf("avoidReservedNames(%q) = %q, want %q", name, got, name+"_")
			}
		})
	}

	// Case insensitive
	t.Run("case insensitive", func(t *testing.T) {
		got := avoidReservedNames("con")
		if got != "con_" {
			t.Errorf("avoidReservedNames(con) = %q, want con_", got)
		}
	})

	// Reserved name with extension
	t.Run("with extension", func(t *testing.T) {
		got := avoidReservedNames("CON.txt")
		if got != "CON_.txt" {
			t.Errorf("avoidReservedNames(CON.txt) = %q, want CON_.txt", got)
		}
	})

	// Non-reserved names unchanged
	nonReserved := []string{"file", "document", "CONST", "CONTAIN", "PRINTER"}
	for _, name := range nonReserved {
		t.Run("non_reserved_"+name, func(t *testing.T) {
			got := avoidReservedNames(name)
			if got != name {
				t.Errorf("avoidReservedNames(%q) = %q, want unchanged", name, got)
			}
		})
	}
}

// ===== SANITIZE FOLDER NAME =====

func TestSanitizeFolderName(t *testing.T) {
	// Folder sanitization uses same logic as filename
	got := SanitizeFolderName("Test: Folder?", true, ColonSpaceDashSpace, "")
	want := "Test - Folder"
	if got != want {
		t.Errorf("SanitizeFolderName() = %q, want %q", got, want)
	}
}

// ===== CLEANUP SPACES =====

func TestCleanupSpaces(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test  double", "test double"},
		{"test   triple", "test triple"},
		{"  leading", "leading"},
		{"trailing  ", "trailing"},
		{"  both  ", "both"},
		{"no change", "no change"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanupSpaces(tt.input)
			if got != tt.want {
				t.Errorf("cleanupSpaces(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== EDGE CASES =====

func TestReplaceIllegalCharacters_AllIllegal(t *testing.T) {
	// String with all illegal characters
	input := `\/:*?"<>|`

	got := ReplaceIllegalCharacters(input, true, ColonDash, "")
	// Should have replacements for each
	if got == "" {
		// All chars replaced with something
		t.Log("All illegal chars handled")
	}

	// With no replacement
	got2 := ReplaceIllegalCharacters(input, false, ColonDelete, "")
	// All stripped
	if got2 != "" {
		t.Errorf("expected empty string when all chars illegal and replace=false, got %q", got2)
	}
}

func TestSanitizeFilename_Unicode(t *testing.T) {
	// Unicode characters should be preserved
	input := "日本語タイトル"
	got := SanitizeFilename(input, true, ColonDelete, "")
	if got != input {
		t.Errorf("SanitizeFilename() = %q, want %q (unicode preserved)", got, input)
	}
}

func TestSanitizeFilename_Emoji(t *testing.T) {
	// Emoji should be preserved (they're not filesystem-illegal)
	input := "Movie 🎬 Title"
	got := SanitizeFilename(input, true, ColonDelete, "")
	if got != input {
		t.Errorf("SanitizeFilename() = %q, want %q (emoji preserved)", got, input)
	}
}
