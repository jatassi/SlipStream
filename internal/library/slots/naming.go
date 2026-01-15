package slots

import (
	"strings"

	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/quality"
)

// File Naming Validation
// Implements Requirements 4.1.1-4.1.5 from the Multiple Quality Versions spec

// DifferentiatorAttribute represents an attribute that differs between slot profiles
type DifferentiatorAttribute string

const (
	DifferentiatorHDR           DifferentiatorAttribute = "HDR"
	DifferentiatorVideoCodec    DifferentiatorAttribute = "Video Codec"
	DifferentiatorAudioCodec    DifferentiatorAttribute = "Audio Codec"
	DifferentiatorAudioChannels DifferentiatorAttribute = "Audio Channels"
)

// NamingToken represents a filename token that can differentiate files
type NamingToken struct {
	Name        string   `json:"name"`
	Patterns    []string `json:"patterns"`    // Token patterns that satisfy this requirement
	Description string   `json:"description"` // Human-readable description
}

// AttributeTokenMapping maps differentiator attributes to their required filename tokens
var AttributeTokenMapping = map[DifferentiatorAttribute]NamingToken{
	DifferentiatorHDR: {
		Name: "HDR Format",
		Patterns: []string{
			"{mediainfo videodynamicrange}",
			"{mediainfo videodynamicrangetype}",
		},
		Description: "Include {MediaInfo VideoDynamicRange} or {MediaInfo VideoDynamicRangeType} to differentiate HDR/SDR files",
	},
	DifferentiatorVideoCodec: {
		Name: "Video Codec",
		Patterns: []string{
			"{mediainfo videocodec}",
			"{mediainfo simple}",
			"{mediainfo full}",
		},
		Description: "Include {MediaInfo VideoCodec} to differentiate by video codec (x264, x265, etc.)",
	},
	DifferentiatorAudioCodec: {
		Name: "Audio Codec",
		Patterns: []string{
			"{mediainfo audiocodec}",
			"{mediainfo simple}",
			"{mediainfo full}",
		},
		Description: "Include {MediaInfo AudioCodec} to differentiate by audio codec (TrueHD, DTS-HD MA, etc.)",
	},
	DifferentiatorAudioChannels: {
		Name: "Audio Channels",
		Patterns: []string{
			"{mediainfo audiochannels}",
		},
		Description: "Include {MediaInfo AudioChannels} to differentiate by channel layout (7.1, 5.1, 2.0)",
	},
}

// NamingValidationResult contains the result of validating a filename format
// Req 4.1.5: If filename format is missing required differentiators, prompt user to update format
type NamingValidationResult struct {
	Valid              bool                      `json:"valid"`
	MissingTokens      []MissingTokenInfo        `json:"missingTokens,omitempty"`
	RequiredAttributes []DifferentiatorAttribute `json:"requiredAttributes,omitempty"`
	Warnings           []string                  `json:"warnings,omitempty"`
}

// MissingTokenInfo provides details about a missing token
type MissingTokenInfo struct {
	Attribute      DifferentiatorAttribute `json:"attribute"`
	TokenName      string                  `json:"tokenName"`
	Description    string                  `json:"description"`
	SuggestedToken string                  `json:"suggestedToken"`
}

// GetConflictingAttributes identifies attributes that differ between slot profiles.
// Req 4.1.2: Only conflicting differentiators are required (attributes where profiles have opposing requirements)
func GetConflictingAttributes(slots []quality.SlotConfig) []DifferentiatorAttribute {
	var conflicts []DifferentiatorAttribute

	for i := 0; i < len(slots); i++ {
		slotA := slots[i]
		if !slotA.Enabled || slotA.Profile == nil {
			continue
		}

		for j := i + 1; j < len(slots); j++ {
			slotB := slots[j]
			if !slotB.Enabled || slotB.Profile == nil {
				continue
			}

			// Check each attribute for conflicts between the two profiles
			if hasRequiredDifference(slotA.Profile.HDRSettings, slotB.Profile.HDRSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorHDR)
			}
			if hasRequiredDifference(slotA.Profile.VideoCodecSettings, slotB.Profile.VideoCodecSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorVideoCodec)
			}
			if hasRequiredDifference(slotA.Profile.AudioCodecSettings, slotB.Profile.AudioCodecSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorAudioCodec)
			}
			if hasRequiredDifference(slotA.Profile.AudioChannelSettings, slotB.Profile.AudioChannelSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorAudioChannels)
			}
		}
	}

	return conflicts
}

// hasRequiredDifference checks if two attribute settings have different required values.
// Req 4.1.1: Filename format must include tokens for attributes that differ between assigned slot profiles
func hasRequiredDifference(settingsA, settingsB quality.AttributeSettings) bool {
	// Get required values from each settings using the per-item mode system
	requiredA := settingsA.GetRequired()
	requiredB := settingsB.GetRequired()

	// Both must have required values to create a meaningful difference
	if len(requiredA) == 0 || len(requiredB) == 0 {
		return false
	}
	// Values must not overlap (i.e., they require different things)
	return !hasOverlap(requiredA, requiredB)
}

// hasOverlap checks if two string slices share any common elements
func hasOverlap(a, b []string) bool {
	setA := make(map[string]bool, len(a))
	for _, v := range a {
		setA[v] = true
	}
	for _, v := range b {
		if setA[v] {
			return true
		}
	}
	return false
}

// appendIfMissing appends an attribute to the slice if not already present
func appendIfMissing(slice []DifferentiatorAttribute, attr DifferentiatorAttribute) []DifferentiatorAttribute {
	for _, existing := range slice {
		if existing == attr {
			return slice
		}
	}
	return append(slice, attr)
}

// ValidateFilenameFormat checks if a filename format includes required differentiator tokens.
// Req 4.1.4: Validation occurs when saving slot configuration
func ValidateFilenameFormat(format string, requiredAttrs []DifferentiatorAttribute) NamingValidationResult {
	result := NamingValidationResult{
		Valid:              true,
		RequiredAttributes: requiredAttrs,
		MissingTokens:      []MissingTokenInfo{},
		Warnings:           []string{},
	}

	if len(requiredAttrs) == 0 {
		return result
	}

	// Parse the filename format to extract tokens
	tokens := renamer.ParseTokens(format)
	tokenNames := make(map[string]bool)
	for _, token := range tokens {
		tokenNames[strings.ToLower(token.Name)] = true
	}

	// Also normalize the raw format string for pattern matching
	// This handles separator variations like {.MediaInfo.VideoCodec} or {-MediaInfo VideoCodec}
	formatLower := strings.ToLower(format)
	// Replace common separators with spaces for matching
	formatNormalized := strings.NewReplacer(".", " ", "-", " ", "_", " ").Replace(formatLower)

	// Check each required attribute
	for _, attr := range requiredAttrs {
		tokenInfo, exists := AttributeTokenMapping[attr]
		if !exists {
			continue
		}

		found := false
		for _, pattern := range tokenInfo.Patterns {
			// Check if the token name exists in parsed tokens
			patternName := strings.Trim(pattern, "{}")
			if tokenNames[patternName] {
				found = true
				break
			}
			// Also check normalized format string (handles separator variations)
			if strings.Contains(formatNormalized, patternName) {
				found = true
				break
			}
		}

		if !found {
			result.Valid = false
			result.MissingTokens = append(result.MissingTokens, MissingTokenInfo{
				Attribute:      attr,
				TokenName:      tokenInfo.Name,
				Description:    tokenInfo.Description,
				SuggestedToken: getSuggestedToken(attr),
			})
		}
	}

	return result
}

// getSuggestedToken returns the recommended token to add for an attribute
func getSuggestedToken(attr DifferentiatorAttribute) string {
	switch attr {
	case DifferentiatorHDR:
		return "{MediaInfo VideoDynamicRangeType}"
	case DifferentiatorVideoCodec:
		return "{MediaInfo VideoCodec}"
	case DifferentiatorAudioCodec:
		return "{MediaInfo AudioCodec}"
	case DifferentiatorAudioChannels:
		return "{MediaInfo AudioChannels}"
	default:
		return ""
	}
}

// ValidateSlotNaming performs full naming validation for slot configuration.
// Returns validation results for both movie and episode filename formats.
type SlotNamingValidation struct {
	MovieFormatValid   bool                    `json:"movieFormatValid"`
	EpisodeFormatValid bool                    `json:"episodeFormatValid"`
	MovieValidation    NamingValidationResult  `json:"movieValidation"`
	EpisodeValidation  NamingValidationResult  `json:"episodeValidation"`
	RequiredAttributes []DifferentiatorAttribute `json:"requiredAttributes"`
	CanProceed         bool                    `json:"canProceed"` // True if validation passes or user acknowledged
}

// ValidateSlotNaming validates naming formats against slot configuration
func ValidateSlotNaming(slots []quality.SlotConfig, movieFormat, episodeFormat string) SlotNamingValidation {
	conflictingAttrs := GetConflictingAttributes(slots)

	result := SlotNamingValidation{
		RequiredAttributes: conflictingAttrs,
		CanProceed:         true,
	}

	if len(conflictingAttrs) == 0 {
		result.MovieFormatValid = true
		result.EpisodeFormatValid = true
		result.MovieValidation = NamingValidationResult{Valid: true}
		result.EpisodeValidation = NamingValidationResult{Valid: true}
		return result
	}

	// Validate movie format
	result.MovieValidation = ValidateFilenameFormat(movieFormat, conflictingAttrs)
	result.MovieFormatValid = result.MovieValidation.Valid

	// Validate episode format
	result.EpisodeValidation = ValidateFilenameFormat(episodeFormat, conflictingAttrs)
	result.EpisodeFormatValid = result.EpisodeValidation.Valid

	// Can proceed only if both formats are valid
	result.CanProceed = result.MovieFormatValid && result.EpisodeFormatValid

	return result
}

// BuildNamingValidationWarnings creates user-friendly warning messages
func BuildNamingValidationWarnings(validation SlotNamingValidation) []string {
	var warnings []string

	if !validation.MovieFormatValid {
		for _, missing := range validation.MovieValidation.MissingTokens {
			warnings = append(warnings,
				"Movie filename format is missing "+missing.TokenName+" token. "+missing.Description)
		}
	}

	if !validation.EpisodeFormatValid {
		for _, missing := range validation.EpisodeValidation.MissingTokens {
			warnings = append(warnings,
				"Episode filename format is missing "+missing.TokenName+" token. "+missing.Description)
		}
	}

	return warnings
}
