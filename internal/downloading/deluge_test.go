package downloading_test

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"slices"
	"testing"

	fakedeluge "github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// TestADelugeTestThatNeedsADaemonAnswersTheDaemons tests a draft of a
// Deluge web UI that lists two daemons and names neither.
func TestADelugeTestThatNeedsADaemonAnswersTheDaemons(t *testing.T) {
	daemon := func(host string) fakedeluge.Daemon {
		return fakedeluge.Daemon{Host: host, Port: 58846, Dir: t.TempDir(), Folder: "/downloads", Rate: 100}
	}
	served := faketest.InMemory(t, fakedeluge.Playing(fakedeluge.Client{
		Node: clientNode, Version: "2.1", Password: clientPassword, Daemons: []fakedeluge.Daemon{daemon("127.0.0.1"), daemon("10.0.0.5")},
	}), scenario.Scenario{Seed: "deluge"}, faketest.As(clientNode))
	n := downloading.NewNode(nil, downloading.NodeConfig{
		HTTP: served.Client, Timings: timings, Log: slog.New(slog.DiscardHandler),
		Rand:   rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Folder: func(context.Context, string, bool) (storing.CheckResult, error) { return storing.CheckResult{}, nil },
	})
	res, err := n.TestClient(t.Context(), &dlv1.TestClientRequest{
		Client: &dlv1.Client{
			Id: storage.NewUUID().Bytes(), Kind: dlv1.Kind_KIND_DELUGE, Name: "Deluge",
			Address: served.Address.URL("/json"), Label: clients.DefaultLabel, SettingsVersion: 1,
		},
		Secrets: &dlv1.Secrets{Password: clientPassword},
	})
	if err != nil {
		t.Fatal(err)
	}
	failed := res.GetFailed()
	var listed []string
	for _, d := range failed.GetDaemons() {
		listed = append(listed, d.GetHost())
	}
	if failed.GetCause() != dlv1.Cause_CAUSE_DAEMON_NEEDED || !slices.Equal(listed, []string{"127.0.0.1", "10.0.0.5"}) {
		t.Fatalf("the Test answered %v, want it to need a daemon and list both", res)
	}
}
