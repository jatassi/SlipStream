package clients

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct{ major, minor, patch int }

// ParseVersion reads the leading numbers of a version a product reports,
// such as "4.1.1 (a8bd3e2b)", "v4.6.2" or "2.1.1". Missing numbers are 0.
func ParseVersion(s string) (Version, error) {
	rest := strings.TrimPrefix(s, "v")
	end := strings.IndexFunc(rest, func(r rune) bool { return (r < '0' || r > '9') && r != '.' })
	if end >= 0 {
		rest = rest[:end]
	}
	parts := strings.Split(rest, ".")
	var n [3]int
	for i := range min(len(parts), len(n)) {
		var err error
		if n[i], err = strconv.Atoi(parts[i]); err != nil {
			return Version{}, fmt.Errorf("the version %q doesn't start with a number", s)
		}
	}
	return Version{n[0], n[1], n[2]}, nil
}

func (v Version) Less(w Version) bool {
	if v.major != w.major {
		return v.major < w.major
	}
	if v.minor != w.minor {
		return v.minor < w.minor
	}
	return v.patch < w.patch
}

// String drops a patch of 0, as sentences name versions: "4.6", "4.1.1".
func (v Version) String() string {
	if v.patch == 0 {
		return fmt.Sprintf("%d.%d", v.major, v.minor)
	}
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}
