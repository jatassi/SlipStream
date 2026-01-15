package quality

import (
	"slices"
	"strings"
)

// AttributeMatchResult contains the result of matching an attribute
type AttributeMatchResult struct {
	Matches bool    // Whether the attribute matches (passes filter)
	Score   float64 // Scoring bonus (0.0 if no bonus, positive if preferred match)
	Reason  string  // Explanation if not matching
}

// MatchAttribute checks if a single release attribute value matches per-item profile settings
// Logic:
// - If release value is "notAllowed" → reject
// - If there are "required" values and release doesn't match any → reject
// - If release matches a "preferred" value → pass with bonus
// - Otherwise → pass with no bonus
func MatchAttribute(releaseValue string, settings AttributeSettings) AttributeMatchResult {
	// If no settings configured, pass with no bonus
	if !settings.HasNonDefaultSettings() {
		return AttributeMatchResult{Matches: true, Score: 0}
	}

	// Handle unknown/empty values (Req 2.5.1, 2.5.2)
	if releaseValue == "" || releaseValue == "unknown" {
		requiredValues := settings.GetRequired()
		if len(requiredValues) > 0 {
			// Req 2.5.1: Unknown attributes fail if there are required values
			return AttributeMatchResult{Matches: false, Reason: "unknown value, but profile requires specific values"}
		}
		// Req 2.5.2: Unknown attributes pass if no required values
		return AttributeMatchResult{Matches: true}
	}

	// Check if release value is explicitly not allowed
	if settings.GetMode(releaseValue) == AttributeModeNotAllowed {
		return AttributeMatchResult{Matches: false, Reason: releaseValue + " is not allowed"}
	}

	// Check required values - if any exist, release must match one
	requiredValues := settings.GetRequired()
	if len(requiredValues) > 0 {
		if !slices.Contains(requiredValues, releaseValue) {
			return AttributeMatchResult{Matches: false, Reason: releaseValue + " not in required values"}
		}
	}

	// Check for preferred bonus
	if settings.GetMode(releaseValue) == AttributeModePreferred {
		return AttributeMatchResult{Matches: true, Score: 1.0}
	}

	return AttributeMatchResult{Matches: true, Score: 0}
}

// MatchHDRAttribute checks if release HDR formats match per-item profile settings (Req 2.3.2, 2.3.3)
// A release with multiple HDR layers (e.g., "DV HDR10") matches if:
// - NONE of its formats are "notAllowed"
// - If there are "required" values, at least ONE format matches
// - Preferred matches accumulate scoring bonuses
func MatchHDRAttribute(releaseFormats []string, settings AttributeSettings) AttributeMatchResult {
	// If no settings configured, pass with no bonus
	if !settings.HasNonDefaultSettings() {
		return AttributeMatchResult{Matches: true, Score: 0}
	}

	// Handle empty/unknown
	if len(releaseFormats) == 0 {
		requiredValues := settings.GetRequired()
		if len(requiredValues) > 0 {
			return AttributeMatchResult{Matches: false, Reason: "no HDR format detected, but profile requires HDR"}
		}
		return AttributeMatchResult{Matches: true}
	}

	// Check if ANY release format is "notAllowed" → reject entire release
	notAllowed := settings.GetNotAllowed()
	for _, format := range releaseFormats {
		if slices.Contains(notAllowed, format) {
			return AttributeMatchResult{Matches: false, Reason: format + " is not allowed"}
		}
	}

	// Check required values - if any exist, at least one release format must match (Req 2.3.2)
	requiredValues := settings.GetRequired()
	if len(requiredValues) > 0 {
		hasRequiredMatch := false
		for _, format := range releaseFormats {
			if slices.Contains(requiredValues, format) {
				hasRequiredMatch = true
				break
			}
		}
		if !hasRequiredMatch {
			return AttributeMatchResult{Matches: false, Reason: "none of detected HDR formats match required values"}
		}
	}

	// Calculate preferred bonus - sum up matches
	var score float64
	preferredValues := settings.GetPreferred()
	for _, format := range releaseFormats {
		if slices.Contains(preferredValues, format) {
			score += 1.0
		}
	}

	return AttributeMatchResult{Matches: true, Score: score}
}

// MatchAudioAttribute checks if release audio tracks match per-item profile settings (Req 2.4.1)
// Releases with multiple audio tracks match if:
// - NONE of its tracks are "notAllowed"
// - If there are "required" values, at least ONE track matches
// - Preferred matches accumulate scoring bonuses
func MatchAudioAttribute(releaseTracks []string, settings AttributeSettings) AttributeMatchResult {
	// If no settings configured, pass with no bonus
	if !settings.HasNonDefaultSettings() {
		return AttributeMatchResult{Matches: true, Score: 0}
	}

	// Handle empty/unknown
	if len(releaseTracks) == 0 {
		requiredValues := settings.GetRequired()
		if len(requiredValues) > 0 {
			return AttributeMatchResult{Matches: false, Reason: "no audio detected, but profile requires specific audio"}
		}
		return AttributeMatchResult{Matches: true}
	}

	// Check if ANY track is "notAllowed" → reject entire release
	notAllowed := settings.GetNotAllowed()
	for _, track := range releaseTracks {
		if slices.Contains(notAllowed, track) {
			return AttributeMatchResult{Matches: false, Reason: track + " is not allowed"}
		}
	}

	// Check required values - if any exist, at least one track must match (Req 2.4.1)
	requiredValues := settings.GetRequired()
	if len(requiredValues) > 0 {
		hasRequiredMatch := false
		for _, track := range releaseTracks {
			if slices.Contains(requiredValues, track) {
				hasRequiredMatch = true
				break
			}
		}
		if !hasRequiredMatch {
			return AttributeMatchResult{Matches: false, Reason: "none of detected audio matches required values"}
		}
	}

	// Calculate preferred bonus - sum up matches
	var score float64
	preferredValues := settings.GetPreferred()
	for _, track := range releaseTracks {
		if slices.Contains(preferredValues, track) {
			score += 1.0
		}
	}

	return AttributeMatchResult{Matches: true, Score: score}
}

// ReleaseAttributes contains parsed attributes from a release for matching
type ReleaseAttributes struct {
	HDRFormats    []string // Parsed HDR formats (e.g., ["DV", "HDR10"])
	VideoCodec    string   // Normalized video codec (e.g., "x265")
	AudioCodecs   []string // Normalized audio codecs (e.g., ["TrueHD", "AAC"])
	AudioChannels []string // Normalized channel configs (e.g., ["7.1", "2.0"])
}

// ProfileAttributeMatchResult contains results for all profile attribute checks
type ProfileAttributeMatchResult struct {
	HDRMatch          AttributeMatchResult
	VideoCodecMatch   AttributeMatchResult
	AudioCodecMatch   AttributeMatchResult
	AudioChannelMatch AttributeMatchResult

	AllMatch   bool    // True if all required attributes match
	TotalScore float64 // Sum of all scoring bonuses
}

// RejectionReasons returns a list of human-readable reasons why attributes didn't match
func (r ProfileAttributeMatchResult) RejectionReasons() []string {
	var reasons []string
	if !r.HDRMatch.Matches && r.HDRMatch.Reason != "" {
		reasons = append(reasons, "HDR: "+r.HDRMatch.Reason)
	}
	if !r.VideoCodecMatch.Matches && r.VideoCodecMatch.Reason != "" {
		reasons = append(reasons, "Video: "+r.VideoCodecMatch.Reason)
	}
	if !r.AudioCodecMatch.Matches && r.AudioCodecMatch.Reason != "" {
		reasons = append(reasons, "Audio: "+r.AudioCodecMatch.Reason)
	}
	if !r.AudioChannelMatch.Matches && r.AudioChannelMatch.Reason != "" {
		reasons = append(reasons, "Channels: "+r.AudioChannelMatch.Reason)
	}
	return reasons
}

// MatchProfileAttributes checks a release against all profile attribute settings
func MatchProfileAttributes(release ReleaseAttributes, profile *Profile) ProfileAttributeMatchResult {
	result := ProfileAttributeMatchResult{}

	// Match HDR
	result.HDRMatch = MatchHDRAttribute(release.HDRFormats, profile.HDRSettings)

	// Match video codec
	result.VideoCodecMatch = MatchAttribute(release.VideoCodec, profile.VideoCodecSettings)

	// Match audio codec (multi-track)
	result.AudioCodecMatch = MatchAudioAttribute(release.AudioCodecs, profile.AudioCodecSettings)

	// Match audio channels (multi-track)
	result.AudioChannelMatch = MatchAudioAttribute(release.AudioChannels, profile.AudioChannelSettings)

	// Calculate overall results
	result.AllMatch = result.HDRMatch.Matches &&
		result.VideoCodecMatch.Matches &&
		result.AudioCodecMatch.Matches &&
		result.AudioChannelMatch.Matches

	result.TotalScore = result.HDRMatch.Score +
		result.VideoCodecMatch.Score +
		result.AudioCodecMatch.Score +
		result.AudioChannelMatch.Score

	return result
}

// IsAttributeMatch is a convenience method to check if a release matches profile attributes
func (p *Profile) IsAttributeMatch(release ReleaseAttributes) bool {
	result := MatchProfileAttributes(release, p)
	return result.AllMatch
}

// GetAttributeScore returns the total attribute scoring bonus for a release
func (p *Profile) GetAttributeScore(release ReleaseAttributes) float64 {
	result := MatchProfileAttributes(release, p)
	return result.TotalScore
}

// QualityMatchResult contains the result of matching a release's quality against a profile
type QualityMatchResult struct {
	Matches          bool    // Whether a matching quality was found
	MatchedQualityID int     // ID of the matched quality (0 if no match)
	MatchedQuality   string  // Name of the matched quality
	Score            float64 // Weight/score of the matched quality
	Reason           string  // Explanation if not matching
}

// MatchQuality checks if a release's quality (resolution + source) matches any allowed quality in the profile
// resolution: "2160p", "1080p", "720p", "480p"
// source: "BluRay", "WEB-DL", "WEBRip", "HDTV", "DVDRip", "SDTV", "Remux"
func MatchQuality(resolution, source string, profile *Profile) QualityMatchResult {
	if resolution == "" {
		return QualityMatchResult{
			Matches: false,
			Reason:  "Resolution not detected from release",
		}
	}

	// Parse resolution to int
	resolutionInt := parseResolution(resolution)
	if resolutionInt == 0 {
		return QualityMatchResult{
			Matches: false,
			Reason:  "Unknown resolution: " + resolution,
		}
	}

	// Normalize source for matching
	normalizedSource := normalizeSource(source)

	// Check each allowed quality in the profile
	for _, item := range profile.Items {
		if !item.Allowed {
			continue
		}

		// Match resolution
		if item.Quality.Resolution != resolutionInt {
			continue
		}

		// Match source (with some flexibility)
		if sourceMatches(normalizedSource, item.Quality.Source) {
			return QualityMatchResult{
				Matches:          true,
				MatchedQualityID: item.Quality.ID,
				MatchedQuality:   item.Quality.Name,
				Score:            float64(item.Quality.Weight),
			}
		}
	}

	// If we have a resolution match but no source match, check if ANY quality at that resolution is allowed
	for _, item := range profile.Items {
		if !item.Allowed {
			continue
		}
		if item.Quality.Resolution == resolutionInt {
			// Found a matching resolution with different source - partial match
			return QualityMatchResult{
				Matches:          true,
				MatchedQualityID: item.Quality.ID,
				MatchedQuality:   item.Quality.Name,
				Score:            float64(item.Quality.Weight) * 0.8, // Reduce score for source mismatch
			}
		}
	}

	return QualityMatchResult{
		Matches: false,
		Reason:  "No allowed quality in profile matches " + resolution,
	}
}

// parseResolution converts resolution string to int
func parseResolution(resolution string) int {
	switch resolution {
	case "2160p", "4K", "UHD":
		return 2160
	case "1080p":
		return 1080
	case "720p":
		return 720
	case "480p", "SD":
		return 480
	default:
		return 0
	}
}

// normalizeSource normalizes source names for matching
func normalizeSource(source string) string {
	switch source {
	case "BluRay", "Blu-Ray", "BDRip", "BRRip":
		return "bluray"
	case "WEB-DL", "WEBDL":
		return "webdl"
	case "WEBRip":
		return "webrip"
	case "HDTV":
		return "tv"
	case "DVDRip", "DVD":
		return "dvd"
	case "SDTV", "PDTV":
		return "tv"
	case "Remux":
		return "remux"
	default:
		return strings.ToLower(source)
	}
}

// sourceMatches checks if release source matches quality source
func sourceMatches(releaseSource, qualitySource string) bool {
	if releaseSource == qualitySource {
		return true
	}
	// Allow some flexibility (e.g., "bluray" matches both "bluray" and "remux")
	if releaseSource == "remux" && qualitySource == "remux" {
		return true
	}
	return false
}
