package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/module/parseutil"
	"github.com/slipstream/slipstream/internal/pathutil"
)

// ParsedMedia represents a media file parsed from a filename.
type ParsedMedia struct {
	Title             string   `json:"title"`
	Year              int      `json:"year,omitempty"`
	Season            int      `json:"season,omitempty"`            // 0 for movies or complete series
	EndSeason         int      `json:"endSeason,omitempty"`         // For multi-season packs (S01-S04)
	Episode           int      `json:"episode,omitempty"`           // 0 for movies or season packs
	EndEpisode        int      `json:"endEpisode,omitempty"`        // For multi-episode files
	IsSeasonPack      bool     `json:"isSeasonPack"`                // True for season packs (S01 without episode)
	IsCompleteSeries  bool     `json:"isCompleteSeries,omitempty"`  // True for complete series boxsets
	Quality           string   `json:"quality,omitempty"`           // "720p", "1080p", "2160p"
	Source            string   `json:"source,omitempty"`            // "BluRay", "WEB-DL", "HDTV"
	Codec             string   `json:"codec,omitempty"`             // "x264", "x265", "HEVC" (video codec)
	AudioCodecs       []string `json:"audioCodecs,omitempty"`       // "TrueHD", "DTS-HD MA", "DDP", "AAC"
	AudioChannels     []string `json:"audioChannels,omitempty"`     // "2.0", "5.1", "7.1"
	AudioEnhancements []string `json:"audioEnhancements,omitempty"` // "Atmos", "DTS:X"
	HDRFormats        []string `json:"hdrFormats,omitempty"`        // "DV", "HDR10+", "HDR10", "HDR", "HLG", or "SDR"
	Attributes        []string `json:"attributes,omitempty"`        // HDR + REMUX combined for display (backwards compat)
	ReleaseGroup      string   `json:"releaseGroup,omitempty"`      // "SPARKS", "NTb", "HONE"
	Revision          string   `json:"revision,omitempty"`          // "Proper", "REPACK", "REAL"
	Edition           string   `json:"edition,omitempty"`           // "Directors Cut", "Extended", "Theatrical"
	Languages         []string `json:"languages,omitempty"`         // "German", "French", "Spanish", etc. Empty means English assumed
	IsTV              bool     `json:"isTv"`
	FilePath          string   `json:"filePath"`
	FileSize          int64    `json:"fileSize"`
}

// Regex patterns for parsing
var (
	// TV patterns: Show.S01E02 or Show.1x02
	tvPatternSE = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?[.\s_-]*(.*)$`)
	tvPatternX  = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})[xX](\d{1,2})[.\s_-]*(.*)$`)

	// TV pattern with spelled out Season and Episode: Show.Season.1.Episode.01
	tvPatternSeasonEpisodeSpelled = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]eason[.\s_-]+(\d{1,2})[.\s_-]+[Ee]pisode[.\s_-]+(\d{1,2})[.\s_-]*(.*)$`)

	// TV season pack pattern: Show.S01 (no episode number - for season packs/boxsets)
	tvPatternSeasonPack = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})(?:[.\s_-]|$)(.*)$`)

	// TV season pack pattern with spelled out "Season": Show.Season.1 or Show Season 01
	tvPatternSeasonSpelled = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]eason[.\s_-]+(\d{1,2})(?:[.\s_-]|$)(.*)$`)

	// TV multi-season range pattern: Show.S01-04 or Show.S01-S04 (complete series boxsets)
	tvPatternSeasonRange = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})-s?(\d{1,2})[.\s_-]+(.*)$`)

	// TV complete series pattern: Show.COMPLETE or Show.Complete.Series (no season number)
	// Must NOT have S## pattern - those are handled by season pack patterns above
	tvPatternComplete = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(?:complete[.\s_-]*(?:series)?|the[.\s_-]+complete[.\s_-]+series)[.\s_-]+(.*)$`)

	// Movie pattern: Title.Year or Title (Year)
	moviePatternDot    = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})[.\s_-]+(.*)$`)
	moviePatternParen  = regexp.MustCompile(`^(.+?)\s*\((\d{4})\)\s*(.*)$`)
	moviePatternSimple = regexp.MustCompile(`^(.+?)[.\s_-]+(\d{4})$`)
)

// ParseFilename parses a media filename into structured data.
func ParseFilename(filename string) *ParsedMedia {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	parsed := parseMediaName(name)
	parsed.FilePath = filename
	return parsed
}

// parseMediaName parses a media name (without file extension) into structured data.
// This is the core parsing logic shared by filename and directory name parsing.
func parseMediaName(name string) *ParsedMedia {
	parsed := &ParsedMedia{}

	if tryParseTVPatterns(name, parsed) {
		return parsed
	}

	if tryParseMoviePatterns(name, parsed) {
		return parsed
	}

	parsed.Title = cleanTitle(name)
	parseQualityInfo(name, parsed)
	return parsed
}

func tryParseTVPatterns(name string, parsed *ParsedMedia) bool {
	if match := tvPatternSE.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode, _ = strconv.Atoi(match[3])
		if match[4] != "" {
			parsed.EndEpisode, _ = strconv.Atoi(match[4])
		}
		parseQualityInfo(match[5], parsed)
		return true
	}

	if match := tvPatternX.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode, _ = strconv.Atoi(match[3])
		parseQualityInfo(match[4], parsed)
		return true
	}

	if match := tvPatternSeasonEpisodeSpelled.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode, _ = strconv.Atoi(match[3])
		parseQualityInfo(match[4], parsed)
		return true
	}

	return tryParseTVPackPatterns(name, parsed)
}

func tryParseTVPackPatterns(name string, parsed *ParsedMedia) bool {
	if match := tvPatternSeasonRange.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.IsCompleteSeries = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.EndSeason, _ = strconv.Atoi(match[3])
		parsed.Episode = 0
		parseQualityInfo(match[4], parsed)
		return true
	}

	if match := tvPatternSeasonPack.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode = 0
		parseQualityInfo(match[3], parsed)
		return true
	}

	if match := tvPatternSeasonSpelled.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season, _ = strconv.Atoi(match[2])
		parsed.Episode = 0
		parseQualityInfo(match[3], parsed)
		return true
	}

	if match := tvPatternComplete.FindStringSubmatch(name); match != nil {
		parsed.IsTV = true
		parsed.IsSeasonPack = true
		parsed.IsCompleteSeries = true
		parsed.Title = cleanTitle(match[1])
		parsed.Season = 0
		parsed.Episode = 0
		parseQualityInfo(match[2], parsed)
		return true
	}

	return false
}

func tryParseMoviePatterns(name string, parsed *ParsedMedia) bool {
	if match := moviePatternParen.FindStringSubmatch(name); match != nil {
		parsed.Title = cleanTitle(match[1])
		parsed.Year, _ = strconv.Atoi(match[2])
		parseQualityInfo(match[3], parsed)
		return true
	}

	if match := moviePatternDot.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			parsed.Title = cleanTitle(match[1])
			parsed.Year = year
			parseQualityInfo(match[3], parsed)
			return true
		}
	}

	if match := moviePatternSimple.FindStringSubmatch(name); match != nil {
		year, _ := strconv.Atoi(match[2])
		if year >= 1900 && year <= 2100 {
			parsed.Title = cleanTitle(match[1])
			parsed.Year = year
			return true
		}
	}

	return false
}

// cleanTitle cleans up a parsed title by replacing separators with spaces.
func cleanTitle(title string) string {
	return parseutil.CleanTitle(title)
}

// parseQualityInfo extracts quality, source, codec, and attributes from remaining text.
func parseQualityInfo(text string, parsed *ParsedMedia) {
	parseVideoQuality(text, parsed)
	parseHDRAndAttributes(text, parsed)
	parseAudioInfo(text, parsed)
	parseReleaseGroup(text, parsed)
	parseRevisionAndEdition(text, parsed)
	parseLanguages(text, parsed)
}

func parseVideoQuality(text string, parsed *ParsedMedia) {
	q, src, codec := parseutil.ParseVideoQuality(text)
	parsed.Quality = q
	parsed.Source = src
	if codec != "" {
		parsed.Codec = quality.NormalizeVideoCodec(codec)
	}
}

func parseHDRAndAttributes(text string, parsed *ParsedMedia) {
	hdrFormats := parseutil.ParseHDRFormats(text)

	if len(hdrFormats) > 0 {
		parsed.HDRFormats = hdrFormats
	} else {
		parsed.HDRFormats = []string{"SDR"}
	}

	var attributes []string
	if parsed.Source == "Remux" {
		attributes = append(attributes, "REMUX")
	}
	attributes = append(attributes, hdrFormats...)
	if len(attributes) > 0 {
		parsed.Attributes = attributes
	}
}

func parseAudioInfo(text string, parsed *ParsedMedia) {
	codecs, channels, enhancements := parseutil.ParseAudioInfo(text)

	for _, codec := range codecs {
		parsed.AudioCodecs = append(parsed.AudioCodecs, quality.NormalizeAudioCodec(codec))
	}
	for _, ch := range channels {
		parsed.AudioChannels = append(parsed.AudioChannels, quality.NormalizeAudioChannels(ch))
	}
	parsed.AudioEnhancements = enhancements
}

func parseReleaseGroup(text string, parsed *ParsedMedia) {
	parsed.ReleaseGroup = parseutil.ParseReleaseGroup(text)
}

func parseRevisionAndEdition(text string, parsed *ParsedMedia) {
	parsed.Revision = parseutil.ParseRevision(text)
	parsed.Edition = parseutil.ParseEdition(text)
}

func parseLanguages(text string, parsed *ParsedMedia) {
	parsed.Languages = parseutil.ParseLanguages(text)
}

// ParsePath tries to extract media info from a folder path.
// This is useful when the filename doesn't contain all information.

// tryInheritYearFromFolder attempts to get year from parent folder name
func tryInheritYearFromFolder(parsed *ParsedMedia, folderName string) {
	if parsed.IsTV || parsed.Year != 0 || folderName == "." || folderName == "/" {
		return
	}

	folderParsed := parseMediaName(folderName)
	if folderParsed.Year != 0 {
		parsed.Year = folderParsed.Year
		if folderParsed.Title != "" {
			parsed.Title = folderParsed.Title
		}
	}
}

// tryInheritYearFromSeriesFolder attempts to get year from the series folder for TV episodes.
// The series folder may be the parent (e.g., "ShowName (2020)/file.mkv") or the grandparent
// (e.g., "Vanished (2026)/Season 1/file.mkv") depending on whether season subfolders are used.
func tryInheritYearFromSeriesFolder(parsed *ParsedMedia, fullPath string) {
	if !parsed.IsTV || parsed.Year != 0 {
		return
	}

	dir := filepath.Dir(fullPath)

	// Try parent folder first (handles no season subfolder case)
	parentName := filepath.Base(dir)
	if parentName != "." && parentName != "/" {
		folderParsed := parseMediaName(parentName)
		if folderParsed.Year != 0 {
			parsed.Year = folderParsed.Year
			return
		}
	}

	// Try grandparent folder (handles season subfolder case)
	grandparentDir := filepath.Dir(dir)
	grandparentName := filepath.Base(grandparentDir)
	if grandparentName == "." || grandparentName == "/" {
		return
	}

	folderParsed := parseMediaName(grandparentName)
	if folderParsed.Year != 0 {
		parsed.Year = folderParsed.Year
	}
}

// tryInheritQualityFromFolder attempts to get quality info from parent folder
func tryInheritQualityFromFolder(parsed *ParsedMedia, folderName string) {
	if parsed.Quality != "" || parsed.Source != "" || folderName == "." || folderName == "/" {
		return
	}

	folderParsed := parseMediaName(folderName)
	inheritQualityInfo(parsed, folderParsed)
}

func ParsePath(fullPath string) *ParsedMedia {
	filename := filepath.Base(fullPath)
	parsed := ParseFilename(filename)

	dir := filepath.Dir(fullPath)
	folderName := filepath.Base(dir)

	tryInheritYearFromFolder(parsed, folderName)
	tryInheritYearFromSeriesFolder(parsed, fullPath)
	tryInheritQualityFromFolder(parsed, folderName)

	parsed.FilePath = pathutil.NormalizePath(fullPath)
	return parsed
}

// inheritQualityInfo copies quality-related fields from src to dst when dst has no quality info.
func inheritQualityInfo(dst, src *ParsedMedia) {
	if src.Quality == "" && src.Source == "" {
		return
	}
	inheritVideoFields(dst, src)
	inheritAudioFields(dst, src)
	inheritMetadataFields(dst, src)
}

func inheritVideoFields(dst, src *ParsedMedia) {
	if dst.Quality == "" {
		dst.Quality = src.Quality
	}
	if dst.Source == "" {
		dst.Source = src.Source
	}
	if dst.Codec == "" {
		dst.Codec = src.Codec
	}
	if len(dst.HDRFormats) == 1 && dst.HDRFormats[0] == "SDR" && len(src.HDRFormats) > 0 {
		dst.HDRFormats = src.HDRFormats
	}
	if len(dst.Attributes) == 0 {
		dst.Attributes = src.Attributes
	}
}

func inheritAudioFields(dst, src *ParsedMedia) {
	if len(dst.AudioCodecs) == 0 {
		dst.AudioCodecs = src.AudioCodecs
	}
	if len(dst.AudioChannels) == 0 {
		dst.AudioChannels = src.AudioChannels
	}
	if len(dst.AudioEnhancements) == 0 {
		dst.AudioEnhancements = src.AudioEnhancements
	}
}

func inheritMetadataFields(dst, src *ParsedMedia) {
	if dst.ReleaseGroup == "" {
		dst.ReleaseGroup = src.ReleaseGroup
	}
	if dst.Revision == "" {
		dst.Revision = src.Revision
	}
	if dst.Edition == "" {
		dst.Edition = src.Edition
	}
	if len(dst.Languages) == 0 {
		dst.Languages = src.Languages
	}
}

// ToReleaseAttributes converts ParsedMedia to quality.ReleaseAttributes for profile matching.
// This bridges the parser output to the quality matching system.
func (p *ParsedMedia) ToReleaseAttributes() quality.ReleaseAttributes {
	return quality.ReleaseAttributes{
		HDRFormats:    p.HDRFormats,
		VideoCodec:    p.Codec,
		AudioCodecs:   p.AudioCodecs,
		AudioChannels: p.AudioChannels,
	}
}
