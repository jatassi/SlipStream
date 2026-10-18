package storing

import (
	"cmp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// naturalLess orders names naturally and without case: "Season 2" before
// "Season 10". Names equal but for case or leading zeros fall back to byte
// order, so the order is total.
func naturalLess(a, b string) bool {
	if c := naturalCompare(a, b); c != 0 {
		return c < 0
	}
	return a < b
}

func naturalCompare(x, y string) int {
	for x != "" && y != "" {
		if digit(x[0]) && digit(y[0]) {
			nx, ny := digitRun(x), digitRun(y)
			if c := compareNumbers(x[:nx], y[:ny]); c != 0 {
				return c
			}
			x, y = x[nx:], y[ny:]
			continue
		}
		rx, sx := utf8.DecodeRuneInString(x)
		ry, sy := utf8.DecodeRuneInString(y)
		if c := cmp.Compare(unicode.ToLower(rx), unicode.ToLower(ry)); c != 0 {
			return c
		}
		x, y = x[sx:], y[sy:]
	}
	return cmp.Compare(len(x), len(y))
}

func digit(c byte) bool { return '0' <= c && c <= '9' }

func digitRun(s string) int {
	i := 0
	for i < len(s) && digit(s[i]) {
		i++
	}
	return i
}

// compareNumbers compares two digit runs by value, of any length.
func compareNumbers(a, b string) int {
	a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
	if c := cmp.Compare(len(a), len(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}
