package titleutil

import (
	"regexp"
	"strings"
)

var (
	apostropheRegex    = regexp.MustCompile(`[''\x60\x{2018}\x{2019}\x{02BC}]`) //nolint:gocritic // intentional character duplication
	specialCharsRegex  = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	multipleSpaceRegex = regexp.MustCompile(`\s+`)
	trailingYearRegex  = regexp.MustCompile(`\s+(19|20)\d{2}$`)
)

// NormalizeTitle converts a title to a normalized form for comparison.
// It converts to lowercase, strips apostrophes (within-word punctuation),
// replaces remaining special characters with spaces, and collapses multiple spaces.
// Apostrophes are stripped rather than replaced with spaces so that titles like
// "Schitt's Creek" and "Schitts Creek" both normalize to "schitts creek".
func NormalizeTitle(title string) string {
	normalized := strings.ToLower(title)
	normalized = apostropheRegex.ReplaceAllString(normalized, "")
	normalized = specialCharsRegex.ReplaceAllString(normalized, " ")
	normalized = multipleSpaceRegex.ReplaceAllString(normalized, " ")
	normalized = strings.TrimSpace(normalized)
	return normalized
}

// TitlesMatch performs strict matching of two titles after normalization.
// Returns true only if the normalized titles are exactly equal.
func TitlesMatch(parsedTitle, searchQuery string) bool {
	return NormalizeTitle(parsedTitle) == NormalizeTitle(searchQuery)
}

// TVTitlesMatch matches TV titles with year-awareness. TV releases commonly
// include the year to disambiguate (e.g., "Vanished 2026 S01E01") but the
// database title may not include it (or vice versa). This tries an exact
// match first, then retries after stripping trailing years from both titles.
func TVTitlesMatch(parsedTitle, searchQuery string) bool {
	normalized := NormalizeTitle(parsedTitle)
	query := NormalizeTitle(searchQuery)
	if normalized == query {
		return true
	}
	return stripTrailingYear(normalized) == stripTrailingYear(query)
}

// stripTrailingYear removes a trailing 4-digit year (1900-2099) from a
// normalized title. Returns the title unchanged if no trailing year is found
// or if the year is the entire title.
func stripTrailingYear(normalized string) string {
	stripped := trailingYearRegex.ReplaceAllString(normalized, "")
	if stripped == "" {
		return normalized
	}
	return stripped
}
