package plex_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/plex"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/plex"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

func TestFakePlexConforms(t *testing.T) {
	for _, version := range plex.Versions {
		for _, root := range []string{"/media", `C:\Users\jackson\Media`} {
			for _, pass := range []bool{false, true} {
				srv := plex.Server{
					Name: "Den", Version: version, Identifier: "fake-plex-den", Tokens: []string{"server-token"}, Owner: "jackson",
					PlexPass: pass, Root: root,
				}
				name := version + "/" + filepath.Base(filepath.ToSlash(root)) + map[bool]string{false: "", true: "/plex pass"}[pass]
				t.Run(name+"/loopback", func(t *testing.T) {
					media := seeded(t)
					conformance.Run(t, target(t, faketest.Loopback(t, sections(t, srv), scenario.Scenario{Seed: "plex"}, faketest.WithMedia(media), faketest.As("Den")), srv))
				})
				t.Run(name+"/in memory", func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						media := seeded(t)
						conformance.Run(t, target(t, faketest.InMemory(t, sections(t, srv), scenario.Scenario{Seed: "plex"}, faketest.WithMedia(media), faketest.As("Den")), srv))
					})
				})
			}
		}
	}
}

// seeded is a media dir whose Movies folder holds one film, so its
// section was scanned with content, and whose Fresh folder is empty.
func seeded(t *testing.T) string {
	t.Helper()
	media := t.TempDir()
	file := filepath.Join(media, "Movies", "Metropolis (1927)", "Metropolis (1927).mkv")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(media, "Fresh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("Metropolis"), 0o600); err != nil {
		t.Fatal(err)
	}
	return media
}

func folder(t *testing.T, srv plex.Server, name string) link.ServerPath {
	t.Helper()
	root, err := link.ParsePath(srv.Root)
	if err != nil {
		t.Fatal(err)
	}
	r, err := storing.ParseRel(name)
	if err != nil {
		t.Fatal(err)
	}
	return root.Join(r)
}

func sections(t *testing.T, srv plex.Server) fake.Spec {
	t.Helper()
	srv.Sections = []plex.Section{
		{Key: "3", Title: "Movies", Type: "movie", Folders: []string{folder(t, srv, "Movies").String()}, Scanned: true},
		{Key: "4", Title: "Fresh", Type: "movie", Folders: []string{folder(t, srv, "Fresh").String()}},
	}
	return plex.Playing(srv)
}

func target(t *testing.T, f faketest.Served, srv plex.Server) conformance.Target {
	t.Helper()
	return conformance.Target{
		Address: f.Address.URL(""), Token: "server-token", HTTP: f.Client,
		Section: "3", Folder: folder(t, srv, "Movies"),
		Disk:  conformance.DirDisk{Dir: filepath.Join(f.Media, "Movies")},
		Fresh: &conformance.Fresh{Library: "4", Folder: folder(t, srv, "Fresh"), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "Fresh")}},
		Play: func(t *testing.T, p link.ServerPath, d link.Decision) func() {
			played, _ := f.Cue(t, "plex-play", plex.Play{Path: p.String(), User: "jackson", Player: "Living Room", Decision: d}).(plex.Played)
			return func() { f.Cue(t, "plex-stop", plex.Stop(played)) }
		},
		Fake: true, Settle: time.Second, Bound: 10 * time.Second,
		Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
	}
}
