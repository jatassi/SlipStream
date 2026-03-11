package parseutil

import "regexp"

// Quality resolution patterns
var (
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
		"HDR10":  regexp.MustCompile(`(?i)hdr10(?:[^\+p]|$)`),
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
)

// Ordered slices for deterministic iteration (Go maps have random order).
var (
	sourceOrder       = []string{"Remux", "BluRay", "WEBRip", "WEB-DL", "HDTV", "DVDRip", "SDTV", "CAM"}
	hdrOrder          = []string{"DV", "HDR10+", "HDR10", "HDR", "HLG"}
	audioCodecOrder   = []string{"TrueHD", "DTS-HD MA", "DTS-HD", "DTS", "DDP", "DD", "AAC", "FLAC", "LPCM", "PCM", "Opus", "MP3"}
	audioChannelOrder = []string{"7.1", "5.1", "2.0", "1.0"}
)

// QualityAttributes holds all quality-related parsed info.
type QualityAttributes struct {
	Quality           string
	Source            string
	Codec             string
	HDRFormats        []string
	AudioCodecs       []string
	AudioChannels     []string
	AudioEnhancements []string
}

// ParseVideoQuality extracts resolution, source, and codec from a filename.
func ParseVideoQuality(filename string) (quality, source, codec string) {
	for q, pattern := range qualityPatterns {
		if pattern.MatchString(filename) {
			quality = q
			break
		}
	}

	for _, src := range sourceOrder {
		if pattern, ok := sourcePatterns[src]; ok && pattern.MatchString(filename) {
			source = src
			break
		}
	}

	for c, pattern := range codecPatterns {
		if pattern.MatchString(filename) {
			codec = c
			break
		}
	}

	return quality, source, codec
}

// ParseHDRFormats extracts HDR format info from a filename.
func ParseHDRFormats(filename string) []string {
	var formats []string
	for _, hdr := range hdrOrder {
		if pattern, ok := hdrPatterns[hdr]; ok && pattern.MatchString(filename) {
			formats = append(formats, hdr)
		}
	}
	return formats
}

// ParseAudioInfo extracts audio codecs, channels, and enhancements from a filename.
func ParseAudioInfo(filename string) (codecs, channels, enhancements []string) {
	for _, codec := range audioCodecOrder {
		if pattern, ok := audioCodecPatterns[codec]; ok && pattern.MatchString(filename) {
			codecs = append(codecs, codec)
		}
	}

	for _, ch := range audioChannelOrder {
		if pattern, ok := audioChannelPatterns[ch]; ok && pattern.MatchString(filename) {
			channels = append(channels, ch)
		}
	}

	if audioEnhancementPatterns["Atmos"].MatchString(filename) {
		enhancements = append(enhancements, "Atmos")
	}
	if audioEnhancementPatterns["DTS:X"].MatchString(filename) {
		enhancements = append(enhancements, "DTS:X")
	}

	return codecs, channels, enhancements
}

// DetectQualityAttributes is a convenience that calls all quality parsers and returns a consolidated result.
func DetectQualityAttributes(filename string) QualityAttributes {
	quality, source, codec := ParseVideoQuality(filename)
	hdrFormats := ParseHDRFormats(filename)
	codecs, channels, enhancements := ParseAudioInfo(filename)

	return QualityAttributes{
		Quality:           quality,
		Source:            source,
		Codec:             codec,
		HDRFormats:        hdrFormats,
		AudioCodecs:       codecs,
		AudioChannels:     channels,
		AudioEnhancements: enhancements,
	}
}
