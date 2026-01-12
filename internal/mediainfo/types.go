package mediainfo

import (
	"strings"
	"time"
)

// MediaInfo holds extracted media file information.
type MediaInfo struct {
	// Video
	VideoCodec       string `json:"videoCodec"`
	VideoBitDepth    int    `json:"videoBitDepth"`
	VideoResolution  string `json:"videoResolution"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	DynamicRange     string `json:"dynamicRange"`
	DynamicRangeType string `json:"dynamicRangeType"`

	// Audio
	AudioCodec    string   `json:"audioCodec"`
	AudioChannels string   `json:"audioChannels"`
	AudioLanguages []string `json:"audioLanguages"`

	// Subtitles
	SubtitleLanguages []string `json:"subtitleLanguages"`

	// Container
	ContainerFormat string        `json:"containerFormat"`
	Duration        time.Duration `json:"duration"`
	FileSize        int64         `json:"fileSize"`
}

// VideoCodecMap maps raw codec names to standard display names.
var VideoCodecMap = map[string]string{
	"hevc":      "HEVC",
	"h265":      "HEVC",
	"h.265":     "HEVC",
	"x265":      "x265",
	"h264":      "H.264",
	"h.264":     "H.264",
	"avc":       "H.264",
	"x264":      "x264",
	"av1":       "AV1",
	"vp9":       "VP9",
	"vp8":       "VP8",
	"mpeg2":     "MPEG2",
	"mpeg-2":    "MPEG2",
	"vc1":       "VC-1",
	"xvid":      "XviD",
	"divx":      "DivX",
}

// AudioCodecMap maps raw audio codec names to standard display names.
var AudioCodecMap = map[string]string{
	"dts-hd ma":         "DTS-HD MA",
	"dts-hd master":     "DTS-HD MA",
	"dts-hd":            "DTS-HD",
	"dts":               "DTS",
	"truehd":            "TrueHD",
	"truehd atmos":      "TrueHD Atmos",
	"dolby truehd":      "TrueHD",
	"dolby truehd atmos": "TrueHD Atmos",
	"e-ac-3":            "EAC3",
	"eac3":              "EAC3",
	"e-ac-3 atmos":      "EAC3 Atmos",
	"ac3":               "AC3",
	"ac-3":              "AC3",
	"dolby digital":     "AC3",
	"aac":               "AAC",
	"he-aac":            "HE-AAC",
	"flac":              "FLAC",
	"opus":              "Opus",
	"mp3":               "MP3",
	"pcm":               "PCM",
	"vorbis":            "Vorbis",
}

// HDRType represents different HDR formats.
type HDRType string

const (
	HDRTypeNone       HDRType = ""
	HDRTypeSDR        HDRType = "SDR"
	HDRTypeHDR10      HDRType = "HDR10"
	HDRTypeHDR10Plus  HDRType = "HDR10+"
	HDRTypeDolbyVision HDRType = "DV"
	HDRTypeDVHDR10    HDRType = "DV HDR10"
	HDRTypeHLG        HDRType = "HLG"
	HDRTypeGenericHDR HDRType = "HDR"
)

// NormalizeVideoCodec normalizes a video codec name to its standard form.
func NormalizeVideoCodec(codec string) string {
	lower := normalizeString(codec)

	// Check for x264/x265 encoder signatures
	if containsAny(lower, "x264") {
		return "x264"
	}
	if containsAny(lower, "x265") {
		return "x265"
	}

	if normalized, ok := VideoCodecMap[lower]; ok {
		return normalized
	}

	// Check partial matches
	for key, value := range VideoCodecMap {
		if containsAny(lower, key) {
			return value
		}
	}

	return codec
}

// NormalizeAudioCodec normalizes an audio codec name to its standard form.
func NormalizeAudioCodec(codec string, hasAtmos bool) string {
	lower := normalizeString(codec)

	// Handle Atmos variants
	if hasAtmos {
		if containsAny(lower, "truehd") {
			return "TrueHD Atmos"
		}
		if containsAny(lower, "eac3", "e-ac-3", "ec3") {
			return "EAC3 Atmos"
		}
	}

	if normalized, ok := AudioCodecMap[lower]; ok {
		return normalized
	}

	// Check partial matches
	for key, value := range AudioCodecMap {
		if containsAny(lower, key) {
			return value
		}
	}

	return codec
}

// FormatChannels formats audio channel layout.
func FormatChannels(channels int, layout string) string {
	switch {
	case containsAny(normalizeString(layout), "7.1", "8 channels"):
		return "7.1"
	case containsAny(normalizeString(layout), "5.1", "6 channels"):
		return "5.1"
	case containsAny(normalizeString(layout), "stereo", "2 channels", "2.0"):
		return "2.0"
	case containsAny(normalizeString(layout), "mono", "1 channel"):
		return "1.0"
	}

	// Fall back to channel count
	switch channels {
	case 8:
		return "7.1"
	case 6:
		return "5.1"
	case 2:
		return "2.0"
	case 1:
		return "1.0"
	default:
		if channels > 0 {
			return formatChannelCount(channels)
		}
		return ""
	}
}

func formatChannelCount(count int) string {
	// Convert channel count to approximate layout
	switch {
	case count >= 8:
		return "7.1"
	case count >= 6:
		return "5.1"
	case count >= 2:
		return "2.0"
	default:
		return "1.0"
	}
}

// normalizeString lowercases and trims a string.
func normalizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

