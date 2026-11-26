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
	"github.com/jatassi/slipstream/fakes/internal/serverfake"
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

// seeded is a media dir whose Movies and Movies 4K folders each hold one
// film, so their sections were scanned with content, and whose Fresh
// folder is empty.
func seeded(t *testing.T) string {
	t.Helper()
	media := t.TempDir()
	if err := os.MkdirAll(filepath.Join(media, "Fresh"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{"Movies", "Movies 4K"} {
		file := filepath.Join(media, section, "Metropolis (1927)", "Metropolis (1927).mkv")
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("Metropolis "+section), 0o600); err != nil {
			t.Fatal(err)
		}
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
		{Key: "5", Title: "Movies 4K", Type: "movie", Folders: []string{folder(t, srv, "Movies 4K").String()}, Scanned: true},
	}
	return plex.Playing(srv)
}

func target(t *testing.T, f faketest.Served, srv plex.Server) conformance.Target {
	t.Helper()
	return conformance.Target{
		Address: f.Address.URL(""), Token: "server-token", HTTP: f.Client,
		Section: "3", Folder: folder(t, srv, "Movies"),
		Disk:   conformance.DirDisk{Dir: filepath.Join(f.Media, "Movies")},
		Fresh:  &conformance.Library{Library: "4", Folder: folder(t, srv, "Fresh"), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "Fresh")}},
		Second: &conformance.Library{Library: "5", Folder: folder(t, srv, "Movies 4K"), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "Movies 4K")}},
		Play: func(t *testing.T, p conformance.Play) func() {
			played, _ := f.Cue(t, "plex-play", cued(p)).(plex.Played)
			return func() { f.Cue(t, "plex-stop", plex.Stop(played)) }
		},
		Fake: true, Settle: time.Second, Bound: 10 * time.Second,
		Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
	}
}

// cued is the play cue for p, by jackson on Living Room.
func cued(p conformance.Play) plex.Play {
	return plex.Play{
		Path: p.Path.String(), Version: p.Version.String(), User: "jackson", Player: "Living Room", PlayerKey: p.PlayerKey, Platform: p.Platform,
		Video: p.Streams.Video, Audio: p.Streams.Audio, Subtitles: p.Streams.Subtitles, ContainerChanged: p.Streams.ContainerChanged,
		AudioTrack: serverfake.Track(p.Audio), SubtitleTrack: serverfake.Track(p.Subtitle), Reasons: p.Reasons,
	}
}
