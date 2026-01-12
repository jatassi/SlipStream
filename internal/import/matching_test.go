package importer

import "testing"

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Tron.Ares", "Tron Ares"},
		{"Tron: Ares", "Tron Ares"},
		{"John Wick: Chapter 2", "John Wick Chapter 2"},
		{"Deadpool & Wolverine", "Deadpool Wolverine"},
		{"Spider-Man: No Way Home", "Spider Man No Way Home"},
		{"It's a Wonderful Life", "Its a Wonderful Life"},
		{"Mission: Impossible", "Mission Impossible"},
		{"Star.Trek.Discovery", "Star Trek Discovery"},
		{"The_Walking_Dead", "The Walking Dead"},
		{"Lord of the Rings/Return of King", "Lord of the Rings Return of King"},
		{"Title (2020)", "Title"},
		{"Title [2020]", "Title"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanTitle(tt.input)
			if got != tt.expected {
				t.Errorf("cleanTitle(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Tron: Ares", "tron ares"},
		{"The Matrix", "matrix"},
		{"A Quiet Place", "quiet place"},
		{"An American Werewolf", "american werewolf"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTitle(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCalculateTitleSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		titleA   string
		titleB   string
		minScore float64
	}{
		{
			name:     "exact match after normalization - colon vs dot",
			titleA:   "Tron.Ares",
			titleB:   "Tron: Ares",
			minScore: 1.0,
		},
		{
			name:     "john wick with colon",
			titleA:   "John.Wick.Chapter.2",
			titleB:   "John Wick: Chapter 2",
			minScore: 1.0,
		},
		{
			name:     "deadpool with ampersand",
			titleA:   "Deadpool.and.Wolverine",
			titleB:   "Deadpool & Wolverine",
			minScore: 0.6, // "and" vs "&" creates word count mismatch
		},
		{
			name:     "completely different",
			titleA:   "The Matrix",
			titleB:   "Star Wars",
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateTitleSimilarity(tt.titleA, tt.titleB)
			if got < tt.minScore {
				t.Errorf("calculateTitleSimilarity(%q, %q) = %v, want >= %v", tt.titleA, tt.titleB, got, tt.minScore)
			}
		})
	}
}
