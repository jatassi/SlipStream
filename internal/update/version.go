package update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

func ParseVersion(s string) (*Version, error) {
	matches := versionRegex.FindStringSubmatch(strings.TrimSpace(s))
	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %s", s)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	prerelease := ""
	if len(matches) > 4 {
		prerelease = matches[4]
	}

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, nil
}

func (v *Version) String() string {
	if v.Prerelease != "" {
		return fmt.Sprintf("%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Prerelease)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns:
//
//	-1 if v < other
//	 0 if v == other
//	 1 if v > other
func (v *Version) Compare(other *Version) int {
	if cmp := compareInt(v.Major, other.Major); cmp != 0 {
		return cmp
	}
	if cmp := compareInt(v.Minor, other.Minor); cmp != 0 {
		return cmp
	}
	if cmp := compareInt(v.Patch, other.Patch); cmp != 0 {
		return cmp
	}
	return comparePrerelease(v.Prerelease, other.Prerelease)
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// comparePrerelease compares prerelease strings.
// Empty prerelease (stable) has higher precedence than any prerelease version.
func comparePrerelease(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	if a < b {
		return -1
	}
	return 1
}

func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

func (v *Version) GreaterThan(other *Version) bool {
	return v.Compare(other) > 0
}

func (v *Version) Equal(other *Version) bool {
	return v.Compare(other) == 0
}

// IsNewerThan compares version strings directly.
// Returns true if newVersion is newer than currentVersion.
func IsNewerThan(newVersion, currentVersion string) (bool, error) {
	newV, err := ParseVersion(newVersion)
	if err != nil {
		return false, fmt.Errorf("invalid new version: %w", err)
	}

	currentV, err := ParseVersion(currentVersion)
	if err != nil {
		return false, fmt.Errorf("invalid current version: %w", err)
	}

	return newV.GreaterThan(currentV), nil
}
