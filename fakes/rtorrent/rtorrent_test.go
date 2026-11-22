package rtorrent_test

import (
	"log/slog"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/rtorrent"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/clientfake/clientfaketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// interval is how often fake-tracker answers the announcing torrent, short
// so the last announce case runs in seconds.
const interval = 2 * time.Second

func TestFakeRTorrentConforms(t *testing.T) {
	for _, version := range rtorrent.Versions {
		c := rtorrent.Client{Node: "Media server", Version: version, Username: "jackson", Password: "orchard", Folder: "/downloads", Rate: 100, Check: time.Second}
		target := func(t *testing.T, f faketest.Served, cat *releases.Catalog) conformance.Target {
			return conformance.Target{
				Address: f.Address.URL(""), Username: c.Username, Password: c.Password, HTTP: f.Client, Folder: "/downloads/Movies",
				NodeFolder: filepath.Join(c.Dir, "Movies"), Interval: interval,
				Seeded: conformance.Seeded{Metainfo: cat.Releases[0].Metainfo, Hash: cat.Releases[0].InfoHash, Name: cat.Releases[0].Files[0].Path, Data: clientfaketest.Payload(t, cat.Releases[0])},
				Log:    slog.New(slog.NewTextHandler(t.Output(), nil)),
			}
		}
		t.Run(version+"/loopback", func(t *testing.T) {
			c.Dir = t.TempDir()
			cat, shared := clientfaketest.TrackedEvery(t, interval)
			f := faketest.Loopback(t, rtorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
			conformance.Run(t, target(t, f, cat))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c.Dir = t.TempDir()
				cat, shared := clientfaketest.TrackedEvery(t, interval)
				f := faketest.InMemory(t, rtorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
				conformance.Run(t, target(t, f, cat))
			})
		})
	}
}
