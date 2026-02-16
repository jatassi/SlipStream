package renamer

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MaxPathLength is the Windows MAX_PATH limit.
	MaxPathLength      = 260
	defaultSeriesTitle = "{Series Title}"
)

var (
	ErrPathTooLong    = errors.New("computed path exceeds maximum length")
	ErrInvalidToken   = errors.New("invalid token in pattern")
	ErrEmptyPattern   = errors.New("pattern cannot be empty")
	ErrInvalidPattern = errors.New("invalid pattern syntax")
)

// Settings holds the renaming configuration.
type Settings struct {
	// TV Settings
	RenameEpisodes           bool
	ReplaceIllegalCharacters bool
	ColonReplacement         ColonReplacement
	CustomColonReplacement   string

	// Episode format patterns
	StandardEpisodeFormat string
	DailyEpisodeFormat    string
	AnimeEpisodeFormat    string

	// Folder patterns
	SeriesFolderFormat   string
	SeasonFolderFormat   string
	SpecialsFolderFormat string

	// Multi-episode
	MultiEpisodeStyle MultiEpisodeStyle

	// Movie Settings
	RenameMovies      bool
	MovieFolderFormat string
	MovieFileFormat   string

	// Case transformation
	CaseMode CaseMode
}

// DefaultSettings returns the default renaming settings matching Sonarr defaults.
func DefaultSettings() Settings {
	return Settings{
		RenameEpisodes:           true,
		ReplaceIllegalCharacters: true,
		ColonReplacement:         ColonSmart,

		StandardEpisodeFormat: "{Series Title} - S{season:00}E{episode:00} - {Quality Title} {MediaInfo VideoDynamicRangeType}",
		DailyEpisodeFormat:    "{Series Title} - {Air-Date} - {Episode Title} {Quality Full}",
		AnimeEpisodeFormat:    "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",

		SeriesFolderFormat:   "{Series Title}",
		SeasonFolderFormat:   "Season {season}",
		SpecialsFolderFormat: "Specials",

		MultiEpisodeStyle: StyleExtend,

		RenameMovies:      true,
		MovieFolderFormat: "{Movie Title} ({Year})",
		MovieFileFormat:   "{Movie Title} ({Year}) - {Quality Title}",

		CaseMode: CaseDefault,
	}
}

// Resolver handles pattern resolution for filenames and folders.
type Resolver struct {
	settings Settings
}

// NewResolver creates a new Resolver with the given settings.
func NewResolver(settings *Settings) *Resolver {
	return &Resolver{settings: *settings}
}

// ResolveEpisodeFilename resolves the full episode filename from a pattern.
func (r *Resolver) ResolveEpisodeFilename(ctx *TokenContext, extension string) (string, error) {
	if !r.settings.RenameEpisodes {
		return ctx.OriginalFile, nil
	}

	pattern := r.getEpisodePattern(ctx.SeriesType)
	if pattern == "" {
		return "", ErrEmptyPattern
	}

	filename, err := r.resolvePattern(pattern, ctx)
	if err != nil {
		return "", err
	}

	// Handle multi-episode formatting
	if len(ctx.EpisodeNumbers) > 1 {
		filename = r.applyMultiEpisodeFormat(filename, ctx)
	}

	// Sanitize the filename
	filename = SanitizeFilename(
		filename,
		r.settings.ReplaceIllegalCharacters,
		r.settings.ColonReplacement,
		r.settings.CustomColonReplacement,
	)

	// Apply case transformation
	filename = ApplyCase(filename, r.settings.CaseMode)

	// Add extension (preserve original exactly)
	if extension != "" {
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		filename += extension
	}

	return filename, nil
}

// ResolveMovieFilename resolves the full movie filename from a pattern.
func (r *Resolver) ResolveMovieFilename(ctx *TokenContext, extension string) (string, error) {
	if !r.settings.RenameMovies {
		return ctx.OriginalFile, nil
	}

	pattern := r.settings.MovieFileFormat
	if pattern == "" {
		return "", ErrEmptyPattern
	}

	filename, err := r.resolvePattern(pattern, ctx)
	if err != nil {
		return "", err
	}

	// Sanitize the filename
	filename = SanitizeFilename(
		filename,
		r.settings.ReplaceIllegalCharacters,
		r.settings.ColonReplacement,
		r.settings.CustomColonReplacement,
	)

	// Apply case transformation
	filename = ApplyCase(filename, r.settings.CaseMode)

	// Add extension (preserve original exactly)
	if extension != "" {
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		filename += extension
	}

	return filename, nil
}

// ResolveSeriesFolderName resolves the series folder name.
func (r *Resolver) ResolveSeriesFolderName(ctx *TokenContext) (string, error) {
	pattern := r.settings.SeriesFolderFormat
	if pattern == "" {
		pattern = defaultSeriesTitle
	}

	folder, err := r.resolvePattern(pattern, ctx)
	if err != nil {
		return "", err
	}

	return SanitizeFolderName(
		folder,
		r.settings.ReplaceIllegalCharacters,
		r.settings.ColonReplacement,
		r.settings.CustomColonReplacement,
	), nil
}

// ResolveSeasonFolderName resolves the season folder name.
func (r *Resolver) ResolveSeasonFolderName(seasonNumber int) string {
	if seasonNumber == 0 {
		folder := r.settings.SpecialsFolderFormat
		if folder == "" {
			return "Specials"
		}
		return folder
	}

	pattern := r.settings.SeasonFolderFormat
	if pattern == "" {
		pattern = "Season {season}"
	}

	// Simple replacement for season folder
	ctx := &TokenContext{SeasonNumber: seasonNumber}
	folder, _ := r.resolvePattern(pattern, ctx)

	return SanitizeFolderName(
		folder,
		r.settings.ReplaceIllegalCharacters,
		r.settings.ColonReplacement,
		r.settings.CustomColonReplacement,
	)
}

// ResolveMovieFolderName resolves the movie folder name.
func (r *Resolver) ResolveMovieFolderName(ctx *TokenContext) (string, error) {
	pattern := r.settings.MovieFolderFormat
	if pattern == "" {
		pattern = "{Movie Title} ({Year})"
	}

	folder, err := r.resolvePattern(pattern, ctx)
	if err != nil {
		return "", err
	}

	return SanitizeFolderName(
		folder,
		r.settings.ReplaceIllegalCharacters,
		r.settings.ColonReplacement,
		r.settings.CustomColonReplacement,
	), nil
}

// ResolveFullPath resolves the complete destination path and validates length.
func (r *Resolver) ResolveFullPath(rootPath, relativePath, filename string) (string, error) {
	fullPath := filepath.Join(rootPath, relativePath, filename)

	if len(fullPath) > MaxPathLength {
		return "", fmt.Errorf("%w: path length %d exceeds limit %d", ErrPathTooLong, len(fullPath), MaxPathLength)
	}

	return fullPath, nil
}

// ValidatePattern checks if a pattern is syntactically valid.
func (r *Resolver) ValidatePattern(pattern string) error {
	if pattern == "" {
		return ErrEmptyPattern
	}

	// Check for balanced braces
	if !hasBalancedBraces(pattern) {
		return fmt.Errorf("%w: unbalanced braces", ErrInvalidPattern)
	}

	// Extract and validate all tokens
	tokens := ParseTokens(pattern)
	for _, token := range tokens {
		if !IsValidTokenName(token.Name) {
			return fmt.Errorf("%w: unknown token {%s}", ErrInvalidToken, token.Name)
		}
	}

	return nil
}

// PreviewPattern generates a preview of the pattern with sample data.
func (r *Resolver) PreviewPattern(pattern string, sampleCtx *TokenContext) (string, error) {
	if err := r.ValidatePattern(pattern); err != nil {
		return "", err
	}

	return r.resolvePattern(pattern, sampleCtx)
}

// GetSampleContext returns sample data for pattern preview.
func GetSampleContext() *TokenContext {
	return &TokenContext{
		SeriesTitle:       "The Series Title",
		SeriesYear:        2024,
		SeriesType:        "standard",
		SeasonNumber:      1,
		EpisodeNumber:     1,
		EpisodeTitle:      "Pilot Episode",
		Quality:           "1080p",
		Source:            "WEBDL",
		Codec:             "x264",
		VideoCodec:        "x264",
		VideoBitDepth:     10,
		VideoDynamicRange: "HDR10",
		AudioCodec:        "DTS",
		AudioChannels:     "5.1",
		AudioLanguages:    []string{"EN", "DE"},
		SubtitleLanguages: []string{"EN"},
		ReleaseGroup:      "GROUP",
		OriginalTitle:     "The.Series.Title.S01E01.1080p.WEB-DL.x264-GROUP",
		OriginalFile:      "the.series.title.s01e01.1080p.web-dl.x264-group.mkv",
	}
}

// getEpisodePattern returns the appropriate episode pattern for the series type.
func (r *Resolver) getEpisodePattern(seriesType string) string {
	switch seriesType {
	case "daily":
		return r.settings.DailyEpisodeFormat
	case "anime":
		return r.settings.AnimeEpisodeFormat
	default:
		return r.settings.StandardEpisodeFormat
	}
}

// resolvePattern resolves all tokens in a pattern.
func (r *Resolver) resolvePattern(pattern string, ctx *TokenContext) (string, error) {
	tokens := ParseTokens(pattern)
	result := pattern

	for _, token := range tokens {
		value := token.Resolve(ctx)
		result = r.replaceTokenWithCleanup(result, token.Raw, value)
	}

	// Final cleanup: remove orphaned separators and normalize spaces
	result = cleanupOrphanedSeparators(result)
	result = strings.TrimSpace(result)

	return result, nil
}

// replaceTokenWithCleanup replaces a token and cleans up surrounding separators if value is empty.
func (r *Resolver) replaceTokenWithCleanup(s, token, value string) string {
	if value != "" {
		return strings.Replace(s, token, value, 1)
	}

	// Empty value: need to clean up surrounding separators
	// Find the token position
	idx := strings.Index(s, token)
	if idx == -1 {
		return s
	}

	// Look for separator patterns before and after the token
	before := s[:idx]
	after := s[idx+len(token):]

	// Remove trailing separator from before (e.g., " - " or " ")
	before = strings.TrimRight(before, " -._")

	// Remove leading separator from after
	after = strings.TrimLeft(after, " -._")

	// Reconstruct
	if before != "" && after != "" {
		return before + " " + after
	}
	return before + after
}

// cleanupOrphanedSeparators removes orphaned separators and double spaces.
func cleanupOrphanedSeparators(s string) string {
	// Remove patterns like " - -" or "- -" or "  "
	patterns := []string{
		" - - ", " -  ", "  - ", " -- ",
		"- -", " - -", "- - ", " . . ",
		"  ",
	}

	for _, pattern := range patterns {
		for strings.Contains(s, pattern) {
			s = strings.ReplaceAll(s, pattern, " ")
		}
	}

	// Remove leading/trailing separators from segments
	s = regexp.MustCompile(`^\s*[-._]+\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s*[-._]+\s*$`).ReplaceAllString(s, "")

	return s
}

// applyMultiEpisodeFormat applies multi-episode formatting to the filename.
func (r *Resolver) applyMultiEpisodeFormat(filename string, ctx *TokenContext) string {
	if len(ctx.EpisodeNumbers) <= 1 {
		return filename
	}

	// Find the episode identifier in the filename and replace with multi-episode format
	// Look for patterns like S01E01, s01e01, etc.
	episodePattern := regexp.MustCompile(`(?i)S(\d+)E(\d+)`)
	match := episodePattern.FindStringSubmatch(filename)
	if match == nil {
		return filename
	}

	// Determine padding from the original match
	padding := len(match[2])
	if padding < 2 {
		padding = 2
	}

	// Generate multi-episode identifier
	multiEp := FormatMultiEpisode(ctx.SeasonNumber, ctx.EpisodeNumbers, r.settings.MultiEpisodeStyle, padding)

	// Replace the first episode identifier with multi-episode format
	return episodePattern.ReplaceAllStringFunc(filename, func(s string) string {
		return multiEp
	})
}

// hasBalancedBraces checks if all braces are properly balanced.
func hasBalancedBraces(s string) bool {
	depth := 0
	for _, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

// TokenBreakdown provides detailed token resolution info for debugging.
type TokenBreakdown struct {
	Token    string // The raw token
	Name     string // Token name
	Value    string // Resolved value
	Empty    bool   // Whether value was empty
	Modified bool   // Whether modifiers were applied
}

// GetTokenBreakdown returns detailed info about each token's resolution.
func (r *Resolver) GetTokenBreakdown(pattern string, ctx *TokenContext) []TokenBreakdown {
	tokens := ParseTokens(pattern)
	breakdown := make([]TokenBreakdown, len(tokens))

	for i, token := range tokens {
		value := token.Resolve(ctx)
		breakdown[i] = TokenBreakdown{
			Token:    token.Raw,
			Name:     token.Name,
			Value:    value,
			Empty:    value == "",
			Modified: token.Separator != " " || token.Modifier != "",
		}
	}

	return breakdown
}
