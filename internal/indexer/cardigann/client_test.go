package cardigann

import (
	"testing"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

func TestBuildEnhancedQueryKeywords(t *testing.T) {
	tests := []struct {
		name     string
		criteria types.SearchCriteria
		expected string
	}{
		{
			name:     "TV search with season and episode",
			criteria: types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			expected: "Dark S01E01",
		},
		{
			name:     "TV search with season only",
			criteria: types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 0},
			expected: "Dark S01",
		},
		{
			name:     "TV search with no season",
			criteria: types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 0, Episode: 0},
			expected: "Dark",
		},
		{
			name:     "TV search with double-digit season/episode",
			criteria: types.SearchCriteria{Query: "Game of Thrones", Type: "tvsearch", Season: 10, Episode: 12},
			expected: "Game of Thrones S10E12",
		},
		{
			name:     "Movie search with year",
			criteria: types.SearchCriteria{Query: "It", Type: "movie", Year: 2017},
			expected: "It 2017",
		},
		{
			name:     "Movie search without year",
			criteria: types.SearchCriteria{Query: "It", Type: "movie", Year: 0},
			expected: "It",
		},
		{
			name:     "General search passes through",
			criteria: types.SearchCriteria{Query: "Dark Knight", Type: "search"},
			expected: "Dark Knight",
		},
		{
			name:     "Empty query returns empty",
			criteria: types.SearchCriteria{Query: "", Type: "tvsearch", Season: 1, Episode: 1},
			expected: "",
		},
		{
			name:     "Multi-word movie title with year",
			criteria: types.SearchCriteria{Query: "The Dark Knight", Type: "movie", Year: 2008},
			expected: "The Dark Knight 2008",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEnhancedQueryKeywords(tt.criteria)
			if result != tt.expected {
				t.Errorf("buildEnhancedQueryKeywords(%+v) = %q, want %q", tt.criteria, result, tt.expected)
			}
		})
	}
}
