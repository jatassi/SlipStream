package qbittorrent_test

import (
	"log/slog"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/qbittorrent"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/clientfake/clientfaketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/qbittorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// interval is how often fake-tracker answers the announcing torrent, short
// so the last announce case runs in seconds.
const interval = 2 * time.Second

func TestFakeQBittorrentConforms(t *testing.T) {
	for _, version := range qbittorrent.Versions {
		c := qbittorrent.Client{Node: "Media server", Version: version, Username: "jackson", Password: "orchard", BanAfter: 5, Folder: "/downloads", Rate: 100, Check: time.Second}
		if version >= "5.2" {
			c.APIKey = "qbt_0123456789abcdefghijklmnopqrstuv"
		}
		// The real 4.6.7 and 5.3.0 ran cold, just after their machine booted.
		c.Cold = version == "4.6" || version == "5.3"
		target := func(t *testing.T, f faketest.Served, cat *releases.Catalog) conformance.Target {
			return conformance.Target{
				Address: f.Address.URL(""), Username: c.Username, Password: c.Password, APIKey: c.APIKey, HTTP: f.Client, Folder: "/downloads/Movies",
				NodeFolder: filepath.Join(c.Dir, "Movies"), Interval: interval,
				Seeded: conformance.Seeded{Metainfo: cat.Releases[0].Metainfo, Hash: cat.Releases[0].InfoHash, Name: cat.Releases[0].Files[0].Path, Data: clientfaketest.Payload(t, cat.Releases[0])},
				Log:    slog.New(slog.NewTextHandler(t.Output(), nil)),
			}
		}
		t.Run(version+"/loopback", func(t *testing.T) {
			c.Dir = t.TempDir()
			cat, shared := clientfaketest.TrackedEvery(t, interval)
			f := faketest.Loopback(t, qbittorrent.Playing(c), scenario.Scenario{Seed: "qbittorrent"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
			conformance.Run(t, target(t, f, cat))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c.Dir = t.TempDir()
				cat, shared := clientfaketest.TrackedEvery(t, interval)
				f := faketest.InMemory(t, qbittorrent.Playing(c), scenario.Scenario{Seed: "qbittorrent"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
				conformance.Run(t, target(t, f, cat))
			})
		})
	}
}
