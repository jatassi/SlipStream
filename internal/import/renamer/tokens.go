package renamer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	seriesTypeDaily = "daily"
)

// TokenContext contains all metadata available for token resolution.
type TokenContext struct {
	// Series info
	SeriesTitle string
	SeriesYear  int
	SeriesType  string // "standard", "daily", "anime"

	// Episode info
	SeasonNumber   int
	EpisodeNumber  int
	EpisodeNumbers []int // For multi-episode files
	AbsoluteNumber int   // For anime
	EpisodeTitle   string
	AirDate        time.Time

	// Quality info
	Quality  string // "1080p", "720p", etc.
	Source   string // "WEBDL", "BluRay", etc.
	Codec    string // "x264", "x265", etc.
	Revision string // "Proper", "Repack", "v2", etc.

	// MediaInfo (from actual file analysis)
	VideoCodec        string
	VideoBitDepth     int
	VideoDynamicRange string // "HDR10", "DV", "SDR", etc.
	AudioCodec        string
	AudioChannels     string
	AudioLanguages    []string
	SubtitleLanguages []string

	// Other
	ReleaseGroup   string
	EditionTags    string // "Director's Cut", "Extended", etc.
	OriginalTitle  string
	OriginalFile   string
	CustomFormats  []string
	ReleaseVersion int // For anime v2, v3, etc.

	// Movie info (for movie renaming)
	MovieTitle string
	MovieYear  int
}

// Token represents a parsed token from a format pattern.
type Token struct {
	Raw       string // Original token string including braces
	Name      string // Token name (e.g., "Series Title", "season")
	Separator string // Separator modifier: "", ".", "-", "_"
	Modifier  string // Additional modifier (padding, truncation, filter)
}

// tokenPattern matches tokens like {Series Title}, {season:00}, {Episode Title:30}
var tokenPattern = regexp.MustCompile(`\{([^}]+)\}`)

// ParseTokens extracts all tokens from a format pattern.
func ParseTokens(pattern string) []Token {
	matches := tokenPattern.FindAllStringSubmatch(pattern, -1)
	tokens := make([]Token, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		token := parseTokenContent(match[0], match[1])
		tokens = append(tokens, token)
	}

	return tokens
}

// parseTokenContent parses the content inside a token's braces.
func parseTokenContent(raw, content string) Token {
	token := Token{Raw: raw}

	// Check for separator prefix (first character before token name)
	// e.g., {Series.Title} -> separator=".", name="Series Title"
	if content != "" {
		switch content[0] {
		case '.':
			token.Separator = "."
			content = content[1:]
		case '-':
			token.Separator = "-"
			content = content[1:]
		case '_':
			token.Separator = "_"
			content = content[1:]
		default:
			token.Separator = " " // Default space separator
		}
	}

	// Check for modifier suffix (after colon)
	// e.g., {season:00} -> name="season", modifier="00"
	// e.g., {Episode Title:30} -> name="Episode Title", modifier="30"
	if colonIdx := strings.LastIndex(content, ":"); colonIdx > 0 {
		token.Name = strings.TrimSpace(content[:colonIdx])
		token.Modifier = strings.TrimSpace(content[colonIdx+1:])
	} else {
		token.Name = strings.TrimSpace(content)
	}

	return token
}

// Resolve returns the resolved value for this token given the context.
func (t *Token) Resolve(ctx *TokenContext) string {
	value := t.resolveValue(ctx)

	// Apply separator transformation if value contains spaces
	if t.Separator != " " && t.Separator != "" {
		value = strings.ReplaceAll(value, " ", t.Separator)
	}

	return value
}

// resolveValue returns the raw value for the token before separator transformation.
func (t *Token) resolveValue(ctx *TokenContext) string {
	name := strings.ToLower(t.Name)

	switch {
	case strings.HasPrefix(name, "series "):
		return t.resolveSeriesToken(name, ctx)
	case strings.HasPrefix(name, "episode"):
		return t.resolveEpisodeToken(name, ctx)
	case strings.HasPrefix(name, "air"):
		return t.resolveAirDate(name, ctx)
	case strings.HasPrefix(name, "quality "):
		return t.resolveQualityToken(name, ctx)
	case strings.HasPrefix(name, "mediainfo"):
		return t.resolveMediaInfoToken(name, ctx)
	case strings.HasPrefix(name, "movie "):
		return t.resolveMovieToken(name, ctx)
	default:
		return t.resolveSimpleToken(name, ctx)
	}
}

func (t *Token) resolveSeriesToken(name string, ctx *TokenContext) string {
	switch name {
	case "series title":
		return ctx.SeriesTitle
	case "series titleyear":
		if ctx.SeriesYear > 0 {
			return fmt.Sprintf("%s (%d)", ctx.SeriesTitle, ctx.SeriesYear)
		}
		return ctx.SeriesTitle
	case "series cleantitle":
		return CleanTitle(ctx.SeriesTitle)
	default: // series cleantitleyear
		clean := CleanTitle(ctx.SeriesTitle)
		if ctx.SeriesYear > 0 {
			return fmt.Sprintf("%s %d", clean, ctx.SeriesYear)
		}
		return clean
	}
}

func (t *Token) resolveEpisodeToken(name string, ctx *TokenContext) string {
	switch name {
	case "episode":
		if len(ctx.EpisodeNumbers) > 1 {
			return t.formatNumber(ctx.EpisodeNumbers[0])
		}
		return t.formatNumber(ctx.EpisodeNumber)
	case "episode title", "episode cleantitle":
		return t.resolveEpisodeTitle(name, ctx)
	default:
		return ""
	}
}

func (t *Token) resolveAirDate(name string, ctx *TokenContext) string {
	if ctx.AirDate.IsZero() {
		return ""
	}
	if name == "air-date" {
		return ctx.AirDate.Format("2006-01-02")
	}
	return ctx.AirDate.Format("2006 01 02")
}

func (t *Token) resolveEpisodeTitle(name string, ctx *TokenContext) string {
	title := ctx.EpisodeTitle
	if title == "" && ctx.SeriesType == seriesTypeDaily && !ctx.AirDate.IsZero() {
		title = ctx.AirDate.Format("January 2, 2006")
	}
	if name == "episode cleantitle" {
		title = CleanTitle(title)
	}
	return t.applyTruncation(title)
}

func (t *Token) resolveQualityToken(name string, ctx *TokenContext) string {
	if name == "quality full" {
		return t.buildQualityFull(ctx)
	}
	return t.buildQualityTitle(ctx)
}

func (t *Token) resolveMediaInfoToken(name string, ctx *TokenContext) string {
	switch name {
	case "mediainfo simple":
		return t.buildMediaInfoSimple(ctx)
	case "mediainfo full":
		return t.buildMediaInfoFull(ctx)
	case "mediainfo audiolanguages":
		return t.formatLanguages(ctx.AudioLanguages)
	case "mediainfo subtitlelanguages":
		return t.formatLanguages(ctx.SubtitleLanguages)
	default:
		return t.resolveMediaInfoField(name, ctx)
	}
}

func (t *Token) resolveMediaInfoField(name string, ctx *TokenContext) string {
	switch name {
	case "mediainfo videocodec":
		return ctx.VideoCodec
	case "mediainfo videobitdepth":
		if ctx.VideoBitDepth > 0 {
			return strconv.Itoa(ctx.VideoBitDepth)
		}
		return ""
	case "mediainfo videodynamicrange", "mediainfo videodynamicrangetype":
		if ctx.VideoDynamicRange == "" || ctx.VideoDynamicRange == "SDR" {
			return ""
		}
		if name == "mediainfo videodynamicrange" {
			return "HDR"
		}
		return ctx.VideoDynamicRange
	case "mediainfo audiocodec":
		return ctx.AudioCodec
	default: // mediainfo audiochannels
		return ctx.AudioChannels
	}
}

func (t *Token) resolveOtherToken(name string, ctx *TokenContext) string {
	switch name {
	case "release group":
		return ctx.ReleaseGroup
	case "custom formats":
		if len(ctx.CustomFormats) > 0 {
			return strings.Join(ctx.CustomFormats, " ")
		}
		return ""
	case "original title":
		return ctx.OriginalTitle
	case "original filename":
		return ctx.OriginalFile
	case "revision":
		return ctx.Revision
	case "edition tags":
		return ctx.EditionTags
	default:
		return ""
	}
}

func (t *Token) resolveAnimeToken(name string, ctx *TokenContext) string {
	if name == "absolute" {
		if ctx.AbsoluteNumber > 0 {
			return t.formatNumber(ctx.AbsoluteNumber)
		}
		return ""
	}
	// version
	if ctx.ReleaseVersion > 1 {
		return fmt.Sprintf("v%d", ctx.ReleaseVersion)
	}
	return ""
}

func (t *Token) resolveMovieToken(name string, ctx *TokenContext) string {
	switch name {
	case "movie title":
		return ctx.MovieTitle
	case "movie titleyear":
		if ctx.MovieYear > 0 {
			return fmt.Sprintf("%s (%d)", ctx.MovieTitle, ctx.MovieYear)
		}
		return ctx.MovieTitle
	case "movie cleantitle":
		return CleanTitle(ctx.MovieTitle)
	default: // movie cleantitleyear
		clean := CleanTitle(ctx.MovieTitle)
		if ctx.MovieYear > 0 {
			return fmt.Sprintf("%s %d", clean, ctx.MovieYear)
		}
		return clean
	}
}

func (t *Token) resolveSimpleToken(name string, ctx *TokenContext) string {
	switch name {
	case "season":
		return t.formatNumber(ctx.SeasonNumber)
	case "year":
		if ctx.MovieYear > 0 {
			return strconv.Itoa(ctx.MovieYear)
		}
		if ctx.SeriesYear > 0 {
			return strconv.Itoa(ctx.SeriesYear)
		}
		return ""
	case "absolute", "version":
		return t.resolveAnimeToken(name, ctx)
	case "custom format":
		return t.resolveCustomFormat(ctx)
	default:
		return t.resolveOtherToken(name, ctx)
	}
}

func (t *Token) resolveCustomFormat(ctx *TokenContext) string {
	if t.Modifier == "" {
		return ""
	}
	for _, cf := range ctx.CustomFormats {
		if strings.EqualFold(cf, t.Modifier) {
			return cf
		}
	}
	return ""
}

// formatNumber formats a number with the specified padding from the modifier.
func (t *Token) formatNumber(num int) string {
	if t.Modifier == "" {
		return strconv.Itoa(num)
	}

	// Count leading zeros in modifier to determine padding
	padding := len(t.Modifier)
	if padding > 0 && t.Modifier[0] == '0' {
		return fmt.Sprintf("%0*d", padding, num)
	}

	return strconv.Itoa(num)
}

// applyTruncation applies truncation modifier if present.
func (t *Token) applyTruncation(value string) string {
	if t.Modifier == "" {
		return value
	}

	// Try to parse modifier as truncation length
	limit, err := strconv.Atoi(t.Modifier)
	if err != nil {
		return value
	}

	if limit == 0 {
		return value
	}

	runes := []rune(value)

	if limit > 0 {
		// Truncate from end
		if len(runes) > limit {
			return string(runes[:limit-1]) + "…"
		}
	} else {
		// Negative: truncate from beginning
		limit = -limit
		if len(runes) > limit {
			return "…" + string(runes[len(runes)-limit+1:])
		}
	}

	return value
}

// buildQualityFull builds the full quality string including revision.
func (t *Token) buildQualityFull(ctx *TokenContext) string {
	parts := []string{}

	if ctx.Source != "" {
		parts = append(parts, ctx.Source)
	}
	if ctx.Quality != "" {
		if len(parts) > 0 {
			parts[len(parts)-1] += "-" + ctx.Quality
		} else {
			parts = append(parts, ctx.Quality)
		}
	}
	if ctx.Revision != "" {
		parts = append(parts, ctx.Revision)
	}

	return strings.Join(parts, " ")
}

// buildQualityTitle builds the quality string without revision.
func (t *Token) buildQualityTitle(ctx *TokenContext) string {
	if ctx.Source != "" && ctx.Quality != "" {
		return ctx.Source + "-" + ctx.Quality
	}
	if ctx.Source != "" {
		return ctx.Source
	}
	if ctx.Quality != "" {
		return ctx.Quality
	}
	return ""
}

// buildMediaInfoSimple builds a simple media info string.
func (t *Token) buildMediaInfoSimple(ctx *TokenContext) string {
	parts := []string{}

	if ctx.VideoCodec != "" {
		parts = append(parts, ctx.VideoCodec)
	}
	if ctx.AudioCodec != "" {
		parts = append(parts, ctx.AudioCodec)
	}

	return strings.Join(parts, " ")
}

// buildMediaInfoFull builds a full media info string with languages.
func (t *Token) buildMediaInfoFull(ctx *TokenContext) string {
	parts := []string{}

	if ctx.VideoCodec != "" {
		parts = append(parts, ctx.VideoCodec)
	}
	if ctx.AudioCodec != "" {
		parts = append(parts, ctx.AudioCodec)
	}

	if len(ctx.AudioLanguages) > 0 {
		parts = append(parts, t.formatLanguages(ctx.AudioLanguages))
	}

	return strings.Join(parts, " ")
}

// formatLanguages formats a slice of language codes.
func (t *Token) formatLanguages(langs []string) string {
	if len(langs) == 0 {
		return ""
	}

	// Apply language filter if present in modifier
	filtered := t.filterLanguages(langs)
	if len(filtered) == 0 {
		return ""
	}

	// Default format: [EN+DE]
	return "[" + strings.Join(filtered, "+") + "]"
}

// filterLanguages applies the language filter from the modifier.
func (t *Token) filterLanguages(langs []string) []string {
	if t.Modifier == "" {
		return langs
	}

	if strings.HasPrefix(t.Modifier, "-") {
		return t.filterLanguagesExclude(langs)
	}

	if strings.Contains(t.Modifier, "+") || strings.Contains(t.Modifier, ",") {
		return t.filterLanguagesInclude(langs)
	}

	return langs
}

func (t *Token) filterLanguagesExclude(langs []string) []string {
	exclude := strings.ToUpper(strings.TrimPrefix(t.Modifier, "-"))
	result := make([]string, 0, len(langs))
	for _, lang := range langs {
		if !strings.EqualFold(lang, exclude) {
			result = append(result, lang)
		}
	}
	return result
}

func (t *Token) filterLanguagesInclude(langs []string) []string {
	includes := strings.FieldsFunc(t.Modifier, func(r rune) bool {
		return r == '+' || r == ','
	})
	includeMap := make(map[string]bool)
	for _, inc := range includes {
		includeMap[strings.ToUpper(strings.TrimSpace(inc))] = true
	}

	result := make([]string, 0, len(langs))
	for _, lang := range langs {
		if includeMap[strings.ToUpper(lang)] {
			result = append(result, lang)
		}
	}
	return result
}

// GetAllTokenNames returns all supported token names for validation.
func GetAllTokenNames() []string {
	return []string{
		// Series tokens
		"Series Title", "Series TitleYear", "Series CleanTitle", "Series CleanTitleYear",
		// Season/Episode tokens
		"season", "episode",
		// Air date tokens
		"Air-Date", "Air Date",
		// Episode title tokens
		"Episode Title", "Episode CleanTitle",
		// Quality tokens
		"Quality Full", "Quality Title",
		// MediaInfo tokens
		"MediaInfo Simple", "MediaInfo Full", "MediaInfo VideoCodec",
		"MediaInfo VideoBitDepth", "MediaInfo VideoDynamicRange", "MediaInfo VideoDynamicRangeType",
		"MediaInfo AudioCodec", "MediaInfo AudioChannels",
		"MediaInfo AudioLanguages", "MediaInfo SubtitleLanguages",
		// Other tokens
		"Release Group", "Edition Tags", "Custom Formats", "Original Title", "Original Filename", "Revision",
		// Anime tokens
		"absolute", "version",
		// Movie tokens
		"Movie Title", "Movie TitleYear", "Movie CleanTitle", "Movie CleanTitleYear", "Year",
	}
}

// IsValidTokenName checks if a token name is valid (case-insensitive).
func IsValidTokenName(name string) bool {
	lower := strings.ToLower(name)

	// Check for custom format token
	if strings.HasPrefix(lower, "custom format:") {
		return true
	}

	for _, valid := range GetAllTokenNames() {
		if strings.EqualFold(valid, lower) {
			return true
		}
	}
	return false
}
