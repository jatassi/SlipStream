//go:build !windows

package library

import (
	"path/filepath"

	"golang.org/x/sys/unix"
)

// checkWritable tests whether the file at path can be deleted.
// On Unix, deletion permission depends on the parent directory being writable.
func checkWritable(path string) error {
	return unix.Access(filepath.Dir(path), unix.W_OK)
}
