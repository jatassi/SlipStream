package quality

import (
	"slices"
	"strings"
)

const (
	sourceBluray = "bluray"
	sourceRemux  = "remux"
)

type AttributeMatchResult struct {
	Matches bool
	Score   float64
	Reason  string
}

func MatchAttribute(releaseValue string, settings AttributeSettings) AttributeMatchResult {
	if !settings.HasNonDefaultSettings() {
		return AttributeMatchResult{Matches: true, Score: 0}
	}

	if releaseValue == "" || releaseValue == "unknown" {
		return handleUnknownSingleValue(settings)
	}

	if settings.GetMode(releaseValue) == AttributeModeNotAllowed {
		return AttributeMatchResult{Matches: false, Reason: releaseValue + " is not allowed"}
	}

	requiredValues := settings.GetRequired()
	if len(requiredValues) > 0 {
		if !slices.Contains(requiredValues, releaseValue) {
			return AttributeMatchResult{Matches: false, Reason: releaseValue + " not in required values"}
		}
	}

	if settings.GetMode(releaseValue) == AttributeModePreferred {
		return AttributeMatchResult{Matches: true, Score: 1.0}
	}

	return AttributeMatchResult{Matches: true, Score: 0}
}

func handleUnknownSingleValue(settings AttributeSettings) AttributeMatchResult {
	if len(settings.GetRequired()) > 0 {
		return AttributeMatchResult{Matches: false, Reason: "unknown value, but profile requires specific values"}
	}
	return AttributeMatchResult{Matches: true}
}

func MatchHDRAttribute(releaseFormats []string, settings AttributeSettings) AttributeMatchResult {
	if !settings.HasNonDefaultSettings() {
		return AttributeMatchResult{Matches: true, Score: 0}
	}

	if len(releaseFormats) == 0 {
		return handleEmptyMultiValue(settings, "no HDR format detected, but profile requires HDR")
	}

	if result := checkNotAllowedValues(releaseFormats, settings); !result.Matches {
		return result
	}

	if result := checkRequiredValues(releaseFormats, settings, "none of detected HDR formats match required values"); !result.Matches {
		return result
	}

	score := calculateMultiValueScore(releaseFormats, settings.GetPreferred())
	return AttributeMatchResult{Matches: true, Score: score}
}

func MatchAudioAttribute(releaseTracks []string, settings AttributeSettings) AttributeMatchResult {
	if !settings.HasNonDefaultSettings() {
		return AttributeMatchResult{Matches: true, Score: 0}
	}

	if len(releaseTracks) == 0 {
		return handleEmptyMultiValue(settings, "no audio detected, but profile requires specific audio")
	}

	if result := checkNotAllowedValues(releaseTracks, settings); !result.Matches {
		return result
	}

	if result := checkRequiredValues(releaseTracks, settings, "none of detected audio matches required values"); !result.Matches {
		return result
	}

	score := calculateMultiValueScore(releaseTracks, settings.GetPreferred())
	return AttributeMatchResult{Matches: true, Score: score}
}

func handleEmptyMultiValue(settings AttributeSettings, requiredReason string) AttributeMatchResult {
	if len(settings.GetRequired()) > 0 {
		return AttributeMatchResult{Matches: false, Reason: requiredReason}
	}
	return AttributeMatchResult{Matches: true}
}

func checkNotAllowedValues(values []string, settings AttributeSettings) AttributeMatchResult {
	notAllowed := settings.GetNotAllowed()
	for _, value := range values {
		if slices.Contains(notAllowed, value) {
			return AttributeMatchResult{Matches: false, Reason: value + " is not allowed"}
		}
	}
	return AttributeMatchResult{Matches: true}
}

func checkRequiredValues(values []string, settings AttributeSettings, reason string) AttributeMatchResult {
	requiredValues := settings.GetRequired()
	if len(requiredValues) == 0 {
		return AttributeMatchResult{Matches: true}
	}

	for _, value := range values {
		if slices.Contains(requiredValues, value) {
			return AttributeMatchResult{Matches: true}
		}
	}

	return AttributeMatchResult{Matches: false, Reason: reason}
}

func calculateMultiValueScore(values, preferredValues []string) float64 {
	var score float64
	for _, value := range values {
		if slices.Contains(preferredValues, value) {
			score += 1.0
		}
	}
	return score
}

type ReleaseAttributes struct {
	HDRFormats    []string
	VideoCodec    string
	AudioCodecs   []string
	AudioChannels []string
}

type ProfileAttributeMatchResult struct {
	HDRMatch          AttributeMatchResult
	VideoCodecMatch   AttributeMatchResult
	AudioCodecMatch   AttributeMatchResult
	AudioChannelMatch AttributeMatchResult

	AllMatch   bool
	TotalScore float64
}

func (r *ProfileAttributeMatchResult) RejectionReasons() []string {
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

func MatchProfileAttributes(release *ReleaseAttributes, profile *Profile) ProfileAttributeMatchResult {
	result := ProfileAttributeMatchResult{}

	result.HDRMatch = MatchHDRAttribute(release.HDRFormats, profile.HDRSettings)
	result.VideoCodecMatch = MatchAttribute(release.VideoCodec, profile.VideoCodecSettings)
	result.AudioCodecMatch = MatchAudioAttribute(release.AudioCodecs, profile.AudioCodecSettings)
	result.AudioChannelMatch = MatchAudioAttribute(release.AudioChannels, profile.AudioChannelSettings)

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

func (p *Profile) IsAttributeMatch(release *ReleaseAttributes) bool {
	result := MatchProfileAttributes(release, p)
	return result.AllMatch
}

func (p *Profile) GetAttributeScore(release *ReleaseAttributes) float64 {
	result := MatchProfileAttributes(release, p)
	return result.TotalScore
}

type QualityMatchResult struct {
	Matches          bool
	MatchedQualityID int
	MatchedQuality   string
	Score            float64
	Reason           string
}

func MatchQuality(resolution, source string, profile *Profile) QualityMatchResult {
	if resolution == "" {
		return QualityMatchResult{
			Matches: false,
			Reason:  "Resolution not detected from release",
		}
	}

	resolutionInt := parseResolution(resolution)
	if resolutionInt == 0 {
		return QualityMatchResult{
			Matches: false,
			Reason:  "Unknown resolution: " + resolution,
		}
	}

	normalizedSource := normalizeSource(source)

	for _, item := range profile.Items {
		if !item.Allowed {
			continue
		}

		if item.Quality.Resolution != resolutionInt {
			continue
		}

		if sourceMatches(normalizedSource, item.Quality.Source) {
			return QualityMatchResult{
				Matches:          true,
				MatchedQualityID: item.Quality.ID,
				MatchedQuality:   item.Quality.Name,
				Score:            float64(item.Quality.Weight),
			}
		}
	}

	for _, item := range profile.Items {
		if !item.Allowed {
			continue
		}
		if item.Quality.Resolution == resolutionInt {
			return QualityMatchResult{
				Matches:          true,
				MatchedQualityID: item.Quality.ID,
				MatchedQuality:   item.Quality.Name,
				Score:            float64(item.Quality.Weight) * 0.8,
			}
		}
	}

	return QualityMatchResult{
		Matches: false,
		Reason:  "No allowed quality in profile matches " + resolution,
	}
}

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

func normalizeSource(source string) string {
	switch source {
	case "BluRay", "Blu-Ray", "BDRip", "BRRip":
		return sourceBluray
	case "WEB-DL", "WEBDL", "WEB":
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
		return sourceRemux
	default:
		return strings.ToLower(source)
	}
}

func sourceMatches(releaseSource, qualitySource string) bool {
	if releaseSource == qualitySource {
		return true
	}
	if releaseSource == sourceRemux && qualitySource == sourceRemux {
		return true
	}
	return false
}
