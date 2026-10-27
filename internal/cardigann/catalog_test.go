package cardigann

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestCatalog loads every definition in the directory that
// SLIPSTREAM_CARDIGANN_CATALOG names, such as an unzipped package.zip. Each
// must load, or be refused as using something SlipStream doesn't run.
func TestCatalog(t *testing.T) {
	dir := os.Getenv("SLIPSTREAM_CARDIGANN_CATALOG")
	if dir == "" {
		t.Skip("SLIPSTREAM_CARDIGANN_CATALOG names no catalog directory")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no definitions in %s: %v", dir, err)
	}
	slices.Sort(files)
	loaded, unsupported, broken := 0, 0, 0
	uses := map[Feature]int{}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".yml")
		d, branches, err := load(src)
		for _, b := range branches {
			t.Logf("%s: a branch doesn't compile: %s", name, b)
		}
		broken += min(len(branches), 1)
		switch {
		case err == nil:
			loaded++
			for _, u := range d.Summary().Uses {
				uses[u]++
			}
		case errors.Is(err, errUnsupported):
			unsupported++
			t.Logf("%s: unsupported: %v", name, err)
		default:
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, c := range featureChecks {
		t.Logf("%4d use %s", uses[c.feature], c.feature)
	}
	t.Logf("%d definitions: %d load, %d unsupported, %d with a branch that doesn't compile", len(files), loaded, unsupported, broken)
	if loaded+unsupported != len(files) {
		t.Errorf("%d of %d definitions failed to read or compile", len(files)-loaded-unsupported, len(files))
	}
}
