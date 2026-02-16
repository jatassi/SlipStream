package renamer

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewResolver(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)
	if r == nil {
		t.Error("NewResolver() returned nil")
	}
}

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	if !s.RenameEpisodes {
		t.Error("RenameEpisodes should be true by default")
	}
	if !s.ReplaceIllegalCharacters {
		t.Error("ReplaceIllegalCharacters should be true by default")
	}
	if s.ColonReplacement != ColonSmart {
		t.Errorf("ColonReplacement = %v, want %v", s.ColonReplacement, ColonSmart)
	}
	if s.StandardEpisodeFormat == "" {
		t.Error("StandardEpisodeFormat should not be empty")
	}
	if s.MultiEpisodeStyle != StyleExtend {
		t.Errorf("MultiEpisodeStyle = %v, want %v", s.MultiEpisodeStyle, StyleExtend)
	}
}

// ===== EPISODE FILENAME RESOLUTION =====

func TestResolveEpisodeFilename(t *testing.T) {
	settings := DefaultSettings()
	settings.StandardEpisodeFormat = "{Series Title} - S{season:00}E{episode:00} - {Episode Title}"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		SeriesTitle:   "Breaking Bad",
		SeasonNumber:  2,
		EpisodeNumber: 5,
		EpisodeTitle:  "Breakage",
	}

	got, err := r.ResolveEpisodeFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("ResolveEpisodeFilename() error = %v", err)
	}

	want := "Breaking Bad - S02E05 - Breakage.mkv"
	if got != want {
		t.Errorf("ResolveEpisodeFilename() = %q, want %q", got, want)
	}
}

func TestResolveEpisodeFilename_RenameDisabled(t *testing.T) {
	settings := DefaultSettings()
	settings.RenameEpisodes = false
	r := NewResolver(&settings)

	ctx := &TokenContext{
		OriginalFile: "original.file.mkv",
	}

	got, err := r.ResolveEpisodeFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("ResolveEpisodeFilename() error = %v", err)
	}

	if got != "original.file.mkv" {
		t.Errorf("ResolveEpisodeFilename() = %q, want original filename", got)
	}
}

func TestResolveEpisodeFilename_EmptyPattern(t *testing.T) {
	settings := DefaultSettings()
	settings.StandardEpisodeFormat = ""
	r := NewResolver(&settings)

	ctx := &TokenContext{SeriesTitle: "Test"}

	_, err := r.ResolveEpisodeFilename(ctx, ".mkv")
	if !errors.Is(err, ErrEmptyPattern) {
		t.Errorf("ResolveEpisodeFilename() error = %v, want ErrEmptyPattern", err)
	}
}

func TestResolveEpisodeFilename_DailyFormat(t *testing.T) {
	settings := DefaultSettings()
	settings.DailyEpisodeFormat = "{Series Title} - {Air-Date}"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		SeriesTitle: "The Daily Show",
		SeriesType:  "daily",
		AirDate:     time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
	}

	got, err := r.ResolveEpisodeFilename(ctx, ".mp4")
	if err != nil {
		t.Fatalf("ResolveEpisodeFilename() error = %v", err)
	}

	want := "The Daily Show - 2024-03-15.mp4"
	if got != want {
		t.Errorf("ResolveEpisodeFilename() = %q, want %q", got, want)
	}
}

func TestResolveEpisodeFilename_AnimeFormat(t *testing.T) {
	settings := DefaultSettings()
	settings.AnimeEpisodeFormat = "{Series Title} - {absolute:000} - {Episode Title}"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		SeriesTitle:    "Naruto",
		SeriesType:     "anime",
		AbsoluteNumber: 42,
		EpisodeTitle:   "The Great Battle",
	}

	got, err := r.ResolveEpisodeFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("ResolveEpisodeFilename() error = %v", err)
	}

	want := "Naruto - 042 - The Great Battle.mkv"
	if got != want {
		t.Errorf("ResolveEpisodeFilename() = %q, want %q", got, want)
	}
}

func TestResolveEpisodeFilename_WithQuality(t *testing.T) {
	settings := DefaultSettings()
	settings.StandardEpisodeFormat = "{Series Title} - S{season:00}E{episode:00} - {Quality Title}"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		SeriesTitle:   "Test Show",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Source:        "BluRay",
		Quality:       "1080p",
	}

	got, err := r.ResolveEpisodeFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("ResolveEpisodeFilename() error = %v", err)
	}

	want := "Test Show - S01E01 - BluRay-1080p.mkv"
	if got != want {
		t.Errorf("ResolveEpisodeFilename() = %q, want %q", got, want)
	}
}

func TestResolveEpisodeFilename_ExtensionHandling(t *testing.T) {
	settings := DefaultSettings()
	settings.StandardEpisodeFormat = "{Series Title}"
	r := NewResolver(&settings)

	ctx := &TokenContext{SeriesTitle: "Test"}

	tests := []struct {
		name string
		ext  string
		want string
	}{
		{"with dot", ".mkv", "Test.mkv"},
		{"without dot", "mkv", "Test.mkv"},
		{"empty", "", "Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveEpisodeFilename(ctx, tt.ext)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}

// ===== MOVIE FILENAME RESOLUTION =====

func TestResolveMovieFilename(t *testing.T) {
	settings := DefaultSettings()
	settings.MovieFileFormat = "{Movie Title} ({Year}) - {Quality Title}"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		MovieTitle: "The Matrix",
		MovieYear:  1999,
		Source:     "BluRay",
		Quality:    "1080p",
	}

	got, err := r.ResolveMovieFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("ResolveMovieFilename() error = %v", err)
	}

	want := "The Matrix (1999) - BluRay-1080p.mkv"
	if got != want {
		t.Errorf("ResolveMovieFilename() = %q, want %q", got, want)
	}
}

func TestResolveMovieFilename_RenameDisabled(t *testing.T) {
	settings := DefaultSettings()
	settings.RenameMovies = false
	r := NewResolver(&settings)

	ctx := &TokenContext{
		OriginalFile: "the.matrix.1999.mkv",
	}

	got, err := r.ResolveMovieFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if got != "the.matrix.1999.mkv" {
		t.Errorf("got = %q, want original filename", got)
	}
}

func TestResolveMovieFilename_WithEdition(t *testing.T) {
	settings := DefaultSettings()
	settings.MovieFileFormat = "{Movie Title} ({Year}) {Edition Tags}"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		MovieTitle:  "Blade Runner",
		MovieYear:   1982,
		EditionTags: "Final Cut",
	}

	got, err := r.ResolveMovieFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	want := "Blade Runner (1982) Final Cut.mkv"
	if got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}

// ===== FOLDER NAME RESOLUTION =====

func TestResolveSeriesFolderName(t *testing.T) {
	settings := DefaultSettings()
	settings.SeriesFolderFormat = "{Series Title} ({Year})"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		SeriesTitle: "Breaking Bad",
		SeriesYear:  2008,
	}

	got, err := r.ResolveSeriesFolderName(ctx)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	want := "Breaking Bad (2008)"
	if got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}

func TestResolveSeriesFolderName_DefaultPattern(t *testing.T) {
	settings := DefaultSettings()
	settings.SeriesFolderFormat = ""
	r := NewResolver(&settings)

	ctx := &TokenContext{SeriesTitle: "Test Show"}

	got, err := r.ResolveSeriesFolderName(ctx)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if got != "Test Show" {
		t.Errorf("got = %q, want %q", got, "Test Show")
	}
}

func TestResolveSeasonFolderName(t *testing.T) {
	settings := DefaultSettings()
	settings.SeasonFolderFormat = "Season {season:00}"
	r := NewResolver(&settings)

	tests := []struct {
		name   string
		season int
		want   string
	}{
		{"season 1", 1, "Season 01"},
		{"season 10", 10, "Season 10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.ResolveSeasonFolderName(tt.season)
			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveSeasonFolderName_Specials(t *testing.T) {
	settings := DefaultSettings()
	settings.SpecialsFolderFormat = "Extras"
	r := NewResolver(&settings)

	got := r.ResolveSeasonFolderName(0)
	if got != "Extras" {
		t.Errorf("got = %q, want %q", got, "Extras")
	}
}

func TestResolveSeasonFolderName_SpecialsDefault(t *testing.T) {
	settings := DefaultSettings()
	settings.SpecialsFolderFormat = ""
	r := NewResolver(&settings)

	got := r.ResolveSeasonFolderName(0)
	if got != "Specials" {
		t.Errorf("got = %q, want %q", got, "Specials")
	}
}

func TestResolveMovieFolderName(t *testing.T) {
	settings := DefaultSettings()
	settings.MovieFolderFormat = "{Movie Title} ({Year})"
	r := NewResolver(&settings)

	ctx := &TokenContext{
		MovieTitle: "The Matrix",
		MovieYear:  1999,
	}

	got, err := r.ResolveMovieFolderName(ctx)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	want := "The Matrix (1999)"
	if got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}

// ===== FULL PATH RESOLUTION =====

func TestResolveFullPath(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)

	got, err := r.ResolveFullPath("/media/tv", "Series/Season 01", "episode.mkv")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// Path separator is OS-dependent, so just check it contains all parts
	if !strings.Contains(got, "media") || !strings.Contains(got, "episode.mkv") {
		t.Errorf("got = %q, expected path with all components", got)
	}
}

func TestResolveFullPath_TooLong(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)

	// Create a path that exceeds 260 characters
	longName := strings.Repeat("a", 300)

	_, err := r.ResolveFullPath("/media", longName, "file.mkv")
	if err == nil {
		t.Error("expected error for path too long")
	}
	if !strings.Contains(err.Error(), "exceeds maximum length") {
		t.Errorf("error = %v, want path length error", err)
	}
}

// ===== PATTERN VALIDATION =====

func TestValidatePattern(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)

	validPatterns := []string{
		"{Series Title} - S{season:00}E{episode:00}",
		"{Movie Title} ({Year})",
		"Season {season}",
		"plain text",
		"{Quality Full}",
	}

	for _, pattern := range validPatterns {
		t.Run("valid_"+pattern, func(t *testing.T) {
			err := r.ValidatePattern(pattern)
			if err != nil {
				t.Errorf("ValidatePattern(%q) error = %v, want nil", pattern, err)
			}
		})
	}
}

func TestValidatePattern_Errors(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)

	tests := []struct {
		name    string
		pattern string
		wantErr error
	}{
		{"empty", "", ErrEmptyPattern},
		{"unbalanced open", "{Series Title", ErrInvalidPattern},
		{"unbalanced close", "Series Title}", ErrInvalidPattern},
		{"invalid token", "{Invalid Token}", ErrInvalidToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.ValidatePattern(tt.pattern)
			if err == nil {
				t.Errorf("ValidatePattern(%q) = nil, want error", tt.pattern)
			}
		})
	}
}

func TestHasBalancedBraces(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"{}", true},
		{"{a}", true},
		{"{{a}}", true},
		{"{a}{b}", true},
		{"no braces", true},
		{"{", false},
		{"}", false},
		{"{a", false},
		{"a}", false},
		{"{{}", false},
		{"{}}", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := hasBalancedBraces(tt.input)
			if got != tt.want {
				t.Errorf("hasBalancedBraces(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ===== PREVIEW AND SAMPLE =====

func TestPreviewPattern(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)
	ctx := GetSampleContext()

	pattern := "{Series Title} - S{season:00}E{episode:00}"
	got, err := r.PreviewPattern(pattern, ctx)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	want := "The Series Title - S01E01"
	if got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}

func TestPreviewPattern_InvalidPattern(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)
	ctx := GetSampleContext()

	_, err := r.PreviewPattern("{Invalid}", ctx)
	if err == nil {
		t.Error("expected error for invalid pattern")
	}
}

func TestGetSampleContext(t *testing.T) {
	ctx := GetSampleContext()

	if ctx.SeriesTitle == "" {
		t.Error("SeriesTitle should not be empty")
	}
	if ctx.SeriesYear == 0 {
		t.Error("SeriesYear should not be 0")
	}
	if ctx.SeasonNumber == 0 {
		t.Error("SeasonNumber should not be 0")
	}
	if ctx.EpisodeNumber == 0 {
		t.Error("EpisodeNumber should not be 0")
	}
	if ctx.Quality == "" {
		t.Error("Quality should not be empty")
	}
}

// ===== TOKEN BREAKDOWN =====

func TestGetTokenBreakdown(t *testing.T) {
	settings := DefaultSettings()
	r := NewResolver(&settings)
	ctx := &TokenContext{
		SeriesTitle:   "Test Show",
		SeasonNumber:  1,
		EpisodeNumber: 5,
		EpisodeTitle:  "",
	}

	pattern := "{Series Title} S{season:00}E{episode:00} {Episode Title}"
	breakdown := r.GetTokenBreakdown(pattern, ctx)

	if len(breakdown) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(breakdown))
	}

	// Check Series Title
	if breakdown[0].Token != "{Series Title}" {
		t.Errorf("token 0 = %q, want {Series Title}", breakdown[0].Token)
	}
	if breakdown[0].Value != "Test Show" {
		t.Errorf("value 0 = %q, want Test Show", breakdown[0].Value)
	}
	if breakdown[0].Empty {
		t.Error("Series Title should not be empty")
	}

	// Check Episode Title is empty
	if !breakdown[3].Empty {
		t.Error("Episode Title should be marked empty")
	}

	// Check modifier detection
	if !breakdown[1].Modified {
		t.Error("season:00 should be marked as modified")
	}
}

// ===== EMPTY TOKEN CLEANUP =====

func TestResolvePattern_EmptyTokenCleanup(t *testing.T) {
	settings := DefaultSettings()
	settings.StandardEpisodeFormat = "{Series Title} - {Episode Title} - {Quality Title}"
	r := NewResolver(&settings)

	// Episode title is empty
	ctx := &TokenContext{
		SeriesTitle: "Test Show",
		Source:      "WEBDL",
		Quality:     "1080p",
	}

	got, err := r.ResolveEpisodeFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// Should not have orphaned " - - " in the result
	if strings.Contains(got, " - - ") {
		t.Errorf("got = %q, should not contain orphaned separators", got)
	}
}

func TestCleanupOrphanedSeparators(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Test - - Show", "Test Show"},
		{"Test  Show", "Test Show"},
		{" - Test", "Test"},
		{"Test - ", "Test"},
		{"a . . b", "a b"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanupOrphanedSeparators(tt.input)
			got = strings.TrimSpace(got)
			if got != tt.want {
				t.Errorf("cleanupOrphanedSeparators(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== MULTI-EPISODE FORMATTING IN RESOLVER =====

func TestResolveEpisodeFilename_MultiEpisode(t *testing.T) {
	settings := DefaultSettings()
	settings.StandardEpisodeFormat = "{Series Title} - S{season:00}E{episode:00}"
	settings.MultiEpisodeStyle = StyleExtend
	r := NewResolver(&settings)

	ctx := &TokenContext{
		SeriesTitle:    "Test Show",
		SeasonNumber:   1,
		EpisodeNumber:  1,
		EpisodeNumbers: []int{1, 2, 3},
	}

	got, err := r.ResolveEpisodeFilename(ctx, ".mkv")
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	want := "Test Show - S01E01-02-03.mkv"
	if got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}

// ===== CASE MODE APPLICATION =====

func TestResolveEpisodeFilename_CaseMode(t *testing.T) {
	tests := []struct {
		name string
		mode CaseMode
		want string
	}{
		{"default", CaseDefault, "Test Show - S01E01.mkv"},
		{"upper", CaseUpper, "TEST SHOW - S01E01.mkv"},
		{"lower", CaseLower, "test show - s01e01.mkv"},
		{"title", CaseTitle, "Test Show - S01E01.mkv"}, // S##E## patterns preserved in uppercase
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := DefaultSettings()
			settings.StandardEpisodeFormat = "{Series Title} - S{season:00}E{episode:00}"
			settings.CaseMode = tt.mode
			r := NewResolver(&settings)

			ctx := &TokenContext{
				SeriesTitle:   "Test Show",
				SeasonNumber:  1,
				EpisodeNumber: 1,
			}

			got, err := r.ResolveEpisodeFilename(ctx, ".mkv")
			if err != nil {
				t.Fatalf("error = %v", err)
			}

			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}
