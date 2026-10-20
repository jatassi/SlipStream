// Package imagecdn is the conformance suite for TMDB's image host: it reads
// known images at TMDB's sizes from fake-image-cdn on every PR and from
// image.tmdb.org in the online job, so a fake that drifts from the real
// host fails.
package imagecdn

import (
	"bytes"
	"fmt"
	"image"
	// Register JPEG and PNG with image.DecodeConfig.
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"testing"
)

// Known is a file name the host holds, as TMDB's file_path gives it, and
// the content type it answers it with.
type Known struct {
	Path string
	Type string
}

type Target struct {
	// Images is the host's root, the fake's front or TMDB's real one,
	// before the size.
	Images string
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	Known  []Known
}

// sizes name widths, a height and the original.
var sizes = []string{"w92", "w185", "w500", "w1280", "h632", "original"}

// formats are the names image.DecodeConfig gives each raster type.
var formats = map[string]string{"image/jpeg": "jpeg", "image/png": "png"}

// Run reads every Known image at every size and asserts what holds on
// both sides: each answers 200 under its content type, a raster decodes as
// that type at the size's width or height, never upscaled past the
// original, and an SVG is an svg element. A size TMDB doesn't offer
// answers 400.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	for _, k := range tgt.Known {
		original := get(t, tgt, "original"+k.Path)
		for _, size := range sizes {
			if problem := check(get(t, tgt, size+k.Path), original, k, size); problem != "" {
				t.Errorf("GET %s%s %s", size, k.Path, problem)
			}
		}
	}
	if r := get(t, tgt, "w93"+tgt.Known[0].Path); r.status != http.StatusBadRequest {
		t.Errorf("GET w93%s answered %d, want 400", tgt.Known[0].Path, r.status)
	}
}

type reply struct {
	status      int
	contentType string
	body        []byte
}

// check is what is wrong with r as k at size, or "".
func check(r, original reply, k Known, size string) string {
	if r.status != http.StatusOK || r.contentType != k.Type {
		return fmt.Sprintf("answered %d %s, want 200 %s", r.status, r.contentType, k.Type)
	}
	if k.Type == "image/svg+xml" {
		if !bytes.Contains(r.body, []byte("<svg")) {
			return fmt.Sprintf("is no svg element: %.80q", r.body)
		}
		return ""
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(r.body))
	if err != nil || format != formats[k.Type] {
		return fmt.Sprintf("decodes as %q, %v; want %s", format, err, formats[k.Type])
	}
	n, err := strconv.Atoi(size[1:])
	if err != nil {
		return ""
	}
	full, _, err := image.DecodeConfig(bytes.NewReader(original.body))
	if err != nil {
		return fmt.Sprintf("has an original that doesn't decode: %v", err)
	}
	if size[0] == 'w' && cfg.Width != min(n, full.Width) || size[0] == 'h' && cfg.Height != min(n, full.Height) {
		return fmt.Sprintf("is %dx%d, want %s of a %dx%d original", cfg.Width, cfg.Height, size, full.Width, full.Height)
	}
	return ""
}

func get(t *testing.T, tgt Target, path string) reply {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, tgt.Images+"/"+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := tgt.Client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return reply{status: res.StatusCode, contentType: res.Header.Get("Content-Type"), body: body}
}
