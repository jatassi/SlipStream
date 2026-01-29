package availability

import (
	"testing"
)

func TestFormatSeasonRanges(t *testing.T) {
	tests := []struct {
		name     string
		seasons  []int
		expected string
	}{
		{
			name:     "empty",
			seasons:  []int{},
			expected: "Unavailable",
		},
		{
			name:     "single season",
			seasons:  []int{3},
			expected: "Season 3 Available",
		},
		{
			name:     "contiguous range from 1",
			seasons:  []int{1, 2, 3},
			expected: "Seasons 1-3 Available",
		},
		{
			name:     "contiguous range not from 1",
			seasons:  []int{2, 3, 4},
			expected: "Seasons 2-4 Available",
		},
		{
			name:     "non-contiguous with gap",
			seasons:  []int{1, 2, 3, 5},
			expected: "Seasons 1-3, 5 Available",
		},
		{
			name:     "multiple ranges",
			seasons:  []int{1, 2, 4, 5},
			expected: "Seasons 1-2, 4-5 Available",
		},
		{
			name:     "scattered singles",
			seasons:  []int{1, 3, 5},
			expected: "Seasons 1, 3, 5 Available",
		},
		{
			name:     "complex mix",
			seasons:  []int{1, 2, 3, 5, 7, 8, 9},
			expected: "Seasons 1-3, 5, 7-9 Available",
		},
		{
			name:     "just two adjacent",
			seasons:  []int{1, 2},
			expected: "Seasons 1-2 Available",
		},
		{
			name:     "two non-adjacent",
			seasons:  []int{1, 3},
			expected: "Seasons 1, 3 Available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSeasonRanges(tt.seasons)
			if got != tt.expected {
				t.Errorf("formatSeasonRanges(%v) = %q, want %q", tt.seasons, got, tt.expected)
			}
		})
	}
}

func TestParseSeasonNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single season",
			input:    "3",
			expected: []int{3},
		},
		{
			name:     "multiple seasons",
			input:    "1,2,3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "unsorted input",
			input:    "3,1,2",
			expected: []int{1, 2, 3},
		},
		{
			name:     "with spaces",
			input:    "1, 2, 3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "skip invalid",
			input:    "1,abc,3",
			expected: []int{1, 3},
		},
		{
			name:     "skip zero",
			input:    "0,1,2",
			expected: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSeasonNumbers(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("parseSeasonNumbers(%q) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseSeasonNumbers(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestFormatSeriesAvailability(t *testing.T) {
	tests := []struct {
		name             string
		totalSeasons     int64
		availableSeasons string
		expected         string
	}{
		{
			name:             "no seasons",
			totalSeasons:     0,
			availableSeasons: "",
			expected:         "Unavailable",
		},
		{
			name:             "no downloads",
			totalSeasons:     3,
			availableSeasons: "",
			expected:         "Unavailable",
		},
		{
			name:             "all downloaded",
			totalSeasons:     3,
			availableSeasons: "1,2,3",
			expected:         "Available",
		},
		{
			name:             "partial - only season 3",
			totalSeasons:     3,
			availableSeasons: "3",
			expected:         "Season 3 Available",
		},
		{
			name:             "partial - seasons 1-2 of 5",
			totalSeasons:     5,
			availableSeasons: "1,2",
			expected:         "Seasons 1-2 Available",
		},
		{
			name:             "non-contiguous",
			totalSeasons:     5,
			availableSeasons: "1,2,3,5",
			expected:         "Seasons 1-3, 5 Available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSeriesAvailability(tt.totalSeasons, tt.availableSeasons)
			if got != tt.expected {
				t.Errorf("formatSeriesAvailability(%d, %q) = %q, want %q", tt.totalSeasons, tt.availableSeasons, got, tt.expected)
			}
		})
	}
}
