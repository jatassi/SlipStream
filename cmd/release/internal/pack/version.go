package pack

import (
	"fmt"
	"regexp"
	"strings"
)

// Version is a package's version: the release's tag without its v, such
// as 1.2.0 or 1.2.0-beta.1.
type Version string

var tagPattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.]+)?$`)

// ParseTag reads a release tag, vX.Y.Z with an optional pre-release suffix.
func ParseTag(tag string) (Version, error) {
	if !tagPattern.MatchString(tag) {
		return "", fmt.Errorf("tag %q is not vX.Y.Z or vX.Y.Z-<pre-release>", tag)
	}
	return Version(strings.TrimPrefix(tag, "v")), nil
}

// Described is the version of a build git describe named: the tag's, its
// distance and commit after a tag, or 0.0.0-g<commit> where no tag reaches.
func Described(described string) Version {
	if strings.HasPrefix(described, "v") && len(described) > 1 && described[1] >= '0' && described[1] <= '9' {
		return Version(described[1:])
	}
	return Version("0.0.0-g" + described)
}

func (v Version) String() string { return string(v) }

// Prerelease reports a version with a suffix after X.Y.Z.
func (v Version) Prerelease() bool { return strings.Contains(string(v), "-") }

// Numeric is X.Y.Z alone, the only form an MSI and a bundle version take.
func (v Version) Numeric() string {
	numeric, _, _ := strings.Cut(string(v), "-")
	return numeric
}

// ImageTags are the tags the Docker image of a release takes: immutable
// X.Y.Z and moving X.Y, X and latest, or beta alone for a pre-release.
func (v Version) ImageTags() []string {
	if v.Prerelease() {
		return []string{"beta"}
	}
	parts := strings.Split(string(v), ".")
	return []string{string(v), parts[0] + "." + parts[1], parts[0], "latest"}
}
