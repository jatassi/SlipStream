// Package fsinfo is a file as the side that received it sees it: its
// size, its link count and the BLAKE3 of its bytes, the same on every OS.
package fsinfo

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/zeebo/blake3"
)

type Info struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Links uint64 `json:"links"`
	// BLAKE3 is the hex BLAKE3 of the file's bytes.
	BLAKE3 string `json:"blake3"`
}

// Stat reads the regular file at path.
func Stat(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, fmt.Errorf("stat: %w", err)
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return Info{}, fmt.Errorf("stat: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return Info{}, fmt.Errorf("stat %s: not a regular file", path)
	}
	n, err := links(f, fi)
	if err != nil {
		return Info{}, fmt.Errorf("stat %s: %w", path, err)
	}
	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return Info{}, fmt.Errorf("stat %s: %w", path, err)
	}
	return Info{Path: path, Size: fi.Size(), Links: n, BLAKE3: hex.EncodeToString(h.Sum(nil))}, nil
}
