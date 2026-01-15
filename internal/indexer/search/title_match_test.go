package search

import (
	"math"
	"testing"
)

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dots to spaces",
			input:    "The.Dark.Knight.2008",
			expected: "the dark knight 2008",
		},
		{
			name:     "TV release format",
			input:    "Dark.S01E01.1080p.WEB-DL",
			expected: "dark s01e01 1080p web dl",
		},
		{
			name:     "parentheses and year",
			input:    "It (2017)",
			expected: "it 2017",
		},
		{
			name:     "multiple spaces",
			input:    "  Multiple   Spaces  ",
			expected: "multiple spaces",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "special characters",
			input:    "Spider-Man: Into the Spider-Verse",
			expected: "spider man into the spider verse",
		},
		{
			name:     "numbers preserved",
			input:    "2001: A Space Odyssey",
			expected: "2001 a space odyssey",
		},
		{
			name:     "already lowercase",
			input:    "dark",
			expected: "dark",
		},
		{
			name:     "mixed case",
			input:    "DaRk KnIgHt",
			expected: "dark knight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTitlesMatch(t *testing.T) {
	tests := []struct {
		name        string
		parsedTitle string
		searchQuery string
		expected    bool
	}{
		{
			name:        "exact match",
			parsedTitle: "Dark",
			searchQuery: "Dark",
			expected:    true,
		},
		{
			name:        "case insensitive",
			parsedTitle: "Dark",
			searchQuery: "dark",
			expected:    true,
		},
		{
			name:        "different title - longer",
			parsedTitle: "The Dark Knight",
			searchQuery: "Dark",
			expected:    false,
		},
		{
			name:        "different title - prefix match",
			parsedTitle: "Dark Matter",
			searchQuery: "Dark",
			expected:    false,
		},
		{
			name:        "short title exact match",
			parsedTitle: "It",
			searchQuery: "It",
			expected:    true,
		},
		{
			name:        "short title mismatch",
			parsedTitle: "It Follows",
			searchQuery: "It",
			expected:    false,
		},
		{
			name:        "multi-word exact match",
			parsedTitle: "Game of Thrones",
			searchQuery: "Game of Thrones",
			expected:    true,
		},
		{
			name:        "multi-word with special chars",
			parsedTitle: "Game.of.Thrones",
			searchQuery: "Game of Thrones",
			expected:    true,
		},
		{
			name:        "empty strings",
			parsedTitle: "",
			searchQuery: "",
			expected:    true,
		},
		{
			name:        "one empty",
			parsedTitle: "Dark",
			searchQuery: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TitlesMatch(tt.parsedTitle, tt.searchQuery)
			if result != tt.expected {
				t.Errorf("TitlesMatch(%q, %q) = %v, want %v", tt.parsedTitle, tt.searchQuery, result, tt.expected)
			}
		})
	}
}

func TestCalculateTitleSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		title1   string
		title2   string
		expected float64
	}{
		{
			name:     "identical titles",
			title1:   "Dark",
			title2:   "Dark",
			expected: 1.0,
		},
		{
			name:     "partial match - one of three",
			title1:   "Dark",
			title2:   "The Dark Knight",
			expected: 1.0 / 3.0, // 1 matching token, 3 unique tokens
		},
		{
			name:     "partial match - three of four",
			title1:   "The Dark Knight",
			title2:   "The Dark Knight Rises",
			expected: 3.0 / 4.0,
		},
		{
			name:     "no common tokens",
			title1:   "Dark",
			title2:   "Sunshine",
			expected: 0.0,
		},
		{
			name:     "both empty",
			title1:   "",
			title2:   "",
			expected: 1.0,
		},
		{
			name:     "one empty",
			title1:   "Dark",
			title2:   "",
			expected: 0.0,
		},
		{
			name:     "case insensitive",
			title1:   "DARK",
			title2:   "dark",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTitleSimilarity(tt.title1, tt.title2)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("CalculateTitleSimilarity(%q, %q) = %v, want %v", tt.title1, tt.title2, result, tt.expected)
			}
		})
	}
}
