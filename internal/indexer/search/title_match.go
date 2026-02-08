package search

import (
	"regexp"
	"strings"
)

var (
	apostropheRegex    = regexp.MustCompile(`[''\x60\x{2018}\x{2019}\x{02BC}]`)
	specialCharsRegex  = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	multipleSpaceRegex = regexp.MustCompile(`\s+`)
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

// CalculateTitleSimilarity calculates the Jaccard similarity between two titles.
// Returns a value between 0.0 (no match) and 1.0 (identical).
// Used for debugging/logging, not for filtering decisions.
func CalculateTitleSimilarity(title1, title2 string) float64 {
	tokens1 := tokenize(NormalizeTitle(title1))
	tokens2 := tokenize(NormalizeTitle(title2))

	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, t := range tokens1 {
		set1[t] = true
	}

	set2 := make(map[string]bool)
	for _, t := range tokens2 {
		set2[t] = true
	}

	intersection := 0
	for t := range set1 {
		if set2[t] {
			intersection++
		}
	}

	union := len(set1)
	for t := range set2 {
		if !set1[t] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}
