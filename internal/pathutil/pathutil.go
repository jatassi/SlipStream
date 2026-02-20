package pathutil

import "path/filepath"

// NormalizePath converts all path separators to forward slashes.
// Go's os.Open/os.Stat accept forward slashes on all platforms.
func NormalizePath(p string) string {
	return filepath.ToSlash(p)
}
