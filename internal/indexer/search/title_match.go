package search

import (
	"strings"

	"github.com/slipstream/slipstream/internal/module/titleutil"
)

func NormalizeTitle(title string) string { return titleutil.NormalizeTitle(title) }
func TitlesMatch(a, b string) bool       { return titleutil.TitlesMatch(a, b) }
func TVTitlesMatch(a, b string) bool     { return titleutil.TVTitlesMatch(a, b) }

// CalculateTitleSimilarity calculates the Jaccard similarity between two titles.
// Returns a value between 0.0 (no match) and 1.0 (identical).
// Used for debugging/logging, not for filtering decisions.

// buildTokenSet creates a set from a slice of tokens
func buildTokenSet(tokens []string) map[string]bool {
	set := make(map[string]bool)
	for _, t := range tokens {
		set[t] = true
	}
	return set
}

// calculateIntersection counts common elements between two sets
func calculateIntersection(set1, set2 map[string]bool) int {
	intersection := 0
	for t := range set1 {
		if set2[t] {
			intersection++
		}
	}
	return intersection
}

// calculateUnion counts total unique elements in both sets
func calculateUnion(set1, set2 map[string]bool) int {
	union := len(set1)
	for t := range set2 {
		if !set1[t] {
			union++
		}
	}
	return union
}

func CalculateTitleSimilarity(title1, title2 string) float64 {
	tokens1 := tokenize(NormalizeTitle(title1))
	tokens2 := tokenize(NormalizeTitle(title2))

	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	set1 := buildTokenSet(tokens1)
	set2 := buildTokenSet(tokens2)

	intersection := calculateIntersection(set1, set2)
	union := calculateUnion(set1, set2)

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
