package link

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a product version's leading numbers: "1.43.5.11029-1bd7c5a7e"
// is 1.43.5.11029, "12.2.0" is 12.2.0.
type Version struct {
	n    int
	part [4]int
}

func ParseVersion(s string) (Version, error) {
	var v Version
	rest := s
	for v.n < len(v.part) {
		end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
		if end < 0 {
			end = len(rest)
		}
		if end == 0 {
			break
		}
		n, err := strconv.Atoi(rest[:end])
		if err != nil {
			return Version{}, fmt.Errorf("version %q: %w", s, err)
		}
		v.part[v.n] = n
		v.n++
		if end == len(rest) || rest[end] != '.' {
			break
		}
		rest = rest[end+1:]
	}
	if v.n == 0 {
		return Version{}, fmt.Errorf("%q doesn't start with a version number", s)
	}
	return v, nil
}

// MustVersion is ParseVersion for a literal floor.
func MustVersion(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

func (v Version) IsZero() bool { return v.n == 0 }

func (v Version) String() string {
	parts := make([]string, v.n)
	for i := range parts {
		parts[i] = strconv.Itoa(v.part[i])
	}
	return strings.Join(parts, ".")
}

// Compare orders versions by their numbers, a missing number reading as 0,
// so 12.2 and 12.2.0 are equal.
func (v Version) Compare(w Version) int {
	for i := range v.part {
		if c := v.part[i] - w.part[i]; c != 0 {
			if c < 0 {
				return -1
			}
			return 1
		}
	}
	return 0
}
