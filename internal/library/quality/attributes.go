package quality

import (
	"encoding/json"
	"strings"
)

// AttributeMode defines how an attribute setting should be applied (Req 2.1.5)
type AttributeMode string

const (
	AttributeModeAcceptable AttributeMode = "acceptable" // No filtering, accept anything
	AttributeModePreferred  AttributeMode = "preferred"  // Scoring bonus for matches
	AttributeModeRequired   AttributeMode = "required"   // Hard filter, must match
	AttributeModeNotAllowed AttributeMode = "notAllowed" // Hard reject, must not match
)

// AttributeSettings holds per-item mode configuration for an attribute category
type AttributeSettings struct {
	Items map[string]AttributeMode `json:"items"` // value -> mode mapping (e.g., "DV" -> "required")
}

// DefaultAttributeSettings returns settings with all values set to "acceptable"
func DefaultAttributeSettings() AttributeSettings {
	return AttributeSettings{
		Items: make(map[string]AttributeMode),
	}
}

// GetMode returns the mode for a specific value, defaulting to "acceptable" if not set
func (s AttributeSettings) GetMode(value string) AttributeMode {
	if mode, ok := s.Items[value]; ok {
		return mode
	}
	return AttributeModeAcceptable
}

// GetRequired returns all values with "required" mode
func (s AttributeSettings) GetRequired() []string {
	var result []string
	for value, mode := range s.Items {
		if mode == AttributeModeRequired {
			result = append(result, value)
		}
	}
	return result
}

// GetPreferred returns all values with "preferred" mode
func (s AttributeSettings) GetPreferred() []string {
	var result []string
	for value, mode := range s.Items {
		if mode == AttributeModePreferred {
			result = append(result, value)
		}
	}
	return result
}

// GetNotAllowed returns all values with "notAllowed" mode
func (s AttributeSettings) GetNotAllowed() []string {
	var result []string
	for value, mode := range s.Items {
		if mode == AttributeModeNotAllowed {
			result = append(result, value)
		}
	}
	return result
}

// HasNonDefaultSettings returns true if any item has a mode other than "acceptable"
func (s AttributeSettings) HasNonDefaultSettings() bool {
	for _, mode := range s.Items {
		if mode != AttributeModeAcceptable {
			return true
		}
	}
	return false
}

// Req 2.2: All supported attribute values

// HDRFormats lists all supported HDR format identifiers
var HDRFormats = []string{
	"DV",     // Dolby Vision
	"HDR10+", // HDR10+
	"HDR10",  // HDR10
	"HDR",    // Generic HDR
	"HLG",    // Hybrid Log-Gamma
	"SDR",    // Standard Dynamic Range (no HDR)
}

// VideoCodecs lists all supported video codec identifiers
var VideoCodecs = []string{
	"x264",  // H.264/AVC
	"x265",  // H.265/HEVC
	"AV1",   // AV1
	"VP9",   // VP9
	"XviD",  // XviD
	"DivX",  // DivX
	"MPEG2", // MPEG-2
}

// AudioCodecs lists all supported audio codec identifiers
var AudioCodecs = []string{
	"TrueHD",    // Dolby TrueHD
	"DTS-HD MA", // DTS-HD Master Audio
	"DTS-HD",    // DTS-HD (non-MA)
	"DTS",       // DTS
	"DDP",       // Dolby Digital Plus (E-AC3)
	"DD",        // Dolby Digital (AC3)
	"AAC",       // AAC
	"FLAC",      // FLAC
	"LPCM",      // Linear PCM
	"Opus",      // Opus
	"MP3",       // MP3
}

// AudioChannels lists all supported audio channel configurations
var AudioChannels = []string{
	"7.1", // 7.1 surround
	"5.1", // 5.1 surround
	"2.0", // Stereo
	"1.0", // Mono
}

// SerializeAttributeSettings converts AttributeSettings to JSON string
func SerializeAttributeSettings(settings AttributeSettings) (string, error) {
	data, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// legacyAttributeSettings represents the old format for migration purposes
type legacyAttributeSettings struct {
	Mode   AttributeMode `json:"mode"`
	Values []string      `json:"values"`
}

// DeserializeAttributeSettings parses JSON to AttributeSettings
// Handles migration from old format (mode + values) to new format (items map)
func DeserializeAttributeSettings(data string) (AttributeSettings, error) {
	if data == "" {
		return DefaultAttributeSettings(), nil
	}

	// Try new format first (has "items" key)
	var settings AttributeSettings
	if err := json.Unmarshal([]byte(data), &settings); err == nil && settings.Items != nil {
		return settings, nil
	}

	// Try old format and convert
	var legacy legacyAttributeSettings
	if err := json.Unmarshal([]byte(data), &legacy); err != nil {
		return DefaultAttributeSettings(), err
	}

	// Convert old format to new format
	return convertLegacySettings(legacy), nil
}

// convertLegacySettings converts old category-level mode to per-item modes
func convertLegacySettings(legacy legacyAttributeSettings) AttributeSettings {
	settings := AttributeSettings{
		Items: make(map[string]AttributeMode),
	}

	// If mode was "acceptable", just return empty items (all default to acceptable)
	if legacy.Mode == AttributeModeAcceptable || len(legacy.Values) == 0 {
		return settings
	}

	// Apply the mode to each value in the list
	for _, value := range legacy.Values {
		settings.Items[value] = legacy.Mode
	}

	return settings
}

// ParseHDRFormats splits a combo HDR string into individual formats (Req 2.3.1)
// Example: "DV HDR10" -> ["DV", "HDR10"]
func ParseHDRFormats(input string) []string {
	if input == "" {
		return nil
	}

	upperInput := strings.ToUpper(input)
	var formats []string

	if containsDolbyVision(upperInput) {
		formats = append(formats, "DV")
	}

	hasHDR10Plus, hasHDR10 := detectHDR10Variants(upperInput)
	if hasHDR10Plus {
		formats = append(formats, "HDR10+")
	} else if hasHDR10 {
		formats = append(formats, "HDR10")
	}

	if strings.Contains(upperInput, "HLG") {
		formats = append(formats, "HLG")
	}

	if !hasHDR10 && !hasHDR10Plus && strings.Contains(upperInput, "HDR") {
		formats = append(formats, "HDR")
	}

	if len(formats) == 0 {
		return []string{"SDR"}
	}

	return formats
}

func containsDolbyVision(upper string) bool {
	return strings.Contains(upper, "DOLBY VISION") ||
		strings.Contains(upper, "DOLBYVISION") ||
		strings.Contains(upper, "DOVI") ||
		strings.Contains(upper, "DV")
}

func detectHDR10Variants(upper string) (hasPlus, hasBase bool) {
	hasPlus = strings.Contains(upper, "HDR10+") || strings.Contains(upper, "HDR10PLUS")
	if !hasPlus {
		hasBase = strings.Contains(upper, "HDR10")
	}
	return
}

// NormalizeVideoCodec normalizes a parsed video codec to a standard identifier
func NormalizeVideoCodec(codec string) string {
	codec = strings.ToUpper(strings.TrimSpace(codec))

	switch codec {
	case "H264", "H.264", "AVC", "X264":
		return "x264"
	case "H265", "H.265", "HEVC", "X265":
		return "x265"
	case "AV1":
		return "AV1"
	case "VP9":
		return "VP9"
	case "XVID":
		return "XviD"
	case "DIVX":
		return "DivX"
	case "MPEG2", "MPEG-2":
		return "MPEG2"
	default:
		return codec
	}
}

// audioCodecMap maps uppercase codec names to their normalized form.
var audioCodecMap = map[string]string{
	"TRUEHD": "TrueHD", "TRUE-HD": "TrueHD",
	"DTS-HD MA": "DTS-HD MA", "DTSHD MA": "DTS-HD MA", "DTS-HDMA": "DTS-HD MA", "DTSHDMA": "DTS-HD MA",
	"DTS-HD": "DTS-HD", "DTSHD": "DTS-HD",
	"DTS": "DTS",
	"DDP": "DDP", "DD+": "DDP", "DOLBY DIGITAL PLUS": "DDP", "EAC3": "DDP", "E-AC3": "DDP", "E-AC-3": "DDP",
	"DD": "DD", "AC3": "DD", "AC-3": "DD", "DOLBY DIGITAL": "DD",
	"AAC": "AAC", "FLAC": "FLAC",
	"LPCM": "LPCM", "PCM": "LPCM",
	"OPUS": "Opus", "MP3": "MP3",
}

// NormalizeAudioCodec normalizes a parsed audio codec to a standard identifier
func NormalizeAudioCodec(codec string) string {
	upper := strings.ToUpper(strings.TrimSpace(codec))
	if normalized, ok := audioCodecMap[upper]; ok {
		return normalized
	}
	return upper
}

// NormalizeAudioChannels normalizes channel configuration to standard format
func NormalizeAudioChannels(channels string) string {
	channels = strings.TrimSpace(channels)

	switch channels {
	case "7.1", "8":
		return "7.1"
	case "5.1", "6":
		return "5.1"
	case "2.0", "2", "stereo":
		return "2.0"
	case "1.0", "1", "mono":
		return "1.0"
	default:
		return channels
	}
}
