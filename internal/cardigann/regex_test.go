package cardigann

import "testing"

func TestRegex(t *testing.T) {
	tests := []struct {
		name, pattern, in string
		group1            string
		matches           bool
	}{
		{"group", `(\d+) seeders`, "12 seeders", "12", true},
		{"no match", `(\d+) seeders`, "none", "", false},
		{"no group", `\d+`, "12", "", true},
		{"named group", `(?<n>\d+)x`, "4x", "4", true},
		{"lookbehind", `(?<=id=)(\d+)`, "?id=42", "42", true},
		{"block cyrillic", `([\p{IsCyrillic}]+)`, "abc Привет", "Привет", true},
		{"block negated", `([^\p{IsBasicLatin}]+)`, "abc日本", "日本", true},
		{"block as a class", `(\p{IsCJKUnifiedIdeographs}+)`, "x日本y", "日本", true},
		{"block negated outside a class", `(\P{IsBasicLatin}+)`, "abcé", "é", true},
		{"case-insensitive option", `(?i)(abc)`, "xABC", "ABC", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileRegex(tt.pattern)
			if err != nil {
				t.Fatal(err)
			}
			got, err := re.Group1(tt.in)
			if err != nil || got != tt.group1 {
				t.Errorf("Group1(%q) = %q, %v; want %q", tt.in, got, err, tt.group1)
			}
			ok, err := re.MatchString(tt.in)
			if err != nil || ok != tt.matches {
				t.Errorf("MatchString(%q) = %v, %v; want %v", tt.in, ok, err, tt.matches)
			}
		})
	}
}

func TestRegexReplace(t *testing.T) {
	tests := []struct{ pattern, in, replacement, want string }{
		{`(\w+) (\w+)`, "abc def", "$2 $1", "def abc"},
		{`(?<a>\w+)`, "x y", "[${a}]", "[x] [y]"},
		{`a`, "aa", "$$", "$$"},
		{`b`, "abc", "$0$0", "abbc"},
		{`\s+`, "a  b   c", " ", "a b c"},
	}
	for _, tt := range tests {
		re, err := compileRegex(tt.pattern)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := re.Replace(tt.in, tt.replacement); err != nil || got != tt.want {
			t.Errorf("Replace(%q, %q, %q) = %q, %v; want %q", tt.pattern, tt.in, tt.replacement, got, err, tt.want)
		}
	}
}

func TestCompileRegexErrors(t *testing.T) {
	for _, p := range []string{`(`, `\p{IsKlingon}`, `[`} {
		if _, err := compileRegex(p); err == nil {
			t.Errorf("compileRegex(%q): want an error", p)
		}
	}
}
