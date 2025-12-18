package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParsedMedia represents a media file parsed from a filename.
type ParsedMedia struct {
	Title      string `json:"title"`
	Year       int    `json:"year,omitempty"`
	Season     int    `json:"season,omitempty"`     // 0 for movies
	Episode    int    `json:"episode,omitempty"`    // 0 for movies
	EndEpisode int    `json:"endEpisode,omitempty"` // For multi-episode files
	Quality    string `json:"quality,omitempty"`    // "720p", "1080p", "2160p"
	Resolution int    `json:"resolution,omitempty"` // 720, 1080, 2160
	Source     string `json:"source,omitempty"`     // "BluRay", "WEB-DL", "HDTV"
	Codec      string `json:"codec,omitempty"`      // "x264", "x265", "HEVC"
	IsTV       bool   `json:"isTv"`
	FilePath   string `json:"filePath"`
	FileSize   int64  `json:"fileSize"`
}

// Regex patterns for parsing
var (
	// TV patterns: Show.S01E02 or Show.1x02
	tvPatternSE = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[\.\s_-]*(.*)$`)
	tvPatternX  = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+(\d{1,2})[xX](\d{1,2})[\.\s_-]*(.*)$`)

	// Movie pattern: Title.Year or Title (Year)
	moviePatternDot    = regexp.MustCompile(`^(.+?)[\.\s_-]+(\d{4})[\.\s_-]+(.*)$`)
	moviePatternParen  = regexp.MustCompile(`^(.+?)\s*\((\d{4})\)\s*(.*)$`)
	moviePatternSimple = regexp.MustCompile(`^(.+?)[\.\s_-]+(\d{4})$`)

	// Quality patterns
	qualityPatterns = map[string]*regexp.Regexp{
		"2160p": regexp.MustCompile(`(?i)(2160p|4k|uhd)`),
		"1080p": regexp.MustCompile(`(?i)1080p`),
		"720p":  regexp.MustCompile(`(?i)720p`),
		"480p":  regexp.MustCompile(`(?i)(480p|sd)`),
	}

	// Source patterns
	sourcePatterns = map[string]*regexp.Regexp{
		"BluRay":  regexp.MustCompile(`(?i)(blu-?ray|bdrip|brrip|bdremux)`),
		"WEB-DL":  regexp.MustCompile(`(?i)(web-?dl|webdl)`),
		"WEBRip":  regexp.MustCompile(`(?i)webrip`),
		"HDTV":    regexp.MustCompile(`(?i)hdtv`),
		"DVDRip":  regexp.MustCompile(`(?i)(dvdrip|dvd-?r)`),
		"SDTV":    regexp.MustCompile(`(?i)(sdtv|pdtv|dsr)`),
		"CAM":     regexp.MustCompile(`(?i)(cam|hdcam|ts|telesync)`),
		"Remux":   regexp.MustCompile(`(?i)remux`),
	}

	// Codec patterns
	codecPatterns = map[string]*regexp.Regexp{
		"x265":  regexp.MustCompile(`(?i)(x265|h\.?265|hevc)`),
		"x264":  regexp.MustCompile(`(?i)(x264|h\.?264|avc)`),
		"AV1":   regexp.MustCompile(`(?i)av1`),
		"VP9":   regexp.MustCompile(`(?i)vp9`),
		"XviD":  regexp.MustCompile(`(?i)xvid`),
		"DivX":  regexp.MustCompile(`(?i)divx`),
		"MPEG2": regexp.MustCompile(`(?i)mpeg-?2`),
	}

	// Clean up patterns
	cleanupPattern = regexp.MustCompile(`[\.\s_-]+`)
)

// ParseFilename parses a media filename into structured data.
func ParseFilename(filename string) *ParsedMedia {
	// Remove extension for parsing
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	parsed := &ParsedMedia{
		FilePath: filename,
	}

	// Try TV patterns first
	if match := tvPatternSE.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			parsed.EndEpisode, _ = strconv.Atoi(match[4])
		}
		parseQualityInfo(match[5], parsed)
		return parsed
	}

	if match := tvPatternX.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode, _ = strconv.Atoi(match[3])
		parseQualityInfo(match[4], parsed)
		return parsed
	}

	// Try movie patterns
	if match := moviePatternParen.FindStringSubmatch(name); match != nil {
		parsed.Title = cleanTitle(match[1])
		parsed.Year, _ = strconv.Atoi(match[2])
		parseQualityInfo(match[3], parsed)
		return parsed
	}

	if match := moviePatternDot.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			parsed.Title = cleanTitle(match[1])
			parsed.Year = year
			parseQualityInfo(match[3], parsed)
			return parsed
		}
	}

	if match := moviePatternSimple.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			parsed.Title = cleanTitle(match[1])
			parsed.Year = year
			return parsed
		}
	}

	// Fallback: just use the filename as title
	parsed.Title = cleanTitle(name)
	parseQualityInfo(name, parsed)
	return parsed
}

// cleanTitle cleans up a parsed title by replacing separators with spaces.
func cleanTitle(title string) string {
	// Replace common separators with spaces
	cleaned := cleanupPattern.ReplaceAllString(title, " ")
	// Trim and normalize whitespace
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

// parseQualityInfo extracts quality, source, and codec from remaining text.
func parseQualityInfo(text string, parsed *ParsedMedia) {
	// Quality
	for quality, pattern := range qualityPatterns {
		if pattern.MatchString(text) {
			parsed.Quality = quality
			switch quality {
			case "2160p":
				parsed.Resolution = 2160
			case "1080p":
				parsed.Resolution = 1080
			case "720p":
				parsed.Resolution = 720
			case "480p":
				parsed.Resolution = 480
			}
			break
		}
	}

	// Source
	for source, pattern := range sourcePatterns {
		if pattern.MatchString(text) {
			parsed.Source = source
			break
		}
	}

	// Codec
	for codec, pattern := range codecPatterns {
		if pattern.MatchString(text) {
			parsed.Codec = codec
			break
		}
	}
}

// ParsePath tries to extract media info from a folder path.
// This is useful when the filename doesn't contain all information.
func ParsePath(fullPath string) *ParsedMedia {
	// First try the filename
	filename := filepath.Base(fullPath)
	parsed := ParseFilename(filename)

	// If we didn't get a year for a movie, try the parent folder
	if !parsed.IsTV && parsed.Year == 0 {
		dir := filepath.Dir(fullPath)
		folderName := filepath.Base(dir)
		folderParsed := ParseFilename(folderName)
		if folderParsed.Year != 0 {
			parsed.Year = folderParsed.Year
			// Use folder's title when folder parsing was successful (has year)
			// This handles cases like "The Matrix (1999)/The.Matrix.1080p.BluRay.mkv"
			// where the filename lacks the proper title format
			if folderParsed.Title != "" {
				parsed.Title = folderParsed.Title
			}
		}
	}

	parsed.FilePath = fullPath
	return parsed
}
