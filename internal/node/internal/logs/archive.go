package logs

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Archive zips every file in the log folder into a new file in tmp: the
// current file first, then the rotated ones by name, the compressed ones
// stored as they are. It returns the zip at offset 0 and its size.
func (l *Log) Archive(tmp string) (zipped *os.File, size int64, err error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, 0, fmt.Errorf("list the log folder: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			names = append(names, e.Name())
		}
	}
	slices.SortStableFunc(names, func(a, b string) int {
		if (a == File) != (b == File) {
			if a == File {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})
	f, err := os.CreateTemp(tmp, "logs-*.zip")
	if err != nil {
		return nil, 0, fmt.Errorf("create the log archive: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, f.Close(), os.Remove(f.Name()))
		}
	}()
	zw := zip.NewWriter(f)
	for _, name := range names {
		if err := addFile(zw, filepath.Join(l.dir, name)); err != nil {
			return nil, 0, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, 0, fmt.Errorf("write the log archive: %w", err)
	}
	if size, err = f.Seek(0, io.SeekCurrent); err == nil {
		_, err = f.Seek(0, io.SeekStart)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("rewind the log archive: %w", err)
	}
	return f, size, nil
}

// addFile adds path as it is now: a file still being written ends at the
// size it had when opened.
func addFile(zw *zip.Writer, path string) error {
	src, err := os.Open(path) //nolint:gosec // a file of the Node's own logs dir
	if err != nil {
		return fmt.Errorf("open %s: %w", filepath.Base(path), err)
	}
	defer func() { _ = src.Close() }()
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", filepath.Base(path), err)
	}
	method := zip.Deflate
	if filepath.Ext(path) == ".gz" {
		method = zip.Store
	}
	w, err := zw.CreateHeader(&zip.FileHeader{Name: info.Name(), Method: method, Modified: info.ModTime()})
	if err != nil {
		return fmt.Errorf("archive %s: %w", info.Name(), err)
	}
	if _, err := io.Copy(w, io.LimitReader(src, info.Size())); err != nil {
		return fmt.Errorf("archive %s: %w", info.Name(), err)
	}
	return nil
}
