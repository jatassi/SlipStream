package organizer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// NamingConfig holds naming template configuration.
type NamingConfig struct {
	// Movie naming
	MovieFolderFormat string `json:"movieFolderFormat"`
	MovieFileFormat   string `json:"movieFileFormat"`

	// TV naming
	SeriesFolderFormat  string `json:"seriesFolderFormat"`
	SeasonFolderFormat  string `json:"seasonFolderFormat"`
	EpisodeFileFormat   string `json:"episodeFileFormat"`
	MultiEpisodeFormat  string `json:"multiEpisodeFormat"`

	// Options
	ReplaceSpaces      bool   `json:"replaceSpaces"`
	SpaceReplacement   string `json:"spaceReplacement"`
	ColonReplacement   string `json:"colonReplacement"`
	CleanSpecialChars  bool   `json:"cleanSpecialChars"`
}

// DefaultNamingConfig returns default naming configuration.
func DefaultNamingConfig() NamingConfig {
	return NamingConfig{
		MovieFolderFormat:   "{Movie Title} ({Year})",
		MovieFileFormat:     "{Movie Title} ({Year})",
		SeriesFolderFormat:  "{Series Title}",
		SeasonFolderFormat:  "Season {Season:00}",
		EpisodeFileFormat:   "{Series Title} - S{Season:00}E{Episode:00} - {Episode Title}",
		MultiEpisodeFormat:  "{Series Title} - S{Season:00}E{Episode:00}-E{EndEpisode:00} - {Episode Title}",
		ReplaceSpaces:       false,
		SpaceReplacement:    ".",
		ColonReplacement:    " -",
		CleanSpecialChars:   true,
	}
}

// MovieTokens contains tokens for movie naming.
type MovieTokens struct {
	Title      string
	Year       int
	Quality    string
	Resolution string
	Source     string
	Codec      string
	ImdbID     string
	TmdbID     int
}

// SeriesTokens contains tokens for series naming.
type SeriesTokens struct {
	SeriesTitle string
	Year        int
}

// EpisodeTokens contains tokens for episode naming.
type EpisodeTokens struct {
	SeriesTitle   string
	SeasonNumber  int
	EpisodeNumber int
	EndEpisode    int
	EpisodeTitle  string
	Quality       string
	Resolution    string
	Source        string
	Codec         string
}

// tokenPattern matches template tokens like {Token} or {Token:format}
var tokenPattern = regexp.MustCompile(`\{([^}:]+)(?::([^}]+))?\}`)

// FormatMovieFolder formats a movie folder name using the template.
func (c NamingConfig) FormatMovieFolder(tokens MovieTokens) string {
	return c.formatTemplate(c.MovieFolderFormat, func(token, format string) string {
		return c.resolveMovieToken(token, format, tokens)
	})
}

// FormatMovieFile formats a movie filename using the template.
func (c NamingConfig) FormatMovieFile(tokens MovieTokens) string {
	return c.formatTemplate(c.MovieFileFormat, func(token, format string) string {
		return c.resolveMovieToken(token, format, tokens)
	})
}

// FormatSeriesFolder formats a series folder name using the template.
func (c NamingConfig) FormatSeriesFolder(tokens SeriesTokens) string {
	return c.formatTemplate(c.SeriesFolderFormat, func(token, format string) string {
		return c.resolveSeriesToken(token, format, tokens)
	})
}

// FormatSeasonFolder formats a season folder name using the template.
func (c NamingConfig) FormatSeasonFolder(seasonNumber int) string {
	return c.formatTemplate(c.SeasonFolderFormat, func(token, format string) string {
		if strings.EqualFold(token, "Season") {
			return formatNumber(seasonNumber, format)
		}
		return ""
	})
}

// FormatEpisodeFile formats an episode filename using the template.
func (c NamingConfig) FormatEpisodeFile(tokens EpisodeTokens) string {
	template := c.EpisodeFileFormat
	if tokens.EndEpisode > 0 && tokens.EndEpisode != tokens.EpisodeNumber {
		template = c.MultiEpisodeFormat
	}

	return c.formatTemplate(template, func(token, format string) string {
		return c.resolveEpisodeToken(token, format, tokens)
	})
}

// formatTemplate applies token resolution to a template string.
func (c NamingConfig) formatTemplate(template string, resolver func(token, format string) string) string {
	result := tokenPattern.ReplaceAllStringFunc(template, func(match string) string {
		submatch := tokenPattern.FindStringSubmatch(match)
		if len(submatch) >= 2 {
			token := submatch[1]
			format := ""
			if len(submatch) >= 3 {
				format = submatch[2]
			}
			return resolver(token, format)
		}
		return match
	})

	// Clean up the result
	result = c.cleanFilename(result)
	return result
}

// resolveMovieToken resolves a movie template token.
func (c NamingConfig) resolveMovieToken(token, format string, tokens MovieTokens) string {
	switch strings.ToLower(token) {
	case "movie title", "title":
		return tokens.Title
	case "year":
		if tokens.Year > 0 {
			return formatNumber(tokens.Year, format)
		}
		return ""
	case "quality":
		return tokens.Quality
	case "resolution":
		return tokens.Resolution
	case "source":
		return tokens.Source
	case "codec":
		return tokens.Codec
	case "imdb", "imdbid":
		return tokens.ImdbID
	case "tmdb", "tmdbid":
		if tokens.TmdbID > 0 {
			return strconv.Itoa(tokens.TmdbID)
		}
		return ""
	}
	return ""
}

// resolveSeriesToken resolves a series template token.
func (c NamingConfig) resolveSeriesToken(token, format string, tokens SeriesTokens) string {
	switch strings.ToLower(token) {
	case "series title", "title":
		return tokens.SeriesTitle
	case "year":
		if tokens.Year > 0 {
			return formatNumber(tokens.Year, format)
		}
		return ""
	}
	return ""
}

// resolveEpisodeToken resolves an episode template token.
func (c NamingConfig) resolveEpisodeToken(token, format string, tokens EpisodeTokens) string {
	switch strings.ToLower(token) {
	case "series title", "series":
		return tokens.SeriesTitle
	case "season":
		return formatNumber(tokens.SeasonNumber, format)
	case "episode":
		return formatNumber(tokens.EpisodeNumber, format)
	case "endepisode":
		if tokens.EndEpisode > 0 {
			return formatNumber(tokens.EndEpisode, format)
		}
		return formatNumber(tokens.EpisodeNumber, format)
	case "episode title", "title":
		return tokens.EpisodeTitle
	case "quality":
		return tokens.Quality
	case "resolution":
		return tokens.Resolution
	case "source":
		return tokens.Source
	case "codec":
		return tokens.Codec
	}
	return ""
}

// formatNumber formats a number with optional padding.
func formatNumber(n int, format string) string {
	if format == "" {
		return strconv.Itoa(n)
	}

	// Parse format like "00" for zero-padded
	if len(format) > 0 && format[0] == '0' {
		width := len(format)
		return fmt.Sprintf("%0*d", width, n)
	}

	return strconv.Itoa(n)
}

// cleanFilename cleans a filename by removing/replacing invalid characters.
func (c NamingConfig) cleanFilename(name string) string {
	// Replace colons
	if c.ColonReplacement != "" {
		name = strings.ReplaceAll(name, ":", c.ColonReplacement)
	}

	// Replace spaces if configured
	if c.ReplaceSpaces && c.SpaceReplacement != "" {
		name = strings.ReplaceAll(name, " ", c.SpaceReplacement)
	}

	// Remove other invalid characters
	if c.CleanSpecialChars {
		// Characters invalid on Windows filesystems
		invalidChars := []string{"<", ">", "\"", "/", "\\", "|", "?", "*"}
		for _, char := range invalidChars {
			name = strings.ReplaceAll(name, char, "")
		}
	}

	// Clean up multiple spaces/separators
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)

	// Remove empty parentheses like "()" that might result from empty year
	name = regexp.MustCompile(`\s*\(\s*\)\s*`).ReplaceAllString(name, "")

	return name
}
