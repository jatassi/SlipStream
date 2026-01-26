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
	conflicts := make([]DifferentiatorAttribute, 0)

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
			// Use the same logic as mutual exclusivity check: required vs notAllowed conflicts
			if quality.HasAttributeConflict(slotA.Profile.HDRSettings, slotB.Profile.HDRSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorHDR)
			}
			if quality.HasAttributeConflict(slotA.Profile.VideoCodecSettings, slotB.Profile.VideoCodecSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorVideoCodec)
			}
			if quality.HasAttributeConflict(slotA.Profile.AudioCodecSettings, slotB.Profile.AudioCodecSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorAudioCodec)
			}
			if quality.HasAttributeConflict(slotA.Profile.AudioChannelSettings, slotB.Profile.AudioChannelSettings) {
				conflicts = appendIfMissing(conflicts, DifferentiatorAudioChannels)
			}
		}
	}

	return conflicts
}

// checkQualityTierExclusivity checks if slot profiles have non-overlapping quality tiers.
// This means profiles are distinguished by quality (e.g., 1080p vs 4K) and don't need
// special filename tokens beyond {Quality Title}.
func checkQualityTierExclusivity(slots []quality.SlotConfig) bool {
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

			// Use the quality package's exclusivity check for quality tiers
			exclusivity := quality.CheckMutualExclusivity(slotA.Profile, slotB.Profile)
			if exclusivity.AreExclusive && exclusivity.Reason == "profiles have different allowed quality tiers" {
				return true
			}
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

// SlotNamingValidation performs full naming validation for slot configuration.
// Returns validation results for both movie and episode filename formats.
type SlotNamingValidation struct {
	MovieFormatValid        bool                      `json:"movieFormatValid"`
	EpisodeFormatValid      bool                      `json:"episodeFormatValid"`
	MovieValidation         NamingValidationResult    `json:"movieValidation"`
	EpisodeValidation       NamingValidationResult    `json:"episodeValidation"`
	RequiredAttributes      []DifferentiatorAttribute `json:"requiredAttributes"`
	CanProceed              bool                      `json:"canProceed"`              // True if validation passes or user acknowledged
	QualityTierExclusive    bool                      `json:"qualityTierExclusive"`    // Profiles are exclusive via quality tiers only
	NoEnabledSlots          bool                      `json:"noEnabledSlots"`          // No enabled slots with profiles found
}

// ValidateSlotNaming validates naming formats against slot configuration
func ValidateSlotNaming(slots []quality.SlotConfig, movieFormat, episodeFormat string) SlotNamingValidation {
	result := SlotNamingValidation{
		RequiredAttributes: make([]DifferentiatorAttribute, 0),
		CanProceed:         true,
	}

	// Count enabled slots with profiles
	enabledCount := 0
	for _, slot := range slots {
		if slot.Enabled && slot.Profile != nil {
			enabledCount++
		}
	}

	if enabledCount < 2 {
		// Need at least 2 slots to check for conflicts
		result.NoEnabledSlots = true
		result.MovieFormatValid = false
		result.EpisodeFormatValid = false
		result.CanProceed = false
		result.MovieValidation = NamingValidationResult{Valid: false, MissingTokens: []MissingTokenInfo{}, RequiredAttributes: []DifferentiatorAttribute{}}
		result.EpisodeValidation = NamingValidationResult{Valid: false, MissingTokens: []MissingTokenInfo{}, RequiredAttributes: []DifferentiatorAttribute{}}
		return result
	}

	conflictingAttrs := GetConflictingAttributes(slots)
	result.RequiredAttributes = conflictingAttrs

	if len(conflictingAttrs) == 0 {
		// No attribute conflicts - check if profiles are exclusive via quality tiers
		result.QualityTierExclusive = checkQualityTierExclusivity(slots)
		result.MovieFormatValid = true
		result.EpisodeFormatValid = true
		result.MovieValidation = NamingValidationResult{Valid: true, MissingTokens: []MissingTokenInfo{}, RequiredAttributes: []DifferentiatorAttribute{}}
		result.EpisodeValidation = NamingValidationResult{Valid: true, MissingTokens: []MissingTokenInfo{}, RequiredAttributes: []DifferentiatorAttribute{}}
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
