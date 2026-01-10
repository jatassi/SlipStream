package scoring

import (
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
)

func TestQualityMatcher(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		resolution     int
		expectedID     int
		expectedName   string
		minConfidence  float64
	}{
		{
			name:          "BluRay 1080p exact match",
			source:        "BluRay",
			resolution:    1080,
			expectedID:    11,
			expectedName:  "Bluray-1080p",
			minConfidence: 1.0,
		},
		{
			name:          "WEB-DL 2160p exact match",
			source:        "WEB-DL",
			resolution:    2160,
			expectedID:    15,
			expectedName:  "WEBDL-2160p",
			minConfidence: 1.0,
		},
		{
			name:          "Remux 1080p exact match",
			source:        "Remux",
			resolution:    1080,
			expectedID:    12,
			expectedName:  "Remux-1080p",
			minConfidence: 1.0,
		},
		{
			name:          "HDTV 720p exact match",
			source:        "HDTV",
			resolution:    720,
			expectedID:    4,
			expectedName:  "HDTV-720p",
			minConfidence: 1.0,
		},
		{
			name:          "Resolution only - 1080p",
			source:        "",
			resolution:    1080,
			expectedID:    12, // Highest weight for 1080p
			expectedName:  "Remux-1080p",
			minConfidence: 0.5,
		},
		{
			name:          "Source only - BluRay",
			source:        "BluRay",
			resolution:    0,
			expectedID:    16, // Highest resolution for BluRay
			expectedName:  "Bluray-2160p",
			minConfidence: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchQuality(tt.source, tt.resolution)

			if result.Quality == nil {
				t.Fatalf("Expected quality match, got nil")
			}

			if result.QualityID != tt.expectedID {
				t.Errorf("Expected quality ID %d, got %d", tt.expectedID, result.QualityID)
			}

			if result.Quality.Name != tt.expectedName {
				t.Errorf("Expected quality name %s, got %s", tt.expectedName, result.Quality.Name)
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %f, got %f", tt.minConfidence, result.Confidence)
			}
		})
	}
}

func TestNormalizeSource(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"BluRay", "bluray"},
		{"Blu-Ray", "bluray"},
		{"BDRip", "bluray"},
		{"WEB-DL", "webdl"},
		{"WEBDL", "webdl"},
		{"WEBRip", "webrip"},
		{"HDTV", "tv"},
		{"Remux", "remux"},
		{"DVDRip", "dvd"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeSource(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSource(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestScorer_HealthScore(t *testing.T) {
	scorer := NewDefaultScorer()

	tests := []struct {
		name           string
		seeders        int
		leechers       int
		freeleech      bool
		minScore       float64
		maxScore       float64
	}{
		{
			name:     "No seeders",
			seeders:  0,
			leechers: 10,
			minScore: 0,
			maxScore: 5, // Just ratio and no freeleech
		},
		{
			name:     "Many seeders",
			seeders:  100,
			leechers: 10,
			minScore: 30, // Good seeder score + good ratio
			maxScore: 40,
		},
		{
			name:      "Freeleech with seeders",
			seeders:   50,
			leechers:  5,
			freeleech: true,
			minScore:  40, // Seeder score + ratio + freeleech bonus
			maxScore:  50,
		},
		{
			name:     "High ratio",
			seeders:  100,
			leechers: 1,
			minScore: 30,
			maxScore: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrent := &types.TorrentInfo{
				Seeders:              tt.seeders,
				Leechers:             tt.leechers,
				DownloadVolumeFactor: 1,
			}
			if tt.freeleech {
				torrent.DownloadVolumeFactor = 0
			}

			score := scorer.calculateHealthScore(torrent)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Health score = %f, want between %f and %f", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestScorer_IndexerScore(t *testing.T) {
	scorer := NewDefaultScorer()

	tests := []struct {
		name          string
		priority      int
		expectedScore float64
		tolerance     float64
	}{
		{
			name:          "Highest priority (1)",
			priority:      1,
			expectedScore: 20,
			tolerance:     0.5,
		},
		{
			name:          "Default priority (50)",
			priority:      50,
			expectedScore: 10,
			tolerance:     1,
		},
		{
			name:          "Low priority (100)",
			priority:      100,
			expectedScore: 0,
			tolerance:     0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrent := &types.TorrentInfo{
				ReleaseInfo: types.ReleaseInfo{
					IndexerID: 1,
				},
			}
			ctx := ScoringContext{
				IndexerPriorities: map[int64]int{1: tt.priority},
			}

			score := scorer.calculateIndexerScore(torrent, ctx)

			diff := score - tt.expectedScore
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Indexer score = %f, want %f (±%f)", score, tt.expectedScore, tt.tolerance)
			}
		})
	}
}

func TestScorer_MatchScore_Year(t *testing.T) {
	scorer := NewDefaultScorer()

	tests := []struct {
		name        string
		title       string
		searchYear  int
		minScore    float64
		description string
	}{
		{
			name:        "Year matches",
			title:       "Movie.Name.2024.1080p.BluRay",
			searchYear:  2024,
			minScore:    10,
			description: "Full year match points",
		},
		{
			name:        "Year missing in release",
			title:       "Movie.Name.1080p.BluRay",
			searchYear:  2024,
			minScore:    10,
			description: "No penalty for missing year",
		},
		{
			name:        "Year mismatch",
			title:       "Movie.Name.2023.1080p.BluRay",
			searchYear:  2024,
			minScore:    0,
			description: "No points for year mismatch",
		},
		{
			name:        "No year filter",
			title:       "Movie.Name.2024.1080p.BluRay",
			searchYear:  0,
			minScore:    10,
			description: "Full points when not filtering by year",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrent := &types.TorrentInfo{
				ReleaseInfo: types.ReleaseInfo{
					Title: tt.title,
				},
			}
			ctx := ScoringContext{
				SearchYear: tt.searchYear,
			}

			score := scorer.calculateMatchScore(torrent, ctx)

			if score < tt.minScore {
				t.Errorf("%s: Match score = %f, want >= %f", tt.description, score, tt.minScore)
			}
		})
	}
}

func TestScorer_MatchScore_Episode(t *testing.T) {
	scorer := NewDefaultScorer()

	tests := []struct {
		name          string
		title         string
		searchSeason  int
		searchEpisode int
		minScore      float64
		maxScore      float64
		description   string
	}{
		{
			name:          "Exact episode match",
			title:         "Show.Name.S01E05.1080p.WEB-DL",
			searchSeason:  1,
			searchEpisode: 5,
			minScore:      30, // Year + Episode
			maxScore:      35,
			description:   "Full points for exact match",
		},
		{
			name:          "Season pack when looking for episode",
			title:         "Show.Name.S01.Complete.1080p.WEB-DL",
			searchSeason:  1,
			searchEpisode: 5,
			minScore:      10, // Year only - parser doesn't extract season-only format
			maxScore:      12,
			description:   "Parser doesn't detect season packs without episode number",
		},
		{
			name:          "Wrong episode",
			title:         "Show.Name.S01E10.1080p.WEB-DL",
			searchSeason:  1,
			searchEpisode: 5,
			minScore:      10, // Just year
			maxScore:      12,
			description:   "No episode points for wrong episode",
		},
		{
			name:          "Wrong season",
			title:         "Show.Name.S02E05.1080p.WEB-DL",
			searchSeason:  1,
			searchEpisode: 5,
			minScore:      10, // Just year
			maxScore:      12,
			description:   "No points for wrong season",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrent := &types.TorrentInfo{
				ReleaseInfo: types.ReleaseInfo{
					Title: tt.title,
				},
			}
			ctx := ScoringContext{
				SearchSeason:  tt.searchSeason,
				SearchEpisode: tt.searchEpisode,
			}

			score := scorer.calculateMatchScore(torrent, ctx)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("%s: Match score = %f, want between %f and %f", tt.description, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestScorer_AgeScore(t *testing.T) {
	scorer := NewDefaultScorer()
	now := time.Now()

	tests := []struct {
		name     string
		daysAgo  int
		minScore float64
		maxScore float64
	}{
		{
			name:     "Fresh release (1 day)",
			daysAgo:  1,
			minScore: 0,
			maxScore: 0,
		},
		{
			name:     "Within grace period (5 days)",
			daysAgo:  5,
			minScore: 0,
			maxScore: 0,
		},
		{
			name:     "Just past grace (10 days)",
			daysAgo:  10,
			minScore: -1,
			maxScore: 0,
		},
		{
			name:     "Old release (60 days)",
			daysAgo:  60,
			minScore: -10,
			maxScore: -5,
		},
		{
			name:     "Very old release (180 days)",
			daysAgo:  180,
			minScore: -20,
			maxScore: -15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrent := &types.TorrentInfo{
				ReleaseInfo: types.ReleaseInfo{
					PublishDate: now.AddDate(0, 0, -tt.daysAgo),
				},
			}
			ctx := ScoringContext{
				Now: now,
			}

			score := scorer.calculateAgeScore(torrent, ctx)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Age score for %d days = %f, want between %f and %f", tt.daysAgo, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestScorer_QualityScore_WithProfile(t *testing.T) {
	scorer := NewDefaultScorer()

	// Create a profile that only allows 1080p content
	profile := quality.HD1080pProfile()

	tests := []struct {
		name       string
		source     string
		resolution int
		allowed    bool
		minScore   float64
	}{
		{
			name:       "Allowed quality - BluRay 1080p",
			source:     "BluRay",
			resolution: 1080,
			allowed:    true,
			minScore:   60, // Weight 11/17 * 100 ≈ 64
		},
		{
			name:       "Allowed quality - WEB-DL 720p",
			source:     "WEB-DL",
			resolution: 720,
			allowed:    true,
			minScore:   30, // Weight 6/17 * 100 ≈ 35
		},
		{
			name:       "Disallowed quality - 2160p",
			source:     "BluRay",
			resolution: 2160,
			allowed:    false,
			minScore:   -1000,
		},
		{
			name:       "Disallowed quality - 480p",
			source:     "DVDRip",
			resolution: 480,
			allowed:    false,
			minScore:   -1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			torrent := &types.TorrentInfo{
				ReleaseInfo: types.ReleaseInfo{
					Source:     tt.source,
					Resolution: tt.resolution,
				},
			}
			ctx := ScoringContext{
				QualityProfile: &profile,
			}
			breakdown := &types.ScoreBreakdown{}

			score := scorer.calculateQualityScore(torrent, ctx, breakdown)

			if tt.allowed {
				if score < tt.minScore {
					t.Errorf("Quality score = %f, want >= %f", score, tt.minScore)
				}
			} else {
				if score != tt.minScore {
					t.Errorf("Disallowed quality score = %f, want %f", score, tt.minScore)
				}
			}
		})
	}
}

func TestScorer_FullScoring(t *testing.T) {
	scorer := NewDefaultScorer()
	now := time.Now()

	// Create an "Any" profile
	profile := quality.DefaultProfile()

	torrent := types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			Title:       "Movie.Name.2024.1080p.BluRay.x264",
			Source:      "BluRay",
			Resolution:  1080,
			PublishDate: now.AddDate(0, 0, -3), // 3 days ago
			IndexerID:   1,
		},
		Seeders:              100,
		Leechers:             10,
		DownloadVolumeFactor: 0, // Freeleech
	}

	ctx := ScoringContext{
		QualityProfile:    &profile,
		SearchYear:        2024,
		IndexerPriorities: map[int64]int{1: 25}, // Good priority
		Now:               now,
	}

	scorer.ScoreTorrent(&torrent, ctx)

	// Should have high scores across the board
	if torrent.Score < 100 {
		t.Errorf("Expected high total score (>100), got %f", torrent.Score)
	}

	if torrent.NormalizedScore < 50 {
		t.Errorf("Expected normalized score >= 50, got %d", torrent.NormalizedScore)
	}

	if torrent.ScoreBreakdown == nil {
		t.Fatal("Expected score breakdown, got nil")
	}

	// Verify breakdown components are reasonable
	if torrent.ScoreBreakdown.QualityScore < 60 {
		t.Errorf("Expected quality score >= 60, got %f", torrent.ScoreBreakdown.QualityScore)
	}

	if torrent.ScoreBreakdown.HealthScore < 40 {
		t.Errorf("Expected health score >= 40 (freeleech), got %f", torrent.ScoreBreakdown.HealthScore)
	}

	if torrent.ScoreBreakdown.IndexerScore < 10 {
		t.Errorf("Expected indexer score >= 10, got %f", torrent.ScoreBreakdown.IndexerScore)
	}

	if torrent.ScoreBreakdown.MatchScore < 10 {
		t.Errorf("Expected match score >= 10 (year match), got %f", torrent.ScoreBreakdown.MatchScore)
	}

	if torrent.ScoreBreakdown.AgeScore != 0 {
		t.Errorf("Expected age score = 0 (fresh release), got %f", torrent.ScoreBreakdown.AgeScore)
	}
}

func TestScorer_SortByScore(t *testing.T) {
	scorer := NewDefaultScorer()
	now := time.Now()
	profile := quality.DefaultProfile()

	torrents := []types.TorrentInfo{
		{
			ReleaseInfo: types.ReleaseInfo{
				Title:       "Movie.480p.DVDRip",
				Source:      "DVDRip",
				Resolution:  480,
				PublishDate: now,
				IndexerID:   1,
			},
			Seeders:              10,
			DownloadVolumeFactor: 1,
		},
		{
			ReleaseInfo: types.ReleaseInfo{
				Title:       "Movie.1080p.BluRay",
				Source:      "BluRay",
				Resolution:  1080,
				PublishDate: now,
				IndexerID:   1,
			},
			Seeders:              100,
			DownloadVolumeFactor: 0, // Freeleech
		},
		{
			ReleaseInfo: types.ReleaseInfo{
				Title:       "Movie.720p.HDTV",
				Source:      "HDTV",
				Resolution:  720,
				PublishDate: now,
				IndexerID:   1,
			},
			Seeders:              50,
			DownloadVolumeFactor: 1,
		},
	}

	ctx := ScoringContext{
		QualityProfile:    &profile,
		IndexerPriorities: map[int64]int{1: 50},
		Now:               now,
	}

	scorer.ScoreTorrents(torrents, ctx)

	// Should be sorted by score descending
	// 1080p BluRay freeleech should be first
	// 720p HDTV should be second
	// 480p DVDRip should be last
	if torrents[0].Resolution != 1080 {
		t.Errorf("Expected 1080p first, got %dp", torrents[0].Resolution)
	}

	if torrents[1].Resolution != 720 {
		t.Errorf("Expected 720p second, got %dp", torrents[1].Resolution)
	}

	if torrents[2].Resolution != 480 {
		t.Errorf("Expected 480p last, got %dp", torrents[2].Resolution)
	}

	// Verify scores are in descending order
	for i := 0; i < len(torrents)-1; i++ {
		if torrents[i].Score < torrents[i+1].Score {
			t.Errorf("Torrents not sorted correctly: score[%d]=%f < score[%d]=%f",
				i, torrents[i].Score, i+1, torrents[i+1].Score)
		}
	}
}
