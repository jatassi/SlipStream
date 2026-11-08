package formats_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
)

var titles = sync.OnceValues(func() ([]release.Read, error) {
	b, err := os.ReadFile("testdata/titles.txt")
	if err != nil {
		return nil, fmt.Errorf("read the titles: %w", err)
	}
	var reads []release.Read
	for name := range strings.Lines(string(b)) {
		reads = append(reads, release.Title(strings.TrimRight(name, "\r\n"), release.TitleOnly{}, nil))
	}
	return reads, nil
})

func TestTRaSHFormats(t *testing.T) {
	reads, err := titles()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []formats.Dialect{formats.Radarr, formats.Sonarr} {
		files, err := filepath.Glob(filepath.Join("testdata", "trash", string(d), "cf", "*.json"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no %s formats in the snapshot: %v", d, err)
		}
		for _, path := range files {
			t.Run(string(d)+"/"+strings.TrimSuffix(filepath.Base(path), ".json"), func(t *testing.T) {
				t.Parallel()
				runTRaSH(t, readTRaSH(t, path, d), reads)
			})
		}
	}
}

func runTRaSH(t *testing.T, f formats.Format, reads []release.Read) {
	t.Helper()
	snapshotLimit := formats.Timings{Match: formats.DefaultTimings().Match * raceSlowdown}
	if err := formats.Check(f, snapshotLimit); err != nil {
		t.Fatal(err)
	}
	c := formats.Compile([]formats.Format{f}, snapshotLimit)
	for _, r := range reads {
		for i := range slowConditions(c, r) {
			t.Errorf("condition %d (%s) ran past %v on %q, %d times running", i, f.Conditions[i].Name, snapshotLimit.Match, r.Given, attempts)
		}
	}
}

// attempts is how often a condition must run past the limit to fail. A
// pattern that backtracks does so every time. On a busy machine a pattern
// of a few microseconds also does now and then, while it waits for a CPU.
const attempts = 3

// slowConditions are the conditions that ran past the limit on every
// attempt.
func slowConditions(c *formats.Catalog, r release.Read) map[int]bool {
	slow := map[int]bool{}
	for n := range attempts {
		e := c.Explain(r)[0]
		for i, ev := range e.Conditions {
			if n == 0 && ev.TimedOut {
				slow[i] = true
			}
			if !ev.TimedOut {
				delete(slow, i)
			}
		}
		if len(slow) == 0 {
			break
		}
	}
	return slow
}

func readTRaSH(t *testing.T, path string, d formats.Dialect) formats.Format {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fs, err := formats.Paste(b)
	if err != nil {
		t.Fatal(err)
	}
	if app, ok := formats.AppOf(fs); ok && app != d {
		t.Errorf("AppOf = %s for a %s format", app, d)
	}
	f := fs[0]
	f.ID, f.Dialect = formats.ID{1}, d
	return f
}
