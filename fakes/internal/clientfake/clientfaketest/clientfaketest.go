// Package clientfaketest is what the fake Download Clients' tests share:
// a world of fake-tracker and two releases that announce to it, and the
// payload placed where a client finds it.
package clientfaketest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
)

// Size is each release's payload size.
const Size = 1 << 16

// Tracked is fake-tracker served in memory, with the default 30 second
// interval, in shared, and its catalog: orchard and lantern, each one
// file announcing to it. A client served with shared and the catalog
// announces to it.
func Tracked(t *testing.T) (*releases.Catalog, *fake.Shared) {
	t.Helper()
	return TrackedEvery(t, 0)
}

// TrackedEvery is Tracked with fake-tracker answering interval, 0 for
// its default.
func TrackedEvery(t *testing.T, interval time.Duration) (*releases.Catalog, *fake.Shared) {
	t.Helper()
	s := scenario.Scenario{Seed: "clients", Tracker: scenario.Tracker{Interval: interval}}
	dir := t.TempDir()
	c := &releases.Catalog{}
	for _, name := range []string{"orchard", "lantern"} {
		payload := releases.Filler(name, Size)
		if err := os.WriteFile(filepath.Join(dir, name+".mkv"), payload, 0o600); err != nil {
			t.Fatal(err)
		}
		meta, hash, err := releases.Metainfo(name+".mkv", nil, payload, "https://127.0.0.1/announce/"+tracker.Passkey(s, releases.Seeders), false)
		if err != nil {
			t.Fatal(err)
		}
		c.Releases = append(c.Releases, releases.Release{
			Release: scenario.Release{Name: name}, Dir: dir, Files: []releases.File{{Path: name + ".mkv", Size: Size}}, Size: Size, Metainfo: meta, InfoHash: hash,
		})
	}
	shared := fake.NewShared()
	faketest.InMemory(t, tracker.Fake, s, faketest.WithReleases(c), faketest.WithShared(shared))
	return c, shared
}

// Place writes the first n bytes of r's payload into folder, where a
// client saving r there finds it.
func Place(t *testing.T, folder string, r releases.Release, n int) {
	t.Helper()
	data := Payload(t, r)
	if err := os.MkdirAll(folder, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, r.Files[0].Path), data[:n], 0o600); err != nil {
		t.Fatal(err)
	}
}

// Payload is the bytes of r's first file.
func Payload(t *testing.T, r releases.Release) []byte {
	t.Helper()
	data, err := os.ReadFile(r.Disk(r.Files[0]))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
