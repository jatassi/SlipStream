//nolint:revive // Package name intentionally matches stdlib scanner for domain clarity
package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/slipstream/slipstream/internal/library/quality"
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

	// Quality patterns
	qualityPatterns = map[string]*regexp.Regexp{
		"2160p": regexp.MustCompile(`(?i)(2160p|4k|uhd)`),
		"1080p": regexp.MustCompile(`(?i)1080p`),
		"720p":  regexp.MustCompile(`(?i)720p`),
		"480p":  regexp.MustCompile(`(?i)(480p|sd)`),
	}

	// Source patterns
	sourcePatterns = map[string]*regexp.Regexp{
		"BluRay": regexp.MustCompile(`(?i)(blu-?ray|bdrip|brrip|bdremux)`),
		"WEB-DL": regexp.MustCompile(`(?i)(web-?dl|webdl|\bweb\b)`),
		"WEBRip": regexp.MustCompile(`(?i)web-?rip`),
		"HDTV":   regexp.MustCompile(`(?i)hdtv`),
		"DVDRip": regexp.MustCompile(`(?i)(dvdrip|dvd-?r)`),
		"SDTV":   regexp.MustCompile(`(?i)(sdtv|pdtv|dsr)`),
		"CAM":    regexp.MustCompile(`(?i)(cam|hdcam|ts|telesync)`),
		"Remux":  regexp.MustCompile(`(?i)remux`),
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
		"DV":     regexp.MustCompile(`(?i)(dolby[.\s]?vision|dovi|\.dv\.)`),
		"HDR10+": regexp.MustCompile(`(?i)hdr10(\+|plus)`),
		"HDR10":  regexp.MustCompile(`(?i)hdr10(?:[^\+p]|$)`), // HDR10 not followed by + or 'p' (plus)
		"HDR":    regexp.MustCompile(`(?i)[.\s\-]hdr[.\s\-]`),
		"HLG":    regexp.MustCompile(`(?i)hlg`),
	}

	// Audio codec patterns (order matters - more specific patterns first)
	audioCodecPatterns = map[string]*regexp.Regexp{
		"TrueHD":    regexp.MustCompile(`(?i)true[.\-]?hd`),
		"DTS-HD MA": regexp.MustCompile(`(?i)dts[.\-]?hd[.\-]?ma`),
		"DTS-HD":    regexp.MustCompile(`(?i)dts[.\-]?hd`),
		"DTS":       regexp.MustCompile(`(?i)[.\s\-_]dts[.\s\-_]`),
		"DDP":       regexp.MustCompile(`(?i)([.\s\-_]ddp[.\s\-_\d]|dd\+|e[.\-]?ac[.\-]?3)`),
		"DD":        regexp.MustCompile(`(?i)([.\s\-_]dd[.\s\-_\d]|[.\s\-_]ac[.\-]?3[.\s\-_])`),
		"AAC":       regexp.MustCompile(`(?i)[.\s\-_]aac[.\s\-_\d]`),
		"FLAC":      regexp.MustCompile(`(?i)[.\s\-_]flac[.\s\-_]`),
		"LPCM":      regexp.MustCompile(`(?i)[.\s\-_]lpcm[.\s\-_]`),
		"PCM":       regexp.MustCompile(`(?i)[.\s\-_]pcm[.\s\-_]`),
		"Opus":      regexp.MustCompile(`(?i)[.\s\-_]opus[.\s\-_]`),
		"MP3":       regexp.MustCompile(`(?i)[.\s\-_]mp3[.\s\-_]`),
	}

	// Audio channel patterns
	audioChannelPatterns = map[string]*regexp.Regexp{
		"7.1": regexp.MustCompile(`(?i)7[.\-]1`),
		"5.1": regexp.MustCompile(`(?i)5[.\-]1`),
		"2.0": regexp.MustCompile(`(?i)2[.\-]0`),
		"1.0": regexp.MustCompile(`(?i)1[.\-]0`),
	}

	// Audio enhancement patterns (object-based audio layers)
	audioEnhancementPatterns = map[string]*regexp.Regexp{
		"Atmos": regexp.MustCompile(`(?i)atmos`),
		"DTS:X": regexp.MustCompile(`(?i)dts[.\-:]?x`),
	}

	// Release group pattern - typically at the end after a dash: -GroupName
	releaseGroupPattern = regexp.MustCompile(`-([A-Za-z0-9]+)(?:\.[a-z0-9]{2,4})?$`)

	// Revision patterns - PROPER, REPACK, REAL, RERIP
	revisionPatterns = map[string]*regexp.Regexp{
		"Proper": regexp.MustCompile(`(?i)(^|[.\s\-_])proper([.\s\-_]|$)`),
		"REPACK": regexp.MustCompile(`(?i)(^|[.\s\-_])repack([.\s\-_]|$)`),
		"REAL":   regexp.MustCompile(`(?i)(^|[.\s\-_])real([.\s\-_]|$)`),
		"RERIP":  regexp.MustCompile(`(?i)(^|[.\s\-_])rerip([.\s\-_]|$)`),
	}

	// Edition patterns - various special editions
	editionPatterns = map[string]*regexp.Regexp{
		"Director's Cut":      regexp.MustCompile(`(?i)(^|[.\s\-_])directors?[.\s\-_]?cut([.\s\-_]|$)`),
		"Extended":            regexp.MustCompile(`(?i)(^|[.\s\-_])extended([.\s\-_]|$)`),
		"Extended Cut":        regexp.MustCompile(`(?i)(^|[.\s\-_])extended[.\s\-_]?cut([.\s\-_]|$)`),
		"Theatrical":          regexp.MustCompile(`(?i)(^|[.\s\-_])theatrical[.\s\-_]?(?:cut|edition)?([.\s\-_]|$)`),
		"Unrated":             regexp.MustCompile(`(?i)(^|[.\s\-_])unrated([.\s\-_]|$)`),
		"Uncut":               regexp.MustCompile(`(?i)(^|[.\s\-_])uncut([.\s\-_]|$)`),
		"Ultimate Cut":        regexp.MustCompile(`(?i)(^|[.\s\-_])ultimate[.\s\-_]?cut([.\s\-_]|$)`),
		"Final Cut":           regexp.MustCompile(`(?i)(^|[.\s\-_])final[.\s\-_]?cut([.\s\-_]|$)`),
		"Special Edition":     regexp.MustCompile(`(?i)(^|[.\s\-_])special[.\s\-_]?edition([.\s\-_]|$)`),
		"Collector's Edition": regexp.MustCompile(`(?i)(^|[.\s\-_])collectors?[.\s\-_]?edition([.\s\-_]|$)`),
		"Anniversary Edition": regexp.MustCompile(`(?i)(^|[.\s\-_])anniversary[.\s\-_]?edition([.\s\-_]|$)`),
		"Criterion":           regexp.MustCompile(`(?i)(^|[.\s\-_])criterion([.\s\-_]|$)`),
		"IMAX":                regexp.MustCompile(`(?i)(^|[.\s\-_])imax([.\s\-_]|$)`),
		"3D":                  regexp.MustCompile(`(?i)(^|[.\s\-_])3d([.\s\-_]|$)`),
		"Remastered":          regexp.MustCompile(`(?i)(^|[.\s\-_])remastered([.\s\-_]|$)`),
		"Restored":            regexp.MustCompile(`(?i)(^|[.\s\-_])restored([.\s\-_]|$)`),
	}

	// Language patterns - detect non-English releases
	// These patterns look for language indicators in release titles
	languagePatterns = map[string]*regexp.Regexp{
		"German":     regexp.MustCompile(`(?i)(^|[.\s\-_])(german|deutsch|ger|deu)([.\s\-_]|$)`),
		"French":     regexp.MustCompile(`(?i)(^|[.\s\-_])(french|français|fra|fre)([.\s\-_]|$)`),
		"Spanish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(spanish|español|spa|esp)([.\s\-_]|$)`),
		"Italian":    regexp.MustCompile(`(?i)(^|[.\s\-_])(italian|italiano|ita)([.\s\-_]|$)`),
		"Portuguese": regexp.MustCompile(`(?i)(^|[.\s\-_])(portuguese|português|por|pt-br)([.\s\-_]|$)`),
		"Russian":    regexp.MustCompile(`(?i)(^|[.\s\-_])(russian|русский|rus)([.\s\-_]|$)`),
		"Japanese":   regexp.MustCompile(`(?i)(^|[.\s\-_])(japanese|日本語|jpn|jap)([.\s\-_]|$)`),
		"Korean":     regexp.MustCompile(`(?i)(^|[.\s\-_])(korean|한국어|kor)([.\s\-_]|$)`),
		"Chinese":    regexp.MustCompile(`(?i)(^|[.\s\-_])(chinese|中文|chi|chs|cht|mandarin|cantonese)([.\s\-_]|$)`),
		"Dutch":      regexp.MustCompile(`(?i)(^|[.\s\-_])(dutch|nederlands|nld|dut)([.\s\-_]|$)`),
		"Polish":     regexp.MustCompile(`(?i)(^|[.\s\-_])(polish|polski|pol)([.\s\-_]|$)`),
		"Swedish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(swedish|svenska|swe)([.\s\-_]|$)`),
		"Norwegian":  regexp.MustCompile(`(?i)(^|[.\s\-_])(norwegian|norsk|nor)([.\s\-_]|$)`),
		"Danish":     regexp.MustCompile(`(?i)(^|[.\s\-_])(danish|dansk|dan)([.\s\-_]|$)`),
		"Finnish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(finnish|suomi|fin)([.\s\-_]|$)`),
		"Turkish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(turkish|türkçe|tur)([.\s\-_]|$)`),
		"Hindi":      regexp.MustCompile(`(?i)(^|[.\s\-_])(hindi|hin)([.\s\-_]|$)`),
		"Arabic":     regexp.MustCompile(`(?i)(^|[.\s\-_])(arabic|العربية|ara)([.\s\-_]|$)`),
		"Hebrew":     regexp.MustCompile(`(?i)(^|[.\s\-_])(hebrew|עברית|heb)([.\s\-_]|$)`),
		"Czech":      regexp.MustCompile(`(?i)(^|[.\s\-_])(czech|čeština|cze|ces)([.\s\-_]|$)`),
		"Hungarian":  regexp.MustCompile(`(?i)(^|[.\s\-_])(hungarian|magyar|hun)([.\s\-_]|$)`),
		"Greek":      regexp.MustCompile(`(?i)(^|[.\s\-_])(greek|ελληνικά|gre|ell)([.\s\-_]|$)`),
		"Thai":       regexp.MustCompile(`(?i)(^|[.\s\-_])(thai|ไทย|tha)([.\s\-_]|$)`),
		"Vietnamese": regexp.MustCompile(`(?i)(^|[.\s\-_])(vietnamese|tiếng việt|vie)([.\s\-_]|$)`),
		"Indonesian": regexp.MustCompile(`(?i)(^|[.\s\-_])(indonesian|bahasa indonesia|ind)([.\s\-_]|$)`),
		"Romanian":   regexp.MustCompile(`(?i)(^|[.\s\-_])(romanian|română|ron|rum)([.\s\-_]|$)`),
		"Ukrainian":  regexp.MustCompile(`(?i)(^|[.\s\-_])(ukrainian|українська|ukr)([.\s\-_]|$)`),
	}

	// Clean up patterns
	cleanupPattern = regexp.MustCompile(`[.\s_-]+`)
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
	// Replace common separators with spaces
	cleaned := cleanupPattern.ReplaceAllString(title, " ")
	// Trim and normalize whitespace
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
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
	for q, pattern := range qualityPatterns {
		if pattern.MatchString(text) {
			parsed.Quality = q
			break
		}
	}

	sourceOrder := []string{"Remux", "BluRay", "WEBRip", "WEB-DL", "HDTV", "DVDRip", "SDTV", "CAM"}
	for _, source := range sourceOrder {
		if pattern, ok := sourcePatterns[source]; ok && pattern.MatchString(text) {
			parsed.Source = source
			break
		}
	}

	for codec, pattern := range codecPatterns {
		if pattern.MatchString(text) {
			parsed.Codec = quality.NormalizeVideoCodec(codec)
			break
		}
	}
}

func parseHDRAndAttributes(text string, parsed *ParsedMedia) {
	var hdrFormats []string
	hdrOrder := []string{"DV", "HDR10+", "HDR10", "HDR", "HLG"}
	for _, hdr := range hdrOrder {
		if pattern, ok := hdrPatterns[hdr]; ok && pattern.MatchString(text) {
			hdrFormats = append(hdrFormats, hdr)
		}
	}

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
	audioCodecOrder := []string{"TrueHD", "DTS-HD MA", "DTS-HD", "DTS", "DDP", "DD", "AAC", "FLAC", "LPCM", "PCM", "Opus", "MP3"}
	for _, codec := range audioCodecOrder {
		if pattern, ok := audioCodecPatterns[codec]; ok && pattern.MatchString(text) {
			parsed.AudioCodecs = append(parsed.AudioCodecs, quality.NormalizeAudioCodec(codec))
		}
	}

	channelOrder := []string{"7.1", "5.1", "2.0", "1.0"}
	for _, channels := range channelOrder {
		if pattern, ok := audioChannelPatterns[channels]; ok && pattern.MatchString(text) {
			parsed.AudioChannels = append(parsed.AudioChannels, quality.NormalizeAudioChannels(channels))
		}
	}

	if audioEnhancementPatterns["Atmos"].MatchString(text) {
		parsed.AudioEnhancements = append(parsed.AudioEnhancements, "Atmos")
	}
	if audioEnhancementPatterns["DTS:X"].MatchString(text) {
		parsed.AudioEnhancements = append(parsed.AudioEnhancements, "DTS:X")
	}
}

var releaseGroupFalsePositives = map[string]bool{
	"x264": true, "x265": true, "hevc": true, "avc": true,
	"h264": true, "h265": true, "xvid": true, "divx": true,
	"av1": true, "vp9": true, "mkv": true, "mp4": true, "avi": true,
}

func parseReleaseGroup(text string, parsed *ParsedMedia) {
	match := releaseGroupPattern.FindStringSubmatch(text)
	if match == nil {
		return
	}
	if !releaseGroupFalsePositives[strings.ToLower(match[1])] {
		parsed.ReleaseGroup = match[1]
	}
}

func parseRevisionAndEdition(text string, parsed *ParsedMedia) {
	revisionOrder := []string{"Proper", "REPACK", "REAL", "RERIP"}
	for _, rev := range revisionOrder {
		if pattern, ok := revisionPatterns[rev]; ok && pattern.MatchString(text) {
			parsed.Revision = rev
			break
		}
	}

	var editions []string
	editionOrder := []string{"Director's Cut", "Extended Cut", "Extended", "Theatrical", "Unrated", "Uncut", "Ultimate Cut", "Final Cut", "Special Edition", "Collector's Edition", "Anniversary Edition", "Criterion", "IMAX", "3D", "Remastered", "Restored"}
	for _, ed := range editionOrder {
		if pattern, ok := editionPatterns[ed]; ok && pattern.MatchString(text) {
			editions = append(editions, ed)
		}
	}
	if len(editions) > 0 {
		parsed.Edition = strings.Join(editions, " ")
	}
}

func parseLanguages(text string, parsed *ParsedMedia) {
	for lang, pattern := range languagePatterns {
		if pattern.MatchString(text) {
			parsed.Languages = append(parsed.Languages, lang)
		}
	}
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
	tryInheritQualityFromFolder(parsed, folderName)

	parsed.FilePath = fullPath
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
