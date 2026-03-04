//go:build windows

package library

import "os"

// checkWritable tests whether the file at path can be deleted.
// On Windows, file deletion permission is on the file itself.
// We attempt to open the file with write access as a proxy check.
func checkWritable(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	return f.Close()
}
