package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckDeletable verifies that all existing files in the given paths can be
// deleted by the current process. Files that don't exist are silently skipped.
// Returns a descriptive error listing any paths that lack write permission.
func CheckDeletable(paths []string) error {
	var denied []string

	for _, p := range paths {
		if p == "" {
			continue
		}

		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			denied = append(denied, p)
			continue
		}

		if info.IsDir() {
			continue
		}

		if err := checkWritable(p); err != nil {
			denied = append(denied, p)
		}
	}

	if len(denied) > 0 {
		return fmt.Errorf("permission denied for %d file(s): %s", len(denied), strings.Join(denied, ", "))
	}

	return nil
}

// DeleteFiles removes the given file paths from disk. Files that don't exist
// are silently skipped. Returns an error on the first real failure.
func DeleteFiles(paths []string) (int, error) {
	deleted := 0
	for _, p := range paths {
		if p == "" {
			continue
		}

		if err := os.Remove(p); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return deleted, fmt.Errorf("failed to delete %s: %w", filepath.Base(p), err)
		}
		deleted++
	}
	return deleted, nil
}
