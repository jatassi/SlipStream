package jellyfin_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/jellyfin"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func TestFakeJellyfinConforms(t *testing.T) {
	for _, version := range jellyfin.Versions {
		t.Run(version+"/loopback", func(t *testing.T) {
			srv := server(version, time.Second)
			media := seeded(t)
			f := faketest.Loopback(t, jellyfin.Playing(srv), scenario.Scenario{Seed: "jellyfin"}, faketest.WithMedia(media), faketest.As("Den"))
			conformance.Run(t, target(t, f, 2*time.Second, 30*time.Second))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				srv := server(version, 0)
				media := seeded(t)
				f := faketest.InMemory(t, jellyfin.Playing(srv), scenario.Scenario{Seed: "jellyfin"}, faketest.WithMedia(media), faketest.As("Den"))
				conformance.Run(t, target(t, f, 90*time.Second, 5*time.Minute))
			})
		})
	}
}

func server(version string, delay time.Duration) jellyfin.Server {
	return jellyfin.Server{
		Name: "Den", Version: version, ID: "ddc0b5b7f2824180aae9f2f245dd00ca", Keys: []string{"conformance-key"},
		Users:        []jellyfin.User{{Name: "jackson", Password: "conformance-password", Admin: true}},
		MonitorDelay: delay, Root: "/media",
		Libraries: []jellyfin.Library{
			{Name: "Movies", Type: "movies", Folders: []string{"/media/Movies", "/media/New"}, Scanned: true},
			{Name: "Fresh", Type: "movies", Folders: []string{"/media/Fresh"}},
		},
	}
}

// seeded is a media dir whose Movies folder holds one film, so it was
// scanned with content, and whose New and Fresh folders are empty.
func seeded(t *testing.T) string {
	t.Helper()
	media := t.TempDir()
	file := filepath.Join(media, "Movies", "Metropolis (1927)", "Metropolis (1927).mkv")
	for _, dir := range []string{filepath.Dir(file), filepath.Join(media, "New"), filepath.Join(media, "Fresh")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(file, []byte("Metropolis"), 0o600); err != nil {
		t.Fatal(err)
	}
	return media
}

func path(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func target(t *testing.T, f faketest.Served, settle, bound time.Duration) conformance.Target {
	t.Helper()
	return conformance.Target{
		Address: f.Address.URL(""), Key: "conformance-key", HTTP: f.Client,
		Library: "Movies", Folder: path(t, "/media/Movies"),
		Disk:      conformance.DirDisk{Dir: filepath.Join(f.Media, "Movies")},
		Fresh:     &conformance.Fresh{Library: "Fresh", Folder: path(t, "/media/Fresh"), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "Fresh")}},
		FreshRoot: &conformance.Root{Folder: path(t, "/media/New"), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "New")}},
		Admin:     &conformance.Admin{User: "jackson", Password: "conformance-password"},
		Play: func(t *testing.T, p link.ServerPath, d link.Decision) (conformance.Played, func()) {
			played, _ := f.Cue(t, "jellyfin-play", jellyfin.Play{Path: p.String(), User: "jackson", Player: "Living Room", Decision: d}).(jellyfin.Played)
			return conformance.Played{Session: played.Session, Item: played.Item}, func() { f.Cue(t, "jellyfin-stop", jellyfin.Stop{Session: played.Session}) }
		},
		Fake: true, Settle: settle, Bound: bound,
		Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
	}
}
