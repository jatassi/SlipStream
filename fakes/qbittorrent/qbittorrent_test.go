package qbittorrent_test

import (
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/qbittorrent"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/qbittorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func TestFakeQBittorrentConforms(t *testing.T) {
	for _, version := range qbittorrent.Versions {
		c := qbittorrent.Client{Node: "Media server", Version: version, Username: "jackson", Password: "orchard", BanAfter: 5, Folder: "/downloads", Rate: 100, Check: time.Second}
		if version == "5.2" {
			c.APIKey = "qbt_0123456789abcdefghijklmnopqrstuv"
		}
		target := func(t *testing.T, f faketest.Served) conformance.Target {
			return conformance.Target{
				Address: f.Address.URL(""), Username: c.Username, Password: c.Password, APIKey: c.APIKey, HTTP: f.Client, Folder: "/downloads/Movies",
				Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
			}
		}
		t.Run(version+"/loopback", func(t *testing.T) {
			c.Dir = t.TempDir()
			f := faketest.Loopback(t, qbittorrent.Playing(c), scenario.Scenario{Seed: "qbittorrent"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(c.Node))
			conformance.Run(t, target(t, f))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c.Dir = t.TempDir()
				f := faketest.InMemory(t, qbittorrent.Playing(c), scenario.Scenario{Seed: "qbittorrent"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(c.Node))
				conformance.Run(t, target(t, f))
			})
		})
	}
}
