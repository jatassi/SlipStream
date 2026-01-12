package importer

import (
	"regexp"
	"strings"

	"github.com/slipstream/slipstream/internal/library/tv"
)

// specialsPatterns are regex patterns used to detect special episodes from filenames.
var specialsPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)S0+E\d+`),                  // S00E01, S0E01
	regexp.MustCompile(`(?i)Season\s*0+\s*Episode`),    // Season 0 Episode
	regexp.MustCompile(`(?i)[.\s_-]SP\d+[.\s_-]`),      // .SP01. _SP01_ -SP01-
	regexp.MustCompile(`(?i)[.\s_-]Special[.\s_-]`),    // .Special.
	regexp.MustCompile(`(?i)^Special[.\s_-]`),          // Special.
	regexp.MustCompile(`(?i)[.\s_-]Specials?[.\s_-]`),  // .Specials.
	regexp.MustCompile(`(?i)[.\s_-]OVA[.\s_-]`),        // .OVA. (anime)
	regexp.MustCompile(`(?i)[.\s_-]OAV[.\s_-]`),        // .OAV. (anime)
	regexp.MustCompile(`(?i)[.\s_-]ONA[.\s_-]`),        // .ONA. (anime)
	regexp.MustCompile(`(?i)[.\s_-]Omake[.\s_-]`),      // .Omake. (anime extras)
	regexp.MustCompile(`(?i)[.\s_-]Picture\s*Drama`),   // Picture Drama (anime)
	regexp.MustCompile(`(?i)[.\s_-]Movie[.\s_-]`),      // .Movie. (anime movie specials)
	regexp.MustCompile(`(?i)[.\s_-]Pilot[.\s_-]`),      // .Pilot.
	regexp.MustCompile(`(?i)[.\s_-]Bonus[.\s_-]`),      // .Bonus.
	regexp.MustCompile(`(?i)[.\s_-]Extra[s]?[.\s_-]`),  // .Extra. .Extras.
}

// IsSpecialEpisode checks if an episode is a special by examining metadata and filename.
func IsSpecialEpisode(episode *tv.Episode, filename string) bool {
	// Priority 1: Check metadata - Season 0 means it's a special
	if DetectSpecialFromMetadata(episode) {
		return true
	}

	// Priority 2: Check filename patterns
	if DetectSpecialFromFilename(filename) {
		return true
	}

	return false
}

// DetectSpecialFromMetadata checks if an episode is a special based on its metadata.
// Season 0 is the standard location for specials in TVDB/TMDB.
func DetectSpecialFromMetadata(episode *tv.Episode) bool {
	if episode == nil {
		return false
	}
	return episode.SeasonNumber == 0
}

// DetectSpecialFromFilename checks if a filename indicates a special episode.
func DetectSpecialFromFilename(filename string) bool {
	if filename == "" {
		return false
	}

	// Check all patterns
	for _, pattern := range specialsPatterns {
		if pattern.MatchString(filename) {
			return true
		}
	}

	// Also check if the file is in a folder named "Specials" or "Season 0"
	lowerFilename := strings.ToLower(filename)
	if strings.Contains(lowerFilename, "/specials/") || strings.Contains(lowerFilename, "\\specials\\") {
		return true
	}
	if strings.Contains(lowerFilename, "/season 0/") || strings.Contains(lowerFilename, "\\season 0\\") {
		return true
	}
	if strings.Contains(lowerFilename, "/season0/") || strings.Contains(lowerFilename, "\\season0\\") {
		return true
	}

	return false
}

// GetSpecialType attempts to identify the type of special from the filename.
func GetSpecialType(filename string) string {
	lowerFilename := strings.ToLower(filename)

	switch {
	case strings.Contains(lowerFilename, "ova"):
		return "OVA"
	case strings.Contains(lowerFilename, "oav"):
		return "OAV"
	case strings.Contains(lowerFilename, "ona"):
		return "ONA"
	case strings.Contains(lowerFilename, "omake"):
		return "Omake"
	case strings.Contains(lowerFilename, "picture drama"):
		return "Picture Drama"
	case strings.Contains(lowerFilename, "pilot"):
		return "Pilot"
	case strings.Contains(lowerFilename, "bonus"):
		return "Bonus"
	case strings.Contains(lowerFilename, "extra"):
		return "Extra"
	case strings.Contains(lowerFilename, "movie"):
		return "Movie"
	default:
		return "Special"
	}
}

// ParseSpecialEpisodeNumber attempts to extract the special episode number from a filename.
// Returns -1 if no special number could be parsed.
func ParseSpecialEpisodeNumber(filename string) int {
	// Try S00E## format
	s00ePattern := regexp.MustCompile(`(?i)S0+E(\d+)`)
	if match := s00ePattern.FindStringSubmatch(filename); len(match) > 1 {
		var num int
		_, _ = parseIntFromString(match[1], &num)
		return num
	}

	// Try SP## format
	spPattern := regexp.MustCompile(`(?i)SP(\d+)`)
	if match := spPattern.FindStringSubmatch(filename); len(match) > 1 {
		var num int
		_, _ = parseIntFromString(match[1], &num)
		return num
	}

	return -1
}

// parseIntFromString is a helper to parse an int from a string.
func parseIntFromString(s string, result *int) (bool, error) {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	*result = n
	return n > 0, nil
}
