package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParsedMedia represents a media file parsed from a filename.
type ParsedMedia struct {
	Title            string   `json:"title"`
	Year             int      `json:"year,omitempty"`
	Season           int      `json:"season,omitempty"`           // 0 for movies or complete series
	EndSeason        int      `json:"endSeason,omitempty"`        // For multi-season packs (S01-S04)
	Episode          int      `json:"episode,omitempty"`          // 0 for movies or season packs
	EndEpisode       int      `json:"endEpisode,omitempty"`       // For multi-episode files
	IsSeasonPack     bool     `json:"isSeasonPack,omitempty"`     // True for season packs (S01 without episode)
	IsCompleteSeries bool     `json:"isCompleteSeries,omitempty"` // True for complete series boxsets
	Quality          string   `json:"quality,omitempty"`          // "720p", "1080p", "2160p"
	Resolution       int      `json:"resolution,omitempty"`       // 720, 1080, 2160
	Source           string   `json:"source,omitempty"`           // "BluRay", "WEB-DL", "HDTV"
	Codec            string   `json:"codec,omitempty"`            // "x264", "x265", "HEVC"
	Attributes       []string `json:"attributes,omitempty"`       // HDR, Atmos, REMUX, etc.
	IsTV             bool     `json:"isTv"`
	FilePath         string   `json:"filePath"`
	FileSize         int64    `json:"fileSize"`
}

// Regex patterns for parsing
var (
	// TV patterns: Show.S01E02 or Show.1x02
	tvPatternSE = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[\.\s_-]*(.*)$`)
	tvPatternX  = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+(\d{1,2})[xX](\d{1,2})[\.\s_-]*(.*)$`)

	// TV season pack pattern: Show.S01 (no episode number - for season packs/boxsets)
	tvPatternSeasonPack = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+[Ss](\d{1,2})(?:[\.\s_-]|$)(.*)$`)

	// TV season pack pattern with spelled out "Season": Show.Season.1 or Show Season 01
	tvPatternSeasonSpelled = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+[Ss]eason[\.\s_-]+(\d{1,2})(?:[\.\s_-]|$)(.*)$`)

	// TV multi-season range pattern: Show.S01-04 or Show.S01-S04 (complete series boxsets)
	tvPatternSeasonRange = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+[Ss](\d{1,2})-[Ss]?(\d{1,2})[\.\s_-]+(.*)$`)

	// TV complete series pattern: Show.COMPLETE or Show.Complete.Series (no season number)
	// Must NOT have S## pattern - those are handled by season pack patterns above
	tvPatternComplete = regexp.MustCompile(`(?i)^(.+?)[\.\s_-]+(?:complete[\.\s_-]*(?:series)?|the[\.\s_-]+complete[\.\s_-]+series)[\.\s_-]+(.*)$`)

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

	// HDR patterns (order matters - more specific patterns first)
	hdrPatterns = map[string]*regexp.Regexp{
		"DV":     regexp.MustCompile(`(?i)(dolby[\.\s]?vision|dovi|\.dv\.)`),
		"HDR10+": regexp.MustCompile(`(?i)hdr10\+`),
		"HDR10":  regexp.MustCompile(`(?i)hdr10`),
		"HDR":    regexp.MustCompile(`(?i)[\.\s\-]hdr[\.\s\-]`),
		"HLG":    regexp.MustCompile(`(?i)hlg`),
	}

	// Audio patterns (order matters - more specific patterns first)
	audioPatterns = map[string]*regexp.Regexp{
		"Atmos":  regexp.MustCompile(`(?i)atmos`),
		"DTS-X":  regexp.MustCompile(`(?i)dts[\.\-]?x`),
		"DTS-HD": regexp.MustCompile(`(?i)dts[\.\-]?hd([\.\-]?ma)?`),
		"TrueHD": regexp.MustCompile(`(?i)truehd`),
		"DTS":    regexp.MustCompile(`(?i)[\.\s\-]dts[\.\s\-]`),
		"DD+":    regexp.MustCompile(`(?i)(ddp|dd\+|e[\.\-]?ac[\.\-]?3)`),
		"DD":     regexp.MustCompile(`(?i)(dd[25]\.[01]|[\.\s\-]ac[\.\-]?3[\.\s\-])`),
		"AAC":    regexp.MustCompile(`(?i)[\.\s\-]aac[\.\s\-]`),
		"FLAC":   regexp.MustCompile(`(?i)[\.\s\-]flac[\.\s\-]`),
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

	// Try multi-season range pattern FIRST (S01-04 or S01-S04 - complete series boxsets)
	// Must check before single season pack to avoid S01 matching prematurely
	if match := tvPatternSeasonRange.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.IsCompleteSeries = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.EndSeason, _ = strconv.Atoi(match[3])
		parsed.Episode = 0
		parseQualityInfo(match[4], parsed)
		return parsed
	}

	// Try season pack pattern (S01 without episode number)
	if match := tvPatternSeasonPack.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode = 0 // Season pack - no specific episode
		parseQualityInfo(match[3], parsed)
		return parsed
	}

	// Try spelled out "Season X" pattern (Season 1, Season 02, etc.)
	if match := tvPatternSeasonSpelled.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode = 0 // Season pack - no specific episode
		parseQualityInfo(match[3], parsed)
		return parsed
	}

	// Try complete series pattern (COMPLETE without season number)
	if match := tvPatternComplete.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.IsCompleteSeries = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season = 0 // Complete series - all seasons
		parsed.Episode = 0
		parseQualityInfo(match[2], parsed)
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

// parseQualityInfo extracts quality, source, codec, and attributes from remaining text.
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

	// Attributes
	var attributes []string

	// Check for REMUX (from source)
	if parsed.Source == "Remux" {
		attributes = append(attributes, "REMUX")
	}

	// HDR attributes (check in priority order)
	hdrOrder := []string{"DV", "HDR10+", "HDR10", "HDR", "HLG"}
	for _, hdr := range hdrOrder {
		if pattern, ok := hdrPatterns[hdr]; ok && pattern.MatchString(text) {
			attributes = append(attributes, hdr)
			break // Only add the most specific HDR type
		}
	}

	// Audio attributes (check in priority order)
	audioOrder := []string{"Atmos", "DTS-X", "DTS-HD", "TrueHD", "DTS", "DD+", "DD", "AAC", "FLAC"}
	for _, audio := range audioOrder {
		if pattern, ok := audioPatterns[audio]; ok && pattern.MatchString(text) {
			attributes = append(attributes, audio)
			break // Only add the most specific audio type
		}
	}

	if len(attributes) > 0 {
		parsed.Attributes = attributes
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
