package pathutil

import (
	"path"
	"path/filepath"
	"strings"
)

// NormalizePath converts all path separators to forward slashes.
// Go's os.Open/os.Stat accept forward slashes on all platforms.
func NormalizePath(p string) string {
	return filepath.ToSlash(p)
}

// PathsEqual reports whether two paths refer to the same location,
// ignoring differences in separator style (\ vs /) and cleaning relative
// segments. Unlike NormalizePath, this always treats backslashes as
// separators regardless of host OS, so Windows-origin paths in the DB
// compare correctly even when the process is running on Unix.
// Comparison is case-sensitive; do not rely on this to match
// differently-cased paths on case-insensitive filesystems.
func PathsEqual(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	return canonicalize(a) == canonicalize(b)
}

// HasPathPrefix reports whether p lies within prefix (or equals it),
// treating \ and / as equivalent separators and requiring boundary alignment.
// Unlike strings.HasPrefix, "D:/foobar" does NOT have prefix "D:/foo".
// Comparison is case-sensitive.
func HasPathPrefix(p, prefix string) bool {
	_, ok := TrimPathPrefix(p, prefix)
	return ok
}

// TrimPathPrefix removes prefix from path and returns the remainder along
// with ok=true when prefix is a strict path-prefix of path (or equals it).
// Separator style and redundant slashes/dots are normalized before comparing.
// When path equals prefix the remainder is empty; otherwise the remainder
// includes its leading separator, matching strings.TrimPrefix semantics so
// callers can concatenate it directly with a replacement prefix.
//
// Examples:
//
//	TrimPathPrefix("D:/foo/bar", `D:\foo`) → "/bar", true
//	TrimPathPrefix("D:/foo",     "D:/foo") → "",     true
//	TrimPathPrefix("D:/foobar",  "D:/foo") → "",     false
func TrimPathPrefix(p, prefix string) (string, bool) {
	if p == "" || prefix == "" {
		return "", false
	}
	canonPath := canonicalize(p)
	canonPrefix := canonicalize(prefix)
	if canonPath == canonPrefix {
		return "", true
	}
	// Root prefix "/" matches any absolute path; preserve the leading slash.
	if canonPrefix == "/" {
		return canonPath, true
	}
	if strings.HasPrefix(canonPath, canonPrefix+"/") {
		return canonPath[len(canonPrefix):], true
	}
	return "", false
}

func canonicalize(p string) string {
	return path.Clean(strings.ReplaceAll(p, `\`, "/"))
}
