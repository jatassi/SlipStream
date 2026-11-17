package entry

import (
	"slices"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/works"
)

// Count words n of noun: "1 episode", "1,200 episodes".
func Count(n int, noun works.Noun) string {
	if n == 1 {
		return "1 " + noun.One
	}
	return setting.Grouped(n) + " " + noun.Many
}

// Titles is up to three of first as lines, then "{n} more were {verb}."
// for the rest of total.
func Titles(first []string, total int, verb string) []string {
	lines := slices.Clone(first[:min(len(first), 3)])
	switch rest := total - len(lines); {
	case rest == 1:
		return append(lines, "1 more was "+verb+".")
	case rest > 1:
		return append(lines, setting.Grouped(rest)+" more were "+verb+".")
	}
	return lines
}
