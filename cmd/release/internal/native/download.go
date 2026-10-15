package native

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// FetchFile makes d present in the cache, named by its SHA-256 like a
// layer, and returns its path. The file keeps the base name of its URL.
func FetchFile(ctx context.Context, cacheRoot string, d Download) (string, error) {
	dir := filepath.Join(cacheRoot, "file", d.SHA256)
	file := filepath.Join(dir, path.Base(d.URL))
	if _, err := os.Stat(file); err == nil {
		return file, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("check the native cache: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create the native cache: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".fetching-")
	if err != nil {
		return "", fmt.Errorf("create the native cache: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	sum, err := download(ctx, d.URL, tmp)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if sum != d.SHA256 {
		return "", fmt.Errorf("%s has SHA-256 %s, %s pins %s", d.URL, sum, InputsPath, d.SHA256)
	}
	if err := os.Rename(tmp.Name(), file); err != nil {
		return "", fmt.Errorf("move %s into the native cache: %w", d.URL, err)
	}
	return file, nil
}
