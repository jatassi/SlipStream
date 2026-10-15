// Package icons records which inputs rendered the web app's PNG icons, so
// release lint can tell they are stale without rendering them again:
// Chromium rasterises the same SVG differently on each OS.
package icons

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// Renderer draws brand/app-icon.svg into web/public with Playwright's
	// Chromium. Run it with bun from the repo root.
	Renderer        = "brand/render-icons.mjs"
	stampPath       = "brand/icons.sha256"
	generateCommand = "go run ./cmd/release gen icons"
)

var files = []string{
	"brand/app-icon.svg",
	Renderer,
	"web/public/apple-touch-icon.png",
	"web/public/icon-192.png",
	"web/public/icon-512.png",
}

// Record writes the stamp for the icons Renderer just drew.
func Record(root string) error {
	stamp, err := stampOf(root)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(stampPath)), stamp, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", stampPath, err)
	}
	return nil
}

// Check reports a master, renderer or icon that changed since Record.
func Check(root string) error {
	want, err := stampOf(root)
	if err != nil {
		return err
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(stampPath)))
	if err != nil {
		return fmt.Errorf("read %s: %w", stampPath, err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s is stale; run %s", stampPath, generateCommand)
	}
	return nil
}

// stampOf lists each file's SHA-256 in sha256sum's format.
func stampOf(root string) ([]byte, error) {
	var b bytes.Buffer
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		fmt.Fprintf(&b, "%x  %s\n", sha256.Sum256(data), f)
	}
	return b.Bytes(), nil
}
