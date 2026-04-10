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

func canonicalize(p string) string {
	return path.Clean(strings.ReplaceAll(p, `\`, "/"))
}
