package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
)

// api calls run's Nodes over the OS network under the tier's context.
func api(t *testing.T, run *world.Run) *worldtest.API {
	t.Helper()
	return worldtest.NewAPI(worldtest.Context(t), t, http.DefaultClient, run.Manifest())
}

func TestFirstRun(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "first-run")
	m := run.Manifest()
	worldtest.SetupCode(t, run, "home")
	home := m.Nodes[0]
	if home.Setup != scenario.SetupNone || home.Claim != world.ClaimUnclaimed || m.Admin != nil || m.APIKey != "" {
		t.Fatalf("manifest: setup %s, claim %s, admin %+v, apiKey %q; want an Unclaimed Node and no Admin", home.Setup, home.Claim, m.Admin, m.APIKey)
	}
	worldtest.Expect(t, "GET /api/v1/status", api(t, run).Get("home", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"unclaimed"}` + "\n"})
	for name, want := range map[string]bool{"node.key": true, "library.db": false, "library.key": false, "local.db": false} {
		if _, err := os.Stat(filepath.Join(home.DataDir, name)); (err == nil) != want {
			t.Errorf("the Unclaimed data dir holds %s: %v, want %v", name, err == nil, want)
		}
	}
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
	worldtest.Expect(t, "GET /api/v1/status", a.Get("home", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"controller"}` + "\n"})

	rows := a.Nodes("home")
	every := []string{"searching", "downloading", "storage", "converting", "subtitles", "metadata", "media-server-link", "notifying", "serving"}
	if len(rows) != 1 || !rows[0].Self || rows[0].Link.State != "local" || !slices.Equal(rows[0].Roles, every) || rows[0].Facts == nil || !slices.Equal(rows[0].Facts.Assigned, every) {
		t.Fatalf("listNodes = %+v, want one row, self, local, holding and assigned %v", rows, every)
	}
	if home.NodeID != rows[0].ID || !slices.Equal(home.Roles, every) {
		t.Fatalf("the manifest has home as %s holding %v, want %s holding %v", home.NodeID, home.Roles, rows[0].ID, every)
	}

	want := node.Key{Key: "timings.node.stop_deadline", Value: "5s", Default: "30s", Source: "config.yaml", Env: "SLIPSTREAM_TIMINGS_NODE_STOP_DEADLINE"}
	if !slices.Contains(home.Overrides, want) {
		t.Fatalf("overrides = %+v, want %+v among them", home.Overrides, want)
	}
	if census := resolve(worldtest.Context(t), t, m.Binary.Path, home.DataDir).Census; len(census) != 0 {
		t.Fatalf("census = %v, want it empty", census)
	}
}

func TestHome(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	worldtest.CheckHome(t, api(t, run))
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

func TestTwoNodesOwnAddresses(t *testing.T) {
	t.Parallel()
	run := worldtest.LaunchScenario(t, scenario.Scenario{
		Name:  "two-nodes",
		Seed:  "two-nodes",
		Nodes: []scenario.Node{{Name: "a"}, {Name: "b"}},
	})
	type want struct{ name, addr, host string }
	for i, w := range []want{{"a", "127.0.0.2", "a.localhost"}, {"b", "127.0.0.3", "b.localhost"}} {
		n := run.Manifest().Nodes[i]
		if n.Name != w.name || n.Address != w.addr || n.Host != w.host {
			t.Fatalf("node %d = %s on %s at %s, want %s on %s at %s", i, n.Name, n.Address, n.Host, w.name, w.addr, w.host)
		}
		for _, listener := range []string{"http", "torrent"} {
			if addr := netip.MustParseAddrPort(n.Listeners[listener]).Addr().String(); addr != w.addr {
				t.Fatalf("node %s listener %s on %s, want %s", n.Name, listener, addr, w.addr)
			}
		}
		u, err := url.Parse(n.URL)
		if err != nil {
			t.Fatal(err)
		}
		if wantBrowser := "http://" + w.host + ":" + u.Port(); n.BrowserURL != wantBrowser {
			t.Fatalf("node %s browserUrl = %s, want %s", n.Name, n.BrowserURL, wantBrowser)
		}
	}
}
