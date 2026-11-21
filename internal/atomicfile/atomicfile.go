// Package atomicfile writes a small file so that a crash leaves either the
// old file or the whole new one, and renames and deletes files through the
// retry Windows needs.
package atomicfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Write writes data to path with mode 0600 by temp file, fsync, rename and
// a directory fsync.
func Write(path string, data []byte) error { return Replace(path, data, 0) }

// Replace is Write with its rename retried as Rename retries it.
func Replace(path string, data []byte, retry time.Duration) error {
	tmp, err := temp(path, data)
	if err != nil {
		return err
	}
	if err := Rename(tmp, path, retry); err != nil {
		return errors.Join(fmt.Errorf("write %s: %w", path, err), os.Remove(tmp))
	}
	return SyncDir(filepath.Dir(path))
}

// Create is Replace that never replaces: when a file is already at path it
// fails with an error that wraps fs.ErrExist, so of two Creates of one path
// only one succeeds, even in two processes.
func Create(path string, data []byte, retry time.Duration) error {
	tmp, err := temp(path, data)
	if err != nil {
		return err
	}
	err = retried(retry, func() error { return os.Link(tmp, path) })
	if err = errors.Join(err, Remove(tmp, retry)); err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	return SyncDir(filepath.Dir(path))
}

func temp(path string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Sync()
	}
	if err = errors.Join(err, tmp.Close()); err != nil {
		return "", errors.Join(fmt.Errorf("write %s: %w", path, err), os.Remove(tmp.Name()))
	}
	return tmp.Name(), nil
}

// Rename is os.Rename, which replaces a file at newPath. On Windows a
// sharing violation, access denied or file not found is retried for up to
// retry, as cmd/go's robustio retries them: an antivirus scan or the search
// indexer holds a file it has just seen open for a moment.
func Rename(oldPath, newPath string, retry time.Duration) error {
	return retried(retry, func() error { return os.Rename(oldPath, newPath) })
}

// Remove deletes path, retrying as Rename does. A path already gone is no
// error.
func Remove(path string, retry time.Duration) error {
	err := retried(retry, func() error {
		err := os.Remove(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err //nolint:wrapcheck // wrapped once below, after the retries
	})
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// retried runs op until it succeeds, fails with an error that isn't
// ephemeral, or retry has passed since the first failure.
func retried(retry time.Duration, op func() error) error {
	err := op()
	deadline := time.Now().Add(retry)
	for wait := time.Millisecond; err != nil && ephemeral(err) && time.Now().Before(deadline); wait = min(2*wait, 100*time.Millisecond) {
		time.Sleep(min(wait, time.Until(deadline)))
		err = op()
	}
	return err
}
