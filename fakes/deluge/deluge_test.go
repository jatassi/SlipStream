package deluge_test

import (
	"log/slog"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/deluge"
	"github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/clientfake/clientfaketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// interval is how often fake-tracker answers the announcing torrent, short
// so the last announce case runs in seconds.
const interval = 2 * time.Second

func TestFakeDelugeConforms(t *testing.T) {
	for _, version := range deluge.Versions {
		c := deluge.Client{Node: "Media server", Version: version, Password: "orchard", AutoConnect: 1}
		daemon := deluge.Daemon{Host: "127.0.0.1", Port: 58846, Folder: "/downloads", Rate: 100, Check: time.Second}
		target := func(t *testing.T, f faketest.Served, cat *releases.Catalog) conformance.Target {
			return conformance.Target{
				Address: f.Address.URL(""), Password: c.Password, HTTP: f.Client, Folder: "/downloads/Movies",
				NodeFolder: filepath.Join(daemon.Dir, "Movies"), Interval: interval,
				Seeded: conformance.Seeded{Metainfo: cat.Releases[0].Metainfo, Hash: cat.Releases[0].InfoHash, Name: cat.Releases[0].Files[0].Path, Data: clientfaketest.Payload(t, cat.Releases[0])},
				Log:    slog.New(slog.NewTextHandler(t.Output(), nil)),
			}
		}
		t.Run(version+"/loopback", func(t *testing.T) {
			daemon.Dir = t.TempDir()
			cat, shared := clientfaketest.TrackedEvery(t, interval)
			c.Daemons = []deluge.Daemon{daemon}
			f := faketest.Loopback(t, deluge.Playing(c), scenario.Scenario{Seed: "deluge"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
			conformance.Run(t, target(t, f, cat))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				daemon.Dir = t.TempDir()
				cat, shared := clientfaketest.TrackedEvery(t, interval)
				c.Daemons = []deluge.Daemon{daemon}
				f := faketest.InMemory(t, deluge.Playing(c), scenario.Scenario{Seed: "deluge"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
				conformance.Run(t, target(t, f, cat))
			})
		})
	}
}
