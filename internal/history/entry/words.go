package entry

import (
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/works"
)

// Count words n of noun: "1 episode", "1,200 episodes".
func Count(n int, noun works.Noun) string {
	if n == 1 {
		return "1 " + noun.One
	}
	return Grouped(n) + " " + noun.Many
}

// Grouped is n with its thousands grouped by commas, as English writes it.
func Grouped(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Titles is up to three of first as lines, then "{n} more were {verb}."
// for the rest of total.
func Titles(first []string, total int, verb string) []string {
	lines := slices.Clone(first[:min(len(first), 3)])
	switch rest := total - len(lines); {
	case rest == 1:
		return append(lines, "1 more was "+verb+".")
	case rest > 1:
		return append(lines, Grouped(rest)+" more were "+verb+".")
	}
	return lines
}
