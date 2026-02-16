package search

import (
	"testing"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

func TestFilterByCriteria_TV(t *testing.T) {
	tests := []struct {
		name         string
		torrentTitle string
		criteria     types.SearchCriteria
		shouldPass   bool
	}{
		{
			name:         "exact match with season and episode",
			torrentTitle: "Dark.S01E01.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   true,
		},
		{
			name:         "season pack should pass episode check",
			torrentTitle: "Dark.S01.Complete.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   true,
		},
		{
			name:         "wrong episode should fail",
			torrentTitle: "Dark.S01E02.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   false,
		},
		{
			name:         "wrong season should fail",
			torrentTitle: "Dark.S02E01.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   false,
		},
		{
			name:         "movie should fail TV search",
			torrentTitle: "The.Dark.Knight.2008.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   false,
		},
		{
			name:         "different title should fail",
			torrentTitle: "Dark.Matter.S01E01.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   false,
		},
		{
			name:         "multi-word title exact match",
			torrentTitle: "Game.of.Thrones.S01E01.1080p",
			criteria:     types.SearchCriteria{Query: "Game of Thrones", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   true,
		},
		{
			name:         "season search without episode (season pack)",
			torrentTitle: "Dark.S01.Complete.1080p",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 0},
			shouldPass:   true,
		},
		{
			name:         "season search without episode (individual ep)",
			torrentTitle: "Dark.S01E05.1080p",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 0},
			shouldPass:   true,
		},
		{
			name:         "multi-episode release containing target",
			torrentTitle: "Dark.S01E01E02.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1},
			shouldPass:   true,
		},
		{
			name:         "multi-episode double format containing target",
			torrentTitle: "Dark.S01E01E02.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 2},
			shouldPass:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrents := []types.TorrentInfo{
				{ReleaseInfo: types.ReleaseInfo{Title: tt.torrentTitle}},
			}
			result := FilterByCriteria(torrents, &tt.criteria)

			if tt.shouldPass && len(result) == 0 {
				t.Errorf("Expected torrent %q to pass filter for criteria %+v, but it was filtered out", tt.torrentTitle, tt.criteria)
			}
			if !tt.shouldPass && len(result) > 0 {
				t.Errorf("Expected torrent %q to be filtered out for criteria %+v, but it passed", tt.torrentTitle, tt.criteria)
			}
		})
	}
}

func TestFilterByCriteria_Movie(t *testing.T) {
	tests := []struct {
		name         string
		torrentTitle string
		criteria     types.SearchCriteria
		shouldPass   bool
	}{
		{
			name:         "exact match with year",
			torrentTitle: "It.2017.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "It", Type: "movie", Year: 2017},
			shouldPass:   true,
		},
		{
			name:         "year off by one (later) should pass",
			torrentTitle: "It.2018.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "It", Type: "movie", Year: 2017},
			shouldPass:   true,
		},
		{
			name:         "year off by one (earlier) should pass",
			torrentTitle: "It.2016.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "It", Type: "movie", Year: 2017},
			shouldPass:   true,
		},
		{
			name:         "year off by two should fail",
			torrentTitle: "It.2019.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "It", Type: "movie", Year: 2017},
			shouldPass:   false,
		},
		{
			name:         "different title should fail",
			torrentTitle: "It.Follows.2014.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "It", Type: "movie", Year: 2017},
			shouldPass:   false,
		},
		{
			name:         "TV content should fail movie search",
			torrentTitle: "It.S01E01.1080p.WEB-DL",
			criteria:     types.SearchCriteria{Query: "It", Type: "movie", Year: 2017},
			shouldPass:   false,
		},
		{
			name:         "multi-word movie title",
			torrentTitle: "The.Dark.Knight.2008.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "The Dark Knight", Type: "movie", Year: 2008},
			shouldPass:   true,
		},
		{
			name:         "movie with parentheses year",
			torrentTitle: "Dune (2021) 1080p BluRay",
			criteria:     types.SearchCriteria{Query: "Dune", Type: "movie", Year: 2021},
			shouldPass:   true,
		},
		{
			name:         "movie with different year format",
			torrentTitle: "Dune 2021 1080p BluRay",
			criteria:     types.SearchCriteria{Query: "Dune", Type: "movie", Year: 2021},
			shouldPass:   true,
		},
		{
			name:         "short title exact match",
			torrentTitle: "Her.2013.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "Her", Type: "movie", Year: 2013},
			shouldPass:   true,
		},
		{
			name:         "short title prefix mismatch",
			torrentTitle: "Her.Smell.2018.1080p.BluRay",
			criteria:     types.SearchCriteria{Query: "Her", Type: "movie", Year: 2018},
			shouldPass:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrents := []types.TorrentInfo{
				{ReleaseInfo: types.ReleaseInfo{Title: tt.torrentTitle}},
			}
			result := FilterByCriteria(torrents, &tt.criteria)

			if tt.shouldPass && len(result) == 0 {
				t.Errorf("Expected torrent %q to pass filter for criteria %+v, but it was filtered out", tt.torrentTitle, tt.criteria)
			}
			if !tt.shouldPass && len(result) > 0 {
				t.Errorf("Expected torrent %q to be filtered out for criteria %+v, but it passed", tt.torrentTitle, tt.criteria)
			}
		})
	}
}

func TestFilterByCriteria_EmptyQuery(t *testing.T) {
	torrents := []types.TorrentInfo{
		{ReleaseInfo: types.ReleaseInfo{Title: "Dark.S01E01.1080p"}},
		{ReleaseInfo: types.ReleaseInfo{Title: "It.2017.1080p"}},
	}
	criteria := types.SearchCriteria{Query: "", Type: "tvsearch"}

	result := FilterByCriteria(torrents, &criteria)

	if len(result) != 2 {
		t.Errorf("Expected all torrents to pass when query is empty, got %d results", len(result))
	}
}

func TestFilterByCriteria_GeneralSearch(t *testing.T) {
	torrents := []types.TorrentInfo{
		{ReleaseInfo: types.ReleaseInfo{Title: "Dark.S01E01.1080p"}},
		{ReleaseInfo: types.ReleaseInfo{Title: "It.2017.1080p"}},
		{ReleaseInfo: types.ReleaseInfo{Title: "Random.Movie.2020"}},
	}
	criteria := types.SearchCriteria{Query: "Dark", Type: "search"}

	result := FilterByCriteria(torrents, &criteria)

	// General search type doesn't filter by TV/movie criteria
	if len(result) != 3 {
		t.Errorf("Expected all torrents to pass for general 'search' type, got %d results", len(result))
	}
}

func TestFilterByCriteria_MultipleResults(t *testing.T) {
	torrents := []types.TorrentInfo{
		{ReleaseInfo: types.ReleaseInfo{Title: "Dark.S01E01.1080p.WEB-DL", GUID: "1"}},
		{ReleaseInfo: types.ReleaseInfo{Title: "Dark.S01E01.720p.HDTV", GUID: "2"}},
		{ReleaseInfo: types.ReleaseInfo{Title: "Dark.S01E02.1080p.WEB-DL", GUID: "3"}}, // Wrong episode
		{ReleaseInfo: types.ReleaseInfo{Title: "Dark.S01.Complete.1080p", GUID: "4"}},  // Season pack
		{ReleaseInfo: types.ReleaseInfo{Title: "Dark.Matter.S01E01.1080p", GUID: "5"}}, // Wrong title
	}
	criteria := types.SearchCriteria{Query: "Dark", Type: "tvsearch", Season: 1, Episode: 1}

	result := FilterByCriteria(torrents, &criteria)

	if len(result) != 3 {
		t.Errorf("Expected 3 results (two exact matches + season pack), got %d", len(result))
	}

	// Verify the correct ones passed
	guids := make(map[string]bool)
	for _, r := range result {
		guids[r.GUID] = true
	}
	if !guids["1"] || !guids["2"] || !guids["4"] {
		t.Errorf("Expected GUIDs 1, 2, and 4 to pass, got %v", guids)
	}
}
