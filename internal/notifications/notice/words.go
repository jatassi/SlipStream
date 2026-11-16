package notice

import (
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/works"
)

// Count words n of noun: "1 episode", "10 episodes".
func Count(n int, noun works.Noun) string {
	if n == 1 {
		return "1 " + noun.One
	}
	return strconv.Itoa(n) + " " + noun.Many
}

// Titles is up to three of first as lines, then "{n} more were {verb}."
// for the rest of total.
func Titles(first []string, total int, verb string) []string {
	lines := slices.Clone(first[:min(len(first), 3)])
	switch rest := total - len(lines); {
	case rest == 1:
		return append(lines, "1 more was "+verb+".")
	case rest > 1:
		return append(lines, strconv.Itoa(rest)+" more were "+verb+".")
	}
	return lines
}
