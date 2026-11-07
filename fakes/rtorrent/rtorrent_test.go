package rtorrent_test

import (
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/rtorrent"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func TestFakeRTorrentConforms(t *testing.T) {
	for _, version := range rtorrent.Versions {
		c := rtorrent.Client{Node: "Media server", Version: version, Username: "jackson", Password: "orchard", Folder: "/downloads", Rate: 100, Check: time.Second}
		target := func(t *testing.T, f faketest.Served) conformance.Target {
			return conformance.Target{
				Address: f.Address.URL(""), Username: c.Username, Password: c.Password, HTTP: f.Client, Folder: "/downloads/Movies",
				Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
			}
		}
		t.Run(version+"/loopback", func(t *testing.T) {
			c.Dir = t.TempDir()
			f := faketest.Loopback(t, rtorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(c.Node))
			conformance.Run(t, target(t, f))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c.Dir = t.TempDir()
				f := faketest.InMemory(t, rtorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(c.Node))
				conformance.Run(t, target(t, f))
			})
		})
	}
}
