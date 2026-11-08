package transmission_test

import (
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/transmission"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/transmission"
)

func TestFakeTransmissionConforms(t *testing.T) {
	for _, version := range transmission.Versions {
		d := transmission.Daemon{Node: "Media server", Version: version, Username: "jackson", Password: "orchard", Folder: "/downloads", Rate: 100, Check: time.Second}
		target := func(t *testing.T, f faketest.Served) conformance.Target {
			return conformance.Target{
				Address: f.Address.URL(""), Username: d.Username, Password: d.Password, HTTP: f.Client, Folder: "/downloads/Movies",
				Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
			}
		}
		t.Run(version+"/loopback", func(t *testing.T) {
			d.Dir = t.TempDir()
			f := faketest.Loopback(t, transmission.Playing(d), scenario.Scenario{Seed: "transmission"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(d.Node))
			conformance.Run(t, target(t, f))
		})
		t.Run(version+"/in memory", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				d.Dir = t.TempDir()
				f := faketest.InMemory(t, transmission.Playing(d), scenario.Scenario{Seed: "transmission"}, faketest.WithReleases(faketest.Catalog(t, 1)), faketest.As(d.Node))
				conformance.Run(t, target(t, f))
			})
		})
	}
}
