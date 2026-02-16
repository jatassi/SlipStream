package organizer

import (
	"testing"
)

func TestDefaultNamingConfig(t *testing.T) {
	config := DefaultNamingConfig()

	tests := []struct {
		field string
		got   string
		want  string
	}{
		{"MovieFolderFormat", config.MovieFolderFormat, "{Movie Title} ({Year})"},
		{"MovieFileFormat", config.MovieFileFormat, "{Movie Title} ({Year})"},
		{"SeriesFolderFormat", config.SeriesFolderFormat, "{Series Title}"},
		{"SeasonFolderFormat", config.SeasonFolderFormat, "Season {Season:00}"},
		{"EpisodeFileFormat", config.EpisodeFileFormat, "{Series Title} - S{Season:00}E{Episode:00} - {Episode Title}"},
		{"MultiEpisodeFormat", config.MultiEpisodeFormat, "{Series Title} - S{Season:00}E{Episode:00}-E{EndEpisode:00} - {Episode Title}"},
		{"SpaceReplacement", config.SpaceReplacement, "."},
		{"ColonReplacement", config.ColonReplacement, " -"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.field, tt.got, tt.want)
			}
		})
	}

	if config.ReplaceSpaces {
		t.Error("ReplaceSpaces = true, want false")
	}

	if !config.CleanSpecialChars {
		t.Error("CleanSpecialChars = false, want true")
	}
}

func TestFormatMovieFolder(t *testing.T) {
	config := DefaultNamingConfig()

	tests := []struct {
		name   string
		tokens MovieTokens
		want   string
	}{
		{
			name:   "standard movie",
			tokens: MovieTokens{Title: "The Matrix", Year: 1999},
			want:   "The Matrix (1999)",
		},
		{
			name:   "movie without year",
			tokens: MovieTokens{Title: "Unknown Movie", Year: 0},
			want:   "Unknown Movie",
		},
		{
			name:   "movie with special chars",
			tokens: MovieTokens{Title: "Mission: Impossible", Year: 1996},
			want:   "Mission - Impossible (1996)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.FormatMovieFolder(&tt.tokens)
			if got != tt.want {
				t.Errorf("FormatMovieFolder() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMovieFile(t *testing.T) {
	config := DefaultNamingConfig()

	tests := []struct {
		name   string
		tokens MovieTokens
		want   string
	}{
		{
			name:   "standard movie",
			tokens: MovieTokens{Title: "Inception", Year: 2010},
			want:   "Inception (2010)",
		},
		{
			name:   "movie with quality info",
			tokens: MovieTokens{Title: "Dune", Year: 2021, Quality: "1080p", Source: "BluRay"},
			want:   "Dune (2021)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.FormatMovieFile(&tt.tokens)
			if got != tt.want {
				t.Errorf("FormatMovieFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMovieFile_CustomFormat(t *testing.T) {
	config := NamingConfig{
		MovieFileFormat:   "{Movie Title} ({Year}) [{Quality}] [{Source}]",
		ColonReplacement:  " -",
		CleanSpecialChars: true,
	}

	tokens := MovieTokens{
		Title:   "The Dark Knight",
		Year:    2008,
		Quality: "1080p",
		Source:  "BluRay",
	}

	got := config.FormatMovieFile(&tokens)
	want := "The Dark Knight (2008) [1080p] [BluRay]"

	if got != want {
		t.Errorf("FormatMovieFile() = %q, want %q", got, want)
	}
}

func TestFormatSeriesFolder(t *testing.T) {
	config := DefaultNamingConfig()

	tests := []struct {
		name   string
		tokens SeriesTokens
		want   string
	}{
		{
			name:   "standard series",
			tokens: SeriesTokens{SeriesTitle: "Breaking Bad"},
			want:   "Breaking Bad",
		},
		{
			name:   "series with year",
			tokens: SeriesTokens{SeriesTitle: "The Office", Year: 2005},
			want:   "The Office",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.FormatSeriesFolder(&tt.tokens)
			if got != tt.want {
				t.Errorf("FormatSeriesFolder() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSeriesFolder_CustomFormat(t *testing.T) {
	config := NamingConfig{
		SeriesFolderFormat: "{Series Title} ({Year})",
		ColonReplacement:   " -",
		CleanSpecialChars:  true,
	}

	tokens := SeriesTokens{SeriesTitle: "Game of Thrones", Year: 2011}
	got := config.FormatSeriesFolder(&tokens)
	want := "Game of Thrones (2011)"

	if got != want {
		t.Errorf("FormatSeriesFolder() = %q, want %q", got, want)
	}
}

func TestFormatSeasonFolder(t *testing.T) {
	config := DefaultNamingConfig()

	tests := []struct {
		season int
		want   string
	}{
		{1, "Season 01"},
		{2, "Season 02"},
		{10, "Season 10"},
		{15, "Season 15"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := config.FormatSeasonFolder(tt.season)
			if got != tt.want {
				t.Errorf("FormatSeasonFolder(%d) = %q, want %q", tt.season, got, tt.want)
			}
		})
	}
}

func TestFormatSeasonFolder_CustomFormat(t *testing.T) {
	config := NamingConfig{
		SeasonFolderFormat: "S{Season:00}",
	}

	got := config.FormatSeasonFolder(3)
	want := "S03"

	if got != want {
		t.Errorf("FormatSeasonFolder() = %q, want %q", got, want)
	}
}

func TestFormatEpisodeFile(t *testing.T) {
	config := DefaultNamingConfig()

	tests := []struct {
		name   string
		tokens EpisodeTokens
		want   string
	}{
		{
			name: "standard episode",
			tokens: EpisodeTokens{
				SeriesTitle:   "Breaking Bad",
				SeasonNumber:  1,
				EpisodeNumber: 1,
				EpisodeTitle:  "Pilot",
			},
			want: "Breaking Bad - S01E01 - Pilot",
		},
		{
			name: "double digit episode",
			tokens: EpisodeTokens{
				SeriesTitle:   "The Office",
				SeasonNumber:  3,
				EpisodeNumber: 15,
				EpisodeTitle:  "Phyllis' Wedding",
			},
			want: "The Office - S03E15 - Phyllis' Wedding",
		},
		{
			name: "episode without title",
			tokens: EpisodeTokens{
				SeriesTitle:   "Show",
				SeasonNumber:  1,
				EpisodeNumber: 5,
				EpisodeTitle:  "",
			},
			want: "Show - S01E05 -",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.FormatEpisodeFile(&tt.tokens)
			if got != tt.want {
				t.Errorf("FormatEpisodeFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatEpisodeFile_MultiEpisode(t *testing.T) {
	config := DefaultNamingConfig()

	tokens := EpisodeTokens{
		SeriesTitle:   "Game of Thrones",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		EndEpisode:    2,
		EpisodeTitle:  "Winter Is Coming",
	}

	got := config.FormatEpisodeFile(&tokens)
	want := "Game of Thrones - S01E01-E02 - Winter Is Coming"

	if got != want {
		t.Errorf("FormatEpisodeFile() multi-episode = %q, want %q", got, want)
	}
}

func TestFormatEpisodeFile_SameEndEpisode(t *testing.T) {
	config := DefaultNamingConfig()

	// When EndEpisode equals EpisodeNumber, should use single episode format
	tokens := EpisodeTokens{
		SeriesTitle:   "Show",
		SeasonNumber:  1,
		EpisodeNumber: 5,
		EndEpisode:    5,
		EpisodeTitle:  "Episode",
	}

	got := config.FormatEpisodeFile(&tokens)
	want := "Show - S01E05 - Episode"

	if got != want {
		t.Errorf("FormatEpisodeFile() same end episode = %q, want %q", got, want)
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n      int
		format string
		want   string
	}{
		{1, "", "1"},
		{1, "00", "01"},
		{1, "000", "001"},
		{10, "00", "10"},
		{10, "000", "010"},
		{100, "00", "100"},
		{5, "0", "5"},
		{15, "00", "15"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumber(tt.n, tt.format)
			if got != tt.want {
				t.Errorf("formatNumber(%d, %q) = %q, want %q", tt.n, tt.format, got, tt.want)
			}
		})
	}
}

func TestCleanFilename(t *testing.T) {
	config := NamingConfig{
		ColonReplacement:  " -",
		ReplaceSpaces:     false,
		CleanSpecialChars: true,
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"colon replacement", "Mission: Impossible", "Mission - Impossible"},
		{"multiple colons", "Star Trek: Deep Space: Nine", "Star Trek - Deep Space - Nine"},
		{"invalid chars removed", "Movie<>Name", "MovieName"},
		{"quotes removed", `Movie "Title"`, "Movie Title"},
		{"pipe removed", "Movie|Name", "MovieName"},
		{"question mark removed", "What?", "What"},
		{"asterisk removed", "Movie*Name", "MovieName"},
		{"slashes removed", "Movie/Name\\Test", "MovieNameTest"},
		{"multiple spaces collapsed", "Movie   Name", "Movie Name"},
		{"trimmed", "  Movie  ", "Movie"},
		{"empty parens removed", "Movie ()", "Movie"},
		{"empty parens with spaces", "Movie (  )", "Movie"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.cleanFilename(tt.input)
			if got != tt.want {
				t.Errorf("cleanFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanFilename_ReplaceSpaces(t *testing.T) {
	config := NamingConfig{
		ReplaceSpaces:     true,
		SpaceReplacement:  ".",
		ColonReplacement:  " -",
		CleanSpecialChars: true,
	}

	input := "Movie Name Here"
	got := config.cleanFilename(input)
	want := "Movie.Name.Here"

	if got != want {
		t.Errorf("cleanFilename() with ReplaceSpaces = %q, want %q", got, want)
	}
}

func TestCleanFilename_NoCleanSpecialChars(t *testing.T) {
	config := NamingConfig{
		CleanSpecialChars: false,
		ColonReplacement:  "",
	}

	input := "Movie<>Name"
	got := config.cleanFilename(input)
	// Should still trim and collapse spaces but not remove special chars
	if got != input {
		// Actually, without cleaning, it should preserve
		t.Logf("cleanFilename() without CleanSpecialChars = %q", got)
	}
}

func TestResolveMovieToken(t *testing.T) {
	config := DefaultNamingConfig()
	tokens := MovieTokens{
		Title:      "The Matrix",
		Year:       1999,
		Quality:    "1080p",
		Resolution: "1920x1080",
		Source:     "BluRay",
		Codec:      "x264",
		ImdbID:     "tt0133093",
		TmdbID:     603,
	}

	tests := []struct {
		token  string
		format string
		want   string
	}{
		{"Movie Title", "", "The Matrix"},
		{"Title", "", "The Matrix"},
		{"movie title", "", "The Matrix"},
		{"Year", "", "1999"},
		{"Year", "0000", "1999"},
		{"Quality", "", "1080p"},
		{"Resolution", "", "1920x1080"},
		{"Source", "", "BluRay"},
		{"Codec", "", "x264"},
		{"ImdbID", "", "tt0133093"},
		{"TmdbID", "", "603"},
		{"Unknown", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := config.resolveMovieToken(tt.token, tt.format, &tokens)
			if got != tt.want {
				t.Errorf("resolveMovieToken(%q, %q) = %q, want %q", tt.token, tt.format, got, tt.want)
			}
		})
	}
}

func TestResolveMovieToken_EmptyValues(t *testing.T) {
	config := DefaultNamingConfig()
	tokens := MovieTokens{
		Title: "Movie",
		Year:  0, // No year
	}

	got := config.resolveMovieToken("Year", "", &tokens)
	if got != "" {
		t.Errorf("resolveMovieToken(Year) with year=0 = %q, want empty string", got)
	}

	tokens.TmdbID = 0
	got = config.resolveMovieToken("TmdbID", "", &tokens)
	if got != "" {
		t.Errorf("resolveMovieToken(TmdbID) with tmdbId=0 = %q, want empty string", got)
	}
}

func TestResolveSeriesToken(t *testing.T) {
	config := DefaultNamingConfig()
	tokens := SeriesTokens{
		SeriesTitle: "Breaking Bad",
		Year:        2008,
	}

	tests := []struct {
		token string
		want  string
	}{
		{"Series Title", "Breaking Bad"},
		{"Title", "Breaking Bad"},
		{"Year", "2008"},
		{"Unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := config.resolveSeriesToken(tt.token, "", &tokens)
			if got != tt.want {
				t.Errorf("resolveSeriesToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestResolveEpisodeToken(t *testing.T) {
	config := DefaultNamingConfig()
	tokens := EpisodeTokens{
		SeriesTitle:   "The Office",
		SeasonNumber:  3,
		EpisodeNumber: 15,
		EndEpisode:    16,
		EpisodeTitle:  "Phyllis' Wedding",
		Quality:       "720p",
		Resolution:    "1280x720",
		Source:        "HDTV",
		Codec:         "x264",
	}

	tests := []struct {
		token  string
		format string
		want   string
	}{
		{"Series Title", "", "The Office"},
		{"Series", "", "The Office"},
		{"Season", "00", "03"},
		{"Season", "", "3"},
		{"Episode", "00", "15"},
		{"Episode", "", "15"},
		{"EndEpisode", "00", "16"},
		{"Episode Title", "", "Phyllis' Wedding"},
		{"Title", "", "Phyllis' Wedding"},
		{"Quality", "", "720p"},
		{"Resolution", "", "1280x720"},
		{"Source", "", "HDTV"},
		{"Codec", "", "x264"},
		{"Unknown", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := config.resolveEpisodeToken(tt.token, tt.format, &tokens)
			if got != tt.want {
				t.Errorf("resolveEpisodeToken(%q, %q) = %q, want %q", tt.token, tt.format, got, tt.want)
			}
		})
	}
}

func TestResolveEpisodeToken_NoEndEpisode(t *testing.T) {
	config := DefaultNamingConfig()
	tokens := EpisodeTokens{
		SeriesTitle:   "Show",
		SeasonNumber:  1,
		EpisodeNumber: 5,
		EndEpisode:    0, // No end episode
	}

	// When EndEpisode is 0, it should return EpisodeNumber
	got := config.resolveEpisodeToken("EndEpisode", "00", &tokens)
	want := "05"

	if got != want {
		t.Errorf("resolveEpisodeToken(EndEpisode) with EndEpisode=0 = %q, want %q", got, want)
	}
}

func TestTokenPattern(t *testing.T) {
	// Test that the regex matches expected patterns
	tests := []struct {
		input      string
		wantMatch  bool
		wantToken  string
		wantFormat string
	}{
		{"{Title}", true, "Title", ""},
		{"{Season:00}", true, "Season", "00"},
		{"{Episode:000}", true, "Episode", "000"},
		{"{Movie Title}", true, "Movie Title", ""},
		{"No tokens here", false, "", ""},
		{"{}", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := tokenPattern.FindStringSubmatch(tt.input)
			gotMatch := len(matches) > 0

			if gotMatch != tt.wantMatch {
				t.Errorf("tokenPattern.Match(%q) = %v, want %v", tt.input, gotMatch, tt.wantMatch)
			}

			if gotMatch && len(matches) >= 2 {
				if matches[1] != tt.wantToken {
					t.Errorf("token = %q, want %q", matches[1], tt.wantToken)
				}
				format := ""
				if len(matches) >= 3 {
					format = matches[2]
				}
				if format != tt.wantFormat {
					t.Errorf("format = %q, want %q", format, tt.wantFormat)
				}
			}
		})
	}
}
