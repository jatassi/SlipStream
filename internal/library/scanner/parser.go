package scanner

import (
	"path/filepath"
	"strings"
	"sync"

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

// ParseResultData is a local representation of module.ParseResult, defined here to
// avoid importing the module package (which would create an import cycle).
type ParseResultData struct {
	Title         string
	Year          int
	Quality       string
	Source        string
	Codec         string
	HDRFormats    []string
	AudioCodecs   []string
	AudioChannels []string
	ReleaseGroup  string
	Revision      string
	Edition       string
	Languages     []string
	IsTV          bool
	Extra         any
}

// tvExtraAccessor extracts TV-specific fields from ParseResultData.Extra.
type tvExtraAccessor interface {
	TVSeason() int
	TVEndSeason() int
	TVEpisode() int
	TVEndEpisode() int
	TVIsSeasonPack() bool
	TVIsCompleteSeries() bool
}

// ModuleFileParser is the scanner's view of a module's file parsing capability.
type ModuleFileParser interface {
	ID() string
	TryMatch(filename string) (confidence float64, result *ParseResultData)
}

// ModuleParserRegistry provides access to module file parsers for delegation.
type ModuleParserRegistry interface {
	AllFileParsers() []ModuleFileParser
}

// Package-level registry for module-based parsing delegation.
var (
	globalRegistry   ModuleParserRegistry
	globalRegistryMu sync.RWMutex
)

// SetGlobalRegistry sets the module registry used by ParseFilename and ParsePath
// to delegate filename parsing to module FileParsers. This is called once at
// startup after modules are registered.
func SetGlobalRegistry(reg ModuleParserRegistry) {
	globalRegistryMu.Lock()
	defer globalRegistryMu.Unlock()
	globalRegistry = reg
}

func getGlobalRegistry() ModuleParserRegistry {
	globalRegistryMu.RLock()
	defer globalRegistryMu.RUnlock()
	return globalRegistry
}

// ParseFilename parses a media filename into structured data.
// When a module registry is available, delegates to module FileParsers via TryMatch.
// Falls back to standalone parsing (quality/title extraction only) otherwise.
func ParseFilename(filename string) *ParsedMedia {
	if reg := getGlobalRegistry(); reg != nil {
		if parsed := parseFilenameViaModules(filename, reg); parsed != nil {
			return parsed
		}
	}

	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	parsed := parseFallback(name)
	parsed.FilePath = filename
	return parsed
}

// parseFilenameViaModules iterates all registered modules and picks the highest-confidence match.
// If the filename has no recognized video extension (e.g., release/torrent titles), retries
// with a dummy extension so module TryMatch video-extension checks can pass.
func parseFilenameViaModules(filename string, reg ModuleParserRegistry) *ParsedMedia {
	bestResult := tryMatchAllParsers(filename, reg)

	if bestResult == nil && !parseutil.IsVideoFile(filename) {
		bestResult = tryMatchAllParsers(filename+".mkv", reg)
	}

	if bestResult == nil {
		return nil
	}

	parsed := resultToParsedMedia(bestResult)
	parsed.FilePath = filename
	return parsed
}

func tryMatchAllParsers(filename string, reg ModuleParserRegistry) *ParseResultData {
	var bestResult *ParseResultData
	var bestConfidence float64

	for _, parser := range reg.AllFileParsers() {
		confidence, result := parser.TryMatch(filename)
		if confidence > bestConfidence && result != nil {
			bestConfidence = confidence
			bestResult = result
		}
	}

	return bestResult
}

// resultToParsedMedia converts a ParseResultData to ParsedMedia for backward compatibility.
func resultToParsedMedia(result *ParseResultData) *ParsedMedia {
	parsed := &ParsedMedia{
		Title:         result.Title,
		Year:          result.Year,
		Quality:       result.Quality,
		Source:        result.Source,
		Codec:         result.Codec,
		HDRFormats:    result.HDRFormats,
		AudioCodecs:   result.AudioCodecs,
		AudioChannels: result.AudioChannels,
		ReleaseGroup:  result.ReleaseGroup,
		Revision:      result.Revision,
		Edition:       result.Edition,
		Languages:     result.Languages,
		IsTV:          result.IsTV,
	}

	// Extract TV-specific fields from Extra if present.
	if result.Extra != nil {
		populateTVFields(parsed, result.Extra)
	}

	computeAttributes(parsed)
	return parsed
}

// populateTVFields extracts TV-specific data from ParseResultData.Extra.
// Uses interface assertions to avoid importing the tv module package directly.
func populateTVFields(parsed *ParsedMedia, extra any) {
	if accessor, ok := extra.(tvExtraAccessor); ok {
		parsed.Season = accessor.TVSeason()
		parsed.EndSeason = accessor.TVEndSeason()
		parsed.Episode = accessor.TVEpisode()
		parsed.EndEpisode = accessor.TVEndEpisode()
		parsed.IsSeasonPack = accessor.TVIsSeasonPack()
		parsed.IsCompleteSeries = accessor.TVIsCompleteSeries()
	}
}

// computeAttributes builds the Attributes display field from HDRFormats and Source.
func computeAttributes(parsed *ParsedMedia) {
	if len(parsed.Attributes) > 0 {
		return
	}

	var attributes []string
	if parsed.Source == "Remux" {
		attributes = append(attributes, "REMUX")
	}
	hdrFormats := parsed.HDRFormats
	for _, f := range hdrFormats {
		if f != "SDR" {
			attributes = append(attributes, f)
		}
	}
	if len(attributes) > 0 {
		parsed.Attributes = attributes
	}
}

// parseFallback handles names that don't match any module pattern.
// Extracts quality info and cleans the title without media-type-specific parsing.
func parseFallback(name string) *ParsedMedia {
	parsed := &ParsedMedia{}
	parsed.Title = cleanTitle(name)
	parseQualityInfo(name, parsed)
	return parsed
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

// parseFolderName parses a folder name for year/quality extraction only.
// This is used for folder inheritance and does not need module delegation.
func parseFolderName(name string) *ParsedMedia {
	if reg := getGlobalRegistry(); reg != nil {
		if parsed := parseNameViaModules(name, reg); parsed != nil {
			return parsed
		}
	}
	return parseFallback(name)
}

// parseNameViaModules tries to parse a bare name (no extension) via module FileParsers.
// Adds a dummy extension so TryMatch can pass its IsVideoFile check.
func parseNameViaModules(name string, reg ModuleParserRegistry) *ParsedMedia {
	bestResult := tryMatchAllParsers(name+".mkv", reg)
	if bestResult == nil {
		return nil
	}

	return resultToParsedMedia(bestResult)
}

// tryInheritYearFromFolder attempts to get year from parent folder name
func tryInheritYearFromFolder(parsed *ParsedMedia, folderName string) {
	if parsed.IsTV || parsed.Year != 0 || folderName == "." || folderName == "/" {
		return
	}

	folderParsed := parseFolderName(folderName)
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
		folderParsed := parseFolderName(parentName)
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

	folderParsed := parseFolderName(grandparentName)
	if folderParsed.Year != 0 {
		parsed.Year = folderParsed.Year
	}
}

// tryInheritQualityFromFolder attempts to get quality info from parent folder
func tryInheritQualityFromFolder(parsed *ParsedMedia, folderName string) {
	if parsed.Quality != "" || parsed.Source != "" || folderName == "." || folderName == "/" {
		return
	}

	folderParsed := parseFolderName(folderName)
	inheritQualityInfo(parsed, folderParsed)
}

// ParsePath parses a full file path, inheriting info from parent directories when needed.
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
