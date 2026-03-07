package parseutil

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// separatorPattern matches dots, underscores, hyphens, and whitespace runs for basic title cleaning.
	separatorPattern = regexp.MustCompile(`[.\s_-]+`)

	// yearPattern matches a 4-digit year (1900-2100) optionally wrapped in parens/brackets.
	yearExtractPattern = regexp.MustCompile(`\s*[\(\[]?(\d{4})[\)\]]?`)

	// Patterns used by NormalizeTitle for cleaning punctuation and year annotations.
	yearInBracketsPattern = regexp.MustCompile(`\s*[\(\[]\d{4}[\)\]]`)
	multiSpacePattern     = regexp.MustCompile(`\s+`)
)

// CleanTitle replaces separator characters (dots, underscores, hyphens) with spaces and trims.
func CleanTitle(title string) string {
	cleaned := separatorPattern.ReplaceAllString(title, " ")
	return strings.TrimSpace(cleaned)
}

// ExtractYear attempts to extract a 4-digit year (1900-2100) from a string.
// Returns the year, the remaining string with the year removed, and whether a year was found.
func ExtractYear(s string) (year int, remainder string, found bool) {
	match := yearExtractPattern.FindStringSubmatchIndex(s)
	if match == nil {
		return 0, s, false
	}

	yearStr := s[match[2]:match[3]]
	y, err := strconv.Atoi(yearStr)
	if err != nil || y < 1900 || y > 2100 {
		return 0, s, false
	}

	remainder = s[:match[0]] + s[match[1]:]
	remainder = strings.TrimSpace(remainder)
	return y, remainder, true
}

// NormalizeTitle lowercases, removes articles, cleans punctuation and separators,
// strips year annotations in brackets/parens, and normalizes whitespace for comparison.
func NormalizeTitle(title string) string {
	title = strings.ToLower(title)

	// Replace separators with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Replace common punctuation with spaces
	title = strings.ReplaceAll(title, ":", " ")
	title = strings.ReplaceAll(title, "&", " ")
	title = strings.ReplaceAll(title, "/", " ")

	// Remove apostrophes entirely
	title = strings.ReplaceAll(title, "'", "")
	title = strings.ReplaceAll(title, "\u2019", "")

	// Remove year patterns in parentheses/brackets like (2017) or [2017]
	title = yearInBracketsPattern.ReplaceAllString(title, "")

	// Collapse multiple spaces
	title = multiSpacePattern.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	// Remove common article prefixes
	for _, prefix := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(title, prefix) {
			title = strings.TrimPrefix(title, prefix)
			break
		}
	}

	return title
}
