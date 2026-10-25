package node

import (
	"maps"
	"net"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/role"
)

// TestEveryProbeHasACheck holds the Node's checks to the Role table: each
// probe the table names has one, and none checks a probe it doesn't name.
func TestEveryProbeHasACheck(t *testing.T) {
	if got, want := slices.Sorted(maps.Keys(probeChecks)), slices.Sorted(slices.Values(role.Probes())); !slices.Equal(got, want) {
		t.Fatalf("the Node checks %v, the Role table names %v", got, want)
	}
}

// TestATakenTorrentPortFailsTheEngineProbeAndNotTheNode starts a Node whose
// torrent port something else holds: the Node runs, with no torrent
// listener, and its engine-starts probe fails.
func TestATakenTorrentPortFailsTheEngineProbeAndNotTheNode(t *testing.T) {
	taken, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = taken.Close() })
	env, _ := testEnv(t)
	env.Environ = append(env.Environ, "SLIPSTREAM_TORRENT_LISTEN="+taken.Addr().String())
	n := startTest(t, env)
	if got := n.Ready().Listeners.Torrent; got.IsValid() {
		t.Fatalf("on the taken %s the Node's torrent listener is %s, want none", taken.Addr(), got)
	}
	if fit, why := n.engineStarts(); fit != role.Fails || why != role.EngineNotListening {
		t.Fatalf("engine-starts = %s, %s; want %s, %s", fit, why, role.Fails, role.EngineNotListening)
	}
}
