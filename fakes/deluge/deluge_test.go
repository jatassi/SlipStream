package deluge_test

import (
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/deluge"
	"github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func TestFakeDelugeConforms(t *testing.T) {
	for _, version := range deluge.Versions {
		c := deluge.Client{Node: "Media server", Version: version, Password: "orchard", AutoConnect: 1}
		daemon := deluge.Daemon{Host: "127.0.0.1", Port: 58846, Folder: "/downloads", Rate: 100, Check: time.Second}
		target := func(t *testing.T, f faketest.Served) conformance.Target {
			return conformance.Target{
				Address: f.Address.URL(""), Password: c.Password, HTTP: f.Client, Folder: "/downloads/Movies",
				Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
			}
		}
		t.Run(version+"/loopback", func(t *testing.T) {
			daemon.Dir = t.TempDir()
			c.Daemons = []deluge.Daemon{daemon}
			f := faketest.Loopback(t, deluge.Playing(c), scenario.Scenario{Seed: "deluge"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(c.Node))
			conformance.Run(t, target(t, f))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				daemon.Dir = t.TempDir()
				c.Daemons = []deluge.Daemon{daemon}
				f := faketest.InMemory(t, deluge.Playing(c), scenario.Scenario{Seed: "deluge"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(c.Node))
				conformance.Run(t, target(t, f))
			})
		})
	}
}
