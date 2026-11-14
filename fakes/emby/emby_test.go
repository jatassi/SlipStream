package emby_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/emby"
	"github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func TestFakeEmbyConforms(t *testing.T) {
	for _, version := range emby.Versions {
		for _, root := range []string{"/media", `D:\Media`} {
			name := version + "/" + map[bool]string{false: "posix", true: "windows"}[root[1] == ':']
			t.Run(name+"/loopback", func(t *testing.T) {
				f := faketest.Loopback(t, emby.Playing(server(version, root, 2*time.Second)), scenario.Scenario{Seed: "emby"}, faketest.WithMedia(seeded(t)), faketest.As("Den"))
				conformance.Run(t, target(t, f, root, 3*time.Second, 30*time.Second))
			})
			t.Run(name+"/in memory", func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					f := faketest.InMemory(t, emby.Playing(server(version, root, 0)), scenario.Scenario{Seed: "emby"}, faketest.WithMedia(seeded(t)), faketest.As("Den"))
					conformance.Run(t, target(t, f, root, 2*time.Minute, 5*time.Minute))
				})
			})
		}
	}
}

func join(root, rel string) string {
	sep := "/"
	if root[1] == ':' {
		sep = `\`
	}
	return root + sep + rel
}

func server(version, root string, delay time.Duration) emby.Server {
	return emby.Server{
		Name: "Den", Version: version, ID: "5f1b0c3e2a7d4e6f8a9b0c1d2e3f4a5b", Keys: []string{"conformance-key"},
		Users:        []emby.User{{Name: "jackson", Password: "conformance-password", Admin: true}},
		MonitorDelay: delay, Root: root,
		Libraries: []emby.Library{
			{Name: "Movies", Type: "movies", Folders: []string{join(root, "Movies")}, Scanned: true},
			{Name: "Fresh", Type: "movies", Folders: []string{join(root, "Fresh")}},
		},
	}
}

// seeded is a media dir whose Movies folder holds one film and whose
// Fresh folder is empty.
func seeded(t *testing.T) string {
	t.Helper()
	media := t.TempDir()
	file := filepath.Join(media, "Movies", "Metropolis (1927)", "Metropolis (1927).mkv")
	for _, dir := range []string{filepath.Dir(file), filepath.Join(media, "Fresh")} {
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

func target(t *testing.T, f faketest.Served, root string, settle, bound time.Duration) conformance.Target {
	t.Helper()
	return conformance.Target{
		Address: f.Address.URL(""), Key: "conformance-key", HTTP: f.Client,
		Library: "Movies", Folder: path(t, join(root, "Movies")),
		Disk:  conformance.DirDisk{Dir: filepath.Join(f.Media, "Movies")},
		Fresh: &conformance.Fresh{Library: "Fresh", Folder: path(t, join(root, "Fresh")), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "Fresh")}},
		Play: func(t *testing.T, p link.ServerPath, d link.Decision) (conformance.Played, func()) {
			played, _ := f.Cue(t, "emby-play", emby.Play{Path: p.String(), User: "jackson", Player: "Living Room", Decision: d}).(emby.Played)
			return conformance.Played{Session: played.Session, Item: played.Item}, func() { f.Cue(t, "emby-stop", emby.Stop{Session: played.Session}) }
		},
		Fake: true, Settle: settle, Bound: bound,
		Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
	}
}
