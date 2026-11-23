package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
)

// api calls run's Nodes over the OS network under the tier's context.
func api(t *testing.T, run *world.Run) *worldtest.API {
	t.Helper()
	return worldtest.NewAPI(worldtest.Context(t), t, http.DefaultClient, run.Manifest())
}

func TestSingleMachine(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	m := run.Manifest()
	home := m.Nodes[0]
	a := api(t, run)
	if !regexp.MustCompile(`^ssk_[A-Za-z0-9_-]{43}$`).MatchString(m.APIKey) {
		t.Fatalf("apiKey = %q", m.APIKey)
	}

	sys := a.Admin("home", http.MethodGet, "/api/v1/system", "")
	var report struct {
		DataDir   string `json:"dataDir"`
		PID       int    `json:"pid"`
		Seed      string `json:"seed"`
		Listeners struct {
			HTTP string `json:"http"`
		} `json:"listeners"`
	}
	if err := json.Unmarshal([]byte(sys.Body), &report); err != nil || sys.Status != http.StatusOK {
		t.Fatalf("GET /api/v1/system with the API key = %d %s, want 200", sys.Status, sys.Body)
	}
	if report.DataDir != home.DataDir || report.PID != home.PID || report.Seed != "single-machine/home" || report.Listeners.HTTP != home.Listeners["http"] {
		t.Fatalf("getSystem = %s\nwant dataDir %s, pid %d, seed single-machine/home, listeners.http %s", sys.Body, home.DataDir, home.PID, home.Listeners["http"])
	}
	if addr := netip.MustParseAddrPort(report.Listeners.HTTP).Addr(); addr != netip.MustParseAddr("127.0.0.2") {
		t.Fatalf("listeners.http on %s, want 127.0.0.2", addr)
	}
	worldtest.Expect(t, "GET /api/v1/system without a key", a.Get("home", "/api/v1/system"), worldtest.Reply{Status: 401, Body: `{"status":401,"code":"signed_out","title":"Sign in to continue"}` + "\n"})
	worldtest.Expect(t, "GET /api/v1/status", worldtest.WithoutStartedAt(t, a.Get("home", "/api/v1/status")), worldtest.Reply{Status: 200, Body: `{"claim":"controller","passkeys":{"available":false,"reason":"ip_address"}}` + "\n"})

	rows := a.Nodes("home")
	every := []string{"searching", "downloading", "storage", "converting", "subtitles", "metadata", "media-server-link", "notifying", "serving"}
	if len(rows) != 1 || !rows[0].Self || rows[0].Link.State != "local" || !slices.Equal(rows[0].Assigned(), every) || rows[0].Facts == nil || !slices.Equal(rows[0].Facts.Assigned, every) {
		t.Fatalf("listNodes = %+v, want one row, self, local, holding and assigned %v", rows, every)
	}
	if home.NodeID != rows[0].ID || !slices.Equal(home.Roles, every) {
		t.Fatalf("the manifest has home as %s holding %v, want %s holding %v", home.NodeID, home.Roles, rows[0].ID, every)
	}

	want := node.Key{Key: "timings.node.stop_deadline", Value: "15s", Default: "30s", Source: "config.yaml", Env: "SLIPSTREAM_TIMINGS_NODE_STOP_DEADLINE"}
	if !slices.Contains(home.Overrides, want) {
		t.Fatalf("overrides = %+v, want %+v among them", home.Overrides, want)
	}
	if census := resolve(worldtest.Context(t), t, m.Binary.Path, home.DataDir).Census; len(census) != 0 {
		t.Fatalf("census = %v, want it empty", census)
	}
}

func resolve(ctx context.Context, t *testing.T, binary, dataDir string) node.Resolved {
	t.Helper()
	cmd := exec.CommandContext(ctx, binary, "config", "--data-dir", dataDir)
	cmd.Env = []string{"SystemRoot=" + os.Getenv("SystemRoot")}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("slipstream config: %v", err)
	}
	var r node.Resolved
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatal(err)
	}
	return r
}
