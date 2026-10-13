// Package atomicfile writes a small file so that a crash leaves either the
// old file or the whole new one.
package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Write writes data to path with mode 0600 by temp file, fsync, rename and
// a directory fsync.
func Write(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Sync()
	}
	err = errors.Join(err, tmp.Close())
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return errors.Join(fmt.Errorf("write %s: %w", path, err), os.Remove(tmp.Name()))
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
